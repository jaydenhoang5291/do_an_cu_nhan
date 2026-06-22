# Ghi chép nghiên cứu từ `Code` và `Code_HO`

Tài liệu này chỉ ghi những nội dung xác nhận được từ mã nguồn, cấu hình và dữ liệu hiện có.

## 1. Vai trò của hai hệ thống

- `Code` là bộ mô phỏng mức hệ thống vô tuyến: topology gNB lục giác, UE di chuyển Manhattan, LOS/NLOS, path loss, shadow fading, RSRP, công suất thu, SINR và quyết định đổi serving gNB.
- `Code_HO` là bộ giả lập giao thức UE--gNB kết nối 5G Core: nhận chuỗi measurement/handover từ CSV, đăng ký UE, thiết lập PDU session, chạy Xn/N2 handover, chèn loss/latency/jitter, theo dõi timer và xuất kết quả.

Chuỗi nghiên cứu là:

`radio scenario -> measurement CSV -> handover trigger -> protocol execution -> timer/delay evaluation`.

Hai hệ thống chưa dùng chung một schema CSV. `Code` xuất `ue0_connected_bs`; measurement reader của `Code_HO` yêu cầu `connected_gnb` và `gnbX_rsrp`.

TODO: cần xác nhận từ người dùng. Cần xác nhận file CSV chính thức và bước chuyển đổi schema giữa hai hệ thống.

## 2. Hệ thống `Code`

### Đầu vào

`main.py` nhận chiều dài/rộng vùng mô phỏng (mặc định 8000/5000 m), khoảng cách đường (200 m, tối thiểu 10 m), số UE (5), lựa chọn UE mặt đất/trên không, độ cao 1.5--300 m (mặc định 100 m), và lựa chọn hiển thị link. Constructor hỗ trợ `seed`, nhưng CLI không truyền seed.

### Tham số chính

| Tham số | Giá trị |
|---|---:|
| Công suất phát gNB | 46 dBm |
| Gain phát/thu | 2/0 dB |
| Tần số | 2 GHz |
| Băng thông | 10 MHz |
| Noise figure UE | 9 dB |
| HOM | 3 dB |
| LTE RB, subcarrier/RB | 50, 12 |
| Độ cao gNB/ground UE | 25/1.5 m |
| Cập nhật shadow fading | mỗi 25 m |
| Số bước | vòng 0--300, tức 301 mẫu |
| Thời gian mỗi bước | 1 s |

Thermal noise: `N = -174 + 10log10(B) + NF`, xấp xỉ -95 dBm với cấu hình hiện tại.

### Topology và mobility

Vùng nghiên cứu là hình chữ nhật giữa canvas 10000 m. gNB dùng lưới axial lục giác với bán kính hình học 500 m; khoảng cách tâm lân cận xấp xỉ 866 m. Số gNB phụ thuộc kích thước vùng.

UE bắt đầu ngẫu nhiên tại giao điểm, hướng East/North/West/South hợp lệ, tốc độ đều ngẫu nhiên 20--60 km/h, chỉ rẽ tại giao điểm và không ra ngoài biên. Sau 3 lần đổi hướng, UE dừng 9 bước rồi tăng tốc trong 2 bước.

### Mô hình vô tuyến

Xác suất LOS dùng UMa cho 1.5--22.5 m và UMa-AV ở độ cao lớn hơn. Trạng thái LOS/NLOS được lấy mẫu một lần cho từng cặp UE--BS rồi cache. Path loss dùng UMa/UMa-AV LOS hoặc NLOS. Shadow fading Gaussian zero-mean, clip tại +/-3 sigma, được lấy mẫu lại sau 25 m chuyển động tích lũy.

Tổng công suất thu:

`Prx = Ptx + Gtx + Grx - (path loss + shadow fading)`.

RSRP được xấp xỉ:

`RSRP = Prx - 10log10(50*12)`.

Interference lấy từ sáu gNB gần serving gNB nhất. SINR được tính trong miền tuyến tính rồi đổi sang dB.

### Quyết định handover

Ban đầu chọn gNB RSRP lớn nhất. Sau đó chỉ đổi cell nếu:

`RSRP_candidate > RSRP_serving + 3 dB`.

Không có time-to-trigger, A3 state machine, absolute threshold, hoặc quyết định dựa trên SINR. Việc đổi serving BS xảy ra ngay trong bước hiện tại và không có execution delay.

### Đầu ra

CSV chứa Step và, cho mỗi UE: x/y/height/direction, serving BS, LOS probability/state, path loss, shadow fading, RSRP, Prx, SINR, speed, handover flag, cùng tối đa sáu neighbor BS và RSRP.

## 3. Hệ thống `Code_HO`

### Kiến trúc

StormSIM giả lập UE và gNB, giao tiếp 5G Core qua SCTP/NGAP (N2) và GTP-U (N3). UE có state machine 5GMM/5GSM. Trước thí nghiệm, loss được tạm đặt 0, UE đăng ký và thiết lập PDU session; sau đó mới khôi phục impairment.

### Cấu hình và CLI

YAML định nghĩa N2/N3, danh sách gNB, TAC/PLMN/slice, profile và bảo mật UE, AMF, scenario, packet loss/latency/jitter cho radio và Xn. CLI chính: `-c`, `--csv`, `--csv-measurement`, `--gnb-map`, `--step-delay`, `--fail-mode`, `--cho-mode`.

`--csv` đọc `Step/Bước`, `ue0_BS_ketnoi`, `ue0_handover`, `ue0_handover_to_type`. `--csv-measurement` đọc `Step/Bước`, `connected_gnb`, các cột `gnbX_rsrp`, và `Type` tùy chọn. Trong measurement mode thường, trigger được suy ra khi `connected_gnb` thay đổi; StormSIM không tự chọn target từ RSRP.

### Handover và timer

Monitor dùng các pha:

`NULL -> PREPARE -> EXECUTE -> COMPLETE -> SUCCESS/FAIL`.

Delay trial được đo từ `StartHo()` đến khi target báo thành công hoặc scenario timeout.

| Timer | Giá trị trong code |
|---|---:|
| TXnRELOCprep / overall | 3 s / 6 s |
| TNGRELOCprep / overall | 5 s / 10 s |
| T301 | 2 s |
| T304 | 1000 ms |
| T310 | 1 s |
| T311 | 3 s |
| T312 | 500 ms |
| T430 | 30 s |

Test plan và comment cấu hình ghi T304 mặc định 100 ms, nhưng executable source đặt 1000 ms.

TODO: cần xác nhận từ người dùng. T304 chính thức là 100 ms hay 1000 ms.

Nếu T304 hết hạn, code có thể chạy chuỗi T311 rồi T301. T310/T312 hỗ trợ kịch bản radio-link problem. CHO evaluator chạy mỗi 500 ms và có A3/A5; điều kiện NTN position/coverage hiện chỉ trả về `true`.

TODO: cần xác nhận từ người dùng. CHO có thuộc kết quả chính hay chỉ là chức năng mở rộng.

### Đầu ra

- `timer-events-*.csv`: start/stop/timeout/cancel/restart/transport_fail;
- `handover-trial-summary-*.csv`: kết quả, lý do, total handover time;
- `timer_summary_*.csv`: vòng đời timer đã ghép;
- `ho-time-log.csv`: thời điểm các pha;
- anomaly CSV từ script Python.

Trường `pdr` trong log nhận trực tiếp từ biến `PacketLoss`; do đó 0.50 là loss probability 50%, không phải delivery ratio 50%.

## 4. Kết quả xác nhận được

Hai run Xn lớn cùng log delay 20 ms:

| Run | Loss probability (cột `pdr`) | Trial | Success | Rate | Median success delay |
|---|---:|---:|---:|---:|---:|
| 1781614531 | 0.10 | 210 | 210 | 100% | 200.574 ms |
| 1781622441 | 0.50 | 195 | 171 | 87.69% | 200.628 ms |

Run 0.50 có 24 fail `Timeout`; timer log cùng run có 24 `TXnRELOCprep transport_fail`, 24 prep timeout và 24 overall cancel.

Không được dùng T304 event làm mẫu số trial: số event không phủ đủ trial, và run 0.10 có một T304 timeout trong khi summary vẫn báo tất cả handover thành công.

Các hình trong `Images/archived_*.png` được tạo lại bằng `scripts/generate_thesis_figures.py` từ đúng hai run này. Hình `TXnRELOCprep` không cộng `transport_fail` và `timeout` thành hai nhóm trial khác nhau: trong run loss 0.50, đó là hai event trên cùng 24 lifecycle thất bại.

TODO: cần xác nhận từ người dùng. Hai run trên là kết quả chính thức hay log debug.

## 5. Giới hạn phải nêu

- Không có seed mặc định cho Python run.
- LOS state cache cố định theo link.
- Shadow fading lấy mẫu độc lập sau mỗi 25 m.
- RSRP là xấp xỉ từ tổng công suất trên 600 subcarrier.
- Interference chỉ lấy sáu cell tầng thứ nhất.
- Radio simulator không có handover execution delay.
- CSV interface chưa thống nhất.
- `pdr` thực chất chứa packet-loss probability.
- T304 trong code và test plan không thống nhất.
- Normal measurement mode lấy trigger từ `connected_gnb`.
- CHO NTN position/coverage chưa triển khai.
- Chưa có bộ kết quả đầy đủ, được gắn nhãn rõ cho toàn bộ PDR-delay matrix, N2 comparison, T304 breakpoint, RLF cascade và beam failure.
