# TODO: cần xác nhận từ người dùng

Các mục dưới đây không thể suy ra đáng tin cậy từ repository và đang được giữ dưới dạng TODO trong luận văn.

1. **Tên chính thức của đề tài bằng tiếng Anh** để thay placeholder trên bìa.
2. **Nguồn dữ liệu radio chính thức**:
   - file CSV nào của ground UE;
   - file CSV nào của aerial UE;
   - độ cao tương ứng;
   - seed hoặc lệnh chạy nếu có.
3. **Cách nối `Code` sang `Code_HO`**:
   - script/file chuyển `ue0_connected_bs` và neighbor columns sang `connected_gnb`, `gnbX_rsrp`;
   - hoặc xác nhận hai hệ thống được chạy độc lập bằng synthetic CSV.
4. **T304 chính thức**: 100 ms theo test plan hay 1000 ms theo source hiện tại.
5. **Source revision của archived logs**, đặc biệt các run:
   - `1781614531`;
   - `1781622441`;
   - `1781688778`.
6. **Lệnh và config chính thức** dùng để tạo từng run.
7. **Xác nhận archived runs là kết quả luận văn hay log debug**.
8. **CHO có nằm trong phạm vi kết quả chính hay chỉ là future work**.
9. **Tên 5G Core thực tế dùng trong các run chính thức**: Open5GS, free5GC, hay hệ thống khác.
10. **Các thí nghiệm đã thực sự hoàn thành** trong test plan:
    - loss--delay matrix;
    - T304 breakpoint;
    - RLF cascade;
    - Xn versus N2;
    - beam failure/T312.

Sau khi có các xác nhận trên, các TODO trong bìa, Chương 3 và Chương 4 có thể được thay bằng kết luận và bảng/biểu đồ chính thức.
