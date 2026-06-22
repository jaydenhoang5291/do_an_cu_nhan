import unittest

import pandas as pd

from ml_sinr_prediction.data_loading import simulator_wide_to_long
from ml_sinr_prediction.dataset import build_sequence_dataset, time_split_by_series
from ml_sinr_prediction.features import select_feature_columns


class FinalPipelineTests(unittest.TestCase):
    def test_final_sequence_dataset_and_group_split(self):
        df = pd.DataFrame(
            {
                "Step": list(range(8)),
                "ue0_x": list(range(8)),
                "ue0_y": [0] * 8,
                "ue0_height": [1.5] * 8,
                "ue0_direction": [0] * 8,
                "ue0_connected_bs": [1] * 8,
                "ue0_los_probability": [1.0] * 8,
                "ue0_shadow_fading": [0.0] * 8,
                "ue0_rsrp": [-80 + i for i in range(8)],
                "ue0_sinr": [float(i) for i in range(8)],
                "ue0_speed": [30.0] * 8,
                "ue0_handover": [0] * 8,
                "ue1_x": [10 + i for i in range(8)],
                "ue1_y": [0] * 8,
                "ue1_height": [1.5] * 8,
                "ue1_direction": [0] * 8,
                "ue1_connected_bs": [2] * 8,
                "ue1_los_probability": [1.0] * 8,
                "ue1_shadow_fading": [0.0] * 8,
                "ue1_rsrp": [-70 + i for i in range(8)],
                "ue1_sinr": [10.0 + i for i in range(8)],
                "ue1_speed": [30.0] * 8,
                "ue1_handover": [0] * 8,
            }
        )

        long_df = simulator_wide_to_long(df, source_file="sample")
        feature_cols = select_feature_columns(long_df)
        dataset = build_sequence_dataset(long_df, feature_cols, history_len=3, horizon=1)
        split = time_split_by_series(dataset, test_size=0.25)

        _, _, _, _, meta_train, meta_test = split

        self.assertEqual(set(dataset.meta["ue_id"]), {0, 1})
        self.assertEqual(set(meta_test["ue_id"]), {0, 1})
        for ue_id, group in meta_test.groupby("ue_id"):
            train_group = meta_train[meta_train["ue_id"] == ue_id]
            self.assertGreater(group["target_step"].min(), train_group["target_step"].max())


if __name__ == "__main__":
    unittest.main()

