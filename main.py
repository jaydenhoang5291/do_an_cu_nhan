"""
Main entry point: reads user input, initializes the cellular network
simulation, and displays the animation window.
"""

import argparse

import config
from utils import _read_float, _read_int


def _read_ue_height_in_range(prompt: str, default: float) -> float:
    while True:
        height = _read_float(prompt, default)
        if config.UE_HEIGHT_MIN_M <= height <= config.UE_HEIGHT_MAX_M:
            return height
        print(
            "Invalid UE height: outside the research range "
            f"({config.UE_HEIGHT_MIN_M:g}-{config.UE_HEIGHT_MAX_M:g} m). Please enter again."
        )


def _parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run the cellular network simulation.")
    parser.add_argument(
        "--headless",
        action="store_true",
        help="Prompt for simulation parameters, then save CSV data without opening the matplotlib UI.",
    )
    return parser.parse_args()


def _build_simulation_params(headless: bool) -> dict:
    L = _read_float("Enter RECTANGLE LENGTH (m): ", 8000.0)
    W = _read_float("Enter RECTANGLE WIDTH (m): ", 5000.0)
    grid_sp = _read_float("Enter road grid spacing (m, default 200): ", 200.0)
    num_ues = _read_int("Enter number of UEs: ", 5)

    ue_height = _read_ue_height_in_range(
        f"UE height ({config.UE_HEIGHT_MIN_M:g}-{config.UE_HEIGHT_MAX_M:g} m, default {config.H_UT:g}): ",
        config.H_UT,
    )

    show_lines = False
    if not headless:
        show_lines = input("Show UE - BS connection lines? (y/N): ").strip().lower().startswith('y')
    return {
        "num_ues": num_ues,
        "rect_len_m": L,
        "rect_wid_m": W,
        "grid_spacing_m": grid_sp,
        "ue_height_m": ue_height,
        "show_link_lines": show_lines,
        "fast_mode": True,
        "headless": headless,
    }


if __name__ == "__main__":
    args = _parse_args()
    if args.headless:
        import matplotlib

        matplotlib.use("Agg")

    from simulation import CellularNetworkReceivedPower

    sim = CellularNetworkReceivedPower(**_build_simulation_params(args.headless))
    sim.run_animation()
    if not args.headless:
        import matplotlib.pyplot as plt

        plt.ioff()
        plt.show()
