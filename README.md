# recsys_go

面向**通用推荐链路**的 Go 参考实现：多路召回 → 合并去重 → 规则/特征过滤 → 粗排（FM）→ 精排（TF-Serving 可选）→ 展控 → 返回结果。  
不绑定具体业务域（地图/直播等），配置形态对齐常见 **Center + Rank** 双服务拆分，便于对照自有业务推荐中台做迁移或实验。

仓库地址：[https://github.com/gerogexiongle/recsys_go](https://github.com/gerogexiongle/recsys_go)

---

## 1. 系统架构

```mermaid
flowchart TB
  subgraph Client["调用方"]
    APP[App / Gateway / 压测]
  end

  subgraph Recommend["recommend-api :18080"]
    API["POST /v1/recommend"]
    CFG_R["Config: Recall"]
    CFG_F["Config: Filter"]
    CFG_S["Config: ShowControl"]
    RECALL[Recall Registry<br/>多路召回插件]
    MERGE[Merge + Dedupe]
    FEAT_C["featurestore.Session<br/>Redis user/item JSON"]
    FILTER[Rule + Feature Filter]
    SHOW[ShowControl 策略链]
    API --> CFG_R --> RECALL --> MERGE
    MERGE --> FEAT_C --> FILTER --> SHOW
    CFG_F --> FILTER
    CFG_S --> SHOW
  end

  subgraph Redis["Redis（特征 / 过滤侧车）"]
    UK["STRING recsysgo:feat:user:{id}"]
    IK["STRING recsysgo:item:{id}"]
  end

  subgraph Rank["rank-api :18081"]
    RAPI["POST /v1/rank/multi"]
    EXP["RankExpConf.json<br/>PreRank / Rank / ReRank AB"]
    ENG["rankengine.Engine"]
    FM["FM 粗排"]
    TF["TF-Serving 精排<br/>可选"]
    RAPI --> EXP --> ENG
    ENG --> FM --> TF
    FEAT_R["featurestore<br/>拉 user/item JSON"]
    FEAT_R --> FM
  end

  APP --> API
  FEAT_C --> UK
  FEAT_C --> IK
  SHOW -->|HTTP MultiRank| RAPI
  FEAT_R --> UK
  FEAT_R --> IK
  SHOW --> APP
```

| 服务 | 端口 | 职责 |
|------|------|------|
| **recommend-api** | 18080 | 召回、合并、过滤、调用排序、展控 |
| **rank-api** | 18081 | 粗排 / 精排 / 重排，读特征并打分 |

共享库 **`pkg/recsyskit`**（领域无关 item/user/流水线抽象）、**`pkg/featurestore`**（Redis JSON 特征与 FM 语义槽合并）。

---

## 2. 端到端数据流（一次推荐请求）

```mermaid
sequenceDiagram
  participant C as Client
  participant R as recommend-api
  participant Redis as Redis
  participant K as rank-api

  C->>R: POST /v1/recommend<br/>user_id, exp_ids, ret_count

  Note over R: 解析 AB + UserGroup
  R->>R: 多路 Recall（独占 lane → 主 lane）
  R->>R: MergeRecallLanes 去重 + AllMergeNum 截断
  R->>Redis: GET user + MGET items
  Redis-->>R: JSON（曝光/标签/语义特征）
  R->>R: RuleFilter + FeatureFilter + KeepItemNum

  R->>K: POST /v1/rank/multi<br/>item_ids[]
  K->>Redis: UserJSON + ItemJSON（逐 item）
  K->>K: MergeUserItemJSON → FM sparse
  K->>K: FM PreRank → sort
  K->>K: TF Rank（未配置则 Rank=PreRank）
  K->>K: ReRank（mock 加权）
  K-->>R: item_scores[]

  R->>R: 按 Rank 分数重排
  R->>R: ShowControl（分控/截断/强插/MMR…）
  R-->>C: item_ids + recall_type
```

---

## 3. 推荐链路分阶段说明

```mermaid
flowchart LR
  A[请求入参] --> B[AB / UserGroup]
  B --> C[多路召回]
  C --> D[合并去重]
  D --> E[特征加载]
  E --> F[过滤]
  F --> G[粗排 FM]
  G --> H[精排 TF]
  H --> I[展控]
  I --> J[响应]

  style G fill:#e8f4fc
  style H fill:#e8f4fc
  style I fill:#fff4e6
```

| 阶段 | 执行位置 | 配置来源 | 说明 |
|------|----------|----------|------|
| **多路召回** | recommend | `recommend-recall.json` | `ExclusiveRecallList` 优先，再 `RecallAndMergeList`；每路 `RecallNum` / `MergeMaxNum` / `SampleFold` / `UseTopKIndex` |
| **合并** | `pkg/recsyskit` | `AllMergeNum` | 按 item_id 去重；独占路先入队 |
| **特征加载** | `pkg/featurestore` | `FeatureRedis` | 一次 `Session`：user GET + items MGET，供过滤与（rank 侧再次拉取）FM |
| **规则过滤** | `centerconfig` | `recommend-filter.json` → `RuleFilterStrategyList` | 如曝光上限 `LiveExposure` |
| **特征过滤** | `centerconfig` | `FeatureFilterStrategyList` | 如 `FeatureLess`、`LabelTypeWhiteList` |
| **池子截断** | `centerconfig` | `KeepItemNum` | 进入排序前的候选上限 |
| **粗排 PreRank** | rank | `RankExpConf` + FM 模型文件 | FM 对 sparse 特征打分并截断 |
| **精排 Rank** | rank | `RankExpConf` + `RankModelBundles` | TF-Serving HTTP；未配置时 **Rank 分 = PreRank 分**（实验环境） |
| **重排 ReRank** | rank | `ReRankExp`（可选） | 当前为 mock 微调，可替换为业务重排 |
| **展控** | recommend | `recommend-showcontrol.json` | `ScoreControl` / `HomogenContent` / `ForcedInsert` / `MMRRearrange` 等 |

排序相关 AB、截断、模型名**仅在 rank 服务**配置，recommend 只负责召回池与展控，避免重复截断。

---

## 4. 配置架构（与业务解耦的三文件）

```mermaid
flowchart TB
  subgraph RecommendCfg["recommend-api.yaml"]
    P1[CenterRecallPath]
    P2[CenterFilterPath]
    P3[CenterShowControlPath]
    P4[FeatureRedis]
    P5[RankService.BaseURL]
  end

  P1 --> JSON_R["recommend-recall.json<br/>MapRecommend → exp_id → UserGroupList"]
  P2 --> JSON_F["recommend-filter.json<br/>Rule + Feature 策略列表"]
  P3 --> JSON_S["recommend-showcontrol.json<br/>StrategyList"]
  P4 --> Redis[(Redis)]
  P5 --> RankSvc[rank-api]

  subgraph RankCfg["rank-api.yaml"]
    R1[RankEngine / RankModelBundles]
    R2[RankExpConfPath]
    R3[FeatureRedis]
  end

  R1 --> FMFile["fm_5feat.txt 等"]
  R2 --> JSON_E["rank-exp-conf.json<br/>PreRankExp / RankExp / ReRankExp"]
  R3 --> Redis
```

兼容模式：单文件 **`recommend-funnel.json`**（`FunnelConfigPath`）将召回/过滤/展控写在同一 JSON；若配置了 `CenterRecallPath`，则**优先使用三文件模式**。

---

## 5. Rank 内部打分流水线

```mermaid
flowchart TD
  IN[item_ids + user_id + exp_ids] --> LOAD[Redis: UserJSON + ItemJSON]
  LOAD --> MERGE[MergeUserItemJSON<br/>语义槽 field 1..5 + fm_sparse]
  MERGE --> SPARSE[SparseFeature 向量]
  SPARSE --> PR[FM Predict → PreRankScore]
  PR --> SORT1[按 PreRank 降序]
  SORT1 --> TR1{PreRankTrunc > 0?}
  TR1 -->|是| CUT1[截断]
  TR1 -->|否| TFIN
  CUT1 --> TFIN{TF-Serving 已配置?}
  TFIN -->|是| TF[TF Predict → RankScore]
  TFIN -->|否| PASS[RankScore = PreRankScore]
  TF --> SORT2[按 Rank 降序]
  PASS --> SORT2
  SORT2 --> TR2{RankTrunc > 0?}
  TR2 --> RR[ReRank 策略]
  RR --> OUT[返回 item_scores]
```

**特征语义（实验/demo）**：用户 `age` / `gender` / `income_wan`，物品 `ctr_7d` / `revenue_7d` → 映射为 FM 五槽位；亦支持 `fm_sparse` / `tf_dense` 原始字段。详见 `pkg/featurestore/merge.go`。

---

## 6. Redis Key 约定（开源默认）

| 用途 | Key 模式 | 值格式 | key 不存在时 |
|------|----------|--------|--------------|
| 用户画像（FM/展控） | `recsysgo:feat:user:%d` | JSON 画像 | 无用户画像槽位 |
| 物品画像（FM/展控） | `recsysgo:feat:item:%d` | JSON 画像 | 无物品画像槽位 |
| LiveExposure（物品维） | `recsysgo:filter:exposure` | JSON map `item_id→count` | 不按曝光过滤 |
| FeatureLess | `recsysgo:filter:featureless` | JSON 数组 `[910009,…]` | 无 feature-less 物品 |
| Label 白名单 | `recsysgo:filter:label` | JSON map `item_id→label` | 无法按 label 命中 |
| 非个性化召回 | `recsysgo:recall:lane:{RecallType}` | JSON 物品 id 列表 | 回退代码 stub |
| 协同过滤（个性化） | `recsysgo:recall:cf:user:%d` | JSON 物品 id 列表 | 回退 stub |

**仅画像与 CF 按实体分 key**；过滤为单 key 合并 JSON；展控用画像即可。见 `pkg/featurestore/keys.go`。

密码：`FeatureRedis.Crypto=true` 时使用与线上一致的 AES 密文（`pkg/redisdecrypt`），明文密码通过 `EncryptPassword` 生成 hex 写入配置。

---

## 7. 目录结构

```
recsys_go/
├── api/recsys/v1/          # proto 定义（可选生成）
├── pkg/
│   ├── recsyskit/          # 流水线抽象、漏斗配置、合并/过滤工具
│   ├── featurestore/       # Redis 特征、Session、FM JSON 合并
│   ├── featurekit/         # 稀疏特征类型
│   └── redisdecrypt/       # Redis 密码加解密
├── services/
│   ├── recommend/          # Center 侧：召回 → 过滤 → 调 rank → 展控
│   │   ├── etc/            # yaml + recall/filter/show JSON
│   │   └── internal/
│   │       ├── centerconfig/
│   │       ├── recall/     # 召回插件注册表
│   │       └── logic/
│   └── rank/               # Rank 侧：FM + TF + RankExpConf
│       ├── etc/
│       └── internal/rankengine/
└── scripts/
    ├── seed_feature_redis.py
    ├── e2e.sh
    └── e2e_full_chain.sh
```

---

## 8. 快速开始

### 依赖

- Go 1.22+
- Redis（可选；关闭时 `FeatureRedis.Disabled: true`）
- Python 3 + `redis` + `pycryptodome`（仅种子脚本）

### 构建

```bash
git clone https://github.com/gerogexiongle/recsys_go.git
cd recsys_go
make build    # bin/recommend-api, bin/rank-api
```

### 配置

编辑 `services/recommend/etc/recommend-api.yaml` 与 `services/rank/etc/rank-api.yaml` 中的 `FeatureRedis.Host` / `PasswordHex`。

生成 `test123` 的密文：

```bash
go test ./pkg/redisdecrypt/ -run TestEncryptTest123Hex -v
```

### 写入演示数据（2 用户 + 10 物品）

```bash
export RECSYS_SEED_REDIS=1
export RECSYS_REDIS_HOST=127.0.0.1   # 按实际修改
python3 scripts/seed_feature_redis.py
```

### 启动

```bash
./bin/rank-api -f services/rank/etc/rank-api.yaml &
./bin/recommend-api -f services/recommend/etc/recommend-api.yaml &
```

### 调用

```bash
curl -s -X POST http://127.0.0.1:18080/v1/recommend \
  -H 'Content-Type: application/json' \
  -d '{"uuid":"demo","user_id":900001,"exp_ids":[0],"ret_count":5}'
```

### 一键自测

```bash
make e2e          # 轻量 stub 召回 + mock rank
make e2e-full     # Redis + FM pipeline + 三文件 center 全链路
```

---

## 9. HTTP 接口

| 方法 | 路径 | 服务 | 说明 |
|------|------|------|------|
| GET | `/health` | both | 健康检查 |
| GET | `/v1/ready` | recommend | 配置就绪（rank 客户端、center/funnel） |
| POST | `/v1/recommend` | recommend | 完整推荐 |
| POST | `/v1/rank/multi` | rank | 多组候选打分 |

---

## 10. 扩展指南

| 扩展点 | 做法 |
|--------|------|
| 新召回通道 | 在 `services/recommend/internal/recall/registry.go` 增加 `RecallType` 分支，或改为接口+注册表 |
| 新过滤策略 | 在 `centerconfig/apply.go` 增加 `FilterType` 分支 |
| 新展控策略 | 在 `centerconfig/apply.go` 增加 `ShowControlType` 分支 |
| 新模型 / AB | `rank-api.yaml` 的 `RankModelBundles` + `rank-exp-conf.json` |
| 倒排 / 物料 Redis | 实现新 `Fetcher` 接口，独立 key 命名空间，不与 `recsysgo:user/item` 混用 |

---

## 11. 高并发多实例部署

### 11.1 部署拓扑

```mermaid
flowchart LR
  subgraph LB1["入口"]
    GW[Gateway / Ingress]
  end
  subgraph RecPool["recommend-api × N"]
    R1[rec-1]
    R2[rec-2]
    RN[rec-N]
  end
  subgraph RankPool["rank-api × M"]
    K1[rank-1]
    K2[rank-2]
  end
  subgraph TFPool["TF-Serving × P"]
    T1[tf-1:8501]
    T2[tf-2:8501]
  end
  GW --> R1 & R2 & RN
  R1 & R2 & RN --> K1 & K2
  K1 & K2 --> T1 & T2
```

- **recommend / rank**：无状态，水平扩容；实例间不共享会话。
- **Redis**：共享特征存储；用连接池，按集群规模调大 `MaxIdleConns`。
- **下游地址**：客户端侧 **多 Endpoints + 负载均衡**（`pkg/upstream`），不依赖 recommend 进程内嵌 rank 代码。

### 11.2 Recommend → Rank 多地址

`recommend-api.yaml`（**兼容原单地址 `BaseURL`**）：

```yaml
RankService:
  BaseURL: http://rank-svc:18081          # 单 VIP / K8s Service
  # 或直接列多 Pod（与 BaseURL 二选一或并存，会去重）：
  Endpoints:
    - http://10.0.0.11:18081
    - http://10.0.0.12:18081
  LoadBalance: round_robin   # round_robin | random
  TimeoutMs: 800
```

实现：`transporthttp.RankHTTPClient` → `upstream.HTTPDoer`  
- 轮询/随机选实例  
- 连接池复用（`MaxIdleConnsPerHost`）  
- **失败自动换下一个实例**（5xx / 网络错误）

与 go-zero 的关系：rank 侧已是 `go-zero/rest` Server；recommend 调 rank 走 **HTTP 客户端多目标**，无需把 rank 改成 zrpc 即可多实例。若全链路改为 **gRPC + etcd 服务发现**，可再包一层 `zrpc` Client（见下节演进）。

### 11.3 Rank → TF-Serving 多地址

`rank-api.yaml` 的 `TFServing` / `RankModelBundles.*.TFServing`：

```yaml
TFServing:
  Endpoints:
    - http://tf-serving-0:8501
    - http://tf-serving-1:8501
  LoadBalance: round_robin
  ModelName: your_model
  SignatureName: serving_default
  InputTensor: inputs
  FeatureDim: 8
  TimeoutMs: 1500
  OutputName: predictions
```

### 11.4 REST vs gRPC（对照 C++）

| 维度 | C++ `TFModelGrpc` | 本仓库默认 REST |
|------|-------------------|-----------------|
| 协议 | gRPC `:8500` | HTTP JSON `:8501` |
| 性能 | 大批量 tensor 更省 | 单条/小 batch 足够；Go 实现简单 |
| 运维 | 需 proto + grpc 依赖 | 与官方 TF Serving Docker REST 一致 |
| 多实例 | 客户端 LB | `Endpoints` + round_robin（已实现） |

**建议**：开源版与实验环境 **先用 REST**；QPS 极高且 batch 固定时再增加 `Protocol: grpc` 实现（与 REST 共用 `Endpoints` 列表，端口改为 8500）。

### 11.5 生产演进（可选）

| 阶段 | 方案 |
|------|------|
| 现在 | 配置 `Endpoints` + `pkg/upstream` 轮询与 failover |
| 中期 | K8s Service 单 DNS + readiness；或 Nginx/Envoy 做 L7 LB |
| 后期 | go-zero **zrpc** + etcd 注册 rank；rank 调 TF 用 gRPC + 批量 Predict |

---

## 12. 设计原则（开源版）

1. **Center / Rank 分离**：候选扩量与策略在 recommend；算力密集打分在 rank。  
2. **配置驱动**：策略列表用 JSON 描述，按 `exp_id` + `UserGroup` 选桶。  
3. **领域无关命名**：代码中统一使用 `Item` / `User`，不写业务专有名词。  
4. **可测**：单元测试 + `scripts/e2e_full_chain.sh` 覆盖召回→过滤→FM→展控。

---

## License

MIT（如未另行声明，以仓库为准。）
