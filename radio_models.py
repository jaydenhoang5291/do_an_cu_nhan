"""
Implements radio calculations: UE-BS distance, path loss, shadow fading,
received power, SINR, and serving BS selection.
"""

import numpy as np
import config

class RadioModel:
    def __init__(self, sim):
        self.sim = sim

    def calculate_distance_idx(self, ue_x, ue_y, bs_idx: int):
        bs_x, bs_y = self.sim.bs_positions[bs_idx]
        d2 = float(np.hypot(ue_x - bs_x, ue_y - bs_y))
        dz = float(self.sim.bs_heights[bs_idx] - self.sim.h_ut)
        return max(float(np.hypot(d2, dz)), 1.0)

    def calculate_path_loss_idx(self, ue_x, ue_y, bs_idx: int):
        d3 = self.calculate_distance_idx(ue_x, ue_y, bs_idx)
        fspl_1m = 32.4 + 20.0 * np.log10(self.sim.fc)
        if self.sim.bs_is_uav[bs_idx]:
            n = self.sim.ple_uav_los
            los = True
        else:
            n = self.sim.ple_nlos
            los = False
        pl = fspl_1m + 10.0 * n * np.log10(d3)
        return float(pl), los

    def shadow_fading(self, ue_idx, bs_idx, los, ue_pos):
        if self.sim.bs_is_uav[bs_idx]:
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
        for i, (bs_x, bs_y) in enumerate(self.sim.bs_positions):
            d3 = self.calculate_distance_idx(ue_x, ue_y, i)
            dist_list_all.append((i, d3))
            if self.sim.max_bs_range is not None:
                d2 = float(np.hypot(ue_x - bs_x, ue_y - bs_y))
                if d2 > self.sim.max_bs_range:
                    continue
            dist_list.append((i, d3))

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
            pl_base, los_i = self.calculate_path_loss_idx(ue_x, ue_y, i)
            sf_db = self.shadow_fading(ue_idx, i, los_i, (ue_x, ue_y))
            pl = pl_base + sf_db
            prx = self.calculate_received_power_idx(i, pl)
            d3 = self.calculate_distance_idx(ue_x, ue_y, i)
            cand.append((i, float(prx), float(d3)))

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
            pl, los_i = self.calculate_path_loss_idx(ue_x, ue_y, i)
            sf_i = self.shadow_fading(ue_idx, i, los_i, (ue_x, ue_y))
            prx_i = self.calculate_received_power_idx(i, pl + sf_i)
            if prx_i >= self.sim.sensitivity:
                interf += 10 ** (prx_i / 10.0)
        prx_mw = 10 ** (prx_dbm / 10.0)
        sinr_lin = prx_mw / (interf + N_mw)
        sinr_db = 10.0 * np.log10(sinr_lin) if sinr_lin > 0 else -np.inf
        return float(sinr_db)
