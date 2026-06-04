# Cellular Network Simulation and SINR Prediction

This project has two parts:

1. A cellular network simulator that generates UE movement and radio measurements.
2. A machine-learning pipeline that predicts the next-step SINR from simulator logs.

Run every command from the project root, the folder that contains `main.py`, `simulation.py`, and `ml_sinr_prediction/`.

## What Is Predicted?

The ML task predicts the SINR at the next time step.

Current defaults:

- `history_len = 10`
- `horizon = 1`
- `test_size = 0.2`

Example for target `ue0_sinr`:

- Input sample: rows `t-9` to `t`
- Target value: `ue0_sinr` at row `t+1`

So if the model sees 10 previous rows ending at step `100`, it predicts `ue0_sinr` at step `101`.

The input features are stored in:

```text
data/processed/feature_cols.json
```

The target columns are stored in:

```text
data/processed/target_cols.json
```

Prediction outputs are rounded to two decimals, matching the simulator CSV precision.

## Data Folders

```text
data/
  simulation_logs/   real simulator CSV logs
  raw/               legacy or dummy raw files
  processed/         active training dataset, metadata, latest validation files
  test_results/      evaluation bundles for each tested CSV
models/              trained .joblib models
```

## 1. Run the Visual Simulator

Use this if you want to see the animation:

```bash
python main.py
```

The program asks for:

1. Rectangle length and width.
2. Road grid spacing.
3. Number of UEs.
4. Whether to use aerial UE height.
5. Whether to show UE-BS connection lines.

When the simulation ends, a CSV is saved to:

```text
data/simulation_logs/
```

## 2. Generate a Large Data CSV Without Animation

Use this for training/testing because it is much faster than drawing the animation:

```bash
python -m ml_sinr_prediction.generate_training_log
```

It asks for the same core simulator inputs, plus:

```text
Enter number of data rows to generate:
```

The generated CSV is also saved to:

```text
data/simulation_logs/
```

Non-interactive example:

```bash
python -m ml_sinr_prediction.generate_training_log --rows 3000 --num-ues 1 --no-interactive
```

This still runs the real simulator engine. It only disables Matplotlib drawing.

## 3. Train Models on One Source Dataset

Choose one CSV as the training source. This is your "root" training dataset.

Use the newest simulator CSV automatically:

```bash
python -m ml_sinr_prediction.build_dataset
```

Or choose a specific file:

```bash
python -m ml_sinr_prediction.build_dataset --csv data/simulation_logs/YOUR_TRAIN_FILE.csv
```

This creates:

```text
data/processed/X_train.npy
data/processed/y_train.npy
data/processed/X_test.npy
data/processed/y_test.npy
data/processed/dataset_config.json
data/processed/feature_cols.json
data/processed/target_cols.json
```

### How Train/Test Split Works

For the selected training CSV, `build_dataset` creates supervised samples in chronological order.

Then it splits by time:

- First 80% samples: training set.
- Last 20% samples: validation/test split for the same training CSV.

This is controlled by:

```bash
--test-size 0.2
```

Train the models:

```bash
python -m ml_sinr_prediction.train_linear_regression
python -m ml_sinr_prediction.train_random_forest
```

The trained models are saved to:

```text
models/linear_regression.joblib
models/random_forest.joblib
```

## 4. Validate on the Training Dataset Split

After training, run:

```bash
python -m ml_sinr_prediction.evaluate_models
```

This evaluates the trained models on `X_test.npy` / `y_test.npy`, meaning the last 20% split from the same source CSV.

Latest validation outputs are written to:

```text
data/processed/model_results.csv
data/processed/model_predictions.csv
```

It also writes an archived bundle:

```text
data/test_results/<training_dataset_name>/
```

That folder contains:

```text
dataset_config.json
feature_cols.json
target_cols.json
model_results.csv
model_predictions.csv
prediction_plot_linear_regression_ue0_sinr.png
prediction_plot_random_forest_ue0_sinr.png
error_histogram_linear_regression_ue0_sinr.png
error_histogram_random_forest_ue0_sinr.png
parity_plot_linear_regression_ue0_sinr.png
parity_plot_random_forest_ue0_sinr.png
```

## 5. Test Existing Models on a New CSV

Use this when you already trained models on the root training dataset and now want to test a new simulator CSV.

First generate or choose a new CSV in:

```text
data/simulation_logs/
```

Then run:

```bash
python -m ml_sinr_prediction.evaluate_on_csv --csv data/simulation_logs/YOUR_NEW_TEST_FILE.csv
```

This command does not rebuild the training dataset and does not retrain models.

It uses:

- Trained models from `models/`.
- Feature schema from `data/processed/feature_cols.json`.
- Target schema from `data/processed/target_cols.json`.
- `history_len` and `horizon` from `data/processed/dataset_config.json`.

The test result is saved to:

```text
data/test_results/<YOUR_NEW_TEST_FILE>/
```

Inside that folder you will get:

```text
model_results.csv
model_predictions.csv
prediction_plot_linear_regression_ue0_sinr.png
prediction_plot_random_forest_ue0_sinr.png
error_histogram_linear_regression_ue0_sinr.png
error_histogram_random_forest_ue0_sinr.png
parity_plot_linear_regression_ue0_sinr.png
parity_plot_random_forest_ue0_sinr.png
```

## Typical Workflow

Train once:

```bash
python -m ml_sinr_prediction.generate_training_log
python -m ml_sinr_prediction.build_dataset --csv data/simulation_logs/YOUR_TRAIN_FILE.csv
python -m ml_sinr_prediction.train_linear_regression
python -m ml_sinr_prediction.train_random_forest
python -m ml_sinr_prediction.evaluate_models
```

Test later on a new file:

```bash
python -m ml_sinr_prediction.generate_training_log
python -m ml_sinr_prediction.evaluate_on_csv --csv data/simulation_logs/YOUR_NEW_TEST_FILE.csv
```

## Important Notes

- `generate_training_log.py` creates real simulator data, not dummy data.
- `generate_dummy_simulation_log.py` creates fake data only for pipeline smoke tests.
- UE speed is randomized once at initialization, then kept constant for the whole run.
- Stop-and-go is disabled by default to avoid time-varying speed noise.
- `evaluate_models` validates on the held-out 20% split of the training CSV.
- `evaluate_on_csv` tests trained models on a separate CSV.

## Requirements

- Python 3.x
- `numpy`
- `pandas`
- `matplotlib`
- `scikit-learn`
- `joblib`
