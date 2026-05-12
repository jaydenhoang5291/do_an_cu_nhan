import unittest

from ue import AerialUE


class AerialUETests(unittest.TestCase):
    def test_aerial_ue_accepts_heights_in_research_range(self):
        for height_m in (1.5, 50.0, 100.0, 150.0, 200.0, 300.0):
            with self.subTest(height_m=height_m):
                ue = AerialUE(
                    height_m=height_m,
                    speed_mps=12.5,
                    direction_rad=1.0,
                )

                self.assertEqual(ue.ue_type, "aerial")
                self.assertEqual(ue.height_m, height_m)
                self.assertEqual(ue.speed_mps, 12.5)
                self.assertEqual(ue.direction_rad, 1.0)

    def test_aerial_ue_rejects_height_outside_research_range(self):
        for height_m in (1.49, 300.01):
            with self.subTest(height_m=height_m):
                with self.assertRaises(ValueError):
                    AerialUE(height_m=height_m)


if __name__ == "__main__":
    unittest.main()
