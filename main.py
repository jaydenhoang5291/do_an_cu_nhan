"""
Main entry point: reads user input, initializes the cellular network
simulation, and displays the animation window.
"""

import matplotlib.pyplot as plt

import config
from simulation import CellularNetworkReceivedPower
from utils import _read_float, _read_int

if __name__ == "__main__":
    L = _read_float("Enter RECTANGLE LENGTH (m): ", 8000.0)
    W = _read_float("Enter RECTANGLE WIDTH (m): ", 5000.0)
    grid_sp = _read_float("Enter road grid spacing (m, default 200): ", 200.0)
    num_ues = _read_int("Enter number of UEs: ", 5)

    use_aerial_ue = input("Use aerial UE height? (y/N): ").strip().lower().startswith('y')
    ue_height = None
    if use_aerial_ue:
        allowed = "/".join(f"{h:g}" for h in config.AERIAL_UE_HEIGHTS_M)
        ue_height = _read_float(f"Aerial UE height ({allowed} m, default 100): ", 100.0)

    show_lines = input("Show UE - BS connection lines? (y/N): ").strip().lower().startswith('y')
    sim = CellularNetworkReceivedPower(
        num_ues=num_ues,
        rect_len_m=L,
        rect_wid_m=W,
        grid_spacing_m=grid_sp,
        ue_height_m=ue_height,
        show_link_lines=show_lines,
        fast_mode=True
    )
    sim.run_animation()
    plt.ioff()
    plt.show()
