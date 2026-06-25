"""
Compute handover counts and rates from a simulator CSV log.
"""

import argparse
from pathlib import Path

import pandas as pd


def _handover_columns(columns) -> list[str]:
    return [
        column
        for column in columns
        if column.startswith("ue") and column.endswith("_handover")
    ]


def _ue_name(column: str) -> str:
    return column.removesuffix("_handover")


def analyze_handover(csv_path: Path) -> tuple[pd.DataFrame, dict]:
    df = pd.read_csv(csv_path)
    handover_cols = _handover_columns(df.columns)
    if not handover_cols:
        raise ValueError("No handover columns found. Expected columns like ue0_handover.")

    handover_values = df[handover_cols].apply(pd.to_numeric, errors="coerce").fillna(0)
    handover_flags = handover_values.astype(float).gt(0)

    sample_count = len(df)
    ue_count = len(handover_cols)
    total_ue_samples = sample_count * ue_count
    total_handovers = int(handover_flags.sum().sum())
    steps_with_handover = int(handover_flags.any(axis=1).sum())

    rows = []
    for column in handover_cols:
        count = int(handover_flags[column].sum())
        rows.append(
            {
                "ue": _ue_name(column),
                "handover_count": count,
                "samples": sample_count,
                "handover_rate": count / sample_count if sample_count else 0.0,
            }
        )

    summary = {
        "file": str(csv_path),
        "samples": sample_count,
        "ue_count": ue_count,
        "total_ue_samples": total_ue_samples,
        "total_handovers": total_handovers,
        "handover_rate_per_ue_sample": (
            total_handovers / total_ue_samples if total_ue_samples else 0.0
        ),
        "steps_with_handover": steps_with_handover,
        "step_handover_rate": steps_with_handover / sample_count if sample_count else 0.0,
    }
    return pd.DataFrame(rows), summary


def _format_percent(value: float) -> str:
    return f"{value * 100:.4f}%"


def main() -> None:
    parser = argparse.ArgumentParser(description="Summarize handovers in a simulator CSV log.")
    parser.add_argument("csv", type=Path, help="Path to a simulator CSV file.")
    parser.add_argument(
        "--save-summary",
        type=Path,
        default=None,
        help="Optional path to save the per-UE summary as CSV.",
    )
    args = parser.parse_args()

    per_ue, summary = analyze_handover(args.csv)

    print(f"File: {summary['file']}")
    print(f"Samples: {summary['samples']}")
    print(f"UE count: {summary['ue_count']}")
    print(f"Total handovers: {summary['total_handovers']}")
    print(
        "Handover rate per UE-sample: "
        f"{_format_percent(summary['handover_rate_per_ue_sample'])}"
    )
    print(
        "Steps with at least one handover: "
        f"{summary['steps_with_handover']} "
        f"({_format_percent(summary['step_handover_rate'])})"
    )

    print("\nPer UE:")
    display = per_ue.copy()
    display["handover_rate"] = display["handover_rate"].map(_format_percent)
    print(display.to_string(index=False))

    if args.save_summary:
        per_ue.to_csv(args.save_summary, index=False)
        print(f"\nSaved per-UE summary: {args.save_summary}")


if __name__ == "__main__":
    main()
