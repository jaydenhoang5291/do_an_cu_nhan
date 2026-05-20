from __future__ import annotations

import json
from pathlib import Path

import numpy as np
from sklearn.metrics import mean_absolute_error, mean_squared_error


PACKAGE_DIR = Path(__file__).resolve().parent
PROJECT_ROOT = PACKAGE_DIR.parent
RAW_DATA_PATH = PROJECT_ROOT / "data" / "raw" / "simulation_log.csv"
PROCESSED_DIR = PROJECT_ROOT / "data" / "processed"
MODELS_DIR = PROJECT_ROOT / "models"


def display_path(path: Path) -> str:
    return str(path).encode("ascii", errors="backslashreplace").decode("ascii")


def ensure_output_dirs() -> None:
    PROCESSED_DIR.mkdir(parents=True, exist_ok=True)
    MODELS_DIR.mkdir(parents=True, exist_ok=True)


def load_json(path: Path) -> object:
    with path.open("r", encoding="utf-8") as f:
        return json.load(f)


def save_json(path: Path, value: object) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8") as f:
        json.dump(value, f, indent=2, ensure_ascii=False)


def load_processed_arrays() -> tuple[np.ndarray, np.ndarray, np.ndarray, np.ndarray]:
    required = ["X_train.npy", "X_test.npy", "y_train.npy", "y_test.npy"]
    missing = [name for name in required if not (PROCESSED_DIR / name).exists()]
    if missing:
        raise FileNotFoundError(
            "Missing processed dataset files: "
            + ", ".join(missing)
            + ". Run: python -m ml_sinr_prediction.build_dataset"
        )

    return (
        np.load(PROCESSED_DIR / "X_train.npy"),
        np.load(PROCESSED_DIR / "X_test.npy"),
        np.load(PROCESSED_DIR / "y_train.npy"),
        np.load(PROCESSED_DIR / "y_test.npy"),
    )


def load_metadata() -> tuple[list[str], list[str], dict]:
    feature_path = PROCESSED_DIR / "feature_cols.json"
    target_path = PROCESSED_DIR / "target_cols.json"
    config_path = PROCESSED_DIR / "dataset_config.json"
    missing = [p.name for p in [feature_path, target_path, config_path] if not p.exists()]
    if missing:
        raise FileNotFoundError(
            "Missing metadata files: "
            + ", ".join(missing)
            + ". Run: python -m ml_sinr_prediction.build_dataset"
        )

    return (
        list(load_json(feature_path)),
        list(load_json(target_path)),
        dict(load_json(config_path)),
    )


def regression_metrics(y_true: np.ndarray, y_pred: np.ndarray) -> tuple[np.ndarray, np.ndarray]:
    mae = mean_absolute_error(y_true, y_pred, multioutput="raw_values")
    rmse = np.sqrt(mean_squared_error(y_true, y_pred, multioutput="raw_values"))
    return mae, rmse


def print_metrics(title: str, target_cols: list[str], y_true: np.ndarray, y_pred: np.ndarray) -> dict:
    mae, rmse = regression_metrics(y_true, y_pred)

    print(f"\n{title}")
    print("-" * len(title))
    for col, col_mae, col_rmse in zip(target_cols, mae, rmse):
        print(f"{col}: MAE={col_mae:.4f}, RMSE={col_rmse:.4f}")

    result = {
        "MAE_mean": float(np.mean(mae)),
        "RMSE_mean": float(np.mean(rmse)),
    }
    print(f"Mean: MAE={result['MAE_mean']:.4f}, RMSE={result['RMSE_mean']:.4f}")
    return result


def last_step_feature_values(X_flat: np.ndarray, feature_cols: list[str], wanted_cols: list[str]) -> np.ndarray:
    if X_flat.ndim != 2:
        raise ValueError(f"Expected flattened 2D X array, got shape {X_flat.shape}")

    n_features = len(feature_cols)
    if n_features == 0 or X_flat.shape[1] % n_features != 0:
        raise ValueError(
            f"Cannot infer history_len from X shape {X_flat.shape} and {n_features} feature columns."
        )

    history_len = X_flat.shape[1] // n_features
    last_step_start = (history_len - 1) * n_features
    indices = []
    missing = []
    for col in wanted_cols:
        if col in feature_cols:
            indices.append(last_step_start + feature_cols.index(col))
        else:
            missing.append(col)

    if missing:
        raise ValueError(
            "Cannot run persistence baseline because these target columns are not present "
            f"as input features: {missing}"
        )

    return X_flat[:, indices]
