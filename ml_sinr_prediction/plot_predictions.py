from __future__ import annotations

import argparse

import matplotlib

matplotlib.use("Agg")

import matplotlib.pyplot as plt
import pandas as pd

from .common import PROCESSED_DIR, display_path, ensure_output_dirs

MAX_TIME_PLOT_POINTS = 300


def main() -> None:
    parser = argparse.ArgumentParser(description="Plot actual vs predicted SINR from model_predictions.csv.")
    parser.add_argument("--model", default="Random Forest", help="Model name to plot.")
    parser.add_argument("--target", default=None, help="Target column to plot, for example ue0_sinr.")
    parser.add_argument("--output", default=None, help="Output image path.")
    args = parser.parse_args()

    predictions_path = PROCESSED_DIR / "model_predictions.csv"
    if not predictions_path.exists():
        raise FileNotFoundError(
            f"Missing predictions file: {display_path(predictions_path)}. "
            "Run: python -m ml_sinr_prediction.evaluate_models"
        )

    df = pd.read_csv(predictions_path)
    df = df[df["model"] == args.model].copy()
    if args.target is not None:
        df = df[df["target"] == args.target].copy()

    if df.empty:
        raise ValueError(
            f"No prediction rows found for model={args.model!r}, target={args.target!r}."
        )
    df = df.sort_values("sample_index")
    visible = df.head(MAX_TIME_PLOT_POINTS).copy()

    target_name = args.target or str(df["target"].iloc[0])
    output_path = (
        PROCESSED_DIR / f"prediction_plot_{args.model.lower().replace(' ', '_')}_{target_name}.png"
        if args.output is None
        else args.output
    )

    fig, ax = plt.subplots(figsize=(12, 5.5))
    ax.plot(visible["sample_index"], visible["actual_sinr"], label="Actual SINR", linewidth=2.0)
    ax.plot(
        visible["sample_index"],
        visible["predicted_sinr"],
        label="Predicted SINR",
        linewidth=2.0,
        linestyle="--",
    )
    ax.set_title(
        f"{args.model}: actual vs predicted {target_name} "
        f"(first {len(visible)} of {len(df)} test samples)"
    )
    ax.set_xlabel("Test sample index")
    ax.set_ylabel("SINR (dB)")
    ax.grid(True, linestyle=":", alpha=0.6)
    ax.legend()

    fig.tight_layout()
    ensure_output_dirs()
    fig.savefig(output_path, dpi=160)
    plt.close(fig)

    print(f"Saved plot: {display_path(output_path)}")


if __name__ == "__main__":
    main()
