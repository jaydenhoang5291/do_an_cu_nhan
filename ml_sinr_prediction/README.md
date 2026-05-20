# Local SINR Prediction Pipeline

This folder adds a small supervised regression pipeline for predicting future SINR at `t + horizon` from past simulator logs. It does not modify the existing simulator code.

## Expected Input

By default, the dataset builder uses the newest simulator CSV matching:

```bash
data/*_UE_Data.csv
```

You can also pass an explicit file:

```bash
python -m ml_sinr_prediction.build_dataset --csv data/20260520_112557_1_UE_Data.csv
```

The legacy raw-data path is:

```bash
data/raw/simulation_log.csv
```

The dataset builder automatically uses these feature columns when present:

- simulator columns such as `ue0_x`, `ue0_y`, `ue0_height`, `ue0_rsrp`, `ue0_prx`, `ue0_sinr`, and `ue0_bs*_rsrp`
- `x`, `y`, `z`, `vx`, `vy`, `vz`
- columns starting with `rsrp_bs_`
- columns starting with `sinr_bs_`

Targets are simulator SINR columns such as `ue0_sinr`, or legacy columns starting with `sinr_bs_`, shifted by `horizon`.

Optional columns:

- `ue_id`: keeps sequences separated per UE.
- `timestep`, `time_step`, `step`, `time`, `frame`, or `Step`: used for chronological sorting.

## Dummy Data

If `data/raw/simulation_log.csv` does not exist yet, you can generate a small synthetic file only to test the code:

```bash
python -m ml_sinr_prediction.generate_dummy_simulation_log
```

This is dummy data only. Do not use it as research results.

## Run From VS Code Terminal

From the project root:

```bash
python -m ml_sinr_prediction.build_dataset
python -m ml_sinr_prediction.train_persistence
python -m ml_sinr_prediction.train_linear_regression
python -m ml_sinr_prediction.train_random_forest
python -m ml_sinr_prediction.evaluate_models
```

Predict one sample with a trained model:

```bash
python -m ml_sinr_prediction.predict_with_model --model random_forest --sample-index 0
```

## Outputs

Processed arrays and metadata:

```bash
data/processed/X_train.npy
data/processed/X_test.npy
data/processed/y_train.npy
data/processed/y_test.npy
data/processed/feature_cols.json
data/processed/target_cols.json
data/processed/dataset_config.json
data/processed/model_results.csv
```

Trained models:

```bash
models/linear_regression.joblib
models/random_forest.joblib
```

## Configuration

Defaults:

- `history_len = 10`
- `horizon = 10`
- `test_size = 0.2`

Override them when building:

```bash
python -m ml_sinr_prediction.build_dataset --history-len 10 --horizon 10 --test-size 0.2
```
