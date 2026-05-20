from __future__ import annotations

import joblib
from sklearn.linear_model import Ridge

from .common import (
    MODELS_DIR,
    display_path,
    ensure_output_dirs,
    load_metadata,
    load_processed_arrays,
    print_metrics,
)


def main() -> dict:
    ensure_output_dirs()
    X_train, X_test, y_train, y_test = load_processed_arrays()
    _, target_cols, _ = load_metadata()

    model = Ridge(alpha=1.0)
    model.fit(X_train, y_train)
    y_pred = model.predict(X_test)

    joblib.dump(model, MODELS_DIR / "linear_regression.joblib")
    print(f"Saved model: {display_path(MODELS_DIR / 'linear_regression.joblib')}")
    return print_metrics("Ridge regression", target_cols, y_test, y_pred)


if __name__ == "__main__":
    main()
