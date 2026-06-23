# messages_search: multi_match OR-trap 修正

- 日期: 2026-06-23
- 范围: `modules/messages_search/`(reader 层 DSL)
- 不涉及: `octo-search-indexer`(写入/字段投影,与方案 B 部署无关)

## 1. 问题

前端在群内搜索 `"按时缴纳时间的女性"` 这种无语义短语,后端 `_search_all` 返回大量
"无关消息",高亮全打在 `的` 字上。换成其他含 `的` 的胡乱组合也复现。

## 2. 根因

ik_smart 把 `按时缴纳时间的女性` 切成 5 个 token:

```
按时 / 缴纳 / 时间 / 的 / 女性
```

reader 层用 `elastic.NewMultiMatchQuery(req.Keyword, ...)` 构造查询,go-elastic
库默认 `type=best_fields`、`operator=OR`,**任一 token 命中即返回**。`的`
是中文最高频虚词,命中近乎全库 → 实测 OR 召回 10000,AND 召回 0。

## 3. 受影响位点(共 4 处,非 3 处)

| 文件 | 行 | 上下文 | 备注 |
|---|---|---|---|
| `modules/messages_search/search_messages.go` | 135 | `b.Must(elastic.NewMultiMatchQuery(...))` | 搜消息 tab |
| `modules/messages_search/search_all.go` | 137 | `b.Should(elastic.NewMultiMatchQuery(...))` text/mergeForward 分支 | 搜全部 tab |
| `modules/messages_search/search_all.go` | 141 | `b.Should(elastic.NewMultiMatchQuery(...))` file 分支 | 搜全部 tab |
| `modules/messages_search/search_files.go` | 139 | `b.Must(elastic.NewMultiMatchQuery(...))` | **同事漏掉的一处**,搜文件 tab |

`search_media.go` 走 `validateKeywordMustBeEmpty`,不接 keyword,无影响。
`search_around.go` 通过 `messageId` 锚定,不走 multi_match,无影响。

> `search_all.go` 顶层的 `b.MinimumShouldMatch("1")` 只决定两个 Should 分支至少命中
> 一个,**不**等于 multi_match 内部的 token MSM,所以仍然落在 OR-trap 里。

## 4. 修复方案

### 4.1 主推:reader 层条件性 stopword 剥离 + `_analyze`

核心规则:**只在剥离后还剩内容词时才剥离,否则原 keyword 直查**——既屏蔽虚词
噪音,又保留对纯虚词(`"的"`)的字面搜索能力。

切词外包给 OpenSearch(避免 Go 端引 IK 依赖),停用词集合在 Go 这边维护。

流程:

```go
// 1. 调 OS _analyze 拿 IK 切词结果(无索引依赖)
tokens, _ := osClient.IndexAnalyze().
    Analyzer("ik_smart").
    Text(req.Keyword).
    Do(ctx)
// → ["按时","缴纳","时间","的","女性"]

// 2. 滤掉停用词(in-memory set,见 §4.3)
content := filterStopwords(tokens, stopwordsSet)
// → ["按时","缴纳","时间","女性"]

// 3. 分支构造 query
var keywordClause elastic.Query
if len(content) == 0 {
    // 纯虚词路径:原 keyword 直查,不上 MSM
    keywordClause = elastic.NewMultiMatchQuery(req.Keyword, fields...).
        Type("cross_fields")
} else {
    // 内容词路径:用滤后版本,MSM 75% 屏蔽剩余噪音
    filtered := strings.Join(content, " ")
    keywordClause = elastic.NewMultiMatchQuery(filtered, fields...).
        Type("cross_fields").
        MinimumShouldMatch("75%")
}
b.Must(keywordClause)
```

**关键设计点**:

- **stopword 不进 MSM 分母**。3 词查询 `"在公司"` → 滤后 2 词 `[公司]`(假设 "在"
  在停用词集合),MSM 75% 作用于 1 词,trivially 满足,只看 "公司" 是否命中。
  vs. 方案 C(详见 §4.4) MSM 单独的做法:3 词全留 → MSM 75% = ceil(0.75×3) = 3
  = AND,"在" 被迫强制命中,反而误漏。
- **`Type("cross_fields")` 仍然必要**。`best_fields` 下 MSM 作用域是单字段,跨字段
  (text/file.name/file.caption)的 75% 判定按"最佳字段"算,行为反直觉。
  `cross_fields` 把所有字段视作虚拟拼接,token 在哪个字段都算 1 次命中,75% 的
  语义和"4 词中 3"一致。
- **纯虚词路径仍跑 multi_match**(不上 MSM),走 OR 行为是符合直觉的——`"的"`
  alone 就是想要"找含'的'的消息"。
- **代价**:每次 keyword 查询多一次 `_analyze` roundtrip(LAN 内 ~3–8ms)。但因
  为后续 `_search` 阶段命中集更小、回流数据更少,实测总体 RT 通常持平或更短。

### 4.2 实现位点

四处裸 `NewMultiMatchQuery` 都走同一套预处理。建议抽工具函数:

```go
// modules/messages_search/keyword_query.go (新建)
func buildKeywordClause(ctx context.Context, osClient *elastic.Client,
    keyword string, fields ...string) (elastic.Query, error) {
    // ...上述流程
}
```

四处改造:

| 文件 | 行 | 字段集 |
|---|---|---|
| `search_messages.go` | 135 | `payload.text.content^3, payload.image.*, payload.file.*, payload.mergeForward.msgs.searchText` |
| `search_all.go` | 137 | `payload.text.content^3, payload.mergeForward.msgs.searchText` |
| `search_all.go` | 141 | `payload.file.name^2, payload.file.caption` |
| `search_files.go` | 139 | `payload.file.name^2, payload.file.caption` |

`search_all.go` 仍然是两条 Should 分支,只是每条内部都用 `buildKeywordClause`。

### 4.3 停用词集合

放 `modules/messages_search/stopwords.go`,从 hardcoded 列表起步,后续按需挪到配置:

```go
var defaultStopwords = map[string]struct{}{
    "的": {}, "了": {}, "在": {}, "是": {}, "和": {}, "也": {},
    "就": {}, "都": {}, "而": {}, "及": {}, "与": {}, "或": {},
    "把": {}, "被": {}, "对": {}, "向": {}, "从": {}, "到": {},
    "给": {}, "让": {}, "比": {}, "为": {}, "以": {}, "于": {}, "由": {},
    // 多字虚词(IK smart 会切成整 token)
    "的话": {}, "之类": {}, "之中": {},
}
```

迭代建议:线上接入后,把"被 MSM 过滤掉但仍频繁出现在 OR-trap 召回里"的 token
统计出来,逐步加进集合。

### 4.4 退化方案:仅 MSM 75%(不剥离)

如果 `_analyze` roundtrip 不可接受,或前期想最小化变更,可以只做 §4.1 的 step 3
分支里"内容词路径"那一段,跳过 §4.1 step 1/2,直接 `req.Keyword` + MSM 75% +
`cross_fields`。

**这是当前 patch 的最初版本,有局限**:
- 1 token 纯虚词 `"的"`:MSM trivially 满足 → ✓ 可搜
- 5 token 含虚词的胡乱查询:MSM 4/5,虚词单独不够数 → ✓ 屏蔽
- **2~4 token 含虚词的查询**:`"在公司"` 之类,MSM 75% 退化成 AND,"在" 强制命中
  → ✗ 仍有噪音/误漏

如果产品只关心截图里那种极端胡乱组合 case,退化方案够用;否则上 §4.1 主方案。

### 4.5 不采用:indexer 层 `search_analyzer`

考虑过给字段配 `search_analyzer = ik_smart_with_stopwords`(query-time only,
不需要 reindex),被否,原因:

- 纯虚词查询 `"的"` 经过 search_analyzer → 空 token 流 → 0 命中,与产品需求冲突
- 要兼容只能 reader 层叠空结果重试,两次 RT + 状态判断,反而不如 §4.1 简洁
- 集群 mapping 改动需要协调 search-indexer 团队

留作未来选项,如果停用词集合稳定下来、不再需要"纯虚词可搜"语义,可以切到这条
路减少 reader 复杂度。

## 5. 测试用例

回归 spec 拆两层:

**单元(Go,纯逻辑,不连 OS)** — `modules/messages_search/keyword_query_test.go`:

| 用例 | 输入 keyword | 输入 tokens(mock _analyze) | 预期产物 |
|---|---|---|---|
| 纯虚词 | `"的"` | `["的"]` | `multi_match`,query=`"的"`,无 MSM |
| 纯虚词多 token | `"的话"` | `["的话"]` | 同上,query=`"的话"` |
| 全为虚词 | `"的了"` | `["的","了"]` | 同上,query=`"的了"`(剥离后空,走纯虚词路径) |
| 含虚词 | `"按时缴纳时间的女性"` | `["按时","缴纳","时间","的","女性"]` | `multi_match`,query=`"按时 缴纳 时间 女性"`,MSM=75%,type=cross_fields |
| 短 mixed | `"在公司"` | `["在","公司"]` | query=`"公司"`,MSM=75% |
| 纯内容词 | `"季度报告"` | `["季度","报告"]` | query=`"季度 报告"`,MSM=75% |

**集成(连真 OS + IK,跑 `make env-test`)** — `modules/messages_search/or_trap_e2e_test.go`:

| 用例 | 输入 | 预期 |
|---|---|---|
| OR-trap 复现 | `"按时缴纳时间的女性"` | 修复前命中 ≫ 10、修复后 ≤ 极少数(只命中字面 4 词都中的) |
| 纯虚词字面 | `"的"` | 正常返回所有字面含 "的" 的消息,不为 0 |
| 短 mixed | `"在公司"` | 不被 "在" 的高频拖垮,只命中真实含 "公司" 的消息 |
| 正常单实词 | `"会议"` | 召回不退化(对比修复前后召回数差异 < 5%) |
| 模糊 4 词 | `"明天 下午 三点 开会"` | 命中至少 3 词的消息(75%) |
| 文件 tab | `"季度 报告"` | `search_files` 行为与上一致 |

## 6. 上线步骤

1. 新建 `keyword_query.go`(工具函数)+ `stopwords.go`(集合)+ 单测。
2. 改 4 处 `NewMultiMatchQuery`,统一走 `buildKeywordClause`。
3. 跑 `go test ./modules/messages_search/...`——预计 `search_all_test.go` /
   `search_messages` 相关现存断言会因 query 形状变化失败,需要更新断言或改
   mock。重点核对 `dsl_test.go` 里对 `multi_match` 的字面 JSON 断言。
4. 跑 `make env-test` 起真 OS,验证 §5 集成用例。
5. 跑 `make i18n-extract-check` + `make i18n-lint`(本次不动 errcode,应通过)。
6. 灰度环境对照召回:用 §5 集成用例的输入,grafana / 业务侧验证 RT、召回数、误召
   回数三个指标。
7. 灰度过 → 全量。
8. 灰度期间监控 `_analyze` API 调用 RT 和失败率,失败时降级到 §4.4(只 MSM,不剥
   离),建议预留 feature flag `messages_search.stopword_strip.enabled`。

## 7. 关联

- 截图证据: 用户 2026-06-23 反馈,query `按时缴纳时间的女性` 高亮全在 `的` 字。
- 同事原始诊断: 锁定 `multi_match` 默认 OR + ik_smart 切词,建议 MSM 75% + 停用词
  预处理。
- 本文勘误与定型:
  - 补充 `search_files.go:139`(共 4 处而非 3 处)
  - 主推方案改为 reader 层"条件性 stopword 剥离 + `_analyze`",而非纯 MSM 75%
    或 indexer 层 stopword
  - 显式 `Type("cross_fields")` 才能让 MSM 在跨字段场景下行为符合直觉
  - 保留"纯虚词可搜"语义(用户搜 `"的"` 仍应返回字面命中结果)

## 8. Ops kill switch — `OCTO_SEARCH_STOPWORD_STRIP_ENABLED`

落实 §6.8 预留的 `messages_search.stopword_strip.enabled` feature flag,作为线上
误判的一键回退手段,无需重新发版。

- **位置**: `SearchConfig.StopwordStripEnabled`(`modules/messages_search/config.go`)。
- **环境变量**: `OCTO_SEARCH_STOPWORD_STRIP_ENABLED`(`strconv.ParseBool` 规则,
  接受 `true/false`、`1/0`、`TRUE/FALSE` 等)。
- **默认值**: `true`(开启 stopword strip)。
- **关闭路径**(`OCTO_SEARCH_STOPWORD_STRIP_ENABLED=false`):
  - `search_messages` / `search_files` / `search_all` 三个端点的 keyword 查询
    **完全跳过** `_analyze` roundtrip——这是配置开关,不只是 query shape 改动。
  - keyword clause 直接走 §4.4 退化路径:
    `elastic.NewMultiMatchQuery(rawKeyword, fields...).Type("cross_fields").MinimumShouldMatch("75%")`。
  - 行为与"`_analyze` 不可达时的降级"等价,但省掉一次 IK 集群往返。
- **使用建议**: stopword 集合误判某个真实查询时(例如把领域专名 token 当成虚词
  剥离),先翻 flag 应急 → 再迭代 `defaultStopwords` 列表 → flag 翻回。
- **可观测**: flag 关闭后 `_analyze` 调用计数应降到 0;关闭期间 `_search*` RT
  曲线应观察到 ~3–8ms 的均值下行(取决于 LAN 延迟)。

