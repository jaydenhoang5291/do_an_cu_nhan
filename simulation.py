"""
Coordinates the full simulation: initializes the simulation area, draws the
Matplotlib UI, updates UE positions, manages BS connections, and runs animation.
"""

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
from ue import AerialUE

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
        ue_height_m: float | None = None,
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

        # Kept for backward-compatible constructor calls. UAV is modeled as a
        # high-altitude UE; the simulator no longer deploys aerial BS nodes.
        self.add_uav_cover = False
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
        self.ue_height_m = self._validate_ue_height(ue_height_m)

        # Ground BS uses NLOS by default.
        self.ple_nlos = config.PLE_NLOS

        # Aerial radio params kept for UE-at-height formulas with ground BS.
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
        self.ue_heights_m = np.full(self.num_ues, self.ue_height_m, dtype=float)
        self.aerial_ues = [
            AerialUE(height_m=self.ue_height_m, speed_mps=float(self.ue_speeds_ms[i]))
            for i in range(self.num_ues)
        ] if self.is_aerial_ue else []

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
        self.fig = plt.figure(figsize=(11.5, 10))
        self.ax = self.fig.add_axes([0.06, 0.1, 0.72, 0.82])
        self.height_ax = self.fig.add_axes([0.83, 0.18, 0.11, 0.64])
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

    @staticmethod
    def _validate_ue_height(ue_height_m: float | None) -> float:
        if ue_height_m is None:
            return float(config.H_UT)

        height = float(ue_height_m)
        if height not in config.AERIAL_UE_HEIGHTS_M:
            allowed = ", ".join(f"{h:g}" for h in config.AERIAL_UE_HEIGHTS_M)
            raise ValueError(f"UE UAV height must be one of: {allowed} m")
        return height

    @property
    def is_aerial_ue(self) -> bool:
        return float(self.ue_height_m) in config.AERIAL_UE_HEIGHTS_M

    def get_ue_height_m(self, ue_idx: int) -> float:
        return float(self.ue_heights_m[ue_idx])

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

            # Hex boundary.
            size = self.scale_factor
            angles = [60 * i for i in range(7)]
            xs = [cx + size * math.cos(math.radians(a)) for a in angles]
            ys = [cy + size * math.sin(math.radians(a)) for a in angles]
            self.ax.plot(xs, ys, color='black', linestyle='--', linewidth=1.0, alpha=0.6, zorder=2)



        # UE artists
        self.ue_points, self.ue_lines = [], []
        for i in range(self.num_ues):
            color = self.ue_colors[i % len(self.ue_colors)]
            p, = self.ax.plot([], [], 'o', color=color, markersize=3, zorder=7)
            self.ue_points.append(p)
            l, = self.ax.plot([], [], '-', color=color, linewidth=0.8,
                              alpha=(0.6 if self.show_link_lines else 0.0), zorder=6)
            self.ue_lines.append(l)

        # Keep the step label outside the simulation frame, just below the title.
        self.time_text = self.ax.text(
            0.5, 1.015, '', transform=self.ax.transAxes,
            ha='center', va='bottom', fontsize=11, zorder=8, clip_on=False
        )

        self.ax.set_xlim(0, self.total_size)
        self.ax.set_ylim(0, self.total_size)
        self.ax.set_aspect('equal', adjustable='box')
        if hasattr(self.ax, 'set_box_aspect'):
            self.ax.set_box_aspect(1)
        self.ax.set_xlabel('Distance (m)')
        self.ax.set_ylabel('Distance (m)')
        ue_tag = f"UE h={self.ue_height_m:g}m"
        self.ax.set_title(
            f"{int(self.width)}m x{int(self.height)}m | Ground BS | {self.num_ues} UEs | {ue_tag} | Spacing: {int(self.grid_spacing_m)}m",
            fontsize=10,
            pad=28
        )
        self.setup_height_plot()
        plt.draw()

    def setup_height_plot(self):
        self.height_ax.clear()

        ue_heights = [self.get_ue_height_m(i) for i in range(self.num_ues)]
        max_height = max([self.h_bs, *ue_heights, 50.0])
        y_max = max_height * 1.2
        x_ues = np.arange(1, self.num_ues + 1)

        self.height_ax.axhline(
            self.h_bs,
            color='tab:red',
            linestyle='--',
            linewidth=1.5,
            label=f'Ground BS {self.h_bs:g} m'
        )
        self.height_ax.vlines(
            x_ues,
            0,
            ue_heights,
            colors=[self.ue_colors[i % len(self.ue_colors)] for i in range(self.num_ues)],
            linewidth=2.0,
            alpha=0.8
        )
        self.height_ax.scatter(
            x_ues,
            ue_heights,
            c=[self.ue_colors[i % len(self.ue_colors)] for i in range(self.num_ues)],
            s=36,
            zorder=3,
            label='UE height'
        )

        for x, height in zip(x_ues, ue_heights):
            self.height_ax.text(x, height + y_max * 0.025, f'{height:g}', ha='center', va='bottom', fontsize=8)

        self.height_ax.set_xlim(0.5, max(1.5, self.num_ues + 0.5))
        self.height_ax.set_ylim(0, y_max)
        self.height_ax.set_title('Height Profile', fontsize=10)
        self.height_ax.set_ylabel('Height (m)')
        self.height_ax.set_xlabel('UE')
        self.height_ax.grid(axis='y', linestyle=':', linewidth=0.8, alpha=0.6)
        self.height_ax.legend(loc='upper right', fontsize=8)

        if self.num_ues <= 8:
            self.height_ax.set_xticks(x_ues)
            self.height_ax.set_xticklabels([f'UE{i}' for i in range(self.num_ues)], rotation=45, ha='right')
        else:
            self.height_ax.set_xticks([])

    def setup_ues(self):
        self.ue_positions.clear()
        self.ue_dir.clear()

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
        self.current_frame = -1
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

            bs, prx_avg, dist, neighbors = self.radio_model.get_serving_bs(x, y, ue_idx)

            if bs is not None and self.show_link_lines:
                self.ue_lines[ue_idx].set_data([x, self.bs_positions[bs][0]],
                                               [y, self.bs_positions[bs][1]])
            else:
                self.ue_lines[ue_idx].set_data([], [])

            if bs is not None:
                pl_base, los, los_probability = self.radio_model.calculate_path_loss(x, y, bs, ue_idx)
                los_state = 'LOS' if los else 'NLOS'
                sf_db = self.radio_model.shadow_fading(ue_idx, bs, los, (x, y))
                prx_inst = self.radio_model.calculate_received_power(bs, pl_base + sf_db)
                sinr = self.radio_model.calculate_sinr(ue_idx, bs, prx_inst)
            else:
                los_probability = None
                los_state = None
                prx_inst = None
                sinr = None

            prev_bs = self.previous_serving_bs[ue_idx]
            handover_flag = 1 if (prev_bs is not None and bs is not None and bs != prev_bs) else 0
            self.previous_serving_bs[ue_idx] = bs

            self.logger.log_ue_data(
                ue_idx, x, y, bs, prx_inst, sinr, handover_flag, neighbors,
                los_probability=los_probability,
                los_state=los_state
            )

        self.time_text.set_text(f'Step: {frame}')
        self.fig.canvas.draw()
        self.fig.canvas.flush_events()

    def run_animation(self):
        self.animation_running = True
        start_frame = getattr(self, 'current_frame', -1) + 1
        
        # Avoid running if we already completed all steps
        if start_frame > self.steps:
            return

        for frame in range(start_frame, self.steps + 1):
            if not self.animation_running:
                break
            self.update(frame)
            plt.pause(self.pause_s)
            
        if self.animation_running and getattr(self, 'current_frame', -1) == self.steps:
            try:
                self.logger.save_data_to_csv()
            except Exception as e:
                print('CSV save failed:', e)
