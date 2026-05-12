import unittest
from unittest.mock import patch

import matplotlib

matplotlib.use("Agg")

from radio_models import uma_av_shadow_fading_sigma
from simulation import CellularNetworkReceivedPower


class UMaAVShadowFadingTests(unittest.TestCase):
    def test_aerial_los_sigma_uses_height_dependent_model(self):
        actual = uma_av_shadow_fading_sigma(True, 100.0)

        self.assertAlmostEqual(actual, 4.64 * 2.718281828459045 ** (-0.0066 * 100.0))

    def test_aerial_nlos_sigma_is_constant(self):
        self.assertEqual(uma_av_shadow_fading_sigma(False, 50.0), 6.0)

    def test_shadow_fading_is_kept_before_25m(self):
        sim = CellularNetworkReceivedPower(
            num_ues=1,
            rect_len_m=1000.0,
            rect_wid_m=1000.0,
            ue_height_m=100.0,
            fast_mode=True,
            show_link_lines=False,
            seed=1,
        )

        with patch("radio_models.np.random.normal", side_effect=[1.0, 2.0]):
            first = sim.radio_model.shadow_fading(0, 0, True, (5000.0, 5000.0))
            second = sim.radio_model.shadow_fading(0, 0, True, (5024.0, 5000.0))

        self.assertEqual(first, 1.0)
        self.assertEqual(second, 1.0)

    def test_shadow_fading_is_regenerated_after_25m(self):
        sim = CellularNetworkReceivedPower(
            num_ues=1,
            rect_len_m=1000.0,
            rect_wid_m=1000.0,
            ue_height_m=100.0,
            fast_mode=True,
            show_link_lines=False,
            seed=1,
        )

        with patch("radio_models.np.random.normal", side_effect=[1.0, 2.0]):
            first = sim.radio_model.shadow_fading(0, 0, True, (5000.0, 5000.0))
            second = sim.radio_model.shadow_fading(0, 0, True, (5025.0, 5000.0))

        self.assertEqual(first, 1.0)
        self.assertEqual(second, 2.0)

    def test_shadow_fading_is_clipped_to_three_sigma(self):
        sim = CellularNetworkReceivedPower(
            num_ues=1,
            rect_len_m=1000.0,
            rect_wid_m=1000.0,
            fast_mode=True,
            show_link_lines=False,
            seed=1,
        )

        sigma = sim.sf_sigma['LOS']
        with patch("radio_models.np.random.normal", return_value=10.0 * sigma):
            high = sim.radio_model.shadow_fading(0, 0, True, (5000.0, 5000.0))

        with patch("radio_models.np.random.normal", return_value=-10.0 * sigma):
            low = sim.radio_model.shadow_fading(0, 1, True, (5000.0, 5000.0))

        self.assertEqual(high, 3.0 * sigma)
        self.assertEqual(low, -3.0 * sigma)

    def test_aerial_los_state_is_fixed_after_initial_determination(self):
        sim = CellularNetworkReceivedPower(
            num_ues=1,
            rect_len_m=1000.0,
            rect_wid_m=1000.0,
            ue_height_m=50.0,
            fast_mode=True,
            show_link_lines=False,
            seed=1,
        )

        bs_x, bs_y = sim.bs_positions[0]
        with patch("radio_models.np.random.random", return_value=0.99):
            _, initial_los, initial_p_los = sim.radio_model.calculate_path_loss(bs_x + 4000.0, bs_y, 0, 0)
            _, later_los, later_p_los = sim.radio_model.calculate_path_loss(bs_x, bs_y, 0, 0)

        self.assertLess(initial_p_los, 0.5)
        self.assertEqual(initial_los, later_los)
        self.assertEqual(later_los, False)
        self.assertEqual(later_p_los, 1.0)


if __name__ == "__main__":
    unittest.main()
