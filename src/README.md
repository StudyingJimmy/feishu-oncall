# BokeOnCall 后端（Go）

飞书机器人 oncall 中台：机器人发卡片引导用户自助预检 -> 没解决就按规则拉值班同学建服务群 -> 群内服务并记录全过程 -> 结案回写知识库。

**当前状态**：只有「启动代码」和「配置/环境变量加载」以及 **infra 层（与业务无关的客户端封装）** 是已实现代码；业务逻辑（model / repository / service / api / task）只保留目录与 `doc.go` 占位，等逐步实现。

## 端到端流程

```text
用户私聊机器人 / 群里 @机器人 / 发图片附件
      ↓
预检引导（卡片一题题问：系统 → 现象 → 影响范围 → 已尝试）
      ↓
知识库检索（ES）→ 命中给自助建议
      ↓ 点「已解决」                    ↓ 点「没解决，找人帮忙」
   记录自助解决                  定优先级 → 选值班团队 → 拉服务群（提单人 + 当班同学）
                                        ↓
                                  群内发工单卡片：认领 → 处理 → 已解决 → 结案
                                        ↓
                                  全过程时间线（群消息 + 每次状态变化）
                                        ↓
                                  结案：发结案卡片 + 结论回写知识库（闭环）
```

## 事件接入与路由

### 两种接入方式（`lark.event_mode`）

| 模式 | 取值 | 用途 | 前置条件 |
| --- | --- | --- | --- |
| 长连接 | `longconn`（默认） | 本地调试：不需要公网入口，程序主动连飞书 | 开放平台里把订阅方式设为「长连接」；填 `app_id` / `app_secret` |
| HTTP 回调 | `callback` | 生产：飞书推到你的域名，做签名/加密校验，可水平扩容 | 有公网地址，回调地址填 `https://你的域名/lark/event`；再填 `verification_token` / `encrypt_key` |

两种方式产出的是同一个归一化事件（`dto.Event`）：长连接取 SDK 给的原始报文，回调取 HTTP body，都交给 `lark.ParseEvent` 解析，之后的链路完全一致。

### 消息路由规则

机器人收到消息后先标准化成 `dto.MessageEvent`，再由 `router.handleMessage` 用 if/else 直接分流。
四个判断各自独立成函数，方便以后加维度：

| 判断函数 | 现在做什么 | 扩展点 |
| --- | --- | --- |
| `isP2P` | `chat_type == p2p` | — |
| `isFromThread` | `thread_id` 非空（话题消息） | 目前只用于日志；若想让话题内免 @ 也触发预检，在 `handleMessage` 里加这个判断 |
| `isMentionBot` | 是否 @机器人（没拿到 bot open_id 时退化为「@了任意人」） | — |
| `isOnCallGroup` | 是否在 oncall 协作群，**目前恒为 true** | 接团队群配置 / 工单服务群判断 |

| 场景 | 去向 |
| --- | --- |
| 单聊（p2p） | `HandleQuestionP2P` |
| 群里 @机器人 且在 oncall 群、并带可解析命令 | `HandleCommand` |
| 群里 @机器人 且在 oncall 群、其余情况 | `HandleQuestionChat`（预检流式卡片） |
| **群里没有 @机器人**（含话题内消息） | **什么都不做**，只打一行 `忽略：群里没有 @机器人` |

- 命令格式 `/命令`（信息都通过卡片表单填写，不用带参数）。入口在 `service/command/handler.go`，每条命令先发一张**仅发起人可见**的表单卡片：

  | 命令 | 含义 | 卡片标题 | 表单字段 | 确认提交入口 |
  | --- | --- | --- | --- | --- |
  | `/upgrade` | 升级值班层级 | 请填写本次升级的信息 | 具体原因 | `handleConfirmUpgrade`（TODO） |
  | `/transfer` | 转派给其它租户 | 请填写本次转接的信息 | 目标租户（下拉）、具体原因 | `handleConfirmTransfer`（TODO） |
  | `/event` | 升级为故障、拉起作战室 | 请填写本次升级作战室的信息 | 作战室群名、关联项目、具体原因 | `handleConfirmEvent`（TODO） |
  | `/save` | 暂时挂起工单 | 请填写本次工单挂起的信息 | 具体原因 | `handleConfirmSave`（TODO） |
  | `/solve` | 标记已解决 | （暂未做卡片） | — | `handleSolve`（TODO） |

  每张卡片底部都是同一行的「确认提交 / 取消」（`column_set` 分栏，`flex_mode: none` + 列宽 `auto` + 4px 间距，两个按钮贴合不拉开）：
  确认提交走对应的 `handleConfirmXxx`，取消返回「已取消」。
  目标租户的下拉选项来自 `service/tenant`（占位数据，TODO 接真实租户服务）。
- 卡片回调不走这套规则，直接交 `HandleCardAction`，由业务层按 `value.action` 分发。
- 群聊必须 @机器人 才响应，否则一律忽略（避免群里闲聊触发预检）；命令也要求先 @机器人。
- 日志：每条消息打一行 `收到消息`（p2p / thread / mention_bot / oncall_group / 文本 / 摘要），命中命令再打一行 `命中命令`，被忽略的打一行 `忽略：群里没有 @机器人`；卡片回调打 `收到卡片回调`。

### 预检流式卡片（question_chat）

群内提问（`question_chat`）交给 `service/precheck` 处理：

1. **先给用户的消息加一个表情回复**（`precheck.reaction_emoji`，默认 `OnIt` = 敲键盘那个），让用户立刻知道机器人在响应；
2. 再发一张**仅发起人可见**的卡片（群里其他人看不到）：`@发起人 正在匹配知识库...`，分割线下是红色「跳过，转人工」按钮；
3. 检索知识库（`searchKnowledge` 目前是占位实现，返回空结果）；
4. 拿到结果后更新同一张卡片：结果 + 「👍 有帮助 / 👎 无帮助」，分割线下是「已解决，无需转人工 / 仍需转人工」；
5. 点 👍 / 👎 → **这一行按钮**变成「您已选择 👍 有帮助，感谢您的反馈！」，结果内容与下方按钮都保留；
6. 点「已解决，无需转人工」→ **底部这一行按钮**变成「您的问题已确认通过预检解决」；
7. 点「跳过，转人工」/「仍需转人工」→ 原卡片底部那一行变成「请填写下方卡片确认入群信息」，同时**另发一张建群卡片**：问题描述（多行输入）+ 目标租户（下拉选择）+ 同一行的「确认提交」/ 红色「取消」（用 `column_set` 分栏实现并排）。
8. 点「确认提交」→ 建群卡片先变成「已提交，正在建群…」；后台调 `im/v1/chats` 建群（`CreateChat`），成功后同一张卡片换成成功态：

   ```text
   @你 **已提交，建群成功**

   问题描述：xxx
   目标租户：xxx
   ────────────
   [ 前往群聊 ]        <- 飞书 applink：https://applink.feishu.cn/client/chat/open?openChatId=<chat_id>
   ```

   建群失败时卡片显示失败原因，用户可以重新发起。成员目前传空（TODO：按目标租户 / 值班团队拉当班同学进群）。
9. 建群成功后按**建群时间**分流（`WorkTimeCard` / `WorkingCard`）：
   - **工作时间**（`worktime.ranges`，默认 `09:30-12:30`、`14:00-18:30`，按东八区判断）：调 `inviteOnDutyL1` 把该租户的 L1 值班人员拉进群，然后发一张公开的「工单处理中」卡片：@值班人员 + 问题描述 + 本群 5 个可用命令；
   - **非工作时间**：群里发一张公开的「非工作时间提示」卡（工作时间文案由配置渲染），下方分割线 + 红色「需要紧急介入」按钮；用户点击后走同一套 `startService`（拉人 + 工单处理中卡），并把提示卡更新成「已收到紧急介入请求」。

   > `inviteOnDutyL1(tenantID, chatID)` 目前是 **TODO 桩函数**（只打日志、不拉人）：需要先接值班数据源
   > （`oc_duty_shift` / 值班服务）拿到租户对应的 L1 值班同学 open_id，再调 `s.lark.InviteMembers(chatID, openIDs)`。
   > 因此「工单处理中」卡片现在 @ 的是一段占位文案。

用到的飞书接口（都在 `infra/lark/client.go`）：

| 能力 | 接口 | 说明 |
| --- | --- | --- |
| 仅特定人可见卡片 | `POST /open-apis/ephemeral/v1/send` | 只支持普通对话群；被指定的用户不会收到通知，且只有在线时可见；卡片里 @ 人不触发提及通知 |
| 原地更新卡片 | `PATCH /open-apis/im/v1/messages/:message_id` | 用上一步返回的 message_id |
| 话题内回复 | `POST /open-apis/im/v1/messages/:message_id/reply`（`reply_in_thread=true`） | **话题群不支持 ephemeral**，话题内自动退化为公开回复（日志会提示） |
| 删除仅特定人可见卡片 | `POST /open-apis/ephemeral/v1/delete` | ephemeral 卡片**没有更新接口**，所以"更新" = 删掉旧的再发一张；普通消息（话题内回复）仍走 patch |
| 卡片交互后更新卡片 | **回调响应体里回传卡片**：`{"toast": {...}, "card": {"type": "raw", "data": 卡片JSON}}` | 这是飞书推荐的更新方式，点按钮后直接把新卡片回给飞书即可，不用调任何更新接口 |

卡片 JSON 集中在 `service/precheck/card.go`（`SearchingCard` / `ResultCard`），改顺序或按钮文案只动这一个文件。
知识库检索的接入点是 `service/precheck/service.go` 的 `searchKnowledge`（TODO：FAQ + 文档 + 历史工单三源混合检索）。
按钮回调在 `HandleCardAction`：反馈（👍/👎）与转人工（换成建群表单）已实现；`buildGroup`（真正建群建单）和「已解决」还是 TODO。
注意：卡片交互的更新**必须同步返回**——飞书用响应体决定卡片怎么变，返回空响应或 nil 会让客户端报 `200080 / 200672` 这类交互错误。

> 状态处理：更新卡片要回传**整张卡片**，所以需要原卡片内容。这里不用服务端缓存，
> 而是把「结果内容 + 当前状态」一并塞进按钮 value（`result` / `feedback` / `decision`，见 `dto.CardActionValue`），
> 点击时原样带回，据此重建卡片——无状态，重启服务、多实例部署都不受影响。
> 另外：转人工时如果新卡片发送失败，原卡片**不会**被改成决策文案（保留原样，方便用户重试）。

> 关于"可搜索"下拉：飞书卡片 JSON 的下拉选择（`select_static`）在 1.0 / 2.0 组件文档里都**没有**搜索字段，
> 只有人员选择器（`select_person`）自带搜索。选项很多时的替代方案：改成「输入关键词 + 提交后校验」，
> 或先保留下拉，等接入租户服务后再换动态搜索（见 `card.go` 里 `tenantOptions()` 的 TODO）。

### 让事件真正跑起来，飞书后台要做的配置

1. 应用能力：机器人；权限：接收消息、发送消息、群相关操作（建群/拉人）、读通讯录基本信息。
2. 事件订阅：勾选「接收消息 `im.message.receive_v1`」与「卡片回传交互 `card.action.trigger`」。
3. 订阅方式二选一：**长连接**（本地调试）或**将事件发送至开发者服务器**并填 `https://你的域名/lark/event`（生产）。
4. 把 `app_id` / `app_secret`（回调模式再加 `verification_token`、`encrypt_key`）填进 `configs/config.yaml`；机器人 `open_id` **可以不填**——启动时会自动调飞书接口获取，用于判断群里是否 @机器人。
5. 想收到群里所有消息（不只 @机器人 的），还需要开通「读取群消息」相关权限。

## 目录分工

| 路径 | 负责什么 | 状态 |
| --- | --- | --- |
| `cmd/server` | HTTP 服务启动：加载配置 → 初始化日志 → 起服务 → 优雅退出 | ✅ 已实现 |
| `cmd/worker` | 后台进程启动：同上，后续挂定时任务与事件消费 | ✅ 已实现 |
| `configs/config.example.yaml` | 配置模板（入库，逐项注释引导填写） | ✅ 已实现 |
| `configs/config.yaml` | 本地实际配置（**不入库**，由 `make init-config` 生成） | ✅ 已实现 |
| `internal/conf` | 配置 + 环境变量加载与校验；配置型常量也放这里 | ✅ 已实现 |
| `internal/infra/mysql` | MySQL 连接池、建表入口 | ✅ 已实现 |
| `internal/infra/es` | 索引 mapping、切片写入、检索（当前 BM25，向量留 TODO） | ✅ 已实现 |
| `internal/infra/oss` | 对象存储：上传/下载/预签名（本地 MinIO，生产换公司 OSS 只改这里） | ✅ 已实现 |
| `internal/infra/lark` | 飞书开放平台：HTTP 接口封装（消息/群/通讯录/资源）、事件解密与签名、**事件解析 / 回调接入 / 长连接接入** | ✅ 已实现 |
| `internal/infra/mq` | 事件发布/订阅抽象（默认日志实现，RocketMQ 接法见注释） | ✅ 已实现 |
| `pkg/logx` `pkg/idgen` `pkg/timex` | 日志、业务 ID、时间与时区工具 | ✅ 已实现 |
| `internal/service/router` | 事件路由：消息 if/else 分流 + 卡片回调分发，判断函数集中在一处 | ✅ 已实现 |
| `internal/service/handler.go` | 事件处理器入口：命令已接真实入口，预检/卡片仍是日志占位 | 🚧 部分实现 |
| `internal/service/command` | 命令解析 + 5 个命令的处理入口（当前为日志占位） | 🚧 部分实现 |
| `internal/model/enum` | 工单状态、优先级、事件类型、卡片动作等枚举 | 🚧 占位 |
| `internal/model/entity` | 数据库表结构（GORM 模型） | 🚧 占位 |
| `internal/model/dto` | 飞书回调报文、卡片 value、服务入参出参 | 🚧 占位 |
| `internal/repository` | 数据访问（唯一直接碰 DB 的一层） | 🚧 占位 |
| `internal/service/precheck` | 预检问答 → 知识库建议 → 转人工 | 🚧 占位 |
| `internal/service/dispatch` | 选值班团队 → 拉群 → 拉人 → 发工单卡片 | 🚧 占位 |
| `internal/service/ticket` | 工单状态机 + 全过程时间线 + 事件发布 | 🚧 占位 |
| `internal/service/notify` | 卡片渲染与发送 | 🚧 占位 |
| `internal/service/knowledge` | 文档导入（切分 → 对象存储 → ES → 元数据）与检索 | 🚧 占位 |
| `internal/service/command` | 群内文本命令（帮助 / 查工单 / 升级 / 我的工单） | 🚧 占位 |
| `internal/service/view` | 工单视图组装（ticket 与 notify 共用，避免互相依赖） | 🚧 占位 |
| `internal/api` + `handler` `middleware` | 路由、飞书回调解析、幂等、健康检查 | 🚧 占位 |
| `internal/task` | SLA 看门狗、值班表同步、事件消费 | 🚧 占位 |
| `internal/app` | 依赖装配（连 DB / ES / 对象存储 / 飞书 / MQ，构造各 service） | 🚧 占位 |

## 依赖方向（谁能 import 谁）

```text
cmd  →  app / api  →  service  →  repository  →  model(entity/dto/enum)
                        ↓             ↓
                      infra        infra          （infra 只被上层调用，自己不 import service/repository）
                        ↓
                    conf / pkg                     （谁都能用，但它们不依赖任何内部包）
```

- 只有 `repository` 直接使用 `gorm`；`service` 不写 SQL。
- `handler` 只做「解析请求 → 调 service → 组装响应」，不写业务规则。
- `model` 不 import `repository` / `service` / `infra`。

## 建议的实现顺序

1. `model/enum` → `model/entity` → `model/dto`：先把领域词汇和表结构定下来。
2. `repository`：按实体逐个写 repo（建单、时间线追加、超时扫描、幂等事件）。
3 `service/ticket`：先把状态机写扎实（统一流转入口 + 时间线 + 事件），再写 `precheck` / `dispatch`。
4. `api` + `app`：接上飞书回调路由，把依赖装配起来，在 `cmd/server` 里把 `mux` 换成真实路由。
5. `task`：SLA 看门狗、值班表同步、事件消费。

每步的验收：`go build ./... && go vet ./...` 通过；链路能跑通（本地可用 curl 模拟飞书事件，联调期用飞书开放平台的「事件调试」）。

## 数据表规划（表名统一 `oc_` 前缀）

| 表 | 说明 |
| --- | --- |
| `oc_ticket` | 工单主表：工单号、标题、状态、优先级、提单人、服务群 chat_id、负责人、SLA 截止时间 |
| `oc_ticket_event` | 全过程时间线：状态变化、群消息、备注各一条 |
| `oc_precheck_session` | 预检会话：问答进度、答案、检索命中、关联工单号 |
| `oc_team` / `oc_team_member` / `oc_duty_shift` | 值班团队、成员、排班 |
| `oc_kb_document` / `oc_kb_chunk` | 知识库文档元数据、切片元数据（正文与向量在 ES） |
| `oc_processed_event` | 事件幂等表（飞书会重推事件） |

## 飞书应用需要的能力

- **事件订阅**：`im.message.receive_v1`（收消息）、`card.action.trigger`（卡片按钮回调）；回调地址 `POST /lark/event`。
- **权限**：发消息、**添加消息表情回复**（`im:message.reaction`，用于回 OnIt 表情）、建群与拉人、读通讯录基本信息（查姓名）；
  想记录群里全部消息还需要「读取群消息」相关权限。
- **配置项**：`lark.app_id` / `app_secret` / `verification_token` / `encrypt_key` / `bot_open_id` 填到 `configs/config.yaml`（从 `configs/config.example.yaml` 复制，**不入库**），或用环境变量注入。
- 飞书要求回调 **3 秒内响应**，所以 handler 只做校验 + 幂等，重活（检索、拉群）异步处理再返回 200。

## 配置管理（企业协作约定）

仓库里**只有模板**，真实配置不入库：

| 文件 | 是否入库 | 说明 |
| --- | --- | --- |
| `configs/config.example.yaml` | ✅ 提交 | 配置模板，逐项注释；新人照着填即可 |
| `configs/config.yaml` | ❌ 已 gitignore | 每个人自己生成：`make init-config` 或 `cp configs/config.example.yaml configs/config.yaml` |
| `configs/config.local.yaml` | ❌ 已 gitignore | 个人覆盖（改端口、连自己的库），优先级高于 `config.yaml` |

**加载顺序**（后面的覆盖前面的）：

```text
内置默认值 < configs/config.yaml < configs/config.local.yaml < 环境变量
```

也可以用 `-config 路径` 或 `BOKEONCALL_CONFIG=路径` 指定配置文件（这两种情况下不再走默认搜索）。启动日志会打印实际加载了哪个文件；走内置默认值时会打 WARN 提示，避免"以为读到了配置"。

**环境变量**（生产 / CI / K8s 用它，不落文件）：前缀 `BOKEONCALL_`，层级用下划线连接。

```bash
BOKEONCALL_APP_ENV=prod
BOKEONCALL_LARK_APP_ID=cli_xxx
BOKEONCALL_LARK_APP_SECRET=xxx
BOKEONCALL_MYSQL_PASSWORD=xxx
BOKEONCALL_MINIO_ACCESS_KEY=xxx
BOKEONCALL_MINIO_SECRET_KEY=xxx
```

**凭据策略**：数据库密码、对象存储 key、飞书 secret 一律不写进入库文件。本地留空会回退到与 `scripts/.env` 一致的开发默认值（所以刚 clone 下来就能跑）；`app.env` 不是 `local` 时，这些凭据必须显式提供（文件或环境变量都行，不强制落文件），否则**直接启动失败**，避免生产环境带着开发默认密码跑起来。

默认端口与 `../scripts/` 部署的中间件对齐：MySQL `3307`、Elasticsearch `9200`、MinIO `9000`、RocketMQ proxy `8081`（MySQL 用 3307 是因为本机 3306 被原生 MySQL 占用）。

## 本地运行

```bash
# 1) 生成自己的配置（不入库）
make init-config                # 或：cp configs/config.example.yaml configs/config.yaml

# 2) 起中间件（MySQL / ES / MinIO / RocketMQ）
cd ../scripts && ./up.ps1       # Windows；Linux 用 ./up.sh

# 3) 起后端
cd ../src
go run ./cmd/server            # HTTP :8080，健康检查 GET /healthz
go run ./cmd/worker            # 后台进程

make build / make test / make lint
```

## 团队约定

- 时间统一存 UTC，展示用东八区（`pkg/timex`）。
- 跨层传参用 `dto`，落库用 `entity`；不要在 handler 里直接拼 SQL。
- 飞书事件必须幂等（同一 `event_id` 只处理一次）。
- 工单状态流转只走一个统一入口，顺带写时间线，避免「状态改了但过程没记」。
- 卡片 JSON 统一在 `service/notify/card.go` 里拼，颜色、按钮文案集中改一处。
