5GReplay fuzz bằng cách **replay traffic 5G rồi sửa gói theo rule do người dùng định nghĩa** trước khi bơm lại vào target như AMF, SMF hoặc gNB. Nói cách khác, nó không tự sinh fuzz input hoàn toàn từ đầu mà lấy traffic thật hoặc pcap làm nền, dùng DPI để nhận diện đúng gói 5G cần sửa, rồi áp dụng các phép mutation lên các field giao thức. 

## Cơ chế chính

5GReplay hoạt động như một “one-way bridge” giữa input NIC/pcap và output NIC. Với mỗi gói đi vào, nó phân loại giao thức, trích xuất thuộc tính, so khớp với các rule XML; nếu rule khớp thì gói có thể bị sửa, forward hoặc drop, còn nếu không khớp thì xử lý theo default action trong file cấu hình. 

Rule của 5GReplay luôn mô tả ba phần: chọn gói nào, áp dụng hành động gì, và sửa trường nào như thế nào. Việc chọn gói dựa trên điều kiện logic theo field giao thức và cả chuỗi sự kiện theo thời gian, ví dụ “sau NAS Security Mode Command thì trong dưới 1 ms bắt được Security Mode Complete”. 

## Nó “fuzz” bằng gì

Bài báo mô tả 4 **atomic operators** để tạo mutant từ traffic gốc: xóa gói `DEL_PKT(P)`, đổi một thuộc tính header/message `CH_ATTR(P)`, đổi thứ tự hai gói `ORD(P1,P2)` và nhân bản gói `DUP_PKT(P)`; trong đó `ORD` được nêu là chưa triển khai ở thời điểm bài báo. Khi fuzz 5G, công cụ chủ yếu thao tác trên SCTP, NAS-5G và NGAP, tức là sửa các field 5G chuyên biệt thay vì chỉ đụng IP/TCP/UDP như các replay tool thông thường. 

Nền tảng của việc chọn đúng field để sửa là **Deep Packet Inspection** qua MMT-DPI. Nhờ vậy 5GReplay có thể nhận diện và trích các trường như `ngap.procedure_code`, `ngap.ran_ue_id`, `nas_5g.message_type`, `sctp_data.data_ppid` rồi cập nhật trực tiếp bằng rule. 

## Quy trình fuzz thực tế

Quy trình điển hình là: lấy traffic 5G thật hoặc pcap, viết rule XML để chọn các bản tin mục tiêu, đặt phép sửa field, rồi replay gói đã sửa vào mạng hoặc service đích. Target có thể là 5G core service như AMF hoặc môi trường RAN mô phỏng, và công cụ hỗ trợ cả chế độ offline từ pcap lẫn online trên luồng traffic sống. 

Ví dụ trong kịch bản online với 5G core, nhóm tác giả cấu hình 5GReplay để bắt các NGAP message trong quá trình authentication và đổi SCTP protocol identifier từ 60, là PPID của NGAP, thành 0 trước khi gửi tới AMF. Mục tiêu là tạo malformed NGAP packet “đủ đúng để đi qua pipeline” nhưng sai ở field quan trọng để thử độ robust của core service. 

## Ví dụ từ bài báo

Ở một ví dụ khác, 5GReplay sửa `ngap.ran_ue_id` trong Authentication Response bằng cách tăng thêm 100 rồi replay cùng phần còn lại của session, giúp IDS phát hiện bất thường giữa request và response. Điều này cho thấy fuzz ở đây là **field-aware mutation**, tức sửa có chủ đích trên đúng trường 5G đã parse được. 

Trong bài báo, khi gửi malformed NGAP packet tới free5GC thì AMF chỉ cảnh báo nhưng vẫn chạy tiếp, còn với open5GS thì AMF bị crash và không nhận kết nối UE mới nữa. Kết quả đó minh họa cách 5GReplay được dùng để kiểm tra robustness của implementation trước unexpected input ở runtime. 

## Điểm cần hiểu

5GReplay không phải coverage-guided fuzzer như CovFUZZ. Nó là một **traffic mutation and replay fuzzer**: mạnh ở chỗ bám theo traffic thật, lọc/sửa gói 5G rất linh hoạt bằng rule XML, hỗ trợ attack injection, replay attack, malformed packet testing và stress/DoS bằng cách nhân bản gói với tham số `nb-copies`. 

Nếu nhìn theo luồng ngắn gọn, cách 5GReplay fuzz là:
- Bắt hoặc nạp traffic 5G hợp lệ từ pcap/NIC. 
- Parse bằng DPI để hiểu đúng protocol và field 5G. 
- So khớp rule XML theo packet hoặc chuỗi event. 
- Áp dụng mutation như đổi field, xóa gói, nhân bản gói, hoặc replay lại gói. 
- Forward packet đã sửa tới AMF/gNB/IDS để quan sát crash, warning, replay acceptance hoặc DoS behavior. 

Ví dụ dễ hình dung: thay vì tự “bịa” ngẫu nhiên một bản tin NGAP, 5GReplay lấy một bản tin NGAP thật trong phiên đăng nhập UE, rồi sửa một field như PPID hoặc UE ID trước khi bơm lại vào AMF. Cách này thường tạo input thực tế hơn và dễ chạm tới logic xử lý sâu hơn. 

Bạn có muốn mình vẽ luôn sơ đồ so sánh **5GReplay vs CovFUZZ vs 5GHoul** theo mục tiêu fuzz, vị trí chèn gói và loại lỗi tìm được không?