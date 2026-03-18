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
        
        for i in range(self.sim.num_ues):
            # Lọc các cột cơ bản và các cột riêng biệt cho mỗi UE
            ue_cols = ['Step'] + [col for col in df.columns if col.startswith(f'ue{i}_')]
            df_ue = df[ue_cols].copy()
            
            # Xóa tiền tố ue{i}_ khỏi tên cột để file sạch sẽ hơn
            rename_dict = {col: col.replace(f'ue{i}_', '') for col in ue_cols if col.startswith(f'ue{i}_')}
            df_ue.rename(columns=rename_dict, inplace=True)
            
            # Tính tốc độ của UE hiện tại để cho vào tên file
            velocity = df_ue['speed'].max() if 'speed' in df_ue.columns and not df_ue['speed'].isna().all() else 0
            
            filename = f"UE_{i}_{velocity:.2f}_kmh_{timestamp}.csv"
            filepath = os.path.join("data", filename)
            df_ue.to_csv(filepath, index=False, sep=',', decimal='.', encoding='utf-8-sig', float_format='%.2f')
            print(f"Saved CSV for UE {i}: {filepath}")
