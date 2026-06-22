CovFUZZ kiểm thử 5G Core thông qua một kiến trúc **downlink fuzzing** tích hợp, sử dụng srsRAN Project (gNB) kết hợp với Open5GS làm 5G Core Network. Dưới đây là giải thích chi tiết từng bước hoạt động:

***

## Kiến Trúc Tổng Thể

CovFUZZ gồm ba thành phần chính kết hợp với nhau :

- **4G/5G Protocol Stack Implementation** – sinh các gói tin hợp lệ (benign packets) để làm nền cho việc mutation
- **DUT (Device Under Test)** – mục tiêu fuzzing, trong 5G downlink là các thiết bị UE (điện thoại, modem, thiết bị IoT, srsUE giả lập)
- **Fuzzing Controller** – điều phối toàn bộ luồng fuzzing

> **Lưu ý**: CovFUZZ chỉ hỗ trợ **5G downlink fuzzing** (gNB → UE), không hỗ trợ uplink vì srsUE phiên bản 5G chưa có chức năng reset/restart cần thiết giữa các iteration .

***

## Luồng Fuzzing Từng Bước

**Bước 1 – Sinh gói tin hợp lệ (Benign Packet Generation)**

srsGNB và Open5GS CN khởi tạo một Attach Procedure bình thường với UE mục tiêu. Các gói tin trao đổi trong quá trình này (RRC Connection Setup, Authentication Request, Security Mode Command, v.v.) được dùng làm "vật liệu thô" để mutate .

**Bước 2 – Chặn gói tin (Packet Interception)**

CovFUZZ cài **packet interception hooks** vào trong mã nguồn srsGNB tại hai tầng :
- **Tầng RRC** – cho phép thay đổi các trường NAS và RRC (trước khi mã hóa/bảo vệ toàn vẹn)
- **Tầng MAC** – cho phép thay đổi các trường PDCP, RLC, MAC

Cơ chế hook sử dụng **shared memory interface** để gửi gói tin sang Fuzzing Controller và **tạm dừng** luồng protocol cho đến khi gói đã được fuzz được trả về. Đây là lý do tại sao CovFUZZ có thể thao tác trên các trường bên trong mà không bị phá vỡ bởi mã hóa – vì interception xảy ra ở tầng cao hơn nơi mã hóa được áp dụng .

**Bước 3 – Phân tích & Đột biến gói tin (Dissect & Mutate)**

Fuzzing Controller nhận gói tin, sau đó :
1. **Dissector** phân tích cấu trúc gói, xác định loại gói và trích xuất các **Field** (tên, offset, độ dài, mask bit)
2. **Fuzzer** áp dụng mutation dựa trên **mutation probability** \(p_f^i\) của từng field:
   - Nếu số ngẫu nhiên `< p_f`, field đó được chọn để mutate
   - Mutator được chọn ngẫu nhiên trong: `RAND`, `MAX`, `MIN`, `ADD`, `SUB`

**Bước 4 – Gửi lại & Quan sát (Send & Monitor)**

Gói đã bị fuzz được gửi lại qua shared memory vào srsGNB, rồi chuyển tiếp đến DUT (UE). Fuzzing Controller quan sát xem DUT có :
- **Crash** (kiểm tra process còn sống không, hoặc qua AT commands với COTS UE)
- **Hang** (timeout)

**Bước 5 – Thu thập Coverage & Điều chỉnh xác suất**

Đây là điểm cốt lõi của Coverage-based Fuzzer. Sau mỗi iteration, Coordinator thu thập **edge coverage** (các cạnh trong control flow graph được thực thi) từ DUT . Xác suất mutation được cập nhật theo công thức:

\[ p_f^i \leftarrow p_f^{i-1} + \frac{F(c_i, i)}{\log_2(|V_f| + 1)} \]

Trong đó :
- \(F(c_i, i) = f(c) \cdot g(i) / n_i\) — hàm điều chỉnh dựa trên coverage mới tìm được
- \(f(c) = +1\) nếu tìm được coverage mới, \(-1\) nếu không
- \(g(i)\) tăng dần theo iteration để "thưởng" mạnh hơn khi coverage mới tìm được ở giai đoạn sau
- \(\log_2(|V_f| + 1)\) chuẩn hóa theo số giá trị có thể của field (field có nhiều giá trị cần nhiều iteration hơn)

**Bước 6 – Reset & Lặp lại**

Open5GS được patch để **"quên"** UE sau mỗi attach procedure hoàn chỉnh, đảm bảo NAS signaling đầy đủ xuất hiện trong mọi iteration (thay vì dùng fast re-attach) . UE được reset bằng cách bật/tắt airplane mode (AT command hoặc ADB) trước khi bắt đầu iteration tiếp theo.

***

## Black-box vs Grey-box với 5G

| Kịch bản | Coverage Source | Cách hoạt động |
|---|---|---|
| **Grey-box** | Trực tiếp từ DUT | DUT được instrument bằng LLVM coverage sanitizer  |
| **Black-box** | Ước tính từ srsGNB | Dùng coverage của srsGNB (open-source) để **xấp xỉ** coverage của DUT thực  |

Kết quả thực nghiệm cho thấy Coverage-based Fuzzer vượt trội Random Fuzzer tới **47.6% (grey-box)** và **23.9% (black-box)** về code coverage trong downlink fuzzing . Khi áp dụng trên 12 thiết bị COTS 4G/5G thực tế, nhóm nghiên cứu phát hiện lỗ hổng trên **10 trong số 12 thiết bị** .