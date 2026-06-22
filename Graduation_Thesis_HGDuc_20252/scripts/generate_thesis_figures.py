"""Generate thesis figures from archived StormSIM CSV results.

The script intentionally uses only the two archived runs discussed in
Chapter 4. It does not treat them as official experiments; it reproduces
descriptive statistics already present in their CSV files.
"""

from __future__ import annotations

import csv
from collections import Counter
from pathlib import Path

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt


ROOT = Path(__file__).resolve().parents[1]
DATA_DIR = ROOT / "Code_HO"
IMAGE_DIR = ROOT / "Images"

RUNS = {
    "Loss 0.10": "1781614531",
    "Loss 0.50": "1781622441",
}


def read_csv(path: Path) -> list[dict[str, str]]:
    with path.open("r", encoding="utf-8-sig", newline="") as handle:
        return list(csv.DictReader(handle))


def save_figure(filename: str) -> None:
    IMAGE_DIR.mkdir(parents=True, exist_ok=True)
    plt.tight_layout()
    plt.savefig(IMAGE_DIR / filename, dpi=220, bbox_inches="tight")
    plt.close()


def plot_success_rate() -> None:
    labels: list[str] = []
    success_rates: list[float] = []
    trial_counts: list[int] = []

    for label, run_id in RUNS.items():
        rows = read_csv(DATA_DIR / f"handover-trial-summary-{run_id}.csv")
        successful = sum(row["overall_result"] == "success" for row in rows)
        labels.append(label)
        success_rates.append(100.0 * successful / len(rows))
        trial_counts.append(len(rows))

    colors = ["#4C78A8", "#E45756"]
    bars = plt.bar(labels, success_rates, color=colors, width=0.58)
    plt.ylabel("Handover success rate (%)")
    plt.ylim(0, 108)
    plt.grid(axis="y", linestyle=":", alpha=0.45)

    for bar, rate, count in zip(bars, success_rates, trial_counts):
        plt.text(
            bar.get_x() + bar.get_width() / 2,
            rate + 1.3,
            f"{rate:.2f}%\n(n={count})",
            ha="center",
            va="bottom",
            fontsize=9,
        )

    save_figure("archived_ho_success_rate.png")


def plot_success_delay() -> None:
    labels: list[str] = []
    values: list[list[float]] = []

    for label, run_id in RUNS.items():
        rows = read_csv(DATA_DIR / f"handover-trial-summary-{run_id}.csv")
        delays = [
            float(row["total_ho_ms"])
            for row in rows
            if row["overall_result"] == "success"
        ]
        labels.append(label)
        values.append(delays)

    box = plt.boxplot(
        values,
        tick_labels=labels,
        patch_artist=True,
        showfliers=True,
        widths=0.5,
    )
    for patch, color in zip(box["boxes"], ["#4C78A8", "#E45756"]):
        patch.set_facecolor(color)
        patch.set_alpha(0.72)

    plt.ylabel("Successful handover duration (ms)")
    plt.grid(axis="y", linestyle=":", alpha=0.45)
    save_figure("archived_ho_delay_boxplot.png")


def plot_xn_preparation_outcomes() -> None:
    labels: list[str] = []
    stop_counts: list[int] = []
    transport_fail_counts: list[int] = []
    timeout_counts: list[int] = []

    for label, run_id in RUNS.items():
        rows = read_csv(DATA_DIR / f"timer-events-{run_id}.csv")
        events = Counter(
            row["event"]
            for row in rows
            if row["timer_name"] == "TXnRELOCprep"
        )
        labels.append(label)
        stop_counts.append(events["stop"])
        transport_fail_counts.append(events["transport_fail"])
        timeout_counts.append(events["timeout"])

    positions = range(len(labels))
    width = 0.24
    offsets = [-width, 0.0, width]

    plt.bar(
        [position + offsets[0] for position in positions],
        stop_counts,
        width,
        label="stop",
        color="#59A14F",
    )
    plt.bar(
        [position + offsets[1] for position in positions],
        transport_fail_counts,
        width,
        label="transport_fail",
        color="#F28E2B",
    )
    plt.bar(
        [position + offsets[2] for position in positions],
        timeout_counts,
        width,
        label="timeout",
        color="#E15759",
    )
    plt.xticks(list(positions), labels)
    plt.ylabel("Number of timer events")
    plt.grid(axis="y", linestyle=":", alpha=0.45)
    plt.legend()
    save_figure("archived_xn_prep_outcomes.png")


def main() -> None:
    plot_success_rate()
    plot_success_delay()
    plot_xn_preparation_outcomes()
    print(f"Generated thesis figures in {IMAGE_DIR}")


if __name__ == "__main__":
    main()
