Với StormSIM, hướng **đáng làm nhất** không phải replay packet đơn thuần mà là một stateful, structure-aware, grey-box fuzzer cho 5GC: dùng state machine UE sẵn có để đưa hệ thống vào đúng thủ tục rồi mới mutate các NAS/NGAP message ở thời điểm phù hợp. Các công trình gần đây cho thấy 5G fuzzing hiệu quả khi kết hợp ba ý tưởng: tạo message hợp lệ tại runtime rồi chặn để sửa, dùng feedback theo state/coverage để đi sâu hơn vào logic, và thu coverage từ nhiều thành phần thay vì chỉ nhìn một NF đơn lẻ. [arxiv](https://arxiv.org/abs/2602.21794)

## Hướng nên chọn

StormSIM nên được xem là **fuzzing harness + protocol executor**, còn việc là black-box, grey-box hay white-box phụ thuộc chủ yếu vào mức quan sát bên trong DUT mà bạn có được. Với bối cảnh 5GC, CoreCrisis đi theo hướng stateful black-box bằng cách học FSM rồi ưu tiên các state còn ít được khám phá, còn AMFuzz cho thấy với AMF thì tạo thông điệp hợp lệ tại runtime rồi intercept để mutate trước khi gửi sang core thực tế hơn hẳn cách dùng pcap seed thô. [ppl-ai-file-upload.s3.amazonaws](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/335307993/461da88b-eccb-47d2-a18a-0420ff72913f/IEEE-5G-Communication-and-Security-__GlobeComm-2024.pdf)

Vì bạn đã có NAS, NGAP và UE MM/SM state machine, StormSIM đang ở vị trí rất thuận lợi để làm một **hybrid fuzzer**: generation-based ở mức thủ tục, mutation-based ở mức message/IE, và grey-box ở mức feedback nếu bạn instrument được Open5GS, OAI hoặc free5GC. CovFUZZ và MulCovFuzz đều cho thấy coverage-guided fuzzing mang lại coverage và crash discovery tốt hơn các baseline ngẫu nhiên trong môi trường 4G/5G. [arxiv](https://arxiv.org/pdf/2410.20958v1.pdf)

## Kiến trúc nên dựng

AMFuzz dùng pipeline ba bước rất hợp với StormSIM: pre-processing để decode ASN.1 và NAS bảo vệ, fuzzing để chọn field/IE mà sửa, rồi post-processing để encode lại và tính lại integrity/encryption trước khi chuyển vào core.  Với StormSIM, tôi khuyên tách thành 5 module rõ ràng: Scenario Driver, Message Hook, Mutator Engine, Feedback/Oracle, và Corpus Manager.

- Scenario Driver: `set_target_state(ue_id, target_state)`; UE state machine tự chạy các event để đi tới Registration, Security Mode, PDU Session Establishment, Service Request, Deregistration.
- Message Hook: chặn đúng các bản tin NAS/NGAP trước khi gửi ra N1/N2; đây là nơi tiêm mutation thay vì sửa byte sau khi đã mã hóa.
- Security Context Manager: giữ NAS count, KNASenc, KNASint, algorithm, rồi MAC/encrypt lại sau mutation; đây là điểm AMFuzz nhấn mạnh là bắt buộc nếu muốn message đi qua sâu hơn parser. 
- Corpus Manager: lưu `scenario_id`, `target_state`, `message_type`, `IE_path`, `seed`, `delay`, `dup_count`, `response_signature`, `coverage_vector`.

## Feedback nên tối ưu

Đừng dùng mỗi tín hiệu crash/no-crash. CovFUZZ điều chỉnh xác suất mutation theo coverage của field, CoreCrisis dùng FSM học được để nhắm vào underexplored states và tiếp tục refine mô hình bằng phản hồi quan sát được, còn MulCovFuzz thu coverage từ nhiều component và dùng hàm điểm kết hợp coverage với efficiency. [arxiv](https://arxiv.org/abs/2602.21794)

Trong StormSIM, tôi sẽ dùng điểm số nhiều tầng:
- State novelty: state hoặc transition mới của thủ tục UE/AMF/SMF.
- Component coverage: edge/branch/line theo từng NF, ví dụ AMF, SMF, AUSF, UDM, UPF, nếu bạn instrument được.
- Semantic anomaly: reject bất thường, timeout lệch chuẩn, state divergence giữa hai core, crash, restart, memory growth, CPU spike.

Một công thức MVP đủ tốt là: `score = a*state_novelty + b*coverage_novelty + c*semantic_anomaly + d*cross_core_divergence - e*execution_cost`. Ý tưởng này hợp với StormSIM hơn fuzzing truyền thống vì bạn đã kiểm soát được cả thủ tục lẫn số lượng lớn UE/gNB đồng thời.

## Mutator nên có

AMFuzz cho thấy mutation phải “message-type aware”: cùng một seed nhưng mapping khác nhau theo từng NAS/NGAP message type và từng IE; ngoài sửa field, họ còn thử gửi trễ hoặc gửi lặp lại để tác động lên logic trạng thái.  Vì vậy, mutator của StormSIM nên chia thành 4 lớp thay vì một engine chung.

- IE-aware mutators: enum invalid-but-decodable, length boundary, presence/optional IE, spare bits, wrong identity type, inconsistent security header, invalid cause, weird NSSAI/DNN/PDU session attributes.
- Cross-field mutators: giữ message hợp lệ về encoding nhưng phá ràng buộc logic, ví dụ UE security context cũ + NAS count mới, S-NSSAI không khớp DNN, PDU session ID hợp lệ nhưng state chưa sẵn sàng.
- Sequence mutators: bỏ một bước, lặp một bước, gửi sớm, gửi muộn, resend sau reject, xen kẽ thủ tục MM và SM, tạo race giữa Registration Update và PDU Session Release.
- Concurrency mutators: đây là lợi thế riêng của StormSIM; cho nhiều UE dùng cùng SUPI/GUTI, cùng yêu cầu session, handover giả, deregister song song, hoặc nhiều gNB gửi tín hiệu cạnh tranh để tìm race condition và shared-state bugs.

## Lộ trình MVP

AMFuzz trong bản proof-of-concept chỉ tập trung vào một tập con message của registration, session setup và deregistration mà vẫn tìm được 7 bug trên ba open-source cores, nên cách bắt đầu hẹp nhưng sâu là hợp lý.  Tôi sẽ đi theo 4 pha ngắn thay vì xây “fuzzer tổng quát” ngay từ đầu.

1. Pha 1: Chỉ fuzz 2 thủ tục, Registration và PDU Session Establishment; instrument Open5GS/OAI/free5GC để lấy coverage, log seed đầy đủ để tái lập test.  
2. Pha 2: Thêm 20–30 mutator IE-aware và 8–10 sequence mutator; ưu tiên các message protected bằng NAS vì AMFuzz ghi nhận nhiều bug tập trung ở NAS hơn NGAP.   
3. Pha 3: Thêm scheduler theo state coverage và multi-component coverage; đây là bước học từ CoreCrisis, CovFUZZ và MulCovFuzz. [arxiv](https://arxiv.org/pdf/2410.20958v1.pdf)
4. Pha 4: Thêm differential fuzzing giữa Open5GS, free5GC, OAI; chính AMFuzz cũng quan sát khác biệt hành vi giữa các core và xem trace comparison là hướng mở rộng quan trọng. 

Nếu phải chốt một kiến trúc đầu tiên cho StormSIM, tôi sẽ chọn: **state-targeted scenario generation + IE-aware mutation + NAS security re-signing + multi-component coverage + differential oracle**. Bạn muốn tôi phác thảo luôn design chi tiết của vòng lặp fuzz, schema cho corpus, và bộ mutator ưu tiên cho Registration/PDU Session không?