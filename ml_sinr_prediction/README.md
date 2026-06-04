# Local SINR Prediction Pipeline

This folder adds a small supervised regression pipeline for predicting next-step SINR at `t + horizon` from past simulator logs.

## Expected Input

By default, the dataset builder uses the newest simulator CSV matching:

```bash
data/simulation_logs/*_UE_Data.csv
```

You can also pass an explicit file:

```bash
python -m ml_sinr_prediction.build_dataset --csv data/simulation_logs/20260520_112557_1_UE_Data.csv
```

The legacy raw-data path is:

```bash
data/raw/simulation_log.csv
```

The dataset builder automatically uses a focused feature set when present:

- simulator columns such as `ue0_x`, `ue0_y`, `ue0_height`, `ue0_direction`, `ue0_connected_bs`, `ue0_los_probability`, `ue0_pathloss`, `ue0_shadow_fading`, `ue0_rsrp`, `ue0_sinr`, and `ue0_bs*_rsrp`
- `x`, `y`, `z`, `vx`, `vy`, `vz`
- columns starting with `rsrp_bs_`
- columns starting with `sinr_bs_`

Targets are simulator SINR columns such as `ue0_sinr`, or legacy columns starting with `sinr_bs_`, shifted by `horizon`.

Simulator columns such as `prx`, `speed`, `handover`, `los_state`, and neighbor BS indices are intentionally not used as default ML inputs because they are redundant, fixed per UE after initialization, categorical text, or weakly causal for next-step SINR.

Optional columns:

- `ue_id`: keeps sequences separated per UE.
- `timestep`, `time_step`, `step`, `time`, `frame`, or `Step`: used for chronological sorting.

## Dummy Data

If `data/raw/simulation_log.csv` does not exist yet, you can generate a small synthetic file only to test the code:

```bash
python -m ml_sinr_prediction.generate_dummy_simulation_log
```

This is dummy data only. Do not use it as research results.

## Long Training Log

To create a longer real simulator log without waiting for the animation window:

```bash
python -m ml_sinr_prediction.generate_training_log
```

Run it without arguments to enter the same core simulator parameters as `main.py`, plus the number of CSV rows to generate. For automated runs, pass arguments directly:

```bash
python -m ml_sinr_prediction.generate_training_log --rows 3000 --num-ues 1
```

The generated CSV is saved in `data/simulation_logs/` and can then be used by the dataset builder.

## Run From VS Code Terminal

From the project root:

```bash
python -m ml_sinr_prediction.build_dataset
python -m ml_sinr_prediction.train_persistence
python -m ml_sinr_prediction.train_linear_regression
python -m ml_sinr_prediction.train_random_forest
python -m ml_sinr_prediction.evaluate_models
```

To test a different simulator CSV with the already trained models, without rebuilding or retraining:

```bash
python -m ml_sinr_prediction.evaluate_on_csv --csv data/simulation_logs/NEW_FILE.csv
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

Each `evaluate_models` or `evaluate_on_csv` run also creates a per-dataset test bundle:

```bash
data/test_results/<dataset_name>/model_results.csv
data/test_results/<dataset_name>/model_predictions.csv
data/test_results/<dataset_name>/prediction_plot_linear_regression_ue0_sinr.png
data/test_results/<dataset_name>/prediction_plot_random_forest_ue0_sinr.png
data/test_results/<dataset_name>/error_histogram_linear_regression_ue0_sinr.png
data/test_results/<dataset_name>/error_histogram_random_forest_ue0_sinr.png
data/test_results/<dataset_name>/parity_plot_linear_regression_ue0_sinr.png
data/test_results/<dataset_name>/parity_plot_random_forest_ue0_sinr.png
```

Trained models:

```bash
models/linear_regression.joblib
models/random_forest.joblib
```

## Configuration

Defaults:

- `history_len = 10`
- `horizon = 1`
- `test_size = 0.2`

Override them when building:

```bash
python -m ml_sinr_prediction.build_dataset --history-len 10 --horizon 10 --test-size 0.2
```
