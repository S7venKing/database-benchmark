## Kết quả Benchmark

Bảng dưới đây được sinh tự động từ các execution plan của PostgreSQL và MongoDB bằng chương trình:

```bash
go run cmd/benchmark/main.go
```

Thời gian thực thi được lấy trực tiếp từ:

- PostgreSQL: trường `Execution Time` trong `EXPLAIN (ANALYZE, FORMAT JSON)`
- MongoDB: trường `executionTimeMillis` trong `executionStats`

| Query Shape | PostgreSQL (ms) | MongoDB (ms) | Winner |
|------------|---------------:|-------------:|---------|
| Point Lookup | 0.039 | 6 | PostgreSQL |
| Range Scan | 2.129 | 5 | PostgreSQL |
| Aggregate | 2.990 | 13 | PostgreSQL |
| Join / Lookup | 2.276 | 57 | PostgreSQL |

## Giải thích theo Execution Plan

### Point Lookup

**Winner: PostgreSQL**

- PostgreSQL sử dụng `Index Scan` trên `products_pkey`.
- MongoDB thực hiện `COLLSCAN` và phải đọc 10.000 document.

### Range Scan

**Winner: PostgreSQL**

- PostgreSQL thực hiện `Seq Scan` trên 10.000 dòng và trả về 4.008 dòng.
- MongoDB sử dụng `IXSCAN(price_1)` với 4.008 keys examined.
- Trên dataset hiện tại PostgreSQL vẫn có thời gian thấp hơn.

### Aggregate

**Winner: PostgreSQL**

- PostgreSQL sử dụng `Hash Aggregate`.
- MongoDB thực hiện `COLLSCAN + GROUP`.
- Cả hai đều phải đọc toàn bộ dữ liệu nhưng PostgreSQL hoàn thành nhanh hơn.

### Join / Lookup

**Winner: PostgreSQL**

- PostgreSQL sử dụng `Hash Join`.
- MongoDB sử dụng `EQ_LOOKUP (IndexedLoopJoin)` với index `_id_`.
- PostgreSQL có thời gian thực thi thấp hơn trên workload này.

## Raw Plans

Các execution plan được lưu tự động sau mỗi lần benchmark:

### PostgreSQL

- `plans/postgres/point_lookup.json`
- `plans/postgres/range_scan.json`
- `plans/postgres/aggregate.json`
- `plans/postgres/join.json`

### MongoDB

- `plans/mongo/point_lookup.json`
- `plans/mongo/range_scan.json`
- `plans/mongo/aggregate.json`
- `plans/mongo/lookup.json`

Reviewer có thể mở trực tiếp các file trên để kiểm tra và tái lập kết quả benchmark.
