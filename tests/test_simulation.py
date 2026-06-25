import unittest
from unittest.mock import patch
import math
import os
import tempfile

import matplotlib

matplotlib.use("Agg")

import config
from simulation import CellularNetworkReceivedPower


class SimulationBSTests(unittest.TestCase):
    def test_only_ground_bs_nodes_are_deployed(self):
        sim = CellularNetworkReceivedPower(
            num_ues=1,
            rect_len_m=1000.0,
            rect_wid_m=1000.0,
            ue_height_m=100.0,
            fast_mode=True,
            show_link_lines=False,
            seed=1,
        )

        self.assertTrue(all(height == sim.h_bs for height in sim.bs_heights))
        self.assertEqual(sim.get_ue_height_m(0), 100.0)

    def test_aerial_ue_uses_los_probability_with_ground_bs(self):
        sim = CellularNetworkReceivedPower(
            num_ues=1,
            rect_len_m=1000.0,
            rect_wid_m=1000.0,
            ue_height_m=100.0,
            fast_mode=True,
            show_link_lines=False,
            seed=1,
        )

        ue_x, ue_y = sim.ue_positions[0]
        _, _, los_probability = sim.radio_model.calculate_path_loss(ue_x, ue_y, 0, 0)

        self.assertGreaterEqual(los_probability, 0.0)
        self.assertLessEqual(los_probability, 1.0)

    def test_height_profile_axis_is_created(self):
        sim = CellularNetworkReceivedPower(
            num_ues=2,
            rect_len_m=1000.0,
            rect_wid_m=1000.0,
            ue_height_m=200.0,
            fast_mode=True,
            show_link_lines=False,
            seed=1,
        )

        self.assertEqual(sim.height_ax.get_title(), "UE height above ground")
        self.assertEqual(sim.height_ax.get_ylabel(), "Height (m)")

    def test_bs_power_table_has_serving_and_six_neighbors(self):
        sim = CellularNetworkReceivedPower(
            num_ues=1,
            rect_len_m=2000.0,
            rect_wid_m=2000.0,
            fast_mode=True,
            show_link_lines=False,
            seed=1,
        )

        sim.update(0)

        self.assertEqual(len(sim.bs_power_table_rows), 7)
        self.assertTrue(any(row[2] for row in sim.bs_power_table_rows))
        rsrp_values = [row[1] for row in sim.bs_power_table_rows]
        self.assertEqual(rsrp_values, sorted(rsrp_values, reverse=True))

    def test_interference_uses_six_nearest_bs_around_serving_bs(self):
        sim = CellularNetworkReceivedPower(
            num_ues=1,
            rect_len_m=2000.0,
            rect_wid_m=2000.0,
            fast_mode=True,
            show_link_lines=False,
            seed=1,
        )

        serving_bs = len(sim.bs_positions) // 2
        sx, sy = sim.bs_positions[serving_bs]
        expected = sorted(
            (
                (
                    bs_idx,
                    math.hypot(bs_x - sx, bs_y - sy),
                )
                for bs_idx, (bs_x, bs_y) in enumerate(sim.bs_positions)
                if bs_idx != serving_bs
            ),
            key=lambda item: item[1],
        )[:6]

        self.assertEqual(
            sim.radio_model.get_interfering_bs_indices(serving_bs),
            [bs_idx for bs_idx, _ in expected],
        )

    def test_rsrp_from_received_power_uses_lte_10mhz_approximation(self):
        sim = CellularNetworkReceivedPower(
            num_ues=1,
            rect_len_m=1000.0,
            rect_wid_m=1000.0,
            fast_mode=True,
            show_link_lines=False,
            seed=1,
        )

        prx_dbm = config.PTX
        expected = prx_dbm - 10.0 * math.log10(
            config.LTE_N_RB * config.LTE_N_SUBCARRIERS_PER_RB
        )

        self.assertAlmostEqual(sim.radio_model.calculate_rsrp_from_received_power(prx_dbm), expected)
        self.assertAlmostEqual(expected, 18.22, places=2)

    def test_rsrp_is_lower_than_total_received_power_by_resource_grid_factor(self):
        sim = CellularNetworkReceivedPower(
            num_ues=1,
            rect_len_m=1000.0,
            rect_wid_m=1000.0,
            fast_mode=True,
            show_link_lines=False,
            seed=1,
        )

        path_loss_db = 100.0
        prx = sim.radio_model.calculate_total_received_power(0, path_loss_db)
        rsrp = sim.radio_model.calculate_rsrp(0, path_loss_db)
        expected_delta = 10.0 * math.log10(
            config.LTE_N_RB * config.LTE_N_SUBCARRIERS_PER_RB
        )

        self.assertAlmostEqual(prx - rsrp, expected_delta)

    def test_link_quality_color_thresholds(self):
        self.assertEqual(CellularNetworkReceivedPower.link_quality_color(-69.9), '#2ca02c')
        self.assertEqual(CellularNetworkReceivedPower.link_quality_color(-70.0), '#ffbf00')
        self.assertEqual(CellularNetworkReceivedPower.link_quality_color(-90.0), '#ffbf00')
        self.assertEqual(CellularNetworkReceivedPower.link_quality_color(-90.1), '#d62728')
        self.assertEqual(CellularNetworkReceivedPower.link_quality_color(None), '#8c8c8c')

    def test_sampling_interval_uses_config(self):
        sim = CellularNetworkReceivedPower(
            num_ues=1,
            rect_len_m=1000.0,
            rect_wid_m=1000.0,
            fast_mode=True,
            show_link_lines=False,
            seed=1,
        )

        self.assertEqual(sim.steps, config.SIMULATION_STEPS)
        self.assertEqual(sim.time_per_step, config.TIME_PER_STEP_S)

    def test_ue_height_research_range_is_validated(self):
        sim = CellularNetworkReceivedPower(
            num_ues=1,
            rect_len_m=1000.0,
            rect_wid_m=1000.0,
            ue_height_m=150.0,
            fast_mode=True,
            show_link_lines=False,
            seed=1,
        )

        self.assertEqual(sim.ue_height_m, 150.0)

        for height_m in (1.49, 300.01):
            with self.subTest(height_m=height_m):
                with self.assertRaises(ValueError):
                    CellularNetworkReceivedPower(
                        num_ues=1,
                        rect_len_m=1000.0,
                        rect_wid_m=1000.0,
                        ue_height_m=height_m,
                        fast_mode=True,
                        show_link_lines=False,
                        seed=1,
                    )

    def test_ue_speed_range_depends_on_height(self):
        cases = [
            (1.5, 20.0, 60.0),
            (22.5, 20.0, 60.0),
            (50.0, 40.0, 100.0),
            (200.0, 80.0, 160.0),
        ]

        for height_m, expected_min, expected_max in cases:
            with self.subTest(height_m=height_m):
                sim = CellularNetworkReceivedPower(
                    num_ues=3,
                    rect_len_m=1000.0,
                    rect_wid_m=1000.0,
                    ue_height_m=height_m,
                    fast_mode=True,
                    show_link_lines=False,
                    seed=1,
                )

                self.assertEqual(sim.ue_speed_min_kmh, expected_min)
                self.assertEqual(sim.ue_speed_max_kmh, expected_max)
                self.assertTrue(all(expected_min <= speed <= expected_max for speed in sim.ue_speeds))

    def test_multi_ue_log_includes_each_ue_speed_column(self):
        sim = CellularNetworkReceivedPower(
            num_ues=3,
            rect_len_m=1000.0,
            rect_wid_m=1000.0,
            ue_height_m=50.0,
            fast_mode=True,
            show_link_lines=False,
            seed=1,
        )

        for ue_idx in range(3):
            self.assertIn(f'ue{ue_idx}_speed', sim.data_log)

    def test_single_ue_csv_filename_includes_height_and_speed(self):
        sim = CellularNetworkReceivedPower(
            num_ues=1,
            rect_len_m=1000.0,
            rect_wid_m=1000.0,
            ue_height_m=50.0,
            fast_mode=True,
            show_link_lines=False,
            seed=1,
        )
        sim.ue_speeds[0] = 88.5
        sim.logger.log_step(0)
        sim.logger.log_ue_data(
            0,
            x=0.0,
            y=0.0,
            bs=None,
            sinr=None,
            handover_flag=0,
            neighbors=[],
        )

        old_cwd = os.getcwd()
        with tempfile.TemporaryDirectory() as tmpdir:
            try:
                os.chdir(tmpdir)
                sim.logger.save_data_to_csv()
                files = os.listdir(os.path.join(tmpdir, "data"))
            finally:
                os.chdir(old_cwd)

        self.assertEqual(len(files), 1)
        self.assertIn("_1_UE_h50m_v88p5kmh_Data.csv", files[0])

    def test_los_probability_and_state_are_logged(self):
        sim = CellularNetworkReceivedPower(
            num_ues=1,
            rect_len_m=1000.0,
            rect_wid_m=1000.0,
            ue_height_m=100.0,
            fast_mode=True,
            show_link_lines=False,
            seed=1,
        )

        sim.update(0)

        self.assertIn('ue0_los_probability', sim.data_log)
        self.assertIn('ue0_los_state', sim.data_log)
        self.assertIsNotNone(sim.data_log['ue0_los_probability'][0])
        self.assertIn(sim.data_log['ue0_los_state'][0], ('LOS', 'NLOS'))

    def test_los_state_is_sampled_from_probability(self):
        sim = CellularNetworkReceivedPower(
            num_ues=1,
            rect_len_m=1000.0,
            rect_wid_m=1000.0,
            ue_height_m=50.0,
            fast_mode=True,
            show_link_lines=False,
            seed=1,
        )

        with patch("radio_models.np.random.random", return_value=0.2):
            los = sim.radio_model._link_los_state(0, 0, 0.25)

        self.assertTrue(los)

        with patch("radio_models.np.random.random", return_value=0.3):
            los = sim.radio_model._link_los_state(0, 1, 0.25)

        self.assertFalse(los)

    def test_cached_los_state_is_not_resampled(self):
        sim = CellularNetworkReceivedPower(
            num_ues=1,
            rect_len_m=1000.0,
            rect_wid_m=1000.0,
            ue_height_m=50.0,
            fast_mode=True,
            show_link_lines=False,
            seed=1,
        )

        sim.sf_cache[(0, 0)] = {'los': True}

        with patch("radio_models.np.random.random") as random_mock:
            los = sim.radio_model._link_los_state(0, 0, 0.0)

        self.assertTrue(los)
        random_mock.assert_not_called()

    def test_los_probability_one_is_always_los(self):
        sim = CellularNetworkReceivedPower(
            num_ues=1,
            rect_len_m=1000.0,
            rect_wid_m=1000.0,
            ue_height_m=200.0,
            fast_mode=True,
            show_link_lines=False,
            seed=1,
        )

        ue_x, ue_y = sim.ue_positions[0]
        _, los, _ = sim.radio_model.calculate_path_loss(ue_x, ue_y, 0, 0)

        self.assertTrue(los)


if __name__ == "__main__":
    unittest.main()
