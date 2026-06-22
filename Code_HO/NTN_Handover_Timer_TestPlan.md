# NTN Handover Timer Test Plan

## Mục tiêu tổng quan

Đánh giá liệu các giá trị timer handover 5G NR hiện tại có đủ headroom để hoạt động đúng
khi áp dụng lên mạng NTN (Non-Terrestrial Network), nơi delay lan truyền cao hơn đáng kể
so với mạng mặt đất (1.2 – 13.3 ms one-way).

---

## Môi trường test chung

- **Emulator:** gNB + UE viết bằng Go, kết nối 5G Core qua N2/N3
- **RSRP input:** file đo sẵn, inject liên tục để trigger handover decision
- **Delay injection:** thêm propagation delay nhân tạo vào giao tiếp Xn / NG
- **PDR injection:** mô phỏng packet loss trên đường truyền
- **Số lần lặp tối thiểu:** 30 trials mỗi tổ hợp (PDR × delay)

### Giá trị timer cố định trong tất cả kịch bản

| Timer | Giá trị | Ý nghĩa |
|---|---|---|
| TXnRELOCprep | 3s | Xn-based HO preparation |
| TXnRELOCoverall | 6s | Xn-based HO overall |
| TNGRELOCprep | 5s | NG-based HO preparation |
| TNGRELOCoverall | 10s | NG-based HO overall |
| T301 | 2s | RRC Reestablishment request |
| T304 | 100ms | HO execution (timer ngắn nhất — rủi ro cao nhất) |
| T310 | 1s | Radio Link Failure detection |
| T311 | 3s | RRC Reestablishment (sau RLF) |
| T312 | 500ms | Beam Failure Recovery |

---

## Cấu trúc log chung

Mỗi event ghi ra một dòng JSON (NDJSON). Có hai loại entry:

### Timer event entry
Ghi tại mỗi điểm start / stop / timeout của từng timer.

```
ts               time.Time   — timestamp tuyệt đối
trial_id         int         — số thứ tự lần HO trong session
pdr              float64     — mức PDR đang inject (0.0, 0.70, 0.80, 0.90, 0.95, 1.0)
delay_ms         float64     — delay one-way đang inject (1.2, 5.0, 10.0, 13.3)
ho_type          string      — "Xn" | "NG"
timer_name       string      — "T304" | "T310" | "T311" | "T312" | "TXnRELOCprep" | ...
event            string      — "start" | "stop" | "timeout"
elapsed_ms       float64     — thời gian thực tế từ start đến event này
result           string      — "success" | "timeout" | "aborted"
ho_trigger       string      — "rsrp_drop" | "beam_fail" | "manual"
source_gnb       string      — ID gNB nguồn
target_gnb       string      — ID gNB đích
rsrp_at_trigger  float64     — RSRP (dBm) tại thời điểm trigger HO
parent_timer     string      — timer nào chạy trước (dùng cho cascade T310→T311→T301)
rlf_detected     bool        — có RLF xảy ra trong lần HO này không
```

### Trial summary entry
Ghi một lần sau khi toàn bộ HO attempt kết thúc (dù thành công hay thất bại).

```
ts               time.Time
trial_id         int
pdr              float64
delay_ms         float64
ho_type          string
overall_result   string      — "success" | "fail"
fail_reason      string      — timer nào gây fail, hoặc "" nếu success
total_ho_ms      float64     — tổng thời gian từ trigger đến hoàn tất (hoặc fail)
```

> Dùng `event = "summary"` hoặc tách file riêng đều được.
> Trial summary giúp tính success rate mà không cần join nhiều dòng.

---

## Kịch bản 1 — PDR × Delay matrix (kịch bản chính)

### Mục tiêu
Đo hành vi của tất cả timer tại mọi tổ hợp PDR và delay. Đây là kịch bản baseline
để kết luận timer nào "sống được" ở điều kiện NTN nào.

### Setup
- HO type: **Xn-based** (chạy trước), sau đó lặp lại với **NG-based**
- PDR levels: 0% (control), 70%, 80%, 90%, 95%, 100%
- Delay levels: 1.2 ms, 5 ms, 10 ms, 13.3 ms
- Số trials: 30 mỗi ô

**Ma trận test:** 6 PDR × 4 delay = 24 tổ hợp × 2 HO type = **48 tổ hợp**

### Log cần ghi thêm (ngoài common fields)
Không cần thêm — common fields là đủ.

### Metrics cần extract từ log

| Metric | Cách tính |
|---|---|
| Timeout rate T304 | count(result=timeout, timer=T304) / total trials |
| Timeout rate T310/T311 | tương tự |
| Xn/NG RELOC success rate | count(overall_result=success) / total trials |
| P50/P95 latency (elapsed_ms) | percentile của elapsed_ms khi result=success |



---

## Kịch bản 2 — T304 breakpoint (tìm ngưỡng delay tối đa)

### Mục tiêu
T304 = 100 ms là timer ngắn nhất và rủi ro cao nhất. Với NTN delay 13.3 ms one-way
(~26.6 ms RTT), HO execution window thực tế còn lại rất hẹp. Kịch bản này tìm chính xác
ngưỡng delay mà T304 bắt đầu fail.

### Setup
- PDR: **0%** (loại bỏ biến số packet loss, chỉ test delay)
- Delay: tăng từ **1 ms đến 20 ms**, bước nhảy **0.5 ms**
- Số trials: **50 mỗi bước** (cần sample lớn hơn vì tìm breakpoint)
- HO type: Xn-based

### Log cần ghi thêm
```
packet_loss_count   int     — số gói thực tế bị drop trong lần HO này (đếm thật, không phải config PDR)
```

Lý do: dù PDR = 0% vẫn có thể có drop bất thường, cần biết để loại outlier.

### Metrics cần extract từ log

| Metric | Cách tính |
|---|---|
| T304 timeout rate tại mỗi delay step | count(timeout) / 50 |
| Breakpoint delay | delay nhỏ nhất mà timeout rate vượt 5% |
| elapsed_ms distribution | vẽ histogram để thấy T304 bị ăn bao nhiêu ms |

---

## Kịch bản 3 — RLF cascade: T310 → T311 → T301

### Mục tiêu
Khi radio link fail xảy ra **trong lúc HO đang diễn ra**, chuỗi timer T310 → T311 → T301
có đủ thời gian để reestablish không? Đây là worst-case vì UE vừa HO vừa mất link.

### Setup
- Trigger RLF nhân tạo: inject RSRP xuống dưới ngưỡng N310 out-of-sync **sau khi HO đã start**
  nhưng **trước khi HO complete**
- PDR: 0%, 70%, 90%
- Delay: 5 ms và 13.3 ms
- Số trials: 30 mỗi tổ hợp

### Log cần ghi thêm
```
rlf_timestamp_offset_ms   float64   — RLF xảy ra sau bao nhiêu ms kể từ lúc HO start
                                       (biết RLF rơi vào giai đoạn nào: prep / exec / complete)
rsrp_during_ho            float64   — snapshot RSRP tại thời điểm RLF xảy ra
reestab_target_gnb        string    — UE chọn cell nào để reestablish
```

### Metrics cần extract từ log

| Metric | Cách tính |
|---|---|
| T310 expire rate | count(timeout, timer=T310) / total |
| T311 expire rate | count(timeout, timer=T311) / total |
| RRC Reestablishment success rate | count(result=success, timer=T301) / total |
| Correlation: rlf_offset vs cascade result | scatter plot rlf_offset_ms × overall_result |

Correlation cuối giúp biết: RLF càng muộn trong quá trình HO thì cascade càng fail hay không.

---

## Kịch bản 4 — Xn vs NG path comparison

### Mục tiêu
So sánh trực tiếp Xn-based HO và NG-based HO dưới cùng điều kiện NTN.
NG-based đi qua AMF nên thêm latency, nhưng timer dài hơn (5s/10s vs 3s/6s) — liệu có đủ bù không?

### Setup
- Chạy cùng test conditions nhưng lần lượt:
  - **Xn enabled:** HO dùng Xn interface bình thường
  - **Xn disabled:** force NG-based HO qua AMF
- PDR: 0%, 70%, 90%
- Delay: 1.2 ms và 13.3 ms (hai extreme)
- Số trials: 30 mỗi tổ hợp

### Log cần ghi thêm
```
n2_rtt_ms     float64   — RTT đo được trên path N2 (gNB ↔ AMF) cho NG-based HO
                           để tách biệt phần latency do core vs do radio
```

### Metrics cần extract từ log

| Metric | Cách tính |
|---|---|
| Timer headroom còn lại | timer_value - elapsed_ms (trung bình) |
| Success rate Xn vs NG tại cùng PDR/delay | so sánh trực tiếp |
| n2_rtt contribution | n2_rtt_ms / elapsed_ms — core chiếm bao nhiêu % thời gian |

---

## Kịch bản 5 — Beam failure + T312

### Mục tiêu
T312 = 500 ms trigger khi beam failure xảy ra. Trong NTN, beam switching có thể chậm hơn
do satellite geometry thay đổi chậm. Kiểm tra T312 có đủ không.

### Setup
- Inject beam failure: drop RSRP của beam hiện tại xuống dưới ngưỡng beam failure
- Delay: 10 ms và 13.3 ms (chỉ test high-delay vì đây là vấn đề NTN-specific)
- PDR: 0%, 70%, 90%
- Số trials: 30 mỗi tổ hợp

### Log cần ghi thêm
```
beam_id              string    — beam ID đang dùng khi failure xảy ra
candidate_beam_id    string    — beam được chọn để recovery
beam_recovery_ms     float64   — thời gian từ lúc beam fail đến lúc beam mới confirmed
                                  (đây là thứ cần so với T312 = 500ms)
```

### Metrics cần extract từ log

| Metric | Cách tính |
|---|---|
| T312 timeout rate | count(timeout, timer=T312) / total |
| beam_recovery_ms distribution | P50/P95 — bao lâu mới recover được beam |
| Margin còn lại | 500ms - beam_recovery_ms (P95) |

---

## Tổng hợp output CSV

Sau khi chạy xong tất cả kịch bản, parser Go đọc file log và xuất các file CSV:

### results_per_timer.csv
```
scenario, pdr, delay_ms, ho_type, timer_name, total_trials,
success_count, timeout_count, success_rate,
elapsed_p50_ms, elapsed_p95_ms
```

### results_trial_summary.csv
```
scenario, pdr, delay_ms, ho_type, trial_id,
overall_result, fail_reason, total_ho_ms
```

### results_t304_breakpoint.csv (kịch bản 2 riêng)
```
delay_ms, total_trials, timeout_count, timeout_rate
```

### results_cascade.csv (kịch bản 3 riêng)
```
pdr, delay_ms, trial_id, rlf_offset_ms, rsrp_during_ho,
t310_result, t311_result, t301_result, overall_result
```

---

## Thứ tự chạy đề xuất

1. Kịch bản 2 trước (T304 breakpoint) — ngắn nhất, xác định ngay rủi ro lớn nhất
2. Kịch bản 1 (PDR matrix) — dài nhất, chạy qua đêm
3. Kịch bản 4 (Xn vs NG) — phụ thuộc kết quả kịch bản 1
4. Kịch bản 3 (RLF cascade) — cần setup phức tạp hơn
5. Kịch bản 5 (Beam failure) — nếu còn thời gian / beam control sẵn sàng
