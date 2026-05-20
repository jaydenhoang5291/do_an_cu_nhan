from __future__ import annotations

from .common import last_step_feature_values, load_metadata, load_processed_arrays, print_metrics


def main() -> dict:
    _, X_test, _, y_test = load_processed_arrays()
    feature_cols, target_cols, _ = load_metadata()

    y_pred = last_step_feature_values(X_test, feature_cols, target_cols)
    return print_metrics("Persistence baseline", target_cols, y_test, y_pred)


if __name__ == "__main__":
    main()

