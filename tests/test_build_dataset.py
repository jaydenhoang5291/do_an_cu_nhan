import unittest

import pandas as pd

from ml_sinr_prediction import build_dataset


class BuildDatasetFeatureTests(unittest.TestCase):
    def test_default_horizon_predicts_next_step(self):
        self.assertEqual(build_dataset.DEFAULT_HORIZON, 1)

    def test_simulator_feature_detection_keeps_only_relevant_inputs(self):
        df = pd.DataFrame(
            {
                "Step": [0, 1],
                "ue0_x": [1.0, 2.0],
                "ue0_y": [3.0, 4.0],
                "ue0_height": [1.5, 1.5],
                "ue0_direction": [0, 0],
                "ue0_connected_bs": [1, 1],
                "ue0_los_probability": [1.0, 1.0],
                "ue0_los_state": ["LOS", "LOS"],
                "ue0_pathloss": [90.0, 91.0],
                "ue0_shadow_fading": [0.1, 0.2],
                "ue0_rsrp": [-70.0, -71.0],
                "ue0_prx": [-42.0, -43.0],
                "ue0_sinr": [10.0, 9.0],
                "ue0_speed": [40.0, 40.0],
                "ue0_handover": [0, 0],
                "ue0_bs1_idx": [2, 2],
                "ue0_bs1_rsrp": [-80.0, -79.0],
            }
        )

        self.assertEqual(
            build_dataset.detect_feature_cols(df),
            [
                "ue0_bs1_rsrp",
                "ue0_connected_bs",
                "ue0_direction",
                "ue0_height",
                "ue0_los_probability",
                "ue0_pathloss",
                "ue0_rsrp",
                "ue0_shadow_fading",
                "ue0_sinr",
                "ue0_x",
                "ue0_y",
            ],
        )


if __name__ == "__main__":
    unittest.main()
