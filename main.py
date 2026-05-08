"""
Main entry point: reads user input, initializes the cellular network
simulation, and displays the animation window.
"""

import matplotlib.pyplot as plt

from simulation import CellularNetworkReceivedPower
from utils import _read_float, _read_int

if __name__ == "__main__":
    L = _read_float("Enter RECTANGLE LENGTH (m): ", 8000.0)
    W = _read_float("Enter RECTANGLE WIDTH (m): ", 5000.0)
    num_ues = _read_int("Enter number of UEs: ", 5)
    grid_sp = _read_float("Enter road grid spacing (m, default 200): ", 200.0)

    add_uav = input("Add UAV-BS coverage? (y/N): ").strip().lower().startswith('y')
    uav_R = _read_float("UAV radius (m, default 750): ", 750.0) if add_uav else 750.0
    uav_h = _read_float("UAV altitude (m, default 100): ", 100.0) if add_uav else 100.0
    uav_ptx = _read_float("UAV transmit power Ptx (dBm, default 40): ", 40.0) if add_uav else 40.0

    show_lines = input("Show UE - BS connection lines? (y/N): ").strip().lower().startswith('y')
    sim = CellularNetworkReceivedPower(
        num_ues=num_ues,
        rect_len_m=L,
        rect_wid_m=W,
         grid_spacing_m=grid_sp,
        add_uav_cover=add_uav,
        uav_radius_m=uav_R,
        uav_altitude_m=uav_h,
        uav_ptx_dbm=uav_ptx,
        show_link_lines=show_lines,
        fast_mode=True
    )
    sim.run_animation()
    plt.ioff()
    plt.show()
