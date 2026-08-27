# monocache — 泛型编译单态化缓存一致性服务

面向编译器工程师的后端服务。导入泛型定义、类型实参、约束解和目标 ABI，规范化类型图并计算单态化身份键，比对实例缓存、合并等价约束，最终发布一致性验证快照。

## 业务闭环

1. 创建编译批次、泛型定义、类型实参、约束解和目标 ABI。
2. 提交实例请求并规范化：别名解析、实参集排序、约束归一，得到单态化键。
3. 比对实例缓存，判定 unique / duplicate / conflict / abi_mismatch。
4. 合并等价约束，消除同实参集同 ABI 的规范化分歧。
5. 发布验证快照，作为该批次一致性证明。

## 状态机

- 编译批次：`receiving` → `normalizing` → `conflicted` / `releasable` → `sealed`
- 实例请求：`raw` → `normalized` → `equivalent` / `conflict` / `excluded`
- 缓存条目：`candidate` → `unique` / `duplicate` / `conflict` / `abi_mismatch`
- 验证快照：`draft` → `published` / `superseded`

## 标准命令

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test  ./...
go run ./cmd/monocache --smoke-test
go run ./cmd/monocache --addr :8080 --db monocache.db
```

## 目录结构

仓库根目录就是源码根（本地 `env/` 原样推送，远程不再套一层 `env/`）：

```
cmd/monocache/            # 入口（--smoke-test 契约）
internal/model/           # 实体、错误、状态机
internal/store/           # SQLite 持久化与迁移
internal/normalize/       # 类型图规范化
internal/merge/           # 约束解合并
internal/keymake/         # 单态化键计算
internal/cachecmp/        # 缓存比对
internal/snapshot/        # 一致性报告
internal/service/         # 编排层
internal/httpapi/         # /api HTTP 层
go.mod / go.sum
component-versions.json
Dockerfile / benzhi.Dockerfile / build_benzhi_docker.sh
BENZHI_README.md
```

详见 `BENZHI_README.md`。
