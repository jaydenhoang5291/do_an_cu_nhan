import unittest

import matplotlib

matplotlib.use("Agg")

from simulation import CellularNetworkReceivedPower


class SimulationBSTests(unittest.TestCase):
    def test_uav_bs_nodes_are_not_deployed(self):
        sim = CellularNetworkReceivedPower(
            num_ues=1,
            rect_len_m=1000.0,
            rect_wid_m=1000.0,
            add_uav_cover=True,
            ue_height_m=100.0,
            fast_mode=True,
            show_link_lines=False,
            seed=1,
        )

        self.assertFalse(any(sim.bs_is_uav))
        self.assertTrue(all(height == sim.h_bs for height in sim.bs_heights))
        self.assertTrue(sim.is_aerial_ue)

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

        self.assertFalse(sim.bs_is_uav[0])
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

        self.assertEqual(sim.height_ax.get_title(), "Height Profile")
        self.assertEqual(sim.height_ax.get_ylabel(), "Height (m)")

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

    def test_los_state_uses_probability_threshold(self):
        sim = CellularNetworkReceivedPower(
            num_ues=1,
            rect_len_m=1000.0,
            rect_wid_m=1000.0,
            ue_height_m=50.0,
            fast_mode=True,
            show_link_lines=False,
            seed=1,
        )

        ue_x, ue_y = sim.ue_positions[0]
        _, los, p_los = sim.radio_model.calculate_path_loss(ue_x, ue_y, 0, 0)

        self.assertEqual(los, p_los >= 0.5)

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
