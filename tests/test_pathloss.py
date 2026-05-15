import math
import unittest

from radio_models import (
    effective_uma_distances,
    uma_av_los_path_loss,
    uma_av_nlos_path_loss,
    uma_los_path_loss,
)


class UMaAVPathLossTests(unittest.TestCase):
    def test_uma_pathloss_uses_minimum_10m_2d_distance(self):
        fc = 2.0
        h_bs = 25.0
        h_ut = 1.5
        _, d3d_eff = effective_uma_distances(1.0, h_bs, h_ut)
        expected = 28.0 + 22.0 * math.log10(d3d_eff) + 20.0 * math.log10(fc)

        actual = uma_los_path_loss(1.0, 1.0, fc, h_bs=h_bs, h_UT=h_ut)

        self.assertAlmostEqual(actual, expected)

    def test_low_height_los_uses_uma_los_first_segment(self):
        d2d = 100.0
        d3d = math.hypot(d2d, 25.0 - 1.5)
        fc = 4.0
        expected = 28.0 + 22.0 * math.log10(d3d) + 20.0 * math.log10(fc)

        actual = uma_av_los_path_loss(d2d, d3d, fc, h_bs=25.0, h_UT=1.5)

        self.assertAlmostEqual(actual, expected)

    def test_low_height_nlos_uses_uma_nlos_with_los_floor(self):
        d2d = 1000.0
        d3d = math.hypot(d2d, 25.0 - 1.5)
        fc = 4.0
        h_UT = 1.5
        los_pl = uma_los_path_loss(d2d, d3d, fc, h_bs=25.0, h_UT=h_UT)
        nlos_prime = 13.54 + 39.08 * math.log10(d3d) + 20.0 * math.log10(fc) - 0.6 * (h_UT - 1.5)
        expected = max(los_pl, nlos_prime)

        actual = uma_av_nlos_path_loss(d2d, d3d, fc, h_bs=25.0, h_UT=h_UT)

        self.assertAlmostEqual(actual, expected)

    def test_aerial_height_los_uses_uma_av_los(self):
        d2d = 1490.0
        _, d3d = effective_uma_distances(d2d, 25.0, 100.0)
        fc = 4.0
        expected = 28.0 + 22.0 * math.log10(d3d) + 20.0 * math.log10(fc)

        actual = uma_av_los_path_loss(d2d, d3d, fc, h_bs=25.0, h_UT=100.0)

        self.assertAlmostEqual(actual, expected)

    def test_aerial_height_nlos_uses_uma_av_nlos(self):
        d2d = 1490.0
        fc = 4.0
        h_UT = 50.0
        _, d3d = effective_uma_distances(d2d, 25.0, h_UT)
        expected = (
            -17.5
            + (46.0 - 7.0 * math.log10(h_UT)) * math.log10(d3d)
            + 20.0 * math.log10(40.0 * math.pi * fc / 3.0)
        )

        actual = uma_av_nlos_path_loss(d2d, d3d, fc, h_bs=25.0, h_UT=h_UT)

        self.assertAlmostEqual(actual, expected)

    def test_nlos_above_100m_is_not_defined_by_requested_model(self):
        with self.assertRaises(ValueError):
            uma_av_nlos_path_loss(1000.0, 1000.0, 4.0, h_bs=25.0, h_UT=200.0)


if __name__ == "__main__":
    unittest.main()
