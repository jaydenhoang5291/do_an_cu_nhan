import math
import unittest

from radio_models import (
    uma_av_los_path_loss,
    uma_av_nlos_path_loss,
    uma_los_path_loss,
)


class UMaAVPathLossTests(unittest.TestCase):
    def test_low_height_los_uses_uma_los_first_segment(self):
        d2d = 100.0
        d3d = math.hypot(d2d, 25.0 - 1.5)
        fc = 4.0
        expected = 28.0 + 22.0 * math.log10(d3d) + 20.0 * math.log10(fc)

        actual = uma_av_los_path_loss(d2d, d3d, fc, h_bs=25.0, h_ut=1.5)

        self.assertAlmostEqual(actual, expected)

    def test_low_height_nlos_uses_uma_nlos_with_los_floor(self):
        d2d = 1000.0
        d3d = math.hypot(d2d, 25.0 - 1.5)
        fc = 4.0
        h_ut = 1.5
        los_pl = uma_los_path_loss(d2d, d3d, fc, h_bs=25.0, h_ut=h_ut)
        nlos_prime = 13.54 + 39.08 * math.log10(d3d) + 20.0 * math.log10(fc) - 0.6 * (h_ut - 1.5)
        expected = max(los_pl, nlos_prime)

        actual = uma_av_nlos_path_loss(d2d, d3d, fc, h_bs=25.0, h_ut=h_ut)

        self.assertAlmostEqual(actual, expected)

    def test_aerial_height_los_uses_uma_av_los(self):
        d3d = 1500.0
        fc = 4.0
        expected = 28.0 + 22.0 * math.log10(d3d) + 20.0 * math.log10(fc)

        actual = uma_av_los_path_loss(1490.0, d3d, fc, h_bs=25.0, h_ut=100.0)

        self.assertAlmostEqual(actual, expected)

    def test_aerial_height_nlos_uses_uma_av_nlos(self):
        d3d = 1500.0
        fc = 4.0
        h_ut = 50.0
        expected = (
            -17.5
            + (46.0 - 7.0 * math.log10(h_ut)) * math.log10(d3d)
            + 20.0 * math.log10(40.0 * math.pi * fc / 3.0)
        )

        actual = uma_av_nlos_path_loss(1490.0, d3d, fc, h_bs=25.0, h_ut=h_ut)

        self.assertAlmostEqual(actual, expected)

    def test_nlos_above_100m_is_not_defined_by_requested_model(self):
        with self.assertRaises(ValueError):
            uma_av_nlos_path_loss(1000.0, 1000.0, 4.0, h_bs=25.0, h_ut=200.0)


if __name__ == "__main__":
    unittest.main()
