from __future__ import annotations

import argparse

import joblib

from .common import MODELS_DIR, display_path, load_metadata, load_processed_arrays


MODEL_PATHS = {
    "linear": MODELS_DIR / "linear_regression.joblib",
    "random_forest": MODELS_DIR / "random_forest.joblib",
}


def main() -> None:
    parser = argparse.ArgumentParser(description="Predict future SINR for one X_test sample.")
    parser.add_argument(
        "--model",
        choices=sorted(MODEL_PATHS),
        default="random_forest",
        help="Trained model to use.",
    )
    parser.add_argument("--sample-index", type=int, default=0)
    args = parser.parse_args()

    model_path = MODEL_PATHS[args.model]
    if not model_path.exists():
        raise FileNotFoundError(f"Model not found: {display_path(model_path)}. Train it first.")

    _, X_test, _, y_test = load_processed_arrays()
    _, target_cols, _ = load_metadata()

    if args.sample_index < 0 or args.sample_index >= len(X_test):
        raise IndexError(f"sample-index must be in [0, {len(X_test) - 1}], got {args.sample_index}")

    model = joblib.load(model_path)
    x_sample = X_test[args.sample_index : args.sample_index + 1]
    y_pred = model.predict(x_sample)[0]
    y_actual = y_test[args.sample_index]

    best_pred_idx = int(y_pred.argmax())
    best_actual_idx = int(y_actual.argmax())
    y_pred_display = [round(float(value), 2) for value in y_pred]
    y_actual_display = [round(float(value), 2) for value in y_actual]

    print(f"Model: {display_path(model_path)}")
    print(f"Sample index: {args.sample_index}")
    print(f"Target columns: {target_cols}")
    print(f"Predicted SINR vector: {y_pred_display}")
    print(f"Actual SINR vector: {y_actual_display}")
    print(f"Best predicted cell: {target_cols[best_pred_idx]} ({y_pred[best_pred_idx]:.2f})")
    print(f"Best actual cell: {target_cols[best_actual_idx]} ({y_actual[best_actual_idx]:.2f})")


if __name__ == "__main__":
    main()
