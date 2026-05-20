from __future__ import annotations

import joblib
import numpy as np
import pandas as pd

from .common import (
    MODELS_DIR,
    PROCESSED_DIR,
    display_path,
    ensure_output_dirs,
    last_step_feature_values,
    load_metadata,
    load_processed_arrays,
    regression_metrics,
)


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
            rows.append({
                "sample_index": sample_idx,
                "model": model_name,
                "target": target_col,
                "actual_sinr": actual,
                "predicted_sinr": predicted,
                "error": predicted - actual,
                "absolute_error": abs(predicted - actual),
            })
    return rows


def main() -> pd.DataFrame:
    ensure_output_dirs()
    _, X_test, _, y_test = load_processed_arrays()
    feature_cols, target_cols, _ = load_metadata()

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

    out_path = PROCESSED_DIR / "model_results.csv"
    results.to_csv(out_path, index=False)
    print(f"Saved results: {display_path(out_path)}")

    predictions_path = PROCESSED_DIR / "model_predictions.csv"
    predictions = pd.DataFrame(prediction_rows)
    predictions.to_csv(predictions_path, index=False)
    print(f"Saved predictions: {display_path(predictions_path)}")
    return results


if __name__ == "__main__":
    main()
