from __future__ import annotations

import re
from pathlib import Path

import joblib
import matplotlib

matplotlib.use("Agg")

import matplotlib.pyplot as plt
import numpy as np
import pandas as pd

from .common import (
    MODELS_DIR,
    PROCESSED_DIR,
    TEST_RESULTS_DIR,
    display_path,
    ensure_output_dirs,
    last_step_feature_values,
    load_metadata,
    load_processed_arrays,
    regression_metrics,
    save_json,
)


PLOT_MODELS = {"Linear Regression", "Random Forest"}
MAX_TIME_PLOT_POINTS = 300
MAX_PARITY_POINTS = 1000


def _metric_row(model_name: str, y_true, y_pred) -> dict:
    mae, rmse = regression_metrics(y_true, y_pred)
    return {
        "Model": model_name,
        "MAE_mean": float(mae.mean()),
        "RMSE_mean": float(rmse.mean()),
    }


def _as_2d(values) -> np.ndarray:
    values = np.asarray(values)
    if values.ndim == 1:
        return values.reshape(-1, 1)
    return values


def _prediction_rows(model_name: str, target_cols: list[str], y_true, y_pred) -> list[dict]:
    y_true = _as_2d(y_true)
    y_pred = _as_2d(y_pred)
    rows = []
    for sample_idx in range(y_true.shape[0]):
        for target_idx, target_col in enumerate(target_cols):
            actual = float(y_true[sample_idx, target_idx])
            predicted = float(y_pred[sample_idx, target_idx])
            actual_rounded = round(actual, 2)
            predicted_rounded = round(predicted, 2)
            error_rounded = round(predicted_rounded - actual_rounded, 2)
            rows.append({
                "sample_index": sample_idx,
                "model": model_name,
                "target": target_col,
                "actual_sinr": actual_rounded,
                "predicted_sinr": predicted_rounded,
                "error": error_rounded,
                "absolute_error": round(abs(error_rounded), 2),
            })
    return rows


def _safe_name(value: str) -> str:
    return re.sub(r"[^A-Za-z0-9_.-]+", "_", value).strip("_") or "dataset"


def _test_output_dir(dataset_config: dict) -> Path:
    csv_path = dataset_config.get("csv_path", "dataset")
    dataset_name = _safe_name(Path(str(csv_path)).stem)
    return TEST_RESULTS_DIR / dataset_name


def _save_prediction_plot(df: pd.DataFrame, model_name: str, target_name: str, output_path: Path) -> None:
    plot_df = df[(df["model"] == model_name) & (df["target"] == target_name)].copy()
    if plot_df.empty:
        return
    plot_df = plot_df.sort_values("sample_index")
    visible = plot_df.head(MAX_TIME_PLOT_POINTS).copy()

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
        f"{model_name}: actual vs predicted {target_name} "
        f"(first {len(visible)} of {len(plot_df)} test samples)"
    )
    ax.set_xlabel("Test sample index")
    ax.set_ylabel("SINR (dB)")
    ax.grid(True, linestyle=":", alpha=0.6)
    ax.legend()

    fig.tight_layout()
    fig.savefig(output_path, dpi=160)
    plt.close(fig)


def _save_error_histogram(df: pd.DataFrame, model_name: str, target_name: str, output_path: Path) -> None:
    plot_df = df[(df["model"] == model_name) & (df["target"] == target_name)].copy()
    if plot_df.empty:
        return

    mean_abs_error = float(plot_df["absolute_error"].mean())
    median_abs_error = float(plot_df["absolute_error"].median())

    fig, ax = plt.subplots(figsize=(9, 5.5))
    ax.hist(plot_df["error"], bins=40, alpha=0.82, edgecolor="black")
    ax.axvline(0.0, color="black", linestyle="--", linewidth=1.2, label="No error")
    ax.axvline(plot_df["error"].mean(), color="tab:red", linewidth=1.8, label="Mean error")
    ax.set_title(
        f"{model_name}: prediction error distribution {target_name}\n"
        f"Mean abs error={mean_abs_error:.2f} dB, median abs error={median_abs_error:.2f} dB"
    )
    ax.set_xlabel("Prediction error = predicted - actual (dB)")
    ax.set_ylabel("Number of samples")
    ax.grid(True, axis="y", linestyle=":", alpha=0.6)
    ax.legend()
    fig.tight_layout()
    fig.savefig(output_path, dpi=160)
    plt.close(fig)


def _save_parity_plot(df: pd.DataFrame, model_name: str, target_name: str, output_path: Path) -> None:
    plot_df = df[(df["model"] == model_name) & (df["target"] == target_name)].copy()
    if plot_df.empty:
        return
    stride = max(1, int(np.ceil(len(plot_df) / MAX_PARITY_POINTS)))
    sampled = plot_df.iloc[::stride].copy()
    min_value = float(min(sampled["actual_sinr"].min(), sampled["predicted_sinr"].min()))
    max_value = float(max(sampled["actual_sinr"].max(), sampled["predicted_sinr"].max()))

    fig, ax = plt.subplots(figsize=(7.5, 7.0))
    scatter = ax.scatter(
        sampled["actual_sinr"],
        sampled["predicted_sinr"],
        c=sampled["absolute_error"],
        cmap="viridis",
        s=16,
        alpha=0.65,
    )
    ax.plot([min_value, max_value], [min_value, max_value], color="black", linestyle="--", linewidth=1.2)
    ax.set_title(f"{model_name}: predicted vs actual {target_name}")
    ax.set_xlabel("Actual SINR (dB)")
    ax.set_ylabel("Predicted SINR (dB)")
    ax.grid(True, linestyle=":", alpha=0.6)
    fig.colorbar(scatter, ax=ax, label="Absolute error (dB)")
    fig.tight_layout()
    fig.savefig(output_path, dpi=160)
    plt.close(fig)


def _write_test_bundle(
    results: pd.DataFrame,
    predictions: pd.DataFrame,
    feature_cols: list[str],
    target_cols: list[str],
    dataset_config: dict,
) -> Path:
    test_dir = _test_output_dir(dataset_config)
    test_dir.mkdir(parents=True, exist_ok=True)
    test_results_path = test_dir / "model_results.csv"
    test_predictions_path = test_dir / "model_predictions.csv"
    results.to_csv(test_results_path, index=False)
    predictions.to_csv(test_predictions_path, index=False, float_format="%.2f")
    save_json(test_dir / "dataset_config.json", dataset_config)
    save_json(test_dir / "feature_cols.json", feature_cols)
    save_json(test_dir / "target_cols.json", target_cols)

    for model_name in PLOT_MODELS:
        if model_name not in set(predictions["model"]):
            continue
        for target_name in target_cols:
            plot_name = f"prediction_plot_{_safe_name(model_name.lower())}_{_safe_name(target_name)}.png"
            _save_prediction_plot(predictions, model_name, target_name, test_dir / plot_name)
            histogram_name = f"error_histogram_{_safe_name(model_name.lower())}_{_safe_name(target_name)}.png"
            _save_error_histogram(predictions, model_name, target_name, test_dir / histogram_name)
            parity_name = f"parity_plot_{_safe_name(model_name.lower())}_{_safe_name(target_name)}.png"
            _save_parity_plot(predictions, model_name, target_name, test_dir / parity_name)

    print(f"Saved test bundle: {display_path(test_dir)}")
    return test_dir


def _write_outputs(
    results: pd.DataFrame,
    predictions: pd.DataFrame,
    feature_cols: list[str],
    target_cols: list[str],
    dataset_config: dict,
) -> None:
    processed_results_path = PROCESSED_DIR / "model_results.csv"
    processed_predictions_path = PROCESSED_DIR / "model_predictions.csv"
    results.to_csv(processed_results_path, index=False)
    predictions.to_csv(processed_predictions_path, index=False, float_format="%.2f")
    print(f"Saved results: {display_path(processed_results_path)}")
    print(f"Saved predictions: {display_path(processed_predictions_path)}")
    _write_test_bundle(results, predictions, feature_cols, target_cols, dataset_config)


def main() -> pd.DataFrame:
    ensure_output_dirs()
    _, X_test, _, y_test = load_processed_arrays()
    feature_cols, target_cols, dataset_config = load_metadata()

    rows = []
    prediction_rows = []
    persistence_pred = last_step_feature_values(X_test, feature_cols, target_cols)
    rows.append(_metric_row("Persistence", y_test, persistence_pred))
    prediction_rows.extend(_prediction_rows("Persistence", target_cols, y_test, persistence_pred))

    model_specs = [
        ("Linear Regression", MODELS_DIR / "linear_regression.joblib"),
        ("Random Forest", MODELS_DIR / "random_forest.joblib"),
    ]
    for model_name, model_path in model_specs:
        if not model_path.exists():
            print(f"Skipping {model_name}: missing model file {display_path(model_path)}")
            continue
        model = joblib.load(model_path)
        y_pred = model.predict(X_test)
        rows.append(_metric_row(model_name, y_test, y_pred))
        prediction_rows.extend(_prediction_rows(model_name, target_cols, y_test, y_pred))

    results = pd.DataFrame(rows, columns=["Model", "MAE_mean", "RMSE_mean"])
    print(results.to_string(index=False, float_format=lambda value: f"{value:.4f}"))

    predictions = pd.DataFrame(prediction_rows)
    _write_outputs(results, predictions, feature_cols, target_cols, dataset_config)
    return results


if __name__ == "__main__":
    main()
