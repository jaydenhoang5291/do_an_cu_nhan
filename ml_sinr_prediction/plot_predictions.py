from __future__ import annotations

import argparse

import matplotlib

matplotlib.use("Agg")

import matplotlib.pyplot as plt
import pandas as pd

from .common import PROCESSED_DIR, display_path, ensure_output_dirs


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

    target_name = args.target or str(df["target"].iloc[0])
    output_path = (
        PROCESSED_DIR / f"prediction_plot_{args.model.lower().replace(' ', '_')}_{target_name}.png"
        if args.output is None
        else args.output
    )

    fig, (ax_top, ax_bottom) = plt.subplots(
        2,
        1,
        figsize=(11, 7),
        sharex=True,
        gridspec_kw={"height_ratios": [2.2, 1.0]},
    )

    ax_top.plot(df["sample_index"], df["actual_sinr"], label="Actual SINR", linewidth=2.0)
    ax_top.plot(
        df["sample_index"],
        df["predicted_sinr"],
        label="Predicted SINR",
        linewidth=2.0,
        linestyle="--",
    )
    ax_top.set_title(f"{args.model}: actual vs predicted {target_name}")
    ax_top.set_ylabel("SINR (dB)")
    ax_top.grid(True, linestyle=":", alpha=0.6)
    ax_top.legend()

    ax_bottom.axhline(0.0, color="black", linewidth=1.0)
    ax_bottom.bar(df["sample_index"], df["error"], width=0.8)
    ax_bottom.set_xlabel("Test sample index")
    ax_bottom.set_ylabel("Error (dB)")
    ax_bottom.grid(True, axis="y", linestyle=":", alpha=0.6)

    fig.tight_layout()
    ensure_output_dirs()
    fig.savefig(output_path, dpi=160)
    plt.close(fig)

    print(f"Saved plot: {display_path(output_path)}")


if __name__ == "__main__":
    main()
