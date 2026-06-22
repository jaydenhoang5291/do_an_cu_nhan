import math, os
import numpy as np
import matplotlib

matplotlib.use("TkAgg")  # or 'Qt5Agg', 'Qt6Agg', 'wxAgg'
import matplotlib.pyplot as plt
from matplotlib.patches import Rectangle
from matplotlib.widgets import Button
import pandas as pd
from dataclasses import dataclass


@dataclass
class Hex:
    q: int
    r: int
    s: int

    def __post_init__(self):
        assert self.q + self.r + self.s == 0


def axial_to_pixel(q: int, r: int, size: float = 500.0) -> tuple[float, float]:
    x = size * (3 / 2 * q)
    y = size * (math.sqrt(3) * r + (math.sqrt(3) / 2) * q)
    return x, y


class CellularNetworkReceivedPower:
    """
    Phiên bản tối giản theo yêu cầu:
    - KHÔNG dùng ảnh bản đồ / mask đen-trắng.
    - KHÔNG xác định LOS/NLOS theo ngưỡng: BS mặt đất coi như NLOS, còn BS UAV luôn LOS (như bản gốc).
    - Tạo lưới grid biểu thị đường; UE di chuyển kiểu Manhattan trên lưới:
        + Chỉ rẽ khi tới giao điểm (intersection).
        + Ở intersection có thể rẽ theo bất kỳ hướng hợp lệ (E/W/N/S).
    """

    def __init__(
        self,
        num_ues: int = 5,
        rect_len_m: float = 4000.0,
        rect_wid_m: float = 3000.0,
        grid_spacing_m: float = 200.0,
        add_uav_cover: bool = False,
        uav_radius_m: float = 750.0,
        uav_altitude_m: float = 100.0,
        uav_ptx_dbm: float = 40.0,
        fast_mode: bool = True,
        show_link_lines: bool = True,
        seed: int | None = None,
    ):
        if seed is not None:
            np.random.seed(int(seed))

        # Core sim canvas
        self.total_size = 2500.0
        self.center = self.total_size / 2.0

        # Region (HCN)
        self.width, self.height = float(rect_len_m), float(rect_wid_m)
        x_offset_m = -500.0
        self.rect_xmin = self.center - self.width / 2.0 + x_offset_m
        self.rect_xmax = self.rect_xmin + self.width
        self.rect_ymin = self.center - self.height / 2.0
        self.rect_ymax = self.center + self.height / 2.0 - 100

        # Road grid
        self.grid_spacing_m = float(max(10.0, grid_spacing_m))
        self.x_lines = self._make_lines(
            self.rect_xmin, self.rect_xmax, self.grid_spacing_m
        )
        self.y_lines = self._make_lines(
            self.rect_ymin, self.rect_ymax, self.grid_spacing_m
        )

        # UAV deployment (optional)
        self.add_uav_cover = bool(add_uav_cover)
        self.uav_radius_m = float(uav_radius_m)
        self.uav_altitude_m = float(uav_altitude_m)
        self.uav_ptx_dbm = float(uav_ptx_dbm)

        # Radio params
        self.ptx, self.gtx, self.grx = 35.0, 2.0, 2.0
        self.sensitivity = -110.0
        self.fc = 4.0  # GHz
        self.hom = 3.0

        self.max_bs_range = 600.0
        self.max_candidate_bs = 6

        self.h_bs, self.h_ut = 25.0, 1.5

        # Ground BS: cố định NLOS
        self.ple_nlos = 4.1
        self.sf_sigma_nlos = 10.0

        # UAV BS: luôn LOS (giữ như bản gốc)
        self.ple_uav_los = 2.8
        self.sf_sigma_uav = 14.0
        self.sf_decorr = 20.0
        self.sf_cache = {}

        # UE
        self.num_ues = int(num_ues)
        self.ue_colors = [
            "green",
            "purple",
            "orange",
            "cyan",
            "magenta",
            "yellow",
            "black",
        ]
        self.ue_speeds = np.random.uniform(20.0, 60.0, self.num_ues)  # km/h
        self.ue_speeds_ms = self.ue_speeds * (1000.0 / 3600.0)

        self.steps = 100
        self.time_per_step = 3.0
        self.pause_s = 0.03 if fast_mode else 0.08
        self.show_link_lines = bool(show_link_lines)

        # BS hex layout
        self.scale_factor = 500.0
        self.hexes = self._build_hex_cover()
        self.bs_positions, self.bs_heights, self.bs_ptx = [], [], []
        self.bs_is_uav = []

        # UE state
        self.ue_positions = []
        self.ue_dir = []  # 0:E,1:N,2:W,3:S
        self.ue_serving_bs = [None] * self.num_ues
        self.previous_serving_bs = [None] * self.num_ues

        # UI + plot
        plt.ion()
        self.fig = plt.figure(figsize=(15, 10))
        self.ax = self.fig.add_axes([0.08, 0.1, 0.84, 0.82])
        self.toggle_ax = self.fig.add_axes([0.08, 0.02, 0.12, 0.05])
        self.restart_ax = self.fig.add_axes([0.22, 0.02, 0.14, 0.05])
        self.toggle_button = Button(self.toggle_ax, "Pause", color="lightcoral")
        self.restart_button = Button(
            self.restart_ax, "Restart", color="lightgreen"
        )
        self.toggle_button.on_clicked(self.toggle_animation)
        self.restart_button.on_clicked(self.restart_animation)
        self.animation_running = True

        # Logging
        self.data_log = {"Bước": []}
        for i in range(self.num_ues):
            for key in [
                "x",
                "y",
                "huong",
                "BS_ketnoi",
                "prx_hientai",
                "SINR",
                "PacketLoss",
                "speed",
                "handover",
            ]:
                self.data_log[f"ue{i}_{key}"] = []

        self.setup_plot()
        self.setup_ues()

    # ---------- Grid helpers ----------
    @staticmethod
    def _make_lines(a: float, b: float, step: float) -> np.ndarray:
        lines = np.arange(a, b + 1e-9, step, dtype=float)
        if abs(lines[-1] - b) > 1e-6:
            lines = np.append(lines, b)
        if abs(lines[0] - a) > 1e-6:
            lines = np.insert(lines, 0, a)
        lines = np.unique(np.round(lines, 6))  # ổn định số học
        lines.sort()
        return lines

    def _allowed_dirs_at_intersection(self, x: float, y: float) -> list[int]:
        # E/W based on x_lines
        allowed = []
        ix_r = np.searchsorted(self.x_lines, x, side="right")
        ix_l = np.searchsorted(self.x_lines, x, side="left") - 1
        iy_u = np.searchsorted(self.y_lines, y, side="right")
        iy_d = np.searchsorted(self.y_lines, y, side="left") - 1
        if ix_r < len(self.x_lines):
            allowed.append(0)  # E
        if ix_l >= 0:
            allowed.append(2)  # W
        if iy_u < len(self.y_lines):
            allowed.append(1)  # N
        if iy_d >= 0:
            allowed.append(3)  # S
        # Loại bỏ hướng "đi ra ngoài" khi đang ở biên thật sự
        allowed2 = []
        for d in allowed:
            if d == 0 and x >= self.x_lines[-1] - 1e-6:
                continue
            if d == 2 and x <= self.x_lines[0] + 1e-6:
                continue
            if d == 1 and y >= self.y_lines[-1] - 1e-6:
                continue
            if d == 3 and y <= self.y_lines[0] + 1e-6:
                continue
            allowed2.append(d)
        return allowed2

    def _step_manhattan_on_grid(self, ue_idx: int, dist_m: float):
        """Di chuyển UE dọc theo đường (grid line), chỉ đổi hướng tại intersection."""
        x, y = self.ue_positions[ue_idx]
        d = float(dist_m)
        eps = 1e-6

        while d > eps:
            dir = int(self.ue_dir[ue_idx])

            if dir == 0:  # East
                j = np.searchsorted(self.x_lines, x, side="right")
                if j >= len(self.x_lines):  # tại biên, phải rẽ
                    self._turn_at_intersection(ue_idx)
                    continue
                target = float(self.x_lines[j])
                to_next = target - x
                if to_next <= eps:
                    x = target
                    self._turn_at_intersection(ue_idx)
                    continue
                if d < to_next:
                    x += d
                    d = 0.0
                else:
                    x = target
                    d -= to_next
                    self._turn_at_intersection(ue_idx)

            elif dir == 2:  # West
                j = np.searchsorted(self.x_lines, x, side="left") - 1
                if j < 0:
                    self._turn_at_intersection(ue_idx)
                    continue
                target = float(self.x_lines[j])
                to_next = x - target
                if to_next <= eps:
                    x = target
                    self._turn_at_intersection(ue_idx)
                    continue
                if d < to_next:
                    x -= d
                    d = 0.0
                else:
                    x = target
                    d -= to_next
                    self._turn_at_intersection(ue_idx)

            elif dir == 1:  # North
                j = np.searchsorted(self.y_lines, y, side="right")
                if j >= len(self.y_lines):
                    self._turn_at_intersection(ue_idx)
                    continue
                target = float(self.y_lines[j])
                to_next = target - y
                if to_next <= eps:
                    y = target
                    self._turn_at_intersection(ue_idx)
                    continue
                if d < to_next:
                    y += d
                    d = 0.0
                else:
                    y = target
                    d -= to_next
                    self._turn_at_intersection(ue_idx)

            else:  # South (3)
                j = np.searchsorted(self.y_lines, y, side="left") - 1
                if j < 0:
                    self._turn_at_intersection(ue_idx)
                    continue
                target = float(self.y_lines[j])
                to_next = y - target
                if to_next <= eps:
                    y = target
                    self._turn_at_intersection(ue_idx)
                    continue
                if d < to_next:
                    y -= d
                    d = 0.0
                else:
                    y = target
                    d -= to_next
                    self._turn_at_intersection(ue_idx)

        # Clamp (an toàn số học)
        x = float(np.clip(x, self.rect_xmin, self.rect_xmax))
        y = float(np.clip(y, self.rect_ymin, self.rect_ymax))
        self.ue_positions[ue_idx] = [x, y]

    def _turn_at_intersection(self, ue_idx: int):
        """Chỉ được gọi khi UE ở intersection (hoặc sát intersection do epsilon)."""
        x, y = self.ue_positions[ue_idx]
        allowed = self._allowed_dirs_at_intersection(x, y)
        if not allowed:
            return
        self.ue_dir[ue_idx] = int(np.random.choice(allowed))

    # ---------- BS layout ----------
    def _build_hex_cover(self):
        size = self.scale_factor
        w, h = self.width, self.height
        q_range = range(int(-w / (1.5 * size)) - 1, int(w / (1.5 * size)) + 2)
        r_range = range(
            int(-h / (size * math.sqrt(3))) - 1, int(h / (size * math.sqrt(3))) + 2
        )
        hexes = []
        for q in q_range:
            for r in r_range:
                s = -q - r
                cx, cy = axial_to_pixel(q, r, size)
                cx += self.center
                cy += self.center
                buffer = size * math.sqrt(3) / 2
                if (
                    self.rect_xmin - buffer <= cx <= self.rect_xmax + buffer
                    and self.rect_ymin - buffer <= cy <= self.rect_ymax + buffer
                ):
                    hexes.append(Hex(q, r, s))
        return hexes

    def setup_plot(self):
        # Roads (grid)
        for x in self.x_lines:
            self.ax.plot(
                [x, x],
                [self.rect_ymin, self.rect_ymax],
                "gray",
                linestyle="-",
                alpha=0.35,
                linewidth=1.0,
                zorder=1,
            )
        for y in self.y_lines:
            self.ax.plot(
                [self.rect_xmin, self.rect_xmax],
                [y, y],
                "gray",
                linestyle="-",
                alpha=0.35,
                linewidth=1.0,
                zorder=1,
            )

        # HCN boundary
        self.ax.add_patch(
            Rectangle(
                (self.rect_xmin, self.rect_ymin),
                self.width,
                self.height,
                linewidth=2,
                edgecolor="red",
                facecolor="none",
                linestyle="--",
                zorder=5,
            )
        )

        # BS from hex grid
        self.bs_positions.clear()
        self.bs_heights.clear()
        self.bs_ptx.clear()
        bs_colors = [
            "red",
            "green",
            "blue",
            "cyan",
            "magenta",
            "yellow",
            "black",
            "orange",
            "purple",
            "brown",
        ]
        for idx, hex_obj in enumerate(self.hexes):
            x, y = axial_to_pixel(hex_obj.q, hex_obj.r, self.scale_factor)
            cx, cy = x + self.center, y + self.center
            self.bs_positions.append((cx, cy))
            self.bs_heights.append(self.h_bs)
            self.bs_ptx.append(self.ptx)
            self.bs_is_uav.append(False)
            self.ax.plot(
                cx,
                cy,
                marker="^",
                color=bs_colors[idx % len(bs_colors)],
                markersize=10,
                zorder=6,
            )

            # viền hex (đứt)
            size = self.scale_factor
            angles = [60 * i for i in range(7)]
            xs = [cx + size * math.cos(math.radians(a)) for a in angles]
            ys = [cy + size * math.sin(math.radians(a)) for a in angles]
            self.ax.plot(
                xs,
                ys,
                color="black",
                linestyle="--",
                linewidth=1.0,
                alpha=0.6,
                zorder=2,
            )

        # --- UAV BS cover (optional): marker 'v' đỏ ---
        if self.add_uav_cover:
            R = float(self.uav_radius_m)
            dx = math.sqrt(3.0) * R
            dy = 1.5 * R
            y = self.rect_ymin + R
            row = 0
            while y <= self.rect_ymax - R:
                x = self.rect_xmin + R + (dx / 2.0 if (row % 2 == 1) else 0.0)
                while x <= self.rect_xmax - R:
                    self.bs_positions.append((x, y))
                    self.bs_heights.append(float(self.uav_altitude_m))
                    self.bs_ptx.append(float(self.uav_ptx_dbm))
                    self.bs_is_uav.append(True)
                    self.ax.plot(
                        x, y, marker="v", color="tab:red", markersize=10, zorder=7
                    )
                    x += dx
                y += dy
                row += 1

        # UE artists
        self.ue_points, self.ue_lines = [], []
        for i in range(self.num_ues):
            color = self.ue_colors[i % len(self.ue_colors)]
            (p,) = self.ax.plot([], [], "o", color=color, markersize=10, zorder=7)
            self.ue_points.append(p)
            (l,) = self.ax.plot(
                [],
                [],
                "-",
                color=color,
                linewidth=0.8,
                alpha=(0.6 if self.show_link_lines else 0.0),
                zorder=6,
            )
            self.ue_lines.append(l)

        self.time_text = self.ax.text(
            self.rect_xmin + 20, self.rect_ymax + 20, "", fontsize=10, zorder=8
        )

        info_text = (
            "Network Layout:\n"
            "- Hexagonal grid\n"
            "- Cell radius: 750 m\n\n"
            "BS Configuration:\n"
            "- BS height: 25 m\n\n"
            "UE Configuration:\n"
            "- Mobility model: Manhattan"
        )
        self.ax.text(
            0.98,
            0.98,
            info_text,
            transform=self.ax.transAxes,
            va="top",
            ha="right",
            fontsize=14,
            color="black",
            bbox=dict(
                facecolor="white",
                edgecolor="lightgray",
                alpha=0.9,
                boxstyle="round,pad=0.5",
            ),
            zorder=9,
        )

        self.ax.set_xlim(0, self.total_size)
        self.ax.set_ylim(0, self.total_size)
        self.ax.set_xlabel("Distance (m)")
        self.ax.set_ylabel("Distance (m)")
        # self.ax.set_title(
        #     f"Grid road (Manhattan) | TẤT CẢ link = NLOS | {int(self.width)}x{int(self.height)} m | grid_spacing={int(self.grid_spacing_m)} m"
        # )
        plt.draw()

    def setup_ues(self):
        self.ue_positions.clear()
        self.ue_dir.clear()

        # Spawn tại giao điểm ngẫu nhiên
        xs = self.x_lines
        ys = self.y_lines
        for _ in range(self.num_ues):
            x = float(np.random.choice(xs))
            y = float(np.random.choice(ys))
            self.ue_positions.append([x, y])
            # hướng ban đầu ngẫu nhiên nhưng phải hợp lệ tại intersection
            allowed = self._allowed_dirs_at_intersection(x, y)
            self.ue_dir.append(
                int(np.random.choice(allowed))
                if allowed
                else int(np.random.randint(0, 4))
            )

    # ---------- Radio models ----------
    def calculate_distance_idx(self, ue_x, ue_y, bs_idx: int):
        bs_x, bs_y = self.bs_positions[bs_idx]
        d2 = float(np.hypot(ue_x - bs_x, ue_y - bs_y))
        dz = float(self.bs_heights[bs_idx] - self.h_ut)
        return max(float(np.hypot(d2, dz)), 1.0)

    def calculate_path_loss_idx(self, ue_x, ue_y, bs_idx: int):
        """Path loss (dB).
        - BS mặt đất: cố định NLOS.
        - BS UAV: luôn LOS (giữ như bản gốc).
        """
        d3 = self.calculate_distance_idx(ue_x, ue_y, bs_idx)
        fspl_1m = 32.4 + 20.0 * np.log10(self.fc)  # fc (GHz)
        if self.bs_is_uav[bs_idx]:
            n = self.ple_uav_los
            los = True
        else:
            n = self.ple_nlos
            los = False
        pl = fspl_1m + 10.0 * n * np.log10(d3)
        return float(pl), los

    def _shadow_fading(self, ue_idx, bs_idx, ue_pos):
        sigma = self.sf_sigma_uav if self.bs_is_uav[bs_idx] else self.sf_sigma_nlos
        key = (ue_idx, bs_idx)
        state = self.sf_cache.get(key)
        if state is None:
            val = float(np.random.normal(0.0, sigma))
            val = float(np.clip(val, -3 * sigma, 3 * sigma))
            self.sf_cache[key] = {"x": ue_pos[0], "y": ue_pos[1], "val": val}
            return val

        oldx, oldy = state["x"], state["y"]
        oldval = state["val"]
        delta = float(np.hypot(ue_pos[0] - oldx, ue_pos[1] - oldy))
        rho = 0.0 if self.sf_decorr <= 0 else float(np.exp(-delta / self.sf_decorr))
        innov = float(np.random.normal(0.0, sigma))
        newval = rho * oldval + (np.sqrt(max(0.0, 1.0 - rho**2)) * innov)
        newval = float(np.clip(newval, -3 * sigma, 3 * sigma))
        self.sf_cache[key] = {"x": ue_pos[0], "y": ue_pos[1], "val": newval}
        return newval

    def calculate_received_power_idx(self, bs_idx: int, path_loss_db: float):
        ptx = float(self.bs_ptx[bs_idx])
        return ptx + self.gtx + self.grx - float(path_loss_db)

    def get_serving_bs(self, ue_x, ue_y, ue_idx):
        # Find candidate BS by distance
        dist_list = []
        for i, (bs_x, bs_y) in enumerate(self.bs_positions):
            if self.max_bs_range is not None:
                d2 = float(np.hypot(ue_x - bs_x, ue_y - bs_y))
                if d2 > self.max_bs_range:
                    continue
            d3 = self.calculate_distance_idx(ue_x, ue_y, i)
            dist_list.append((i, d3))

        if not dist_list:
            self.ue_serving_bs[ue_idx] = None
            return None, None, None

        dist_list.sort(key=lambda x: x[1])
        candidate_indices = [i for i, _ in dist_list[: self.max_candidate_bs]]

        received_powers = []
        for i in candidate_indices:
            pl, _los = self.calculate_path_loss_idx(ue_x, ue_y, i)
            rsrp_inst = self.calculate_received_power_idx(i, pl)
            if rsrp_inst >= self.sensitivity:
                d3 = self.calculate_distance_idx(ue_x, ue_y, i)
                received_powers.append((i, rsrp_inst, d3, pl))

        if not received_powers:
            self.ue_serving_bs[ue_idx] = None
            return None, None, None

        received_powers.sort(key=lambda x: x[1], reverse=True)
        current_bs = self.ue_serving_bs[ue_idx]

        if current_bs is None:
            best_bs, best_prx, best_d, _ = received_powers[0]
            self.ue_serving_bs[ue_idx] = best_bs
            return best_bs, best_prx, best_d

        current_info = next(
            (info for info in received_powers if info[0] == current_bs), None
        )
        if current_info is None:
            best_bs, best_prx, best_d, _ = received_powers[0]
            self.ue_serving_bs[ue_idx] = best_bs
            return best_bs, best_prx, best_d

        current_prx = current_info[1]
        current_dist = current_info[2]

        for bs_idx, bs_prx, bs_dist, _ in received_powers:
            if bs_idx == current_bs:
                continue
            if bs_prx > current_prx + self.hom:
                self.ue_serving_bs[ue_idx] = bs_idx
                return bs_idx, bs_prx, bs_dist

        return current_bs, current_prx, current_dist

    def calculate_sinr_and_packet_loss(self, ue_idx, serving_bs_idx, prx_dbm):
        N_dbm = -100.0
        N_mw = 10 ** (N_dbm / 10.0)
        interf = 0.0
        ue_x, ue_y = self.ue_positions[ue_idx]
        for i, (bs_x, bs_y) in enumerate(self.bs_positions):
            if i == serving_bs_idx:
                continue
            pl, _ = self.calculate_path_loss_idx(ue_x, ue_y, i)
            prx_i = self.calculate_received_power_idx(i, pl)
            if prx_i >= self.sensitivity:
                interf += 10 ** (prx_i / 10.0)
        prx_mw = 10 ** (prx_dbm / 10.0)
        sinr_lin = prx_mw / (interf + N_mw)
        sinr_db = 10.0 * np.log10(sinr_lin) if sinr_lin > 0 else -np.inf
        k = 0.1
        ploss = np.exp(-k * sinr_db) if sinr_db > 0 else 1.0
        return float(sinr_db), float(min(max(ploss, 0.0), 1.0))

    # ---------- Animation ----------
    def toggle_animation(self, event):
        self.animation_running = not getattr(self, "animation_running", False)
        if self.animation_running:
            self.toggle_button.label.set_text("Pause")
            self.run_animation()
        else:
            self.toggle_button.label.set_text("Resume")
        self.toggle_button.ax.figure.canvas.draw()

    def restart_animation(self, event):
        self.animation_running = True
        self.current_frame = 0
        self.ue_serving_bs = [None] * self.num_ues
        self.previous_serving_bs = [None] * self.num_ues
        self.sf_cache = {}

        # reset logging
        self.data_log = {"Bước": []}
        for i in range(self.num_ues):
            for key in [
                "x",
                "y",
                "huong",
                "BS_ketnoi",
                "prx_hientai",
                "SINR",
                "PacketLoss",
                "speed",
                "handover",
            ]:
                self.data_log[f"ue{i}_{key}"] = []

        self.setup_ues()
        self.toggle_button.label.set_text("Pause")
        self.toggle_button.color = "lightcoral"
        self.toggle_button.ax.figure.canvas.draw()
        self.run_animation()

    def update(self, frame: int):
        if not getattr(self, "animation_running", True):
            return

        self.current_frame = frame
        self.data_log["Bước"].append(frame)

        for ue_idx in range(self.num_ues):
            d = float(self.ue_speeds_ms[ue_idx] * self.time_per_step)

            # Manhattan on grid
            self._step_manhattan_on_grid(ue_idx, d)

            # Draw UE
            x, y = self.ue_positions[ue_idx]
            self.ue_points[ue_idx].set_data([x], [y])

            # Serving BS
            bs, prx_avg, dist = self.get_serving_bs(x, y, ue_idx)

            # Link line
            if bs is not None and self.show_link_lines:
                self.ue_lines[ue_idx].set_data(
                    [x, self.bs_positions[bs][0]], [y, self.bs_positions[bs][1]]
                )
            else:
                self.ue_lines[ue_idx].set_data([], [])

            # Metrics / logging
            if bs is not None:
                pl_base, _ = self.calculate_path_loss_idx(x, y, bs)
                sf_db = self._shadow_fading(ue_idx, bs, (x, y))
                prx_inst = self.calculate_received_power_idx(bs, pl_base + sf_db)
                sinr, pkt = self.calculate_sinr_and_packet_loss(ue_idx, bs, prx_inst)
            else:
                prx_inst = None
                sinr = None
                pkt = None

            self.data_log[f"ue{ue_idx}_x"].append(x)
            self.data_log[f"ue{ue_idx}_y"].append(y)
            self.data_log[f"ue{ue_idx}_huong"].append(int(self.ue_dir[ue_idx]))
            self.data_log[f"ue{ue_idx}_BS_ketnoi"].append(bs)
            self.data_log[f"ue{ue_idx}_prx_hientai"].append(prx_inst)
            self.data_log[f"ue{ue_idx}_SINR"].append(sinr)
            self.data_log[f"ue{ue_idx}_PacketLoss"].append(pkt)
            self.data_log[f"ue{ue_idx}_speed"].append(float(self.ue_speeds[ue_idx]))

            prev_bs = self.previous_serving_bs[ue_idx]
            handover_flag = (
                1 if (prev_bs is not None and bs is not None and bs != prev_bs) else 0
            )
            self.previous_serving_bs[ue_idx] = bs
            self.data_log[f"ue{ue_idx}_handover"].append(handover_flag)

        self.time_text.set_text(f"Step: {frame}")
        self.fig.canvas.draw()
        self.fig.canvas.flush_events()

        if frame == self.steps:
            try:
                self.save_data_to_csv()
            except Exception as e:
                print("CSV save failed:", e)

    def save_data_to_csv(self):
        max_len = max((len(v) for v in self.data_log.values()), default=0)
        for k in self.data_log:
            while len(self.data_log[k]) < max_len:
                self.data_log[k].append(None)

        df = pd.DataFrame(self.data_log)

        # Làm tròn các cột số
        for col in df.columns:
            if any(tag in col for tag in ["prx", "_x", "_y", "SINR"]):
                df[col] = pd.to_numeric(df[col], errors="coerce").round(2)

        filename = "du_lieu_cong_suat_nhan_GRID_ALL_NLOS.csv"
        df.to_csv(
            filename,
            index=False,
            sep=";",
            decimal=",",
            encoding="utf-8-sig",
            float_format="%.2f",
        )
        print(f"Đã lưu CSV: {filename}")

    def run_animation(self):
        self.animation_running = True
        for frame in range(0, self.steps + 1):
            if not self.animation_running:
                break
            self.update(frame)
            plt.pause(self.pause_s)
        try:
            self.save_data_to_csv()
        except Exception as e:
            print("CSV save failed:", e)


def _read_float(prompt: str, default: float) -> float:
    try:
        s = input(prompt).strip()
        return float(s) if s else float(default)
    except Exception:
        return float(default)


def _read_int(prompt: str, default: int) -> int:
    try:
        s = input(prompt).strip()
        return int(s) if s else int(default)
    except Exception:
        return int(default)


if __name__ == "__main__":
    # L = _read_float("Nhập chiều DÀI HCN (m): ", 8000.0)
    # W = _read_float("Nhập chiều RỘNG HCN (m): ", 5000.0)
    # num_ues = _read_int("Nhập số UE: ", 5)
    # grid_sp = _read_float("Nhập khoảng cách lưới đường (m, mặc định 200): ", 200.0)
    #
    # add_uav = input("Thêm UAV-BS phủ (UAV luôn LOS)? (y/N): ").strip().lower().startswith('y')
    # uav_R = _read_float("  UAV radius (m, mặc định 750): ", 750.0) if add_uav else 750.0
    # uav_h = _read_float("  UAV altitude (m, mặc định 100): ", 100.0) if add_uav else 100.0
    # uav_ptx = _read_float("  UAV Ptx (dBm, mặc định 40): ", 40.0) if add_uav else 40.0

    # show_lines = input("Hiện đường UE↔BS? (y/N): ").strip().lower().startswith("y")

    L = 1500.0
    W = 1800.0
    num_ues = 1
    grid_sp = 200.0
    add_uav = True
    uav_R = 350.0
    uav_h = 100.0
    uav_ptx = 40.0
    show_lines = True

    sim = CellularNetworkReceivedPower(
        num_ues=num_ues,
        rect_len_m=L,
        rect_wid_m=W,
        grid_spacing_m=grid_sp,
        add_uav_cover=add_uav,
        uav_radius_m=uav_R,
        uav_altitude_m=uav_h,
        uav_ptx_dbm=uav_ptx,
        show_link_lines=show_lines,
        fast_mode=True,
    )
    sim.run_animation()
    plt.ioff()
    plt.show()
