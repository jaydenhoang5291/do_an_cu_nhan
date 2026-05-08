"""
Implements radio calculations: UE-BS distance, path loss, shadow fading,
received power, SINR, and serving BS selection.
"""

import numpy as np
import config

class RadioModel:
    def __init__(self, sim):
        self.sim = sim

    def is_aerial_link(self, ue_idx: int | None, bs_idx: int) -> bool:
        if self.sim.bs_is_uav[bs_idx]:
            return True
        return ue_idx is not None and self.sim.is_aerial_ue

    @staticmethod
    # UMa LOS probability from TR 38.901 and UMa-AV extension from TR 36.777.
    def uma_av_los_probability(d2D: float, h_UT: float) -> float:
        d2D = max(float(d2D), 0.0)
        h_UT = float(h_UT)

        if not (1.5 <= h_UT <= 300.0):
            raise ValueError(
                "UMa/UMa-AV LOS probability is defined for "
                "1.5 m <= h_UT <= 300 m"
            )

        # TR 38.901 UMa: 1.5 <= h_UT <= 22.5.
        if h_UT <= 22.5:
            if d2D <= 18.0:
                return 1.0

            c_hut = 0.0

            # d2D > 18.0
            if h_UT > 13.0:
                c_hut = ((h_UT - 13.0) / 10.0) ** 1.5

            base = (18.0 / d2D) + (1.0 - 18.0 / d2D) * np.exp(-d2D / 63.0)
            height_gain = 1.0 + (5.0 / 4.0) * c_hut * ((d2D / 100.0) ** 3) * np.exp(-d2D / 150.0)
            return float(np.clip(base * height_gain, 0.0, 1.0))

        # TR 36.777 UMa-AV: 100 < h_UT <= 300.
        if h_UT > 100.0:
            return 1.0

        # TR 36.777 UMa-AV: 22.5 < h_UT <= 100.
        d1 = max(460.0 * np.log10(h_UT) - 700.0, 18.0)
        p1 = 4300.0 * np.log10(h_UT) - 3800.0

        if d2D <= d1:
            return 1.0

        p_los = (d1 / d2D) + np.exp(-d2D / p1) * (1.0 - d1 / d2D)
        return float(np.clip(p_los, 0.0, 1.0))

    def get_los_probability_idx(self, ue_x, ue_y, bs_idx: int, ue_idx: int | None = None) -> float:
        d2D, _ = self.calculate_distances_idx(ue_x, ue_y, bs_idx, ue_idx)
        h_UT = self.sim.get_ue_height_m(ue_idx) if ue_idx is not None else self.sim.h_ut
        return self.uma_av_los_probability(d2D, h_UT)

    def sample_los_state_idx(self, ue_x, ue_y, bs_idx: int, ue_idx: int | None = None) -> bool:
        p_los = self.get_los_probability_idx(ue_x, ue_y, bs_idx, ue_idx)
        return p_los >= 0.5

    def calculate_distances_idx(self, ue_x, ue_y, bs_idx: int, ue_idx: int | None = None):
        bs_x, bs_y = self.sim.bs_positions[bs_idx]
        d2D = float(np.hypot(ue_x - bs_x, ue_y - bs_y))
        h_UT = self.sim.get_ue_height_m(ue_idx) if ue_idx is not None else self.sim.h_ut
        bs_height = float(self.sim.bs_heights[bs_idx])
        delta_height = bs_height - h_UT
        d3D = max(float(np.hypot(d2D, delta_height)), 1.0)
        return d2D, d3D

    def calculate_path_loss_idx(self, ue_x, ue_y, bs_idx: int, ue_idx: int | None = None):
        _, d3D = self.calculate_distances_idx(ue_x, ue_y, bs_idx, ue_idx)
        fspl_1m = 32.4 + 20.0 * np.log10(self.sim.fc)
        los = self.sample_los_state_idx(ue_x, ue_y, bs_idx, ue_idx)
        if los:
            n = self.sim.ple_uav_los
        else:
            n = self.sim.ple_nlos
        pl = fspl_1m + 10.0 * n * np.log10(d3D)
        return float(pl), los

    def shadow_fading(self, ue_idx, bs_idx, los, ue_pos):
        if self.is_aerial_link(ue_idx, bs_idx):
            sigma = self.sim.sf_sigma_uav
        else:
            sigma = self.sim.sf_sigma['LOS'] if los else self.sim.sf_sigma['NLOS']

        key = (ue_idx, bs_idx)
        state = self.sim.sf_cache.get(key)
        if state is None:
            val = float(np.random.normal(0.0, sigma))
            val = float(np.clip(val, -3 * sigma, 3 * sigma))
            self.sim.sf_cache[key] = {'x': ue_pos[0], 'y': ue_pos[1], 'val': val, 'los': bool(los)}
            return val

        oldx, oldy = state['x'], state['y']
        oldval = state['val']

        dx = float(abs(ue_pos[0] - oldx))
        dy = float(abs(ue_pos[1] - oldy))
        delta = dx + dy

        turns = int(self.sim.ue_turns_in_step[ue_idx]) if hasattr(self.sim, 'ue_turns_in_step') else 0
        if turns > 0:
            delta += float(turns) * float(self.sim.sf_turn_penalty_m)

        rho = 0.0 if self.sim.sf_decorr <= 0 else float(np.exp(-delta / self.sim.sf_decorr))
        innov = float(np.random.normal(0.0, sigma))
        newval = rho * oldval + (np.sqrt(max(0.0, 1.0 - rho ** 2)) * innov)
        newval = float(np.clip(newval, -3 * sigma, 3 * sigma))
        self.sim.sf_cache[key] = {'x': ue_pos[0], 'y': ue_pos[1], 'val': newval, 'los': bool(los)}
        return newval

    def calculate_received_power_idx(self, bs_idx: int, path_loss_db: float):
        ptx = float(self.sim.bs_ptx[bs_idx])
        return ptx + self.sim.gtx + self.sim.grx - float(path_loss_db)

    def get_serving_bs(self, ue_x, ue_y, ue_idx):
        neighbor_count = 5
        candidate_target = max(neighbor_count + 1, int(self.sim.max_candidate_bs))
        dist_list_all = []
        dist_list = []
        for i, _ in enumerate(self.sim.bs_positions):
            d2D, d3D = self.calculate_distances_idx(ue_x, ue_y, i, ue_idx)
            dist_list_all.append((i, d3D))
            if self.sim.max_bs_range is not None:
                if d2D > self.sim.max_bs_range:
                    continue
            dist_list.append((i, d3D))

        if not dist_list:
            dist_list = list(dist_list_all)
            if not dist_list:
                self.sim.ue_serving_bs[ue_idx] = None
                return None, None, None, []

        dist_list_all.sort(key=lambda x: x[1])
        dist_list.sort(key=lambda x: x[1])
        candidate_indices = [i for i, _ in dist_list[:max(1, candidate_target)]]

        # Fill short in-range candidate lists with nearest out-of-range BSs so
        # the CSV can still report the top 5 non-serving BSs when available.
        for i, _ in dist_list_all:
            if len(candidate_indices) >= max(1, candidate_target):
                break
            if i not in candidate_indices:
                candidate_indices.append(i)

        current_bs = self.sim.ue_serving_bs[ue_idx]
        if current_bs is not None and current_bs not in candidate_indices:
            candidate_indices.append(current_bs)

        cand = []
        for i in candidate_indices:
            pl_base, los_i = self.calculate_path_loss_idx(ue_x, ue_y, i, ue_idx)
            sf_db = self.shadow_fading(ue_idx, i, los_i, (ue_x, ue_y))
            pl = pl_base + sf_db
            prx = self.calculate_received_power_idx(i, pl)
            _, d3D = self.calculate_distances_idx(ue_x, ue_y, i, ue_idx)
            cand.append((i, float(prx), float(d3D)))

        if not cand:
            self.sim.ue_serving_bs[ue_idx] = None
            return None, None, None, []

        cand.sort(key=lambda x: x[1], reverse=True)
        
        def get_neighbors(final_bs):
            neighbors = [(x[0], x[1]) for x in cand if x[0] != final_bs]
            return neighbors[:neighbor_count]

        if current_bs is None:
            best_bs, best_prx, best_d = cand[0]
            self.sim.ue_serving_bs[ue_idx] = best_bs
            return best_bs, best_prx, best_d, get_neighbors(best_bs)

        cur = next((t for t in cand if t[0] == current_bs), None)
        if cur is None:
            best_bs, best_prx, best_d = cand[0]
            self.sim.ue_serving_bs[ue_idx] = best_bs
            return best_bs, best_prx, best_d, get_neighbors(best_bs)

        cur_prx, cur_d = float(cur[1]), float(cur[2])
        for bs_idx, bs_prx, bs_d in cand:
            if bs_idx == current_bs:
                continue
            if float(bs_prx) > cur_prx + float(self.sim.hom):
                self.sim.ue_serving_bs[ue_idx] = bs_idx
                return bs_idx, float(bs_prx), float(bs_d), get_neighbors(bs_idx)

        return current_bs, cur_prx, cur_d, get_neighbors(current_bs)

    def calculate_sinr(self, ue_idx, serving_bs_idx, prx_dbm):
        N_mw = 10 ** (config.N_DBM / 10.0)
        interf = 0.0
        ue_x, ue_y = self.sim.ue_positions[ue_idx]
        for i, (bs_x, bs_y) in enumerate(self.sim.bs_positions):
            if i == serving_bs_idx:
                continue
            pl, los_i = self.calculate_path_loss_idx(ue_x, ue_y, i, ue_idx)
            sf_i = self.shadow_fading(ue_idx, i, los_i, (ue_x, ue_y))
            prx_i = self.calculate_received_power_idx(i, pl + sf_i)
            if prx_i >= self.sim.sensitivity:
                interf += 10 ** (prx_i / 10.0)
        prx_mw = 10 ** (prx_dbm / 10.0)
        sinr_lin = prx_mw / (interf + N_mw)
        sinr_db = 10.0 * np.log10(sinr_lin) if sinr_lin > 0 else -np.inf
        return float(sinr_db)
