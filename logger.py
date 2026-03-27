import os
import pandas as pd
from datetime import datetime

class SimulationLogger:
    def __init__(self, sim):
        self.sim = sim

    def setup_log(self):
        self.sim.data_log = {'Step': []}
        for i in range(self.sim.num_ues):
            for key in ['x', 'y', 'direction', 'connected_bs', 'current_prx', 'sinr', 'speed', 'handover']:
                self.sim.data_log[f'ue{i}_{key}'] = []

    def log_step(self, frame):
        self.sim.data_log['Step'].append(frame)

    def log_ue_data(self, ue_idx, x, y, bs, prx_inst, sinr, handover_flag):
        self.sim.data_log[f'ue{ue_idx}_x'].append(x)
        self.sim.data_log[f'ue{ue_idx}_y'].append(y)
        self.sim.data_log[f'ue{ue_idx}_direction'].append(int(self.sim.ue_dir[ue_idx]))
        self.sim.data_log[f'ue{ue_idx}_connected_bs'].append(bs)
        self.sim.data_log[f'ue{ue_idx}_current_prx'].append(prx_inst)
        self.sim.data_log[f'ue{ue_idx}_sinr'].append(sinr)
        self.sim.data_log[f'ue{ue_idx}_speed'].append(float(self.sim.ue_speeds[ue_idx] * self.sim.ue_speed_factor[ue_idx]))
        self.sim.data_log[f'ue{ue_idx}_handover'].append(handover_flag)

    def save_data_to_csv(self):
        os.makedirs("data", exist_ok=True)
        max_len = max((len(v) for v in self.sim.data_log.values()), default=0)
        for k in self.sim.data_log:
            while len(self.sim.data_log[k]) < max_len:
                self.sim.data_log[k].append(None)

        df = pd.DataFrame(self.sim.data_log)

        for col in df.columns:
            if any(tag in col for tag in ['prx', '_x', '_y', 'SINR']):
                df[col] = pd.to_numeric(df[col], errors='coerce').round(2)

        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        
        filename = f"{self.sim.num_ues}_UE_Data_{timestamp}.csv"
        filepath = os.path.join("data", filename)
        
        df.to_csv(filepath, index=False, sep=',', decimal='.', encoding='utf-8-sig', float_format='%.2f')
        print(f"Saved single CSV for all {self.sim.num_ues} UEs: {filepath}")
