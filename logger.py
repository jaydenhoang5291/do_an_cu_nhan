"""
Logs simulation data for each step and UE, then exports the collected results
to a CSV file in the data directory.
"""

import os
import pandas as pd
from datetime import datetime

class SimulationLogger:
    def __init__(self, sim):
        self.sim = sim

    def setup_log(self):
        self.sim.data_log = {'Step': []}
        for i in range(self.sim.num_ues):
            keys = [
                'x', 'y', 'height', 'direction', 'connected_bs', 'los_probability',
                'los_state', 'pathloss', 'shadow_fading', 'rsrp', 'prx', 'sinr',
                'speed', 'handover'
            ]
            for j in range(6):
                keys.extend([f'bs{j + 1}_idx', f'bs{j + 1}_rsrp'])
            for key in keys:
                self.sim.data_log[f'ue{i}_{key}'] = []

    def log_step(self, frame):
        self.sim.data_log['Step'].append(frame)

    def log_ue_data(
        self, ue_idx, x, y, bs, sinr, handover_flag, neighbors=None,
        rsrp_dbm=None, prx_dbm=None, los_probability=None,
        los_state=None, pathloss_db=None, shadow_fading_db=None
    ):
        self.sim.data_log[f'ue{ue_idx}_x'].append(x)
        self.sim.data_log[f'ue{ue_idx}_y'].append(y)
        self.sim.data_log[f'ue{ue_idx}_height'].append(self.sim.get_ue_height_m(ue_idx))
        self.sim.data_log[f'ue{ue_idx}_direction'].append(int(self.sim.ue_dir[ue_idx]))
        self.sim.data_log[f'ue{ue_idx}_connected_bs'].append(bs)
        self.sim.data_log[f'ue{ue_idx}_los_probability'].append(los_probability)
        self.sim.data_log[f'ue{ue_idx}_los_state'].append(los_state)
        self.sim.data_log[f'ue{ue_idx}_pathloss'].append(pathloss_db)
        self.sim.data_log[f'ue{ue_idx}_shadow_fading'].append(shadow_fading_db)
        self.sim.data_log[f'ue{ue_idx}_rsrp'].append(rsrp_dbm)
        self.sim.data_log[f'ue{ue_idx}_prx'].append(prx_dbm)
        self.sim.data_log[f'ue{ue_idx}_sinr'].append(sinr)
        self.sim.data_log[f'ue{ue_idx}_speed'].append(float(self.sim.ue_speeds[ue_idx] * self.sim.ue_speed_factor[ue_idx]))
        self.sim.data_log[f'ue{ue_idx}_handover'].append(handover_flag)
        
        if neighbors is None:
            neighbors = []
        for j in range(6):
            idx_key = f'ue{ue_idx}_bs{j+1}_idx'
            rsrp_key = f'ue{ue_idx}_bs{j+1}_rsrp'
            if j < len(neighbors):
                self.sim.data_log[idx_key].append(neighbors[j][0])
                self.sim.data_log[rsrp_key].append(neighbors[j][1])
            else:
                self.sim.data_log[idx_key].append(None)
                self.sim.data_log[rsrp_key].append(None)

    def save_data_to_csv(self):
        os.makedirs("data", exist_ok=True)
        max_len = max((len(v) for v in self.sim.data_log.values()), default=0)
        for k in self.sim.data_log:
            while len(self.sim.data_log[k]) < max_len:
                self.sim.data_log[k].append(None)

        df = pd.DataFrame(self.sim.data_log)

        for col in df.columns:
            if any(tag in col for tag in ['rsrp', 'prx', 'sinr', '_x', '_y', 'los_probability', 'pathloss', 'shadow_fading']):
                df[col] = pd.to_numeric(df[col], errors='coerce').round(2)

        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        
        filename = f"{timestamp}_{self.sim.num_ues}_UE_Data.csv"
        filepath = os.path.join("data", filename)
        
        df.to_csv(filepath, index=False, sep=',', decimal='.', encoding='utf-8-sig', float_format='%.2f')
        print(f"Saved single CSV for all {self.sim.num_ues} UEs: {filepath}")
