AMFuzz fuzz bằng cách **đứng giữa gNB/UE simulator và AMF như một proxy**, chặn các bản tin NGAP/NAS hợp lệ được sinh ra lúc chạy thật, rồi giải mã, sửa một số trường theo seed ngẫu nhiên, sau đó mã hóa/ký lại nếu cần và chuyển tiếp vào AMF. Vì thế nó là mutation-based **black-box fuzzing** cho 5G Core, tập trung vào AMF, nhưng vẫn đủ “protocol-aware” để không bị loại ngay ở bước parser. 

## Ý tưởng chính

Bài báo chọn AMF vì đây là network function có thể bị tác động từ bên ngoài thông qua UE và gNB, đặc biệt qua NAS và NGAP. Thay vì replay pcap cũ hay fuzz byte thô, AMFuzz tạo thông điệp hợp lệ tại runtime bằng simulator, rồi chỉ mutate ngay trước khi gửi sang AMF để vẫn giữ đúng trạng thái thủ tục và dữ liệu phiên hiện tại. 

## Kiến trúc fuzz

AMFuzz dựng một **proxy component** nằm giữa RAN và AMF để transparently forward các bản tin NGAP/NAS, và gắn module fuzzing như một hook trên từng message đi qua proxy. Kiến trúc này tách riêng proxy, simulator và fuzzer nên có thể chạy cùng một thủ tục với bản tin gốc hoặc bản tin đã bị sửa, đồng thời dễ thay simulator mà không phải sửa logic fuzz. 

## Quy trình trên mỗi gói

Mỗi packet đi qua AMFuzz được xử lý theo ba bước: **pre-processing**, fuzzing và post-processing. Ở pre-processing, công cụ giải mã bản tin theo ASN.1, và nếu gói chứa NAS được bảo vệ thì nó còn giải mã NAS và chuẩn bị key/session context; ở post-processing, nó mã hóa lại NAS, tạo lại integrity MAC và encode lại đúng chuẩn trước khi gửi sang AMF. 

Điểm quan trọng là AMFuzz không sửa ciphertext bừa bãi. Nó thu thập thông tin từ quá trình authentication để suy ra session keys, dùng các tham số cấu hình sẵn như OPC và subscriber key cùng với dữ liệu runtime như RAND và thuật toán bảo mật để tạo khóa mã hóa và khóa integrity, nhờ đó vẫn gửi được các NAS message “đúng bảo vệ” sau khi đã mutate. 

## Nó mutate như thế nào

AMFuzz dùng một chuỗi byte ngẫu nhiên gọi là **seed** để quyết định sẽ sửa gì trong packet. Với mỗi message, seed được ánh xạ vào một information element (IE) cụ thể trong NAS hoặc NGAP, rồi đổi giá trị field theo đúng kiểu dữ liệu và theo cấu trúc của đúng loại message đó, vì mỗi message type có tập IE khác nhau. 

Ngoài body, một số seed còn được ánh xạ để sửa header như procedure code, message type, security type hoặc MAC-related fields khi phù hợp. Một số trường hợp khác không sửa nội dung bên trong mà thay đổi hành vi gửi, chẳng hạn gửi trễ hoặc gửi lặp lại nhiều lần để kiểm tra xử lý trạng thái và timing của AMF. 

## Vì sao cách này hiệu quả

Bài báo nhấn mạnh rằng chỉ dùng pcap làm seed corpus rồi mutate thô là không hiệu quả trong 5G, vì AMF là stateful và các thủ tục như registration hay PDU session setup đòi hỏi phản hồi nhất quán với challenge mới do AMF sinh ra. AMFuzz giải quyết điểm này bằng cách để UE/RAN simulator sinh message mới, đúng theo trạng thái hiện tại của mạng, rồi mới mutate ở “phút cuối”, nên vẫn đi sâu hơn vào logic của AMF thay vì chết sớm ở parser hoặc bước xác thực. 

## Theo dõi crash và lặp lại test

Để phát hiện lỗi, proof-of-concept của AMFuzz dùng monitor kiểu grey-box theo dõi container AMF, ví dụ qua Docker API, nhằm biết AMF có crash hay không. Để tái lập test, họ không dựa vào replay pcap vì challenge-response, integrity và timing sẽ thay đổi giữa các lần chạy; thay vào đó, họ lưu **random seed** và chạy lại cùng thủ tục với seed cũ để tái tạo cùng kiểu mutation trên các message mới được sinh ra ở runtime. 

## Khác với 5GReplay

Khác với 5GReplay, AMFuzz không chủ yếu dựa vào replay trace và chỉnh rule trên packet đã capture. Bài báo nói rõ họ lấy cảm hứng từ 5GReplay nhưng mở rộng sang fuzz **message body** và các information element bên trong NAS/NGAP, chứ không chỉ dừng ở việc lọc, nhân bản hay chỉnh các field header của packet. 

Ví dụ dễ hiểu là: thay vì lấy một Registration Request cũ trong pcap và sửa lại để replay, AMFuzz để UE simulator tạo một Registration Request mới phù hợp với phiên hiện tại, proxy chặn nó, giải mã, đổi một IE như 5G Mobility Identity hoặc field khác, rồi mã hóa/ký lại và gửi vào AMF. Cách này giúp test được các bug sâu trong xử lý logic của AMF mà vẫn giữ thủ tục 5G đủ hợp lệ để tiếp tục chạy. 

Bạn có muốn mình làm tiếp bảng **AMFuzz vs 5GReplay vs CovFUZZ** theo tiêu chí: điểm chèn gói, awareness mức giao thức, xử lý NAS security, feedback, và khả năng black-box không?