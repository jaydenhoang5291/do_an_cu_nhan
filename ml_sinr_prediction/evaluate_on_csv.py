from __future__ import annotations

import argparse

import joblib
import numpy as np
import pandas as pd

from .build_dataset import build_sequences
from .common import (
    MODELS_DIR,
    display_path,
    ensure_output_dirs,
    last_step_feature_values,
    load_metadata,
)
from .evaluate_models import _metric_row, _prediction_rows, _write_test_bundle


def _load_test_arrays(csv_path: str, feature_cols: list[str], target_cols: list[str], history_len: int, horizon: int):
    df = pd.read_csv(csv_path)
    missing_features = [col for col in feature_cols if col not in df.columns]
    missing_targets = [col for col in target_cols if col not in df.columns]
    if missing_features or missing_targets:
        raise ValueError(
            "The test CSV does not match the trained feature/target schema. "
            f"Missing features: {missing_features}; missing targets: {missing_targets}"
        )
    return build_sequences(df, feature_cols, target_cols, history_len, horizon)


def main() -> pd.DataFrame:
    parser = argparse.ArgumentParser(description="Evaluate trained models on a new simulator CSV.")
    parser.add_argument("--csv", required=True, help="New simulator CSV to test, for example data/simulation_logs/file.csv.")
    args = parser.parse_args()

    ensure_output_dirs()
    feature_cols, target_cols, train_config = load_metadata()
    history_len = int(train_config.get("history_len", 10))
    horizon = int(train_config.get("horizon", 1))
    X_test, y_test = _load_test_arrays(args.csv, feature_cols, target_cols, history_len, horizon)

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

    test_config = {
        "csv_path": args.csv,
        "train_csv_path": train_config.get("csv_path"),
        "history_len": history_len,
        "horizon": horizon,
        "n_features": len(feature_cols),
        "n_targets": len(target_cols),
        "n_test_samples": int(np.asarray(y_test).shape[0]),
    }
    _write_test_bundle(
        results,
        pd.DataFrame(prediction_rows),
        feature_cols,
        target_cols,
        test_config,
    )
    return results


if __name__ == "__main__":
    main()
