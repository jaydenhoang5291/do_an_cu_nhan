"""
Implements radio calculations: UE-BS distance, path loss, shadow fading,
received power, SINR, and serving BS selection.
"""

import numpy as np
import config


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

#########################################################################################
# CALCULATE PATH LOSS OF UMa/UMa-AV LOS/NLOS
#########################################################################################

# Calculate the breakpoint distance
def effective_breakpoint_distance(fc_ghz: float, h_bs: float, h_UT: float) -> float:
    fc_hz = float(fc_ghz) * 1e9
    h_e = 1.0
    h_bs_effective = float(h_bs) - h_e
    h_UT_effective = float(h_UT) - h_e
    return 4.0 * h_bs_effective * h_UT_effective * fc_hz / 3e8

# d2D distance calculations with minimum 10m 2D distance TS 138.901 Table 7.4.1-1
def effective_uma_distances(d2D: float, h_bs: float, h_UT: float) -> tuple[float, float]:
    d2D_eff = max(float(d2D), 10.0)
    d3D_eff = float(np.hypot(d2D_eff, float(h_bs) - float(h_UT)))
    return d2D_eff, d3D_eff

# UMa LOS path loss based on TR 38.901 Table 7.4.1-1
def uma_los_path_loss(d2D: float, d3D: float, fc_ghz: float, h_bs: float = 25.0, h_UT: float = 1.5) -> float:
    d2D, d3D = effective_uma_distances(d2D, h_bs, h_UT)
    fc_ghz = float(fc_ghz)
    d_bp_effective = effective_breakpoint_distance(fc_ghz, h_bs, h_UT)

    if d2D <= d_bp_effective:
        # TR 38.901 Table 7.4.1-1, UMa LOS, referenced by TR 36.777
        pl = 28.0 + 22.0 * np.log10(d3D) + 20.0 * np.log10(fc_ghz)
    else:
        # TR 38.901 Table 7.4.1-1, UMa LOS/NLOS, referenced by TR 36.777
        pl = (
            28.0
            + 40.0 * np.log10(d3D)
            + 20.0 * np.log10(fc_ghz)
            - 9.0 * np.log10(d_bp_effective ** 2 + (float(h_bs) - float(h_UT)) ** 2)
        )
    return float(pl)

# UMa NLOS path loss based on TR 38.901 Table 7.4.1-1
def uma_nlos_path_loss(d2D: float, d3D: float, fc_ghz: float, h_bs: float = 25.0, h_UT: float = 1.5) -> float:
    d2D, d3D = effective_uma_distances(d2D, h_bs, h_UT)
    h_UT = float(h_UT)
    los_pl = uma_los_path_loss(d2D, d3D, fc_ghz, h_bs, h_UT)
    # TR 38.901 Table 7.4.1-1, UMa LOS/NLOS, referenced by TR 36.777
    nlos_pl = (
        13.54
        + 39.08 * np.log10(d3D)
        + 20.0 * np.log10(float(fc_ghz))
        - 0.6 * (h_UT - 1.5)
    )
    return float(max(los_pl, nlos_pl))

# UMa-AV LOS path loss based on TR 36.777 Annex B Table B-2
def uma_av_los_path_loss(d2D: float, d3D: float, fc_ghz: float, h_bs: float = 25.0, h_UT: float = 1.5) -> float:
    h_UT = float(h_UT)
    if 1.5 <= h_UT <= 22.5:
        return uma_los_path_loss(d2D, d3D, fc_ghz, h_bs, h_UT)
    if 22.5 < h_UT <= 300.0:
        _, d3D = effective_uma_distances(d2D, h_bs, h_UT)
        # TR 36.777 Annex B Table B-2, UMa-AV LOS
        pl = 28.0 + 22.0 * np.log10(d3D) + 20.0 * np.log10(float(fc_ghz))
        return float(pl)
    raise ValueError("UMa-AV LOS pathloss is defined for 1.5 m <= hUT <= 300 m")

# UMa-AV NLOS path loss based on TR 36.777 Annex B Table B-2
def uma_av_nlos_path_loss(d2D: float, d3D: float, fc_ghz: float, h_bs: float = 25.0, h_UT: float = 1.5) -> float:
    h_UT = float(h_UT)
    if 1.5 <= h_UT <= 22.5:
        return uma_nlos_path_loss(d2D, d3D, fc_ghz, h_bs, h_UT)
    if 22.5 < h_UT <= 100.0:
        _, d3D = effective_uma_distances(d2D, h_bs, h_UT)
        # TR 36.777 Annex B Table B-2, UMa-AV NLOS
        pl = (
            -17.5
            + (46.0 - 7.0 * np.log10(h_UT)) * np.log10(d3D)
            + 20.0 * np.log10(40.0 * np.pi * float(fc_ghz) / 3.0)
        )
        return float(pl)
    raise ValueError("UMa-AV NLOS pathloss is defined for 1.5 m <= hUT <= 100 m")

# UMa-AV shadow fading standard deviation based on TR 36.777 Annex B Table B-3
def uma_av_shadow_fading_sigma(los: bool, h_UT: float) -> float:
    h_UT = float(h_UT)
    if 1.5 <= h_UT <= 22.5:
        # TR 38.901 Table 7.4.1-1, UMa shadow fading, referenced by TR 36.777
        return 4.0 if los else 6.0
    if los and 22.5 < h_UT <= 300.0:
        # TR 36.777 Annex B Table B-3, UMa-AV LOS shadow fading standard deviation
        return float(4.64 * np.exp(-0.0066 * h_UT))
    if (not los) and 22.5 < h_UT <= 100.0:
        # TR 36.777 Annex B Table B-3, UMa-AV NLOS shadow fading standard deviation
        return 6.0
    raise ValueError("UMa-AV shadow fading sigma is undefined for this LOS/NLOS and hUT")

##########################################################################################
# Main RadioModel class that uses the above functions to calculate path loss, shadow fading,
# received power, SINR, and serving BS selection.
##########################################################################################
class RadioModel:
    def __init__(self, sim):
        self.sim = sim

    # Calculate 2D and 3D distances between UE and BS, ensuring minimum 3D distance of 1m to avoid singularities
    def calculate_distances(self, ue_x, ue_y, bs_idx: int, ue_idx: int | None = None):
        bs_x, bs_y = self.sim.bs_positions[bs_idx]
        d2D = float(np.hypot(ue_x - bs_x, ue_y - bs_y))
        h_UT = self.sim.get_ue_height_m(ue_idx) if ue_idx is not None else self.sim.h_UT
        bs_height = float(self.sim.bs_heights[bs_idx])
        delta_height = bs_height - h_UT
        d3D = max(float(np.hypot(d2D, delta_height)), 1.0)
        return d2D, d3D

    def _link_state(self, ue_idx: int, bs_idx: int) -> dict:
        return self.sim.sf_cache.setdefault((ue_idx, bs_idx), {})

    def _link_los_state(
        self,
        ue_idx: int | None,
        bs_idx: int,
        p_los: float,
    ) -> bool:
        if ue_idx is not None:
            state = self._link_state(ue_idx, bs_idx)
            if 'los' in state:
                return bool(state['los'])

        p_los = float(np.clip(p_los, 0.0, 1.0))
        if p_los <= 0.0:
            sampled_los = False
        elif p_los >= 1.0:
            sampled_los = True
        else:
            sampled_los = bool(np.random.random() < p_los)

        if ue_idx is None:
            return sampled_los

        state['los'] = sampled_los
        return bool(state['los'])

    def calculate_path_loss(self, ue_x, ue_y, bs_idx: int, ue_idx: int | None = None):
        d2D, d3D = self.calculate_distances(ue_x, ue_y, bs_idx, ue_idx)
        h_UT = self.sim.get_ue_height_m(ue_idx) if ue_idx is not None else self.sim.h_UT
        p_los = uma_av_los_probability(d2D, h_UT)
        los = self._link_los_state(ue_idx, bs_idx, p_los)
        if los:
            pl = uma_av_los_path_loss(d2D, d3D, self.sim.fc, self.sim.h_bs, h_UT)
        else:
            pl = uma_av_nlos_path_loss(d2D, d3D, self.sim.fc, self.sim.h_bs, h_UT)
        return float(pl), los, float(p_los)

    def shadow_fading(self, ue_idx, bs_idx, los, ue_pos):
        state = self._link_state(ue_idx, bs_idx)
        if 'los' not in state:
            state['los'] = bool(los)
        link_los = bool(state['los'])
        h_UT = self.sim.get_ue_height_m(ue_idx)
        sigma = uma_av_shadow_fading_sigma(link_los, h_UT)

        def sample_shadow_fading() -> float:
            return float(np.clip(np.random.normal(0.0, sigma), -3.0 * sigma, 3.0 * sigma))

        x, y = float(ue_pos[0]), float(ue_pos[1])
        if 'shadow_fading_db' not in state:
            state['shadow_fading_db'] = sample_shadow_fading()
            state['last_x'] = x
            state['last_y'] = y
            state['distance_since_sf_update_m'] = 0.0
            state['los'] = link_los
            return float(state['shadow_fading_db'])

        delta = float(np.hypot(x - state['last_x'], y - state['last_y']))
        state['last_x'] = x
        state['last_y'] = y
        state['distance_since_sf_update_m'] += delta

        update_distance = float(getattr(
            self.sim,
            'shadow_fading_update_distance_m',
            config.SHADOW_FADING_UPDATE_DISTANCE_M,
        ))
        # Shadow fading update with each 25m of UE movement
        if state['distance_since_sf_update_m'] >= update_distance:
            state['shadow_fading_db'] = sample_shadow_fading()
            state['distance_since_sf_update_m'] = 0.0

        return float(state['shadow_fading_db'])

    def get_total_bs_tx_power_dbm(self, bs_idx: int) -> float:
        return float(self.sim.bs_ptx[bs_idx])

    def active_lte_subcarrier_count(self) -> float:
        n_subcarriers = float(self.sim.lte_n_rb * self.sim.lte_n_subcarriers_per_rb)
        if n_subcarriers <= 0.0:
            raise ValueError("LTE RB/subcarrier configuration must be positive")
        return n_subcarriers

    # Calculate total received power in dBm at the UE from a given BS index and path loss, including shadow fading.
    def calculate_total_received_power(self, bs_idx: int, path_loss_db: float) -> float:
        ptx_total_dbm = self.get_total_bs_tx_power_dbm(bs_idx)
        return float(ptx_total_dbm + self.sim.gtx + self.sim.grx - float(path_loss_db))

    def calculate_rsrp_from_received_power(self, prx_dbm: float) -> float:
        # System-level approximation: RSRP is derived from total received power
        # by assuming BS power is spread uniformly over active LTE subcarriers.
        # This follows the LTE RB structure and RSRP definition, but is not an
        # exact 3GPP PHY CRS power formula.
        return float(float(prx_dbm) - 10.0 * np.log10(self.active_lte_subcarrier_count()))

    def calculate_rsrp(self, bs_idx: int, path_loss_db: float) -> float:
        prx_dbm = self.calculate_total_received_power(bs_idx, path_loss_db)
        return self.calculate_rsrp_from_received_power(prx_dbm)

    def thermal_noise_power_dbm(self) -> float:
        bandwidth = float(self.sim.bandwidth)              # Hz
        noise_figure = float(self.sim.ue_noise_figure)    # dB
        if bandwidth <= 0.0:
            raise ValueError("Bandwidth must be positive to calculate thermal noise")

        # Thermal noise is a receiver/bandwidth property, not a function of Prx.
        # -174 dBm/Hz is the thermal noise density at room temperature.
        return float(-174.0 + 10.0 * np.log10(bandwidth) + noise_figure)

    def get_interfering_bs_indices(self, serving_bs_idx: int, count: int = 6) -> list[int]:
        serving_x, serving_y = self.sim.bs_positions[serving_bs_idx]
        neighbors = []

        for bs_idx, (bs_x, bs_y) in enumerate(self.sim.bs_positions):
            if bs_idx == serving_bs_idx:
                continue

            # In the hexagonal layout, the six nearest BSs around the serving
            # BS approximate the first-tier co-channel interference cells.
            distance = float(np.hypot(float(bs_x) - serving_x, float(bs_y) - serving_y))
            neighbors.append((bs_idx, distance))

        neighbors.sort(key=lambda item: item[1])
        return [bs_idx for bs_idx, _ in neighbors[:count]]

    def calculate_interference_power_mw(self, ue_idx: int, serving_bs_idx: int) -> float:
        interference_mw = 0.0
        ue_x, ue_y = self.sim.ue_positions[ue_idx]

        for interferer_bs_idx in self.get_interfering_bs_indices(serving_bs_idx):
            # Interference is the received power at this UE from the first-tier
            # neighboring BSs around the serving BS, assuming they transmit on
            # the same downlink time/frequency resource.
            pl, los_i, _ = self.calculate_path_loss(ue_x, ue_y, interferer_bs_idx, ue_idx)
            sf_i = self.shadow_fading(ue_idx, interferer_bs_idx, los_i, (ue_x, ue_y))
            interfering_prx_dbm = self.calculate_total_received_power(
                interferer_bs_idx,
                pl + sf_i,
            )
            interference_mw += 10.0 ** (interfering_prx_dbm / 10.0)

        return float(interference_mw)

    def calculate_sinr(self, ue_idx, serving_bs_idx, prx_dbm) -> float:
        # SINR_i = P_rx,i / (sum_{j != i} P_rx,j + N)
        # All powers must be converted from dBm to mW before summing/dividing.
        signal_mw = 10.0 ** (float(prx_dbm) / 10.0)
        interference_mw = self.calculate_interference_power_mw(ue_idx, serving_bs_idx)
        noise_mw = 10.0 ** (self.thermal_noise_power_dbm() / 10.0)

        sinr_lin = signal_mw / (interference_mw + noise_mw)
        sinr_db = 10.0 * np.log10(sinr_lin) if sinr_lin > 0 else -np.inf
        return float(sinr_db)
    
    def get_serving_bs(self, ue_x, ue_y, ue_idx):
        neighbor_count = 6
        candidate_indices = list(range(len(self.sim.bs_positions)))

        if not candidate_indices:
            self.sim.ue_serving_bs[ue_idx] = None
            return None, None, None, []

        current_bs = self.sim.ue_serving_bs[ue_idx]
        cand = []
        for i in candidate_indices:
            pl_base, los_i, _ = self.calculate_path_loss(ue_x, ue_y, i, ue_idx)
            sf_db = self.shadow_fading(ue_idx, i, los_i, (ue_x, ue_y))
            pl = pl_base + sf_db
            rsrp = self.calculate_rsrp(i, pl)
            _, d3D = self.calculate_distances(ue_x, ue_y, i, ue_idx)
            cand.append((i, float(rsrp), float(d3D)))

        if not cand:
            self.sim.ue_serving_bs[ue_idx] = None
            return None, None, None, []

        cand.sort(key=lambda x: x[1], reverse=True)
        
        def get_neighbors(final_bs):
            neighbors = [(x[0], x[1]) for x in cand if x[0] != final_bs]
            return neighbors[:neighbor_count]

        if current_bs is None:
            best_bs, best_rsrp, best_d = cand[0]
            self.sim.ue_serving_bs[ue_idx] = best_bs
            return best_bs, best_rsrp, best_d, get_neighbors(best_bs)

        cur = next((t for t in cand if t[0] == current_bs), None)
        if cur is None:
            best_bs, best_rsrp, best_d = cand[0]
            self.sim.ue_serving_bs[ue_idx] = best_bs
            return best_bs, best_rsrp, best_d, get_neighbors(best_bs)

        cur_rsrp, cur_d = float(cur[1]), float(cur[2])
        for bs_idx, bs_rsrp, bs_d in cand:
            if bs_idx == current_bs:
                continue
            if float(bs_rsrp) > cur_rsrp + float(self.sim.hom):
                self.sim.ue_serving_bs[ue_idx] = bs_idx
                return bs_idx, float(bs_rsrp), float(bs_d), get_neighbors(bs_idx)

        return current_bs, cur_rsrp, cur_d, get_neighbors(current_bs)
