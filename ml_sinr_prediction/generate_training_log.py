from __future__ import annotations

import argparse
import sys

import matplotlib

matplotlib.use("Agg")

import config
from simulation import CellularNetworkReceivedPower
from utils import _read_float, _read_int


def _read_ue_height() -> float | None:
    use_aerial_ue = input("Use aerial UE height? (y/N): ").strip().lower().startswith("y")
    if not use_aerial_ue:
        return None

    while True:
        height = _read_float(
            f"UE height ({config.UE_HEIGHT_MIN_M:g}-{config.UE_HEIGHT_MAX_M:g} m, default 100): ",
            100.0,
        )
        if config.UE_HEIGHT_MIN_M <= height <= config.UE_HEIGHT_MAX_M:
            return height
        print(
            "Invalid UE height: outside the research range "
            f"({config.UE_HEIGHT_MIN_M:g}-{config.UE_HEIGHT_MAX_M:g} m). Please enter again."
        )


def _interactive_args() -> argparse.Namespace:
    rect_len = _read_float("Enter RECTANGLE LENGTH (m): ", 8000.0)
    rect_wid = _read_float("Enter RECTANGLE WIDTH (m): ", 5000.0)
    grid_spacing = _read_float("Enter road grid spacing (m, default 200): ", 200.0)
    num_ues = _read_int("Enter number of UEs: ", 5)
    ue_height = _read_ue_height()
    rows = _read_int("Enter number of data rows to generate: ", 3000)

    return argparse.Namespace(
        rows=rows,
        steps=None,
        num_ues=num_ues,
        rect_len=rect_len,
        rect_wid=rect_wid,
        grid_spacing=grid_spacing,
        ue_height=ue_height,
        ue_speed_kmh=None,
        seed=None,
    )


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Generate a long simulator CSV quickly without opening the animation UI."
    )
    parser.add_argument("--rows", type=int, default=3000, help="Number of CSV data rows to generate.")
    parser.add_argument("--steps", type=int, default=None, help="Deprecated alias: simulation steps. Rows = steps + 1.")
    parser.add_argument("--num-ues", type=int, default=5)
    parser.add_argument("--rect-len", type=float, default=8000.0)
    parser.add_argument("--rect-wid", type=float, default=5000.0)
    parser.add_argument("--grid-spacing", type=float, default=200.0)
    parser.add_argument("--ue-height", type=float, default=None)
    parser.add_argument("--ue-speed-kmh", type=float, default=None)
    parser.add_argument("--seed", type=int, default=None)
    parser.add_argument(
        "--no-interactive",
        action="store_true",
        help="Use CLI defaults without prompting when no other arguments are passed.",
    )
    cli_args = parser.parse_args()
    args = cli_args if cli_args.no_interactive or len(sys.argv) > 1 else _interactive_args()
    steps_arg = getattr(args, "steps", None)
    rows = int(steps_arg) + 1 if steps_arg is not None else int(args.rows)
    simulation_steps = max(0, rows - 1)

    sim = CellularNetworkReceivedPower(
        num_ues=args.num_ues,
        rect_len_m=args.rect_len,
        rect_wid_m=args.rect_wid,
        grid_spacing_m=args.grid_spacing,
        ue_height_m=args.ue_height,
        show_link_lines=False,
        fast_mode=True,
        seed=args.seed,
        ue_speed_kmh=args.ue_speed_kmh,
        enable_stop_and_go=False,
        simulation_steps=simulation_steps,
        enable_plot=False,
    )
    out_path = sim.run_headless(save_csv=True)
    print(f"Generated training log: {out_path}")


if __name__ == "__main__":
    main()
