# Ghi nho thu muc `data/`

Thu muc nay dung de luu du lieu sinh ra tu mo phong va ket qua cua pipeline du doan SINR.

Chay tat ca lenh tu thu muc goc project:

```powershell
cd "D:\HUST\20252_ĐATN\Code"
```

Neu PowerShell gap loi voi dau tieng Viet trong duong dan, chi can mo terminal dung ngay thu muc chua `main.py`, `simulation.py`, `radio_models.py`.

## 1. Cac thu muc can nho

```text
data/
  simulation_logs/   CSV that sinh tu simulator that
  raw/               du lieu cu hoac du lieu gia de test nhanh
  processed/         dataset da xu ly cho ML
  test_results/      ket qua danh gia tung file CSV
```

Y nghia nhanh:

- `simulation_logs/`: noi quan trong nhat khi sinh du lieu mo phong.
- `processed/`: noi `build_dataset` tao `X_train`, `y_train`, `X_test`, `y_test`.
- `test_results/`: noi luu bang ket qua, prediction va hinh ve sau khi evaluate.
- `raw/`: khong phai du lieu chinh de lam ket qua nghien cuu.

## 2. Chay mo phong co animation

Dung khi muon xem UE di chuyen va ket noi BS truc quan:

```powershell
python main.py
```

Chuong trinh se hoi:

```text
Enter RECTANGLE LENGTH (m)
Enter RECTANGLE WIDTH (m)
Enter road grid spacing
Enter number of UEs
Use aerial UE height?
Show UE - BS connection lines?
```

Sau khi chay xong, file CSV duoc luu vao:

```text
data/simulation_logs/
```

## 3. Sinh du lieu nhanh khong can animation

Dung lenh nay khi can tao file CSV de train/test model:

```powershell
python -m ml_sinr_prediction.generate_training_log
```

Lenh nay van chay simulator that, chi tat phan ve hinh nen nhanh hon.

Vi du chay nhanh khong can nhap tay:

```powershell
python -m ml_sinr_prediction.generate_training_log --rows 3000 --num-ues 1 --no-interactive
```

Ket qua tao ra:

```text
data/simulation_logs/<timestamp>_1_UE_Data.csv
```

## 4. Tao dataset cho hoc may

Neu muon dung file CSV moi nhat trong `data/simulation_logs/`:

```powershell
python -m ml_sinr_prediction.build_dataset
```

Neu muon chi dinh dung mot file cu the:

```powershell
python -m ml_sinr_prediction.build_dataset --csv data/simulation_logs/TEN_FILE.csv
```

Lenh nay tao ra:

```text
data/processed/X_train.npy
data/processed/y_train.npy
data/processed/X_test.npy
data/processed/y_test.npy
data/processed/feature_cols.json
data/processed/target_cols.json
data/processed/dataset_config.json
```

Mac dinh:

```text
history_len = 10
horizon = 1
test_size = 0.2
```

Nghia la model nhin 10 dong qua khu de du doan SINR cua buoc tiep theo.

## 5. Train model

Train Linear Regression:

```powershell
python -m ml_sinr_prediction.train_linear_regression
```

Train Random Forest:

```powershell
python -m ml_sinr_prediction.train_random_forest
```

Model duoc luu vao:

```text
models/linear_regression.joblib
models/random_forest.joblib
```

## 6. Danh gia model

Danh gia tren tap test tach tu file train:

```powershell
python -m ml_sinr_prediction.evaluate_models
```

Ket qua moi nhat nam o:

```text
data/processed/model_results.csv
data/processed/model_predictions.csv
```

Dong thoi chuong trinh tao them mot thu muc rieng:

```text
data/test_results/<ten_dataset>/
```

Thu muc nay thuong co:

```text
model_results.csv
model_predictions.csv
prediction_plot_*.png
error_histogram_*.png
parity_plot_*.png
```

## 7. Test model da train tren CSV moi

Dung khi da train model roi, bay gio co file simulator moi va muon test:

```powershell
python -m ml_sinr_prediction.evaluate_on_csv --csv data/simulation_logs/TEN_FILE_MOI.csv
```

Lenh nay:

- khong train lai model,
- khong build lai dataset train,
- chi dung model da co trong `models/`,
- luu ket qua vao `data/test_results/<TEN_FILE_MOI>/`.

## 8. Quy trinh de nho nhat

Train tu dau:

```powershell
python -m ml_sinr_prediction.generate_training_log
python -m ml_sinr_prediction.build_dataset
python -m ml_sinr_prediction.train_linear_regression
python -m ml_sinr_prediction.train_random_forest
python -m ml_sinr_prediction.evaluate_models
```

Test them mot file moi:

```powershell
python -m ml_sinr_prediction.generate_training_log
python -m ml_sinr_prediction.evaluate_on_csv --csv data/simulation_logs/TEN_FILE_MOI.csv
```

## 9. Nho nhanh

- Muon xem mo phong: `python main.py`
- Muon tao CSV nhanh: `python -m ml_sinr_prediction.generate_training_log`
- Muon bien CSV thanh dataset: `python -m ml_sinr_prediction.build_dataset`
- Muon train: chay `train_linear_regression` va `train_random_forest`
- Muon xem ket qua: `python -m ml_sinr_prediction.evaluate_models`
- CSV that nam trong `data/simulation_logs/`
- Ket qua ML nam trong `data/processed/` va `data/test_results/`
