# 工业窑炉烘炉曲线核验台

本项目为筑炉工程师、窑炉操作员和质量审核员提供烘炉批次建档、升温阶段冻结、温度采集、偏差处置、审核签发和合格证下载的一体化浏览器工作台。数据保存在本地 JSON 快照中，便于追溯和复核。

标准构建、运行和测试方式：

```text
go test ./...
go run ./cmd/ovencheck -addr=127.0.0.1:19081
go run ./cmd/ovencheck -selfcheck -addr=127.0.0.1:19081
```

访问 `http://127.0.0.1:19081/` 使用工作台。监听地址可通过 `-addr` 或 `PORT` 配置，默认仅绑定回环地址。

批次列表支持 `GET /api/batches?q=<关键字>&status=<状态>`，返回索引和状态计数；阶段读数可先调用 `/api/batches/{id}/readings/precheck`（JSON 或 CSV）预检，再提交 `/readings`。详情接口还提供阶段统计、规则矩阵、审核证据和合格证摘要校验。
