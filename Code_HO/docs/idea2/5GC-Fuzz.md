5GC-Fuzz dùng fuzz test theo hướng **stateful black-box fuzzing** cho 5G Core: nó không chỉ sửa ngẫu nhiên từng gói riêng lẻ mà trước hết học cách hệ thống 5GC phản ứng theo chuỗi bản tin, rồi mới sinh các test case để đẩy implementation vào những trạng thái sâu hơn trước khi mutate đầu vào. Theo mô tả tóm tắt của bài báo, framework này nhắm tới việc tìm “deep stateful vulnerabilities” và đạt số trạng thái khám phá cùng số lỗi cao hơn các baseline trước đó. [semanticscholar](https://www.semanticscholar.org/paper/5GC-Fuzz:-Finding-Deep-Stateful-Vulnerabilities-in-Sun-Liu/9454bc47171af2cf1cb125d9aba978b6807e9b5d)

## Ý tưởng chính

Điểm cốt lõi của 5GC-Fuzz là coi 5GC như một hệ thống **phụ thuộc trạng thái**, nơi nhiều lỗi chỉ xuất hiện sau một chuỗi NAS/NGAP hợp lệ chứ không lộ ra khi chỉ fuzz từng message độc lập. Vì vậy, thay vì replay một gói lẻ hoặc mutate byte thô, nó kết hợp ba thành phần: học trạng thái, sinh chuỗi kiểm thử, và mutation có định hướng trên các thông điệp hợp lệ trong tiến trình giao thức. [ieeexplore.ieee](https://ieeexplore.ieee.org/document/11044489/)

## Cách framework fuzz

Từ phần mô tả công khai, 5GC-Fuzz trước tiên xây dựng hiểu biết về các trạng thái mà 5GC có thể đi qua bằng cách quan sát phản hồi của hệ thống đối với các chuỗi bản tin. Sau đó nó tạo hoặc chọn các chuỗi đầu vào có khả năng đưa core network tới các trạng thái sâu, rồi thực hiện fuzz trên các message trong chuỗi đó để kích hoạt các bug logic hoặc bug xử lý chỉ xảy ra sau khi state machine đã tiến đủ xa. [semanticscholar](https://www.semanticscholar.org/paper/5GC-Fuzz:-Finding-Deep-Stateful-Vulnerabilities-in-Sun-Liu/9454bc47171af2cf1cb125d9aba978b6807e9b5d)

Nói đơn giản, luồng của nó có thể hiểu là:
- Gửi các chuỗi message hợp lệ để thăm dò state machine của 5GC. [ieeexplore.ieee](https://ieeexplore.ieee.org/document/11044489/)
- Suy ra hoặc cập nhật các trạng thái/phản hồi quan sát được từ bên ngoài, tức theo kiểu black-box. [semanticscholar](https://www.semanticscholar.org/paper/5GC-Fuzz:-Finding-Deep-Stateful-Vulnerabilities-in-Sun-Liu/9454bc47171af2cf1cb125d9aba978b6807e9b5d)
- Ưu tiên các chuỗi giúp mở rộng số state đã khám phá. [semanticscholar](https://www.semanticscholar.org/paper/5GC-Fuzz:-Finding-Deep-Stateful-Vulnerabilities-in-Sun-Liu/9454bc47171af2cf1cb125d9aba978b6807e9b5d)
- Chèn mutation vào các message trong những chuỗi đó để kiểm tra xử lý sâu trong AMF hay thành phần core liên quan. [ieeexplore.ieee](https://ieeexplore.ieee.org/document/11044489/)

## Điểm khác với các fuzzer kia

So với 5GReplay, 5GC-Fuzz không thiên về packet replay + rule-based mutation trên traffic capture sẵn. So với CovFUZZ, nó không dựa vào code coverage nội bộ của DUT mà vẫn hoạt động ở chế độ black-box, nhưng bù lại tập trung mạnh vào **khám phá trạng thái** để đi sâu vào logic của 5GC. [ieeexplore.ieee](https://ieeexplore.ieee.org/document/11044489/)

## Mục tiêu của cách này

Lý do cách fuzz này hữu ích là vì nhiều lỗ hổng trong 5G Core nằm ở phần xử lý thủ tục nhiều bước, ví dụ chỉ lộ ra sau khi registration, authentication hoặc session management đã đi qua một số pha nhất định. Bài báo nhấn mạnh đúng mục tiêu đó bằng cụm “finding deep stateful vulnerabilities”, nghĩa là lỗi không nằm ở parser đơn giản mà ở phần hành vi phụ thuộc ngữ cảnh và trạng thái trước đó của core network. [semanticscholar](https://www.semanticscholar.org/paper/5GC-Fuzz:-Finding-Deep-Stateful-Vulnerabilities-in-Sun-Liu/9454bc47171af2cf1cb125d9aba978b6807e9b5d)

Hiện mình mới có thể giải thích ở mức đáng tin cậy dựa trên metadata công khai và abstract/indexing của bài báo, vì bản PDF tại OpenReview chưa tải nội dung đầy đủ được trong phiên này. [ieeexplore.ieee](https://ieeexplore.ieee.org/document/11044489/)

Bạn muốn mình tiếp tục bóc tách 5GC-Fuzz theo dạng **pipeline từng bước** như: học state thế nào, chọn seed ra sao, mutate message nào, và phát hiện crash/lỗi logic bằng tín hiệu gì?