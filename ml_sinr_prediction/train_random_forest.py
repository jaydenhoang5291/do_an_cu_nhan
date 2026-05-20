from __future__ import annotations

import joblib
from sklearn.ensemble import RandomForestRegressor
from sklearn.multioutput import MultiOutputRegressor

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

    model = MultiOutputRegressor(
        RandomForestRegressor(
            n_estimators=100,
            random_state=42,
            n_jobs=-1,
        )
    )
    model.fit(X_train, y_train)
    y_pred = model.predict(X_test)

    joblib.dump(model, MODELS_DIR / "random_forest.joblib")
    print(f"Saved model: {display_path(MODELS_DIR / 'random_forest.joblib')}")
    return print_metrics("Random forest regression", target_cols, y_test, y_pred)


if __name__ == "__main__":
    main()
