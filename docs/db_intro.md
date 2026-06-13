# SQLite vs PostgreSQL 对比

项目使用 **GORM + SQLite**，以下是 SQLite 和 PostgreSQL 在当前项目场景下的主要区别。

## 核心区别

| 维度 | SQLite (当前) | PostgreSQL |
|---|---|---|
| **部署方式** | 嵌入式，零配置，单文件 `laws.db` | 需要独立安装和运行数据库服务 |
| **并发写入** | 单写者，写操作会锁整个库 | 支持多并发读写，行级锁 |
| **连接数** | 当前配置 MaxOpenConns=1 | 可支持数百并发连接 |
| **全文搜索** | 需要用 `LIKE` 或 JSON 函数模拟 | 内置 `tsvector` 全文搜索，性能和功能更强 |
| **JSON 查询** | 支持有限 | 原生 JSONB 类型，查询性能好且有索引支持 |
| **数据量** | 适合中小规模（百万级以内表现好） | 适合大规模数据（亿级也没问题） |
| **运维** | 几乎不需要 | 需要备份策略、连接池管理、版本升级等 |

## 对项目的影响

1. **当前阶段 SQLite 完全够用** — 法律数据集规模有限，读多写少，SQLite 性能足够且零运维。
2. **搜索功能** — 项目里用 `detailJson LIKE` 做搜索（`internal/repository/law.go`），如果搜索需求变复杂，PostgreSQL 的 `tsvector` 会好很多。
3. **GORM 抽象层** — 因为用了 GORM，未来如果要切换到 PostgreSQL，改动主要在 `internal/config/config.go` 的连接配置，模型层和仓库层代码基本不用改。GORM 的查询语法是数据库无关的，但项目里有几处 `Raw()` 原生 SQL 需要检查兼容性。

## 结论

如果数据量和并发量不大，继续用 SQLite 没问题。等遇到性能瓶颈或需要更强的搜索能力时，迁移到 PostgreSQL 的成本也不高。
