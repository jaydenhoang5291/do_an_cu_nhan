"""
Handles UE mobility on a Manhattan road grid: creates road lines, selects
valid movement directions, and updates positions at each simulation step.
"""

import numpy as np
from utils import _read_float, _read_int

class GridMobility:
    def __init__(self, sim):
        self.sim = sim

    @staticmethod
    def _make_lines(a: float, b: float, step: float) -> np.ndarray:
        lines = np.arange(a, b + 1e-9, step, dtype=float)
        if abs(lines[-1] - b) > 1e-6:
            lines = np.append(lines, b)
        if abs(lines[0] - a) > 1e-6:
            lines = np.insert(lines, 0, a)
        lines = np.unique(np.round(lines, 6))
        lines.sort()
        return lines

    def allowed_dirs_at_intersection(self, x: float, y: float) -> list[int]:
        allowed = []
        ix_r = np.searchsorted(self.sim.x_lines, x, side='right')
        ix_l = np.searchsorted(self.sim.x_lines, x, side='left') - 1
        iy_u = np.searchsorted(self.sim.y_lines, y, side='right')
        iy_d = np.searchsorted(self.sim.y_lines, y, side='left') - 1
        if ix_r < len(self.sim.x_lines): allowed.append(0)
        if ix_l >= 0: allowed.append(2)
        if iy_u < len(self.sim.y_lines): allowed.append(1)
        if iy_d >= 0: allowed.append(3)
        allowed2 = []
        for d in allowed:
            if d == 0 and x >= self.sim.x_lines[-1] - 1e-6: continue
            if d == 2 and x <= self.sim.x_lines[0] + 1e-6: continue
            if d == 1 and y >= self.sim.y_lines[-1] - 1e-6: continue
            if d == 3 and y <= self.sim.y_lines[0] + 1e-6: continue
            allowed2.append(d)
        return allowed2

    def turn_at_intersection(self, ue_idx: int) -> bool:
        x, y = self.sim.ue_positions[ue_idx]
        allowed = self.allowed_dirs_at_intersection(x, y)
        if not allowed:
            return False

        old_dir = int(self.sim.ue_dir[ue_idx])
        new_dir = int(np.random.choice(allowed))
        self.sim.ue_dir[ue_idx] = new_dir

        if new_dir != old_dir:
            self.sim.ue_turn_count[ue_idx] += 1
            self.sim.ue_turns_in_step[ue_idx] += 1

            stop_and_go_enabled = (
                self.sim.turns_before_stop > 0
                and self.sim.stop_duration_steps > 0
            )
            if stop_and_go_enabled and self.sim.ue_turn_count[ue_idx] >= self.sim.turns_before_stop:
                self.sim.ue_turn_count[ue_idx] = 0
                self.sim.ue_pause_remaining[ue_idx] = self.sim.stop_duration_steps
                self.sim.ue_ramp_step[ue_idx] = -2
                self.sim.ue_speed_factor[ue_idx] = 0.0
                return True

        return False

    def step_manhattan_on_grid(self, ue_idx: int, dist_m: float):
        x, y = self.sim.ue_positions[ue_idx]
        d = float(dist_m)
        eps = 1e-6

        while d > eps:
            dir = int(self.sim.ue_dir[ue_idx])

            if dir == 0:
                j = np.searchsorted(self.sim.x_lines, x, side='right')
                if j >= len(self.sim.x_lines):
                    self.sim.ue_positions[ue_idx] = [x, y]
                    if self.turn_at_intersection(ue_idx): break
                    continue
                target = float(self.sim.x_lines[j])
                to_next = target - x
                if to_next <= eps:
                    x = target
                    self.sim.ue_positions[ue_idx] = [x, y]
                    if self.turn_at_intersection(ue_idx): break
                    continue
                if d < to_next:
                    x += d
                    d = 0.0
                else:
                    x = target
                    d -= to_next
                    self.sim.ue_positions[ue_idx] = [x, y]
                    if self.turn_at_intersection(ue_idx): break

            elif dir == 2:
                j = np.searchsorted(self.sim.x_lines, x, side='left') - 1
                if j < 0:
                    self.sim.ue_positions[ue_idx] = [x, y]
                    if self.turn_at_intersection(ue_idx): break
                    continue
                target = float(self.sim.x_lines[j])
                to_next = x - target
                if to_next <= eps:
                    x = target
                    self.sim.ue_positions[ue_idx] = [x, y]
                    if self.turn_at_intersection(ue_idx): break
                    continue
                if d < to_next:
                    x -= d
                    d = 0.0
                else:
                    x = target
                    d -= to_next
                    self.sim.ue_positions[ue_idx] = [x, y]
                    if self.turn_at_intersection(ue_idx): break

            elif dir == 1:
                j = np.searchsorted(self.sim.y_lines, y, side='right')
                if j >= len(self.sim.y_lines):
                    self.sim.ue_positions[ue_idx] = [x, y]
                    if self.turn_at_intersection(ue_idx): break
                    continue
                target = float(self.sim.y_lines[j])
                to_next = target - y
                if to_next <= eps:
                    y = target
                    self.sim.ue_positions[ue_idx] = [x, y]
                    if self.turn_at_intersection(ue_idx): break
                    continue
                if d < to_next:
                    y += d
                    d = 0.0
                else:
                    y = target
                    d -= to_next
                    self.sim.ue_positions[ue_idx] = [x, y]
                    if self.turn_at_intersection(ue_idx): break

            else:
                j = np.searchsorted(self.sim.y_lines, y, side='left') - 1
                if j < 0:
                    self.sim.ue_positions[ue_idx] = [x, y]
                    if self.turn_at_intersection(ue_idx): break
                    continue
                target = float(self.sim.y_lines[j])
                to_next = y - target
                if to_next <= eps:
                    y = target
                    self.sim.ue_positions[ue_idx] = [x, y]
                    if self.turn_at_intersection(ue_idx): break
                    continue
                if d < to_next:
                    y -= d
                    d = 0.0
                else:
                    y = target
                    d -= to_next
                    self.sim.ue_positions[ue_idx] = [x, y]
                    if self.turn_at_intersection(ue_idx): break

        x = float(np.clip(x, self.sim.rect_xmin, self.sim.rect_xmax))
        y = float(np.clip(y, self.sim.rect_ymin, self.sim.rect_ymax))
        self.sim.ue_positions[ue_idx] = [x, y]
