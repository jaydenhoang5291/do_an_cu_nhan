"""
Generate headless simulation CSV files across the configured UE-height scenarios.
"""

from __future__ import annotations

import argparse
import random
from dataclasses import dataclass
from pathlib import Path

import matplotlib

matplotlib.use("Agg")

import config
from logger import SimulationLogger
from simulation import CellularNetworkReceivedPower


@dataclass(frozen=True)
class HeightScenario:
    name: str
    folder: str
    start_half_units: int
    end_half_units: int

    @property
    def candidate_heights(self) -> list[float]:
        return [half_units / 2.0 for half_units in range(self.start_half_units, self.end_half_units + 1)]


SCENARIOS = [
    HeightScenario("H0", "H0_1p5_13m", 3, 26),
    HeightScenario("H1", "H1_13_22p5m", 27, 45),
    HeightScenario("H2", "H2_22p5_100m", 46, 200),
    HeightScenario("H3", "H3_100_300m", 201, 600),
]


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Generate headless simulation CSV files for H0-H3 UE-height scenarios."
    )
    parser.add_argument("--runs-per-scenario", type=int, default=50)
    parser.add_argument("--seed", type=int, default=None)
    parser.add_argument("--output-root", default="data")
    parser.add_argument("--num-ues", type=int, default=1)
    parser.add_argument("--rect-len-m", type=float, default=8000.0)
    parser.add_argument("--rect-wid-m", type=float, default=5000.0)
    parser.add_argument("--grid-spacing-m", type=float, default=200.0)
    parser.add_argument("--random-area", action="store_true")
    parser.add_argument("--area-min-m", type=int, default=1000)
    parser.add_argument("--area-max-m", type=int, default=10000)
    return parser.parse_args()


def validate_args(args: argparse.Namespace) -> None:
    if args.runs_per_scenario <= 0:
        raise ValueError("--runs-per-scenario must be greater than 0")
    if args.num_ues <= 0:
        raise ValueError("--num-ues must be greater than 0")
    if args.grid_spacing_m <= 0:
        raise ValueError("--grid-spacing-m must be greater than 0")

    if args.random_area:
        if args.area_min_m <= 0:
            raise ValueError("--area-min-m must be greater than 0")
        if args.area_min_m >= args.area_max_m:
            raise ValueError("--area-min-m must be smaller than --area-max-m")
        if args.area_max_m < 2000:
            raise ValueError("--area-max-m must be at least 2000 for random-area generation")
        if max(args.area_min_m, 1500) > min(args.area_max_m - 1, 9000):
            raise ValueError("random-area width range is empty; increase --area-max-m or lower --area-min-m")
    else:
        if args.rect_wid_m <= 0 or args.rect_len_m <= 0:
            raise ValueError("--rect-len-m and --rect-wid-m must be greater than 0")
        if args.rect_len_m <= args.rect_wid_m:
            raise ValueError("--rect-len-m must be greater than --rect-wid-m")
        if args.rect_len_m > 10000 or args.rect_wid_m > 10000:
            raise ValueError("--rect-len-m and --rect-wid-m must not exceed 10000 m")


def choose_area(args: argparse.Namespace, rng: random.Random) -> tuple[float, float]:
    if not args.random_area:
        return float(args.rect_len_m), float(args.rect_wid_m)

    width_min = max(args.area_min_m, 1500)
    width_max = min(args.area_max_m - 1, 9000)
    rect_wid_m = rng.randint(width_min, width_max)

    length_min = max(rect_wid_m + 1, 2000)
    rect_len_m = rng.randint(length_min, args.area_max_m)
    return float(rect_len_m), float(rect_wid_m)


def assert_height_matches_scenario(height_m: float, scenario: HeightScenario) -> None:
    folder = SimulationLogger._height_output_folder(height_m)
    if folder != scenario.folder:
        raise AssertionError(
            f"Selected height {height_m:g} m maps to {folder}, expected {scenario.folder}"
        )


def run_one_simulation(
    *,
    args: argparse.Namespace,
    height_m: float,
    rect_len_m: float,
    rect_wid_m: float,
    sim_seed: int,
) -> str:
    sim = CellularNetworkReceivedPower(
        num_ues=args.num_ues,
        rect_len_m=rect_len_m,
        rect_wid_m=rect_wid_m,
        grid_spacing_m=args.grid_spacing_m,
        ue_height_m=height_m,
        fast_mode=True,
        show_link_lines=False,
        headless=True,
        seed=sim_seed,
    )
    sim.output_root = args.output_root
    sim.run_animation()

    filepath = getattr(sim.logger, "last_saved_csv_path", None)
    if not filepath:
        raise RuntimeError("Simulation finished but logger did not report a saved CSV path")
    if not Path(filepath).exists():
        raise RuntimeError(f"Simulation reported a CSV path that does not exist: {filepath}")
    return filepath


def main() -> None:
    args = parse_args()
    validate_args(args)

    rng = random.Random(args.seed)
    output_root = Path(args.output_root)
    print(f"Output root: {output_root}")
    print(f"Runs per scenario: {args.runs_per_scenario}")
    print(f"Random area: {'on' if args.random_area else 'off'}")

    total_runs = args.runs_per_scenario * len(SCENARIOS)
    completed = 0

    for scenario in SCENARIOS:
        candidates = scenario.candidate_heights
        print(
            f"\nScenario {scenario.name} -> {scenario.folder} "
            f"({len(candidates)} candidate heights, {candidates[0]:g}-{candidates[-1]:g} m)"
        )

        for run_idx in range(1, args.runs_per_scenario + 1):
            completed += 1
            height_m = rng.choice(candidates)
            assert_height_matches_scenario(height_m, scenario)
            rect_len_m, rect_wid_m = choose_area(args, rng)
            if not (rect_len_m > rect_wid_m):
                raise AssertionError("Generated area is invalid: rect_len_m must be greater than rect_wid_m")

            sim_seed = rng.randrange(0, 2**32)
            print(
                f"[{completed}/{total_runs}] {scenario.name} "
                f"run {run_idx}/{args.runs_per_scenario}: "
                f"height={height_m:g} m, rect_len={rect_len_m:g} m, rect_wid={rect_wid_m:g} m"
            )
            filepath = run_one_simulation(
                args=args,
                height_m=height_m,
                rect_len_m=rect_len_m,
                rect_wid_m=rect_wid_m,
                sim_seed=sim_seed,
            )
            print(f"Saved CSV: {filepath}")

    print("\nDone.")


if __name__ == "__main__":
    try:
        main()
    except ValueError as exc:
        raise SystemExit(f"Error: {exc}")
