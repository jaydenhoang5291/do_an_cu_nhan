"""
Generate a StormSIM-ready 7-gNB measurement CSV.

The topology is one center gNB plus its six first-tier hexagonal neighbors.
The output keeps stable gNB indices in columns gnb0_rsrp ... gnb6_rsrp so
StormSIM can map gnbN to config ID 00000N.
"""

from __future__ import annotations

import argparse
import csv
import math
from pathlib import Path

import numpy as np

import config
from simulation import CellularNetworkReceivedPower


DEFAULT_OUTPUT = (
    Path(__file__).resolve().parents[1]
    / "StormSIM-private"
    / "data"
    / "stormsim_7gnb_measurement.csv"
)


def _config_id(gnb_idx: int) -> str:
    return f"{gnb_idx:06d}"


def _build_sim(args: argparse.Namespace) -> CellularNetworkReceivedPower:
    sim = CellularNetworkReceivedPower(
        num_ues=1,
        rect_len_m=args.rect_len,
        rect_wid_m=args.rect_wid,
        grid_spacing_m=args.grid_spacing,
        ue_height_m=args.ue_height,
        fast_mode=True,
        show_link_lines=False,
        headless=True,
        seed=args.seed,
    )

    if len(sim.bs_positions) != 7:
        raise RuntimeError(
            f"Expected exactly 7 gNBs for {args.rect_len:g} m x {args.rect_wid:g} m, "
            f"got {len(sim.bs_positions)}. Use the default 1500 m x 1500 m region."
        )

    # The generic simulator returns q/r loop order. Reorder here so gnb0 is the
    # center gNB and gnb1..gnb6 are the surrounding first-tier cells in angular order.
    center = sim.center
    indexed = []
    for bs_idx, (x, y) in enumerate(sim.bs_positions):
        dx = float(x) - center
        dy = float(y) - center
        distance = math.hypot(dx, dy)
        angle = math.atan2(dy, dx)
        indexed.append((distance, angle, bs_idx))

    center_idx = min(indexed, key=lambda item: item[0])[2]
    neighbor_indices = [
        bs_idx
        for _, _, bs_idx in sorted(
            (item for item in indexed if item[2] != center_idx),
            key=lambda item: item[1],
        )
    ]
    order = [center_idx, *neighbor_indices]

    sim.bs_positions = [sim.bs_positions[i] for i in order]
    sim.bs_heights = [sim.bs_heights[i] for i in order]
    sim.bs_ptx = [sim.bs_ptx[i] for i in order]
    sim.ue_serving_bs = [None]
    sim.previous_serving_bs = [None]
    sim.sf_cache = {}
    return sim


def _hex_loop_waypoints(sim: CellularNetworkReceivedPower) -> list[tuple[float, float]]:
    xmin, xmax = sim.rect_xmin, sim.rect_xmax
    ymin, ymax = sim.rect_ymin, sim.rect_ymax
    cx, cy = sim.center, sim.center
    pad = 20.0
    return [
        (xmin + pad, cy - 0.50 * sim.height),
        (cx, ymin + pad),
        (xmax - pad, cy - 0.50 * sim.height),
        (xmax - pad, cy + 0.50 * sim.height),
        (cx, ymax - pad),
        (xmin + pad, cy + 0.50 * sim.height),
        (xmin + pad, cy - 0.50 * sim.height),
    ]


def _interpolate_waypoints(
    waypoints: list[tuple[float, float]], steps: int
) -> list[tuple[float, float]]:
    if steps < 1:
        raise ValueError("steps must be positive")
    if len(waypoints) < 2:
        raise ValueError("at least two waypoints are required")

    segments = list(zip(waypoints[:-1], waypoints[1:]))
    lengths = [
        math.hypot(x1 - x0, y1 - y0)
        for (x0, y0), (x1, y1) in segments
    ]
    total = sum(lengths)
    positions = []

    for step in range(steps):
        target = 0.0 if steps == 1 else (step / (steps - 1)) * total
        acc = 0.0
        for ((x0, y0), (x1, y1)), length in zip(segments, lengths):
            if target <= acc + length or length == 0:
                frac = 0.0 if length == 0 else (target - acc) / length
                positions.append((x0 + frac * (x1 - x0), y0 + frac * (y1 - y0)))
                break
            acc += length
        else:
            positions.append(waypoints[-1])

    return positions


def _link_measurements(sim: CellularNetworkReceivedPower, x: float, y: float) -> dict[int, float]:
    rsrp_by_gnb = {}
    for gnb_idx in range(len(sim.bs_positions)):
        pl_base, los, _ = sim.radio_model.calculate_path_loss(x, y, gnb_idx, 0)
        sf_db = sim.radio_model.shadow_fading(0, gnb_idx, los, (x, y))
        prx_dbm = sim.radio_model.calculate_total_received_power(gnb_idx, pl_base + sf_db)
        rsrp_by_gnb[gnb_idx] = sim.radio_model.calculate_rsrp_from_received_power(prx_dbm)
    return rsrp_by_gnb


def _select_serving(
    previous_serving: int | None,
    rsrp_by_gnb: dict[int, float],
    hom_db: float,
) -> int:
    best = max(rsrp_by_gnb, key=lambda idx: rsrp_by_gnb[idx])
    if previous_serving is None:
        return best
    if rsrp_by_gnb[best] > rsrp_by_gnb[previous_serving] + hom_db:
        return best
    return previous_serving


def generate_csv(args: argparse.Namespace) -> Path:
    sim = _build_sim(args)
    positions = _interpolate_waypoints(_hex_loop_waypoints(sim), args.steps)
    output = Path(args.output)
    output.parent.mkdir(parents=True, exist_ok=True)

    header = [
        "Step",
        "ue0_x",
        "ue0_y",
        "ue0_height",
        "connected_gnb",
        "ue0_BS_ketnoi",
        "ue0_handover",
        "ue0_handover_to_type",
        "Type",
    ]
    header.extend(f"gnb{i}_rsrp" for i in range(7))

    previous_serving = None
    rows = []
    for step, (x, y) in enumerate(positions):
        sim.ue_positions[0] = [x, y]
        rsrp_by_gnb = _link_measurements(sim, x, y)
        serving = _select_serving(previous_serving, rsrp_by_gnb, sim.hom)
        handover = int(previous_serving is not None and serving != previous_serving)
        row = {
            "Step": step,
            "ue0_x": f"{x:.2f}",
            "ue0_y": f"{y:.2f}",
            "ue0_height": f"{args.ue_height:.2f}",
            "connected_gnb": _config_id(serving),
            "ue0_BS_ketnoi": serving,
            "ue0_handover": handover,
            "ue0_handover_to_type": 1,
            "Type": args.handover_type,
        }
        for gnb_idx in range(7):
            row[f"gnb{gnb_idx}_rsrp"] = f"{rsrp_by_gnb[gnb_idx]:.2f}"
        rows.append(row)
        previous_serving = serving

    with output.open("w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=header)
        writer.writeheader()
        writer.writerows(rows)

    handovers = sum(int(row["ue0_handover"]) for row in rows)
    print(f"Generated {output}")
    print(f"Rows: {len(rows)}")
    print(f"gNBs: 7 (gnb0=center, gnb1..gnb6=first-tier neighbors)")
    print(f"Region: {args.rect_len:g} m x {args.rect_wid:g} m")
    print(f"UE height: {args.ue_height:g} m")
    print(f"Handover events in connected_gnb sequence: {handovers}")
    return output


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Generate a 7-gNB measurement CSV for StormSIM."
    )
    parser.add_argument("--output", default=str(DEFAULT_OUTPUT))
    parser.add_argument("--rect-len", type=float, default=1500.0)
    parser.add_argument("--rect-wid", type=float, default=1500.0)
    parser.add_argument("--grid-spacing", type=float, default=250.0)
    parser.add_argument("--ue-height", type=float, default=50.0)
    parser.add_argument("--steps", type=int, default=config.SIMULATION_STEPS + 1)
    parser.add_argument("--seed", type=int, default=20260628)
    parser.add_argument("--handover-type", choices=["Xn", "N2"], default="Xn")
    return parser.parse_args()


if __name__ == "__main__":
    generate_csv(parse_args())
