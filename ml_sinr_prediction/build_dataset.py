from __future__ import annotations

import argparse
import re

import numpy as np
import pandas as pd

from .common import PROCESSED_DIR, RAW_DATA_PATH, SIMULATION_LOGS_DIR, display_path, ensure_output_dirs, save_json
from .common import PROJECT_ROOT


DEFAULT_HISTORY_LEN = 10
DEFAULT_HORIZON = 1
DEFAULT_TEST_SIZE = 0.2
LATEST_CSV_SENTINEL = "latest"

BASE_FEATURES = ["x", "y", "z", "vx", "vy", "vz"]
TIME_COL_CANDIDATES = ["timestep", "time_step", "step", "time", "frame", "Step"]
UE_COL_CANDIDATES = ["ue_id", "UE_ID", "ue", "UE"]
SIM_UE_COL_RE = re.compile(r"^ue\d+_")
SIM_TARGET_RE = re.compile(r"^ue\d+_sinr$")
SIM_FEATURE_SUFFIXES = (
    "_x",
    "_y",
    "_height",
    "_direction",
    "_connected_bs",
    "_los_probability",
    "_pathloss",
    "_shadow_fading",
    "_rsrp",
    "_sinr",
    "_bs1_rsrp",
    "_bs2_rsrp",
    "_bs3_rsrp",
    "_bs4_rsrp",
    "_bs5_rsrp",
    "_bs6_rsrp",
)


def _find_first_existing(columns: pd.Index, candidates: list[str]) -> str | None:
    lower_to_original = {col.lower(): col for col in columns}
    for candidate in candidates:
        if candidate in columns:
            return candidate
        found = lower_to_original.get(candidate.lower())
        if found is not None:
            return found
    return None


def find_latest_simulator_csv() -> str:
    candidates = list(SIMULATION_LOGS_DIR.glob("*_UE_Data.csv"))
    # Backward compatibility for older logs saved directly under data/.
    candidates.extend((PROJECT_ROOT / "data").glob("*_UE_Data.csv"))
    candidates = sorted(
        candidates,
        key=lambda path: path.stat().st_mtime,
        reverse=True,
    )
    if not candidates:
        raise FileNotFoundError(
            "No simulator CSV found in data/simulation_logs/*_UE_Data.csv. "
            f"Pass --csv {display_path(RAW_DATA_PATH)} for the legacy raw-data path, "
            "or run the simulator first."
        )
    return str(candidates[0])


def detect_feature_cols(df: pd.DataFrame) -> list[str]:
    base_cols = [col for col in BASE_FEATURES if col in df.columns]
    rsrp_cols = sorted(col for col in df.columns if col.startswith("rsrp_bs_"))
    sinr_cols = sorted(col for col in df.columns if col.startswith("sinr_bs_"))
    legacy_cols = base_cols + rsrp_cols + sinr_cols
    if legacy_cols:
        return legacy_cols

    simulator_cols = []
    for col in df.columns:
        if not SIM_UE_COL_RE.match(col):
            continue
        if not col.endswith(SIM_FEATURE_SUFFIXES):
            continue
        values = pd.to_numeric(df[col], errors="coerce")
        if values.notna().any():
            simulator_cols.append(col)
    return sorted(simulator_cols)


def detect_target_cols(df: pd.DataFrame) -> list[str]:
    legacy_targets = sorted(col for col in df.columns if col.startswith("sinr_bs_"))
    if legacy_targets:
        return legacy_targets
    return sorted(col for col in df.columns if SIM_TARGET_RE.match(col))


def build_sequences(
    df: pd.DataFrame,
    feature_cols: list[str],
    target_cols: list[str],
    history_len: int,
    horizon: int,
) -> tuple[np.ndarray, np.ndarray]:
    ue_col = _find_first_existing(df.columns, UE_COL_CANDIDATES)
    time_col = _find_first_existing(df.columns, TIME_COL_CANDIDATES)

    X_parts = []
    y_parts = []

    groups = df.groupby(ue_col, sort=False) if ue_col is not None else [(None, df)]
    for _, group in groups:
        group = group.copy()
        if time_col is not None:
            group = group.sort_values(time_col, kind="mergesort")

        values_x = group[feature_cols].apply(pd.to_numeric, errors="coerce").to_numpy(dtype=float)
        values_y = group[target_cols].apply(pd.to_numeric, errors="coerce").to_numpy(dtype=float)

        max_start = len(group) - history_len - horizon + 1
        if max_start <= 0:
            continue

        for start in range(max_start):
            end = start + history_len
            target_index = end - 1 + horizon
            x_window = values_x[start:end]
            y_target = values_y[target_index]
            if np.isnan(x_window).any() or np.isnan(y_target).any():
                continue
            X_parts.append(x_window.reshape(-1))
            y_parts.append(y_target)

    if not X_parts:
        raise ValueError(
            "No valid supervised samples were created. Check CSV length, NaN values, "
            "history_len, and horizon."
        )

    return np.asarray(X_parts, dtype=float), np.asarray(y_parts, dtype=float)


def time_split(
    X: np.ndarray,
    y: np.ndarray,
    test_size: float,
) -> tuple[np.ndarray, np.ndarray, np.ndarray, np.ndarray]:
    if not 0 < test_size < 1:
        raise ValueError(f"test_size must be between 0 and 1, got {test_size}")

    split_index = int(len(X) * (1 - test_size))
    if split_index <= 0 or split_index >= len(X):
        raise ValueError(
            f"Invalid split for {len(X)} samples and test_size={test_size}. "
            "Need at least one train and one test sample."
        )

    return X[:split_index], X[split_index:], y[:split_index], y[split_index:]


def main() -> None:
    parser = argparse.ArgumentParser(description="Build supervised SINR prediction dataset.")
    parser.add_argument(
        "--csv",
        type=str,
        default=LATEST_CSV_SENTINEL,
        help="Path to simulation CSV log, or 'latest' for newest data/simulation_logs/*_UE_Data.csv.",
    )
    parser.add_argument("--history-len", type=int, default=DEFAULT_HISTORY_LEN)
    parser.add_argument("--horizon", type=int, default=DEFAULT_HORIZON)
    parser.add_argument("--test-size", type=float, default=DEFAULT_TEST_SIZE)
    args = parser.parse_args()

    csv_path = find_latest_simulator_csv() if args.csv == LATEST_CSV_SENTINEL else args.csv
    csv_path = pd.io.common.stringify_path(csv_path)

    ensure_output_dirs()
    if not pd.io.common.file_exists(csv_path):
        raise FileNotFoundError(
            f"CSV log not found: {csv_path}. "
            "Create it from your simulator or run: "
            "python -m ml_sinr_prediction.generate_dummy_simulation_log"
        )

    df = pd.read_csv(csv_path)
    feature_cols = detect_feature_cols(df)
    target_cols = detect_target_cols(df)

    if not target_cols:
        raise ValueError(
            "No target columns found. Expected one or more columns named like "
            "'ue0_sinr' or starting with 'sinr_bs_'."
        )
    if not feature_cols:
        raise ValueError(
            "No feature columns found. Expected x/y/z/vx/vy/vz and/or columns starting "
            "with 'rsrp_bs_' or 'sinr_bs_'."
        )

    X, y = build_sequences(df, feature_cols, target_cols, args.history_len, args.horizon)
    X_train, X_test, y_train, y_test = time_split(X, y, args.test_size)

    np.save(PROCESSED_DIR / "X_train.npy", X_train)
    np.save(PROCESSED_DIR / "X_test.npy", X_test)
    np.save(PROCESSED_DIR / "y_train.npy", y_train)
    np.save(PROCESSED_DIR / "y_test.npy", y_test)
    save_json(PROCESSED_DIR / "feature_cols.json", feature_cols)
    save_json(PROCESSED_DIR / "target_cols.json", target_cols)
    save_json(
        PROCESSED_DIR / "dataset_config.json",
        {
            "csv_path": csv_path,
            "history_len": args.history_len,
            "horizon": args.horizon,
            "test_size": args.test_size,
            "n_features": len(feature_cols),
            "n_targets": len(target_cols),
        },
    )

    print(f"Feature columns ({len(feature_cols)}): {feature_cols}")
    print(f"Target columns ({len(target_cols)}): {target_cols}")
    print(f"Processed data directory: {display_path(PROCESSED_DIR)}")
    print(f"X_train shape: {X_train.shape}")
    print(f"X_test shape: {X_test.shape}")
    print(f"y_train shape: {y_train.shape}")
    print(f"y_test shape: {y_test.shape}")


if __name__ == "__main__":
    main()
