## start

### 1. 安装依赖

```shell
GOTOOLCHAIN=local go mod tidy
```
- GOTOOLCHAIN=local 就是告诉 Go：“别去网上乱下载新的编译器，我电脑上装的是哪个，就用哪个。”
- 

### 2. 启动服务

默认使用：
- HTTP 地址：`:8080`
- SQLite 数据库：`db/laws.db`
- 解析后法律 JSON 目录：`data/law_json`

解析后的法律文件默认按分类放在：

`data/law_json/laws_by_type/type_<lawTypeId>/<versionId>.json`

例如：

`data/law_json/laws_by_type/type_230/2c909fdd678bf17901678bf59da8002d.json`

服务会根据法律本身的 `lawTypeId` 去对应分类目录下读取文件。

也可以通过环境变量覆盖：
- `HTTP_ADDR`
- `LAW_DB_PATH`
- `LAW_DETAIL_JSON_DIR`

启动命令：

```shell
GOTOOLCHAIN=local go run .
```

例如：

```shell
HTTP_ADDR=:9090 LAW_DB_PATH=db/laws.db LAW_DETAIL_JSON_DIR=data/law_json GOTOOLCHAIN=local go run .
```

### 3. 测试接口

健康检查：

```shell
curl http://localhost:8080/healthz
```

接口1: 查看各具体分类的预览列表（每类最多 20 条）：

```shell
curl http://localhost:8080/api/v1/types/previews
```

接口2: 分页查看某个分类下的法律列表：

```shell
curl "http://localhost:8080/api/v1/types/230/laws?page=1&pageSize=20"
```

接口3: 读取某个法律的解析后 JSON：

```shell
curl http://localhost:8080/api/v1/laws/2c909fdd678bf17901678bf59da8002d/parsed
```

如果当前机器还没有准备好解析后的 JSON 文件，接口会返回一个“暂无解析数据”的占位 JSON。

接口4: 大分类统计（按宪法 / 法律 / 行政法规 / 监察法规 / 地方法规 / 司法解释 聚合）：

```shell
curl http://localhost:8080/api/v1/laws/big-groups
```

每个大分类返回：`count`（总数）、`home_tag`（是否进首页：宪法 / 法律 为 true）、`more`（子类型是否超过 3 个）、`subTypes`（按数量倒序取前 3 个子类型，含 `typeId / typeName / count`）。

接口5: 首页聚合数据（homeLaws）：

```shell
curl http://localhost:8080/api/v1/home/laws
```

返回 3 个 section：
- `newLawExpress` — 新法速递。筛选条件：`publishDate` 在最近半年内，或 `effectDate` 为空 / 晚于今天；最多 6 条；`bgColor` 在 `#BF4A90D9 / #BF5C7A9E / #BF7A5C9E` 之间循环。
- `lawCategories` — 法律概览。固定 8 个一级类型（宪法、宪法相关法、刑法、民法商法、诉讼与非诉讼法、行政法、经济法、社会法），`law_type_display` 走 `types` 表，`law_type` / `icon` / `uuid` 与移动端 mock 对齐。
- `commonLaws` — 常用法律。7 个固定分组（婚姻家庭、商品买卖、劳动人事、交通法规、借贷担保、治安案件、刑事案件），`uuid / law_type / law_type_display / icon / type_id` 写死在 `service/home_service.go` 的 `commonLawDefs`；`count` 在每次请求时按各组的 `Keywords` 在 `laws_list.title` 上做 `LIKE %kw%` 聚合得到（粗匹配，分组之间允许重叠命中）。要调整命中口径直接改 `commonLawDefs[i].Keywords`。

接口6: 新法速递分页列表：

```shell
curl "http://localhost:8080/api/v1/new-laws?page=1&pageSize=20"
```

筛选条件同 `newLawExpress`，但支持分页。返回 `page / pageSize / total / totalPages / items`，每个 item 包含 `versionId / title / lawTypeId / lawType / publishDate / effectDate / effectiveStatus / authorityName`。

接口7: 某个常用法律类型下的法律列表（分页）：

```shell
curl "http://localhost:8080/api/v1/common-laws/1/laws?page=1&pageSize=20"
```

返回该常用法律类型下的法律列表，支持分页。返回结构：
- `type` — 常用法律类型信息（`id / lawType / lawTypeDisplay / icon`）
- `page / pageSize / total / totalPages` — 分页信息
- `items` — 法律列表，每项含 `versionId / title / lawTypeId / lawType / publishDate / effectDate / effectiveStatus / authorityName`

目录说明也保存在：

`data/law_json/README.md`
