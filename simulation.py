import math
import numpy as np
import matplotlib.pyplot as plt
from matplotlib.patches import Rectangle
from matplotlib.widgets import Button

from hexagons import axial_to_pixel, build_hex_cover
import config

from mobility import GridMobility
from radio_models import RadioModel
from logger import SimulationLogger

class CellularNetworkReceivedPower:
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
        self.total_size = 10000.0
        self.center = self.total_size / 2.0

        # Region (HCN)
        self.width, self.height = float(rect_len_m), float(rect_wid_m)
        self.rect_xmin = self.center - self.width / 2.0
        self.rect_xmax = self.center + self.width / 2.0
        self.rect_ymin = self.center - self.height / 2.0
        self.rect_ymax = self.center + self.height / 2.0

        self.mobility = GridMobility(self)
        self.radio_model = RadioModel(self)
        self.logger = SimulationLogger(self)

        # Road grid
        self.grid_spacing_m = float(max(10.0, grid_spacing_m))
        self.x_lines = GridMobility._make_lines(self.rect_xmin, self.rect_xmax, self.grid_spacing_m)
        self.y_lines = GridMobility._make_lines(self.rect_ymin, self.rect_ymax, self.grid_spacing_m)

        # UAV deployment (optional)
        self.add_uav_cover = bool(add_uav_cover)
        self.uav_radius_m = float(uav_radius_m)
        self.uav_altitude_m = float(uav_altitude_m)
        self.uav_ptx_dbm = float(uav_ptx_dbm)

        # Radio params
        self.ptx, self.gtx, self.grx = config.PTX, config.GTX, config.GRX
        self.sensitivity = config.SENSITIVITY
        self.fc = config.FC
        self.hom = config.HOM

        self.max_bs_range = config.MAX_BS_RANGE
        self.max_candidate_bs = config.MAX_CANDIDATE_BS

        self.h_bs, self.h_ut = config.H_BS, config.H_UT

        # Ground BS: cố định NLOS
        self.ple_nlos = config.PLE_NLOS

        # UAV BS: luôn LOS (giữ như bản gốc)
        self.ple_uav_los = config.PLE_UAV_LOS

        # Shadow fading 
        self.sf_sigma = config.SF_SIGMA
        self.sf_sigma_uav = config.SF_SIGMA_UAV
        self.sf_decorr = float(max(20.0, 0.25 * self.grid_spacing_m))
        self.sf_turn_penalty_m = float(0.50 * self.grid_spacing_m)
        self.sf_cache = {}

        # UE
        self.num_ues = int(num_ues)
        self.ue_colors = ['green', 'purple', 'orange', 'cyan', 'magenta', 'yellow', 'black']
        self.ue_speeds = np.random.uniform(20.0, 60.0, self.num_ues)  # km/h
        self.ue_speeds_ms = self.ue_speeds * (1000.0 / 3600.0)

        self.steps = 100
        self.time_per_step = 3.0
        self.pause_s = (0.03 if fast_mode else 0.08)
        self.show_link_lines = bool(show_link_lines)

        # BS hex layout
        self.scale_factor = 500.0
        self.hexes = build_hex_cover(self.scale_factor, self.width, self.height, self.center, self.rect_xmin, self.rect_xmax, self.rect_ymin, self.rect_ymax)
        self.bs_positions, self.bs_heights, self.bs_ptx = [], [], []
        self.bs_is_uav = []

        # UE state
        self.ue_positions = []
        self.ue_dir = []  # 0:E,1:N,2:W,3:S
        self.ue_serving_bs = [None] * self.num_ues
        self.previous_serving_bs = [None] * self.num_ues

        # UE stop-and-go after turns:
        self.turns_before_stop = config.TURNS_BEFORE_STOP
        self.stop_duration_steps = config.STOP_DURATION_STEPS
        self.ramp_duration_steps = config.RAMP_DURATION_STEPS

        self.ue_turn_count = [0] * self.num_ues
        self.ue_pause_remaining = [0] * self.num_ues
        self.ue_turns_in_step = [0] * self.num_ues
        self.ue_ramp_step = [-1] * self.num_ues
        self.ue_speed_factor = np.ones(self.num_ues, dtype=float)

        # UI + plot
        plt.ion()
        self.fig = plt.figure(figsize=(10, 10))
        self.ax = self.fig.add_axes([0.08, 0.1, 0.84, 0.82])
        self.toggle_ax = self.fig.add_axes([0.08, 0.02, 0.12, 0.05])
        self.restart_ax = self.fig.add_axes([0.22, 0.02, 0.14, 0.05])
        self.toggle_button = Button(self.toggle_ax, 'Stop', color='lightcoral')
        self.restart_button = Button(self.restart_ax, 'Restart', color='lightgreen')
        self.toggle_button.on_clicked(self.toggle_animation)
        self.restart_button.on_clicked(self.restart_animation)
        self.animation_running = True

        # Logging
        self.logger.setup_log()

        self.setup_plot()
        self.setup_ues()

    def setup_plot(self):
        # Roads (grid)
        for x in self.x_lines:
            self.ax.plot([x, x], [self.rect_ymin, self.rect_ymax], 'gray', linestyle='-', alpha=0.35, linewidth=1.0, zorder=1)
        for y in self.y_lines:
            self.ax.plot([self.rect_xmin, self.rect_xmax], [y, y], 'gray', linestyle='-', alpha=0.35, linewidth=1.0, zorder=1)

        # HCN boundary
        self.ax.add_patch(Rectangle((self.rect_xmin, self.rect_ymin), self.width, self.height,
                                    linewidth=2, edgecolor='red', facecolor='none',
                                    linestyle='--', zorder=5))

        # BS from hex grid
        self.bs_positions.clear()
        self.bs_heights.clear()
        self.bs_ptx.clear()
        self.bs_is_uav.clear()
        bs_colors = ['red', 'green', 'blue', 'cyan', 'magenta', 'yellow', 'black', 'orange', 'purple', 'brown']
        for idx, hex_obj in enumerate(self.hexes):
            x, y = axial_to_pixel(hex_obj.q, hex_obj.r, self.scale_factor)
            cx, cy = x + self.center, y + self.center
            self.bs_positions.append((cx, cy))
            self.bs_heights.append(self.h_bs)
            self.bs_ptx.append(self.ptx)
            self.bs_is_uav.append(False)
            self.ax.plot(cx, cy, marker='^', color=bs_colors[idx % len(bs_colors)],
                         markersize=5, zorder=6)

            # viền hex (đứt)
            size = self.scale_factor
            angles = [60 * i for i in range(7)]
            xs = [cx + size * math.cos(math.radians(a)) for a in angles]
            ys = [cy + size * math.sin(math.radians(a)) for a in angles]
            self.ax.plot(xs, ys, color='black', linestyle='--', linewidth=1.0, alpha=0.6, zorder=2)


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
                    self.ax.plot(x, y, marker='v', color='tab:red', markersize=6, zorder=7)

                    # Viền vùng phủ UAV (xanh nhạt, liền)
                    size = self.scale_factor
                    angles = [60 * i for i in range(7)]
                    xs = [x + size * math.cos(math.radians(a)) for a in angles]
                    ys = [y + size * math.sin(math.radians(a)) for a in angles]
                    self.ax.plot(xs, ys, color='skyblue', linestyle='-', linewidth=1.2, alpha=0.9, zorder=2)
                    x += dx
                y += dy
                row += 1

        # UE artists
        self.ue_points, self.ue_lines = [], []
        for i in range(self.num_ues):
            color = self.ue_colors[i % len(self.ue_colors)]
            p, = self.ax.plot([], [], 'o', color=color, markersize=3, zorder=7)
            self.ue_points.append(p)
            l, = self.ax.plot([], [], '-', color=color, linewidth=0.8,
                              alpha=(0.6 if self.show_link_lines else 0.0), zorder=6)
            self.ue_lines.append(l)

        self.time_text = self.ax.text(self.rect_xmin + 20, self.rect_ymax + 20, '', fontsize=10, zorder=8)

        self.ax.set_xlim(0, self.total_size)
        self.ax.set_ylim(0, self.total_size)
        self.ax.set_xlabel('Khoảng cách (m)')
        self.ax.set_ylabel('Khoảng cách (m)')
        self.ax.set_title(
            f'Grid road (Manhattan) | Ground=NLOS, UAV=LOS | {int(self.width)}x{int(self.height)} m | grid_spacing={int(self.grid_spacing_m)} m'
        )
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
            allowed = self.mobility.allowed_dirs_at_intersection(x, y)
            self.ue_dir.append(int(np.random.choice(allowed)) if allowed else int(np.random.randint(0, 4)))

    def toggle_animation(self, event):
        self.animation_running = not getattr(self, 'animation_running', False)
        if self.animation_running:
            self.toggle_button.label.set_text('Stop')
            self.run_animation()
        else:
            self.toggle_button.label.set_text('Continue')
        self.toggle_button.ax.figure.canvas.draw()

    def restart_animation(self, event):
        self.animation_running = True
        self.current_frame = 0
        self.ue_serving_bs = [None] * self.num_ues
        self.previous_serving_bs = [None] * self.num_ues

        self.turns_before_stop = config.TURNS_BEFORE_STOP
        self.stop_duration_steps = config.STOP_DURATION_STEPS
        self.ramp_duration_steps = config.RAMP_DURATION_STEPS

        self.ue_turn_count = [0] * self.num_ues
        self.ue_pause_remaining = [0] * self.num_ues
        self.ue_ramp_step = [-1] * self.num_ues
        self.ue_speed_factor = np.ones(self.num_ues, dtype=float)
        self.sf_cache = {}

        self.logger.setup_log()
        self.setup_ues()
        
        self.toggle_button.label.set_text('Stop')
        self.toggle_button.color = 'lightcoral'
        self.toggle_button.ax.figure.canvas.draw()
        self.run_animation()

    def update(self, frame: int):
        if not getattr(self, 'animation_running', True):
            return

        self.current_frame = frame
        self.logger.log_step(frame)

        for ue_idx in range(self.num_ues):
            self.ue_turns_in_step[ue_idx] = 0
            base_d = float(self.ue_speeds_ms[ue_idx] * self.time_per_step)

            if self.ue_pause_remaining[ue_idx] > 0:
                self.ue_pause_remaining[ue_idx] -= 1
                self.ue_speed_factor[ue_idx] = 0.0
                d = 0.0
                if self.ue_pause_remaining[ue_idx] == 0:
                    self.ue_ramp_step[ue_idx] = 0
            elif self.ue_ramp_step[ue_idx] >= 0:
                step_k = self.ue_ramp_step[ue_idx] + 1
                factor = min(1.0, float(step_k) / float(self.ramp_duration_steps))
                self.ue_speed_factor[ue_idx] = factor
                d = base_d * factor
                self.ue_ramp_step[ue_idx] += 1
                if self.ue_ramp_step[ue_idx] >= self.ramp_duration_steps:
                    self.ue_ramp_step[ue_idx] = -1
                    self.ue_speed_factor[ue_idx] = 1.0
            else:
                self.ue_speed_factor[ue_idx] = 1.0
                d = base_d

            self.mobility.step_manhattan_on_grid(ue_idx, d)

            x, y = self.ue_positions[ue_idx]
            self.ue_points[ue_idx].set_data([x], [y])

            bs, prx_avg, dist = self.radio_model.get_serving_bs(x, y, ue_idx)

            if bs is not None and self.show_link_lines:
                self.ue_lines[ue_idx].set_data([x, self.bs_positions[bs][0]],
                                               [y, self.bs_positions[bs][1]])
            else:
                self.ue_lines[ue_idx].set_data([], [])

            if bs is not None:
                pl_base, los = self.radio_model.calculate_path_loss_idx(x, y, bs)
                sf_db = self.radio_model.shadow_fading(ue_idx, bs, los, (x, y))
                prx_inst = self.radio_model.calculate_received_power_idx(bs, pl_base + sf_db)
                sinr = self.radio_model.calculate_sinr(ue_idx, bs, prx_inst)
            else:
                prx_inst = None
                sinr = None

            prev_bs = self.previous_serving_bs[ue_idx]
            handover_flag = 1 if (prev_bs is not None and bs is not None and bs != prev_bs) else 0
            self.previous_serving_bs[ue_idx] = bs

            self.logger.log_ue_data(ue_idx, x, y, bs, prx_inst, sinr, handover_flag)

        self.time_text.set_text(f'Step: {frame}')
        self.fig.canvas.draw()
        self.fig.canvas.flush_events()

    def run_animation(self):
        self.animation_running = True
        for frame in range(0, self.steps + 1):
            if not self.animation_running:
                break
            self.update(frame)
            plt.pause(self.pause_s)
        try:
            self.logger.save_data_to_csv()
        except Exception as e:
            print('CSV save failed:', e)
