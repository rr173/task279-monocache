# BENZHI 评测说明

基于 Go 实现的泛型编译单态化缓存一致性后端服务，一款后端服务，完成类型实参与约束规范化、单态化身份键计算、实例缓存比对、等价约束合并与一致性快照发布。

## 启动

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go run ./cmd/monocache --addr :8080 --db monocache.db
```

## 自检（不启动长驻服务）

```bash
go run ./cmd/monocache --smoke-test
```

`--smoke-test` 会真实创建编译批次、泛型定义、类型实参、约束解与 ABI，执行规范化、缓存比对、冲突合并与快照发布，关闭并重新打开数据库验证持久化与重启恢复，最后以 0 退出码结束。

## 构建门禁

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test  ./...
go run ./cmd/monocache --smoke-test
```

## HTTP API（前缀 /api）

批次：`POST /api/batches`、`GET /api/batches`、`GET /api/batches/{id}`、`POST /api/batches/{id}/seal`
定义：`POST /api/definitions`、`GET /api/definitions`、`GET /api/definitions/{id}`、`POST /api/definitions/{id}/args`、`GET /api/definitions/{id}/args`
约束：`POST /api/constraints`、`GET /api/constraints`、`GET /api/constraints/{id}`
ABI：`POST /api/abis`、`GET /api/abis`
请求：`POST /api/requests`、`GET /api/requests`、`GET /api/requests/{id}`、`POST /api/requests/{id}/normalize`
键：`POST /api/keys`、`GET /api/keys/{id}`
缓存：`POST /api/cache/compare`、`GET /api/cache`、`POST /api/cache/merge`、`POST /api/conflicts/{id}/resolve`
快照：`POST /api/snapshots`、`POST /api/snapshots/{id}/publish`、`GET /api/snapshots`、`GET /api/snapshots/{id}`

## 持久化

SQLite（modernc.org/sqlite，CGO 无关）。建表：abis、definitions、type_args、constraints、batches、requests、keys、cache_entries、snapshots。同一请求的单态化键按 request_id upsert；关闭重开后规范化身份、缓存判定与快照状态可恢复。
