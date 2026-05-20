from __future__ import annotations

import argparse
from pathlib import Path

import numpy as np
import pandas as pd

from .common import RAW_DATA_PATH, display_path


def main() -> None:
    parser = argparse.ArgumentParser(
        description=(
            "Generate a small dummy simulation_log.csv for pipeline testing only. "
            "Do not use this synthetic data as research results."
        )
    )
    parser.add_argument("--rows", type=int, default=240)
    parser.add_argument("--num-bs", type=int, default=4)
    parser.add_argument("--output", type=str, default=str(RAW_DATA_PATH))
    args = parser.parse_args()

    if args.rows < 30:
        raise ValueError("rows should be at least 30 so history/horizon splitting has enough samples.")
    if args.num_bs < 1:
        raise ValueError("num-bs must be at least 1.")

    rng = np.random.default_rng(42)
    t = np.arange(args.rows)
    x = 50 + 1.8 * t
    y = 120 + 30 * np.sin(t / 28.0)
    z = np.full(args.rows, 1.5)
    vx = np.gradient(x)
    vy = np.gradient(y)
    vz = np.zeros(args.rows)

    data = {
        "timestep": t,
        "ue_id": np.zeros(args.rows, dtype=int),
        "x": x,
        "y": y,
        "z": z,
        "vx": vx,
        "vy": vy,
        "vz": vz,
    }

    for bs_idx in range(args.num_bs):
        bs_x = 120 + bs_idx * 180
        bs_y = 100 + (bs_idx % 2) * 160
        dist = np.sqrt((x - bs_x) ** 2 + (y - bs_y) ** 2 + 25**2)
        rsrp = -55 - 20 * np.log10(dist / 10.0) + rng.normal(0, 1.2, args.rows)
        interference = 8 + 2.5 * np.sin(t / (18.0 + bs_idx))
        sinr = rsrp + 95 - interference + rng.normal(0, 0.8, args.rows)
        data[f"rsrp_bs_{bs_idx}"] = rsrp
        data[f"sinr_bs_{bs_idx}"] = sinr

    out_path = Path(args.output)
    out_path.parent.mkdir(parents=True, exist_ok=True)
    pd.DataFrame(data).to_csv(out_path, index=False)
    print(f"Generated dummy data for pipeline testing only: {display_path(out_path)}")


if __name__ == "__main__":
    main()
