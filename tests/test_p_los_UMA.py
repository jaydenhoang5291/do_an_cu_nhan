import math
import unittest

from radio_models import uma_av_los_probability


class UMaAVLosProbabilityTests(unittest.TestCase):
    def test_uma_height_1_5_to_22_5_uses_low_height_formula(self):
        h_ut = 20.0
        d2d = 100.0
        c_hut = ((h_ut - 13.0) / 10.0) ** 1.5
        base = (18.0 / d2d) + math.exp(-d2d / 63.0) * (1.0 - 18.0 / d2d)
        height_gain = 1.0 + c_hut * (5.0 / 4.0) * (d2d / 100.0) ** 3 * math.exp(-d2d / 150.0)
        expected = base * height_gain

        actual = uma_av_los_probability(d2d, h_ut)

        self.assertAlmostEqual(actual, expected)

    def test_uma_height_at_1_5_is_valid(self):
        self.assertEqual(uma_av_los_probability(10.0, 1.5), 1.0)

    def test_height_above_100_has_full_los_probability(self):
        self.assertEqual(uma_av_los_probability(1000.0, 200.0), 1.0)
        self.assertEqual(uma_av_los_probability(1000.0, 300.0), 1.0)

    def test_distance_below_d1_has_full_los_probability(self):
        self.assertEqual(uma_av_los_probability(10.0, 50.0), 1.0)

    def test_formula_case(self):
        h_ut = 50.0
        d2d = 100.0

        d1 = max(460.0 * math.log10(h_ut) - 700.0, 18.0)
        p1 = 4300.0 * math.log10(h_ut) - 3800.0
        expected = (d1 / d2d) + math.exp(-d2d / p1) * (1.0 - d1 / d2d)

        actual = uma_av_los_probability(d2d, h_ut)

        self.assertAlmostEqual(actual, expected)

    def test_invalid_height_raises_error(self):
        with self.assertRaises(ValueError):
            uma_av_los_probability(100.0, 1.4)

        with self.assertRaises(ValueError):
            uma_av_los_probability(100.0, 301.0)


if __name__ == "__main__":
    unittest.main()
