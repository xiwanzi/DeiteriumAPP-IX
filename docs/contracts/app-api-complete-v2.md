# DeuteriumAPP 全功能接口文档 v2

配套体验版：**0.7.0-ui-lab (7)**。契约日期：**2026-09-08**。HTTP 基础路径继续使用 **`/api/v1`**；本文档 v2 不表示将路径改为 `/api/v2`。

本文可用于 Android、后端、未来网页商家端及平台管理端对接。配套 `openapi-app-v2.yaml` 为机器可读定义，`api-v2-examples.json` 为合成示例；`bridge-commerce-v2.md` 定义担保和游戏邮箱的内部桥接增量。每个端点下方列出请求模型、响应模型、字段表及示例；同名模型在文末完整定义。

## 1. 实现状态与范围

| 标记 | 含义 |
| --- | --- |
| `existing_v1` | 仓库已有服务端 v1 路由，本轮未改造后端。实际部署版本需由部署方核对。 |
| `existing_route_extension_pending` | 路由已存在，本文件所列新增字段、筛选或能力仍待后端实现。 |
| `contract_only` | 本轮新增的待实现服务端契约。体验 App 使用本机模型，不能据此认为接口已上线。 |

本轮已实现原生体验界面、委托/市场本机状态、模拟扫脸、通知本机偏好，以及更新文件的下载/校验/安装或资源应用流程。生产更新 URL 尚未配置；随包提供明确标识的资源更新样例。商家网页、管理网页和平台介入表单不在本轮实现范围。

沿用 `app-backend-api-v1.md` 的账号、身份、钱包、聊天、AI 和插件桥约定。QQ 注册必填且唯一，对登录用户可见；本轮没有新增 QQ 修改能力。原有 Wiki.js OIDC 和旧 AI 管理 HTML 页面仍分别使用其已有契约，不应把旧管理口令混入 App 公共接口。

## 2. 公共协议、身份和错误

- HTTPS + UTF-8 JSON。除既有公开账号入口和更新检查外，使用 `Authorization: Bearer <opaque-session-token>`。不解析 token，不引入 JWT 或 refresh token。
- 当前账号与服务器 UUID 的绑定由后端/插件确认。公开接口只使用不透明 `playerRef`。客户端不能传付款人、消息发送者、余额或权限来覆盖会话身份。
- 新接口成功响应为 `{requestId,data,serverTime}`，分页追加 `{page:{nextCursor}}`。已有 v1 数据包装保持兼容，例如 `data.records`、`data.messages`。
- 金额是十进制字符串，例如 `"1800.00"`，单位为信用点 `CREDIT`。客户端和服务端使用整数分或精确十进制；付款金额必须正数。商品/委托单价建议范围为 0.01–9,999,999.99。
- 时间采用 UTC ISO-8601。所有正式截止时间由服务端生成。App 使用 `serverTime` 与单调时钟估计剩余时间；手机本地时间不能决定结算结果。
- `X-Request-Id` 用于排障；`clientRequestId` 用于业务幂等，两者不能替代。幂等范围至少为当前账号 + 动作类型 + 键，同键不同正文返回 `IDEMPOTENCY_KEY_REUSED`。
- `expectedVersion` 是最近 GET 返回的版本。状态/编辑冲突返回 409，客户端重新获取数据再让用户确认，不能覆盖他人变更。
- 创建资金操作可能返回 202。新增 operation 只有 `COMPLETED` 才表示完成；既有转账和 AI 购买仍以 v1 的 `success` 作为成功终态。客户端据对应模型的终态播放成功反馈；`PROCESSING`/`UNKNOWN` 保持等待或查询。断网后复用原键，不创建第二笔扣款。
- 普通列表默认 20、最大 100；现有聊天历史限制仍按原接口。游标不透明，使用时间与 ID 稳定排序，避免翻页时重复/漏行。

统一错误体：

```json
{"requestId":"req_demo_001","error":{"code":"STATE_VERSION_CONFLICT","message":"订单状态已变化，请刷新后重试。","details":{"currentVersion":5}}}
```

| HTTP | 常见 code | 客户端处理 |
| --- | --- | --- |
| 400 | `INVALID_REQUEST` | 修正字段，不能盲目重试。 |
| 401 | `UNAUTHORIZED` | 清除失效会话并重新登录；改密后所有旧会话失效。 |
| 403 / 404 | `FORBIDDEN` / `NOT_FOUND` | 无权读取的订单、私聊、证据不可暴露给第三方。 |
| 409 | `STATE_VERSION_CONFLICT`、`INVALID_STATE_TRANSITION`、`IDEMPOTENCY_KEY_REUSED`、`ALREADY_ACCEPTED`、`QUOTE_EXPIRED`、`PRODUCT_CHANGED` | 刷新状态；金额变更必须重新报价并确认。 |
| 422 | `INSUFFICIENT_BALANCE`、`OUT_OF_STOCK`、`REFUND_ALREADY_USED`、`REJECTION_REASON_REQUIRED`、`ASSET_NOT_READY`、`ASSET_IN_USE`、`RESOURCE_INCOMPATIBLE` | 展示明确业务原因。 |
| 429 | `RATE_LIMITED` | 使用 `retryAfterSeconds`；不要并发重试。 |
| 503 | `PLUGIN_BRIDGE_UNAVAILABLE`、`SERVER_UNAVAILABLE` | 依赖不可用；发生过资金指令时先查原操作结果。 |

所有写接口必须限制正文大小、校验图片/证据归属、校验资源版本并审计敏感动作。网页端还需限制 CORS 到实际商家/管理端域名；不能启用任意 Origin 携带管理凭据。权限由服务端授予，普通注册和登录不接收 role/permissions。

### 2.1 图片素材：客户端直传 OSS（2026-09-08 更新）

**生产上传改为：客户端 → OSS 传文件，客户端 → 后端传 JSON；后端只负责授权、验证结果、资产记录及业务写入。** 替换之前的 multipart `POST /assets` 文件代理契约，不实现该旧文件入口；若保留兼容入口，应在读取请求体之前返回 `410 DIRECT_UPLOAD_REQUIRED`。管理员 APK/资源发布包仍使用独立发布接口，本节针对头像、玩家商品、委托、证据及商家图片素材。

| 步骤 | 调用 | 结果 |
| --- | --- | --- |
| 1. 申请授权 | `POST /assets/uploads`，JSON 包含用途、业务范围、最终文件名/类型/字节数/MD5 和幂等键 | 返回 uploadId、预分配 assetId、OSS uploadUrl、完整 POST Policy formFields、到期时间 |
| 2. 直接上传 | 客户端向返回的 OSS 地址发送 multipart 表单，逐项添加 formFields，最后添加 file | 文件流不经过业务 API；按真实上传字节显示进度，OSS 原生结果不使用业务 JSON 包装 |
| 3. 通知验证 | `POST /assets/uploads/{uploadId}/complete`，仅 JSON | 返回 VERIFYING 或 READY/REJECTED；客户端不能自行把 OSS 上传成功视为可发布 |
| 4. 查询恢复 | `GET /assets/uploads/{uploadId}`；授权过期但会话有效时 POST 同路径 `/renew` | 轮询验证；断网重试复用同一会话，已经上传的先验证，不盲目再传 |
| 5. 绑定业务 | 仅使用 READY 的 assetId 调用 profile、商品、委托或证据写接口 | 后端再次检查所有权、用途、交易/店铺权限及数量，再事务性写入业务引用 |

`POST /assets/uploads` 及 complete/renew 全部为 JSON，禁止携带 file 或 Base64。上传签名默认 5 分钟，会话最长 1 小时；过期会话重新申请新 ID，续签不改变文件大小、类型、用途、业务范围或对象 key。客户端必须先完成裁切/压缩再计算上传元信息，文件改变后重新申请。同一 clientRequestId 不重复创建会话，同一 uploadId 多次 complete 也只能生成一个资产。

**OSS 侧约束与权限：** 后端使用官方 SDK 签发 POST Policy，绑定精确随机暂存 key、`content-length-range=[sizeBytes,sizeBytes]`、限定内容类型和 private ACL；禁止客户端选择 bucket/key、公共读写权限或替换签名字段。不下发 AccessKeySecret，也不授予整个 bucket 的读写/list/delete 权限。文件最多 20 MiB，仅 JPEG/PNG/WebP。业务 token 不发送到 OSS；网页 CORS 只放行实际前端域名、POST 及所需响应头，签名表单不进入日志或业务记录。参见 [OSS PostObject](https://www.alibabacloud.com/help/en/oss/developer-reference/postobject)。

**后端验证不能只信“上传成功”：**

1. 只从 uploadId 的可信记录取得对象位置，拒绝客户端指定任意 URL/bucket/key；验证会话归属和当前业务权限。通过 OSS HeadObject 读取对象实际类型、大小、Content-MD5、ETag 和可用的版本信息，必须为本次普通上传对象；大小及实际 MD5 与申请一致。不要把 ETag 一律当 MD5，不能把客户端 x-oss-meta-* 当可信校验结果。此调用只返回元信息，不下载文件正文。参见 [OSS HeadObject](https://www.alibabacloud.com/help/en/oss/developer-reference/headobject)。
2. 最终 assetId 只引用客户端永远无写权限的最终对象。后端通过 OSS 内部 CopyObject 固定暂存对象的已知版本（存在版本号时）或使用源 ETag 条件复制到唯一 final key，然后针对最终对象完成验证；数据在 OSS 内部复制，不绕行后端。条件失败重新检查，不能验证旧内容却绑定新内容。即使授权未过期或 bucket 启用版本控制导致 forbid-overwrite 不生效，重传也不能覆盖最终资产。参见 [OSS 条件复制](https://www.alibabacloud.com/help/zh/oss/developer-reference/copyobject)。
3. 对最终私有图片使用 OSS `image/info` 等可信云端处理，取得真实格式、尺寸和可解码结果；只接受 JPEG/PNG/WebP，宽高各 1–8192。接口只取处理结果 JSON，不把整张图片拉回业务 API；处理能力不可用时维持 VERIFYING/重试，不能跳过校验。扩展内容审核/转码可在 OSS 云端执行，可信结果通过内部授权通道返回，不信任客户端回调。参见 [OSS 图片信息](https://www.alibabacloud.com/help/en/oss/user-guide/query-the-exif-data-of-an-image-4)。
4. 检查通过才原子写入 READY、实际 contentMd5/尺寸/大小和最终对象引用。AssetView.sha256 改为可空：没有可信云端实际计算结果就为 null；不为保留非空字段而重新下载全文件，也不采用客户端自报 SHA-256 冒充验证结果。MD5 仅用于传输完整性，权限由资产归属和业务关系保证；业务成交快照自身的服务端摘要规则不变。

**读取、绑定与失败处理：**

- 已有 GET `/assets/{assetId}` 继续返回有权读取的 AssetView。头像/商品/委托页面从业务响应取得 `url` 后直接请求 OSS/CDN；数据库业务关联存 assetId，不存上传签名或临时下载链接。私有证据/履约附件只返回短期下载 URL 和 urlExpiresAt，过期重新鉴权获取，不把它们公开。
- 新头像、商品及委托草稿可先上传再创建业务；证据必须关联本人参与的订单/委托/案件，商家媒体必须关联有编辑权限的店铺。授权、验证、最终业务写入三个阶段均查权限。预分配 assetId 在 VERIFYING/REJECTED/EXPIRED 时不能用于发布，返回 ASSET_NOT_READY；上传完成不自动注册商品、不扣款、不发布委托。
- 对象尚不存在：`409 UPLOAD_OBJECT_NOT_FOUND`，客户端确认 OSS 结果后可在会话有效期重试 complete。用途/格式/尺寸/校验值错误：REJECTED，rejectionCode 为 UPLOAD_CONTENT_MISMATCH 或 INVALID_IMAGE；当前会话不得重新绑定另一个文件。授权/会话过期：UPLOAD_EXPIRED；限流为 429。依赖超时不视为确定校验失败，保留最后状态供重试。
- 申请/续签都按账号限流并限制未完成会话与存储配额。过期暂存对象、失败验证对象及长期未被引用资产按生命周期回收；必须等待签名失效并再次确认没有业务引用。已被订单快照/证据引用的最终对象不得覆盖或由普通删除接口移除，继续执行 ASSET_IN_USE。

后端验收：接口请求体只有小型 JSON；OSS 上传 Policy 拒绝改 key/ACL/超长文件；A 不能完成 B 的上传；假回执、错 MD5、伪装图片及未知 objectKey 无法 READY；重复通知只产生一个资产；上传网络超时能恢复；授权重放不改变已引用的最终图片；资金/发布必须在 READY 后单独提交。此补充是契约，07 APK 和后端仍待实现直传流程。

## 3. 钱包、担保和交付边界

现有服务器经济系统是可用余额权威，后端保存担保、订单和业务账本。预付从可用余额扣除后进入担保，不能仅在 App 减一个显示数值。钱包显示 `availableAmount` 和 `heldAmount`；冻结细分为市场和委托。

正式上线前必须补齐受控经济桥的预付、结算、退款、状态查询与对账能力。**当前已有的玩家转账接口不等同于已实现担保。** 后端资金操作、业务状态与通知需要可靠事务/可恢复操作记录；插件处理同一 operationId 不得重复扣款、发款或发物品。桥超时使用 UNKNOWN 查证，不能自动把未知当失败后再补一次款。

官方商城还需受控邮箱模板、发放、撤回、领取状态能力。商家只能选择服务端批准的 `deliveryTemplateRef`，不能提交任意 Minecraft 命令。未领取退款与领取必须互斥：若已经领取，退款失败且同步真实领取状态；若撤回获胜，之后不能领取或再次发放。

## 4. 状态和时间规则

| 业务 | 正常流程 | 计时起点与结算 |
| --- | --- | --- |
| 普通市场商品 | 待发货 → 已发货 → 已确认 | 卖家发货后 72 小时自动确认；买家也可手动确认。 |
| 市场建筑服务 | 待开工 → 施工中 → 已完成待验收 → 已验收 | 开工后按总工期计时，工期包含验收预留；提交完成不重置期限。手动验收须已完成；原约定的到期自动验收规则保留。 |
| 委托 | 预付成功 → 待接取 → 进行中 → 已完成待确认 → 已确认 | 接取后计算履约时限；逾期未完成不自动付款。提交完成后另起 72 小时确认期。 |
| 官方商城 | 付款 → 邮箱已发送/未领取 → 已领取 | 未领取可撤回退款，领取后不可退款。领取状态来自游戏端。 |

完成表示工作已经交付，确认表示付款方已验收并释放担保；两者不能合并。金额、照片、要求和交付约定存成不可变成交快照，商品重新上架或商家后台编辑不改变旧订单。

市场分类固定六个代码：`MATERIALS` 建材、`EQUIPMENT` 装备、`SUPPLIES` 补给、`DECORATION` 装饰、`CONSTRUCTION` 建筑服务、`OTHER` 其他。分类取消是客户端移除筛选参数。商品发布 1–5 张图片、首图封面、库存 1–999；退款恢复库存时也不能突破库存与已预留量的约束。

委托发布前在表单编辑；发布时全额预付。尚未接取可取消退款；接取后锁定条款和报酬，不能单方降价或取消。一个委托只能由一个其他玩家接取，接取和取消必须原子互斥。

### 4.1 市场与委托可见性（2026-09-08 补充，后端必须执行）

本节覆盖此前大厅允许查询进行中/已完成委托的规则。本次仅更新接口契约；已交付 07 App 的本机列表尚未按本节改造。这里“公开”指所有已登录玩家可浏览，仍须 Bearer 鉴权。

| 对象 | 公开列表与详情 | 本人/双方订单页面 |
| --- | --- | --- |
| 待接取委托 | 仅 `status=OPEN` 且预付成功、`fundsStatus=HELD` 可见 | 发布者通过 `/commissions/me?role=PUBLISHER` 查看 |
| 已接取委托 | 接取提交成功后立即移出大厅；`ACTIVE`、`COMPLETED`、`CONFIRMED` 永不公开，退款或结算不会重新公开 | 发布者及唯一接取者通过 `/commissions/me` 和受限详情查看，终态仍保留历史记录 |
| 付款中/取消的委托 | `FUNDING`、`CANCELLED` 及预付失败均不公开 | 仅发布者及已存在的接取者按各自关系查看 |
| 市场在售商品 | 仅 `active=true AND stock>0`，其中 stock 是扣除订单预留后的可售库存 | 卖家 `/market/me/listings` 可管理在售、售罄及下架记录 |
| 市场成交订单（含建筑服务） | 从创建起就不是公开信息，任何订单状态均不能进入公开商品列表或玩家公开资料 | `/orders?channel=PLAYER_MARKET&role=BUYER/SELLER` 分别显示本人买入/卖出记录 |
| 市场售罄/下架商品 | `stock=0` 或 `active=false` 后退出公开列表和普通玩家的商品详情 | 卖家仍可管理；买家通过自己的订单和成交快照查看购买内容 |

市场保留已有多库存规则：库存为 1 的商品被购买/预留后隐藏；库存为 5、购买 1 件时，公开列表只展示剩余 4 件的商品，不展示该笔订单、买家、工程地点或履约信息。建筑服务的库存代表可售服务名额，规则相同。付款失败/退款只有经过业务确认释放或回补库存后才可能重新展示，不能仅收到退款请求就回补；卖家已下架时不自动重新上架。

**接口约束：**

- `GET /commissions` 的 status 可省略，默认且唯一合法值为 `OPEN`；传 `ACTIVE` 等其他值返回 `400 INVALID_REQUEST`。服务端必须自行加可见性条件，不能依赖网页传参。搜索、推荐、分类、数量统计及每次游标翻页均执行相同过滤。
- `/commissions/me` 必须按当前会话的 owner/worker 关系筛选；`/orders` 必须按当前会话的 buyer/seller 关系筛选。role 只是本人身份筛选，不能传他人 ID 扩大范围。遗漏 role 时 `/orders` 只取本人参与订单的并集。
- 已接取委托的 `/commissions/{id}`，所有市场 `/orders/{id}`、成交快照、退款记录、介入材料和私有附件必须逐对象鉴权。退款、确认、撤回、关闭后仍仅双方可读；未接取委托的条款快照仅发布者可读。隐藏不删除、不改变履约和结算。
- 有管理职能仍须具备对应案件/资源范围授权，且后台访问留审计；普通商家角色不获得玩家市场或委托订单的全局读取权。管理员通过授权管理流程读取，不能靠前端传 admin=true 或自行声明角色。
- 已登录但非参与者直接访问受限资源时统一返回 `404 NOT_FOUND`，与不存在的 ID 使用相同错误体，不附带金额、状态或参与者；未登录仍返回 401。嵌套 refundId/caseId/assetId 必须属于对应交易，附件下载同样检查授权，不能永久公开证据 URL。
- 接取时原子执行 OPEN 检查、worker 绑定、ACTIVE 迁移与公开资格撤销；购买时原子预留库存并创建私有订单。事务提交后的新查询必须反映隐藏状态；旧页面或旧报价再次操作必须重新校验，不允许二次接取/超卖。
- 大厅/详情/本人订单及错误响应使用 `Cache-Control: private, no-store`；搜索索引只可提供候选 ID，返回前回查当前状态和权限。旧游标不得绕过过滤，不把私有订单混入共享缓存。
- `order.updated`、`commission.updated` 及业务通知只发给有权参与方；大厅不订阅私有履约事件。本轮通过重新查询公开列表同步移除，网页切回前台、刷新或操作后重新读取；不得为了让大厅移除卡片而广播接取者、订单或退款信息。历史上已公开的内容无法从他人设备追溯删除，但接取后的新读取必须拒绝。

**后端验收：** A 发布委托、B 接取后，C 的大厅/搜索/旧游标均无该条，直接 GET 详情或快照为 404；A/B 的本人列表及详情仍可读。A 卖库存为 1 的商品给 B 后，C 看不到该商品及订单；库存为 5 时 C 仅能看剩余 4 件。C 猜测订单/退款/证据 ID 或持有旧链接仍被拒绝；A/B 在完成、退款后仍能查看自己的历史记录。普通商家不能越权，授权管理员可处理案件且有审计。

## 5. 退款与平台介入

市场商品发货/开工后，或委托被接取后，付款方只有一次退款申请机会。申请时暂停当前阶段倒计时；撤回或被拒后继续剩余时间，不恢复次数。拒绝必须填写 2–500 字理由，写入退款记录并通知付款方。同意后完成实际返款才结束订单。

平台介入不是重新增加一次退款机会。申请人提交争议说明、诉求、证据引用和可见聊天消息 ID；服务端捕获成交条款、当前订单、履约时间、退款理由、付款/冻结/结算事件作为不可变案件快照。不能信任客户端上传的“订单 JSON”或自报交易金额。

受理未结算争议时，创建案件、冻结尚未释放的担保款与暂停自动确认应为同一个受控状态变更，防止定时结算抢先释放。已结算的交易也可记录售后争议，但 `fundsHeldForReview=false`；管理员不能凭裁决强制把收款人余额扣成负数，应进入人工追偿/协商处理。支持部分退款时，退款加释放金额不得超过实际被冻结的剩余金额。

同一交易同时最多一个活跃案件。案件领取、索要证据、裁决和资金执行均校验权限/版本/幂等并审计。裁决资金处理尚未完成时维持 RESOLVING，不能提前通知退款成功。本轮只提供这些接口，App 不新增平台介入入口或表单。

## 6. 商家网页需要编辑哪些信息

| 编辑区域 | 字段/接口 |
| --- | --- |
| 店铺名片 | name、intro、logo、cover、contactQq、serviceHours、notice；merchant store GET/PUT。 |
| 首页内容 | intro、sections、区域标题/布局、轮播商品、横幅、品牌与分类顺序；homepage GET/PUT。 |
| 品牌与分类 | 名称、Logo、排序、启用状态；merchant brands/categories。 |
| 商品信息 | title、subtitle、description、brandId、categoryId、price、库存策略/库存、单笔限购。 |
| 商品图片 | coverAssetId、galleryAssetIds、图片说明；封面必须是画廊首图。 |
| 商品详情 | 有序 contentBlocks、包含内容、交付模板、交付说明、预计送达。 |
| 商品展示 | posterTone、accentColor、badges、sortOrder；不覆盖系统和付款主题。 |
| 发布管理 | 草稿 GET/PUT、发布、下架、归档、带原因库存调整；版本锁保护并发编辑。 |
| 订单履约 | 订单列表/详情、退款记录、幂等邮箱重试；不能修改已成交价格或任意拒绝平台规定的未领取退款。 |

官方店铺由平台授权，不开放普通玩家自助取得商家权限。品牌（Apple/NVIDIA/AMD）是商品分类信息，不代表这些公司与本项目存在经营关系；体验版硬件图片和信用点价格是样例。

## 7. 聊天、引用与实时通知

继续使用 `GET /api/v1/chat/ws`，携带 Bearer Header，使用标准 WebSocket ping/pong。已有 chat.send/chat.message/presence/mention/follow 事件保持兼容。新增消息仅扩展 payload，旧客户端忽略未知事件类型。私聊和新业务事件的服务端能力均为待实现扩展。

公共聊天、私聊和 AI 回复引用只提交 `replyToMessageId`。后端校验同一会话且发送者可见，再生成 `ReplySnapshot`。原消息不可用时使用 UNAVAILABLE，不泄漏原私聊；客户端不能自报原作者/原文。复制操作只访问系统剪贴板，不调用服务端。

新增建议事件：`notification.created`、`order.updated`、`commission.updated`、`catalog.changed`、`app.update.available`。事件含 eventId、occurredAt、resourceId、resourceVersion 和通知摘要。客户端按 eventId 去重，并 GET 权威详情；断线后通过 inbox/列表游标补齐。事件本身不能授权付款或结算。

```json
{"type":"commission.updated","requestId":"req_event_001","sentAt":"2026-09-08T02:00:00Z","payload":{"eventId":"event_001","resourceId":"commission_demo_001","resourceVersion":4,"occurredAt":"2026-09-08T02:00:00Z","status":"COMPLETED"}}
```

后台推送依据八类业务偏好和总开关投递；系统权限仍由 Android 决定。关闭推送不会删除收件箱、订单或钱包流水。设备令牌绑定当前用户和 installationId，不能广播给无关玩家。

## 8. APK 与资源更新

`GET /app/update-check` 保留旧版本字段，新增 `apkUpdate`、`resourceUpdate`、`hasUpdates`。没有该类更新返回 null。按包名、频道和兼容版本筛选；旧客户端不传 packageName 时默认正式 App 包名。

1. 从可信更新服务读取清单，显示真实版本、大小和说明，出现红点。
2. 用户触发下载，按实际已下载字节显示进度。服务器建议支持 Range、Content-Length、Content-Range 和不可变文件 URL。网络中断可复用匹配清单的部分文件；服务器不支持 Range 时从头下载。
3. 全量校验文件大小和 SHA-256。APK 再核对真实包名、签名和递增的 versionCode，使用受限 FileProvider URI 交给 Android 安装确认；不静默安装、不绕过来源安装许可。
4. 资源 ZIP 读取 manifest.json：版本兼容、清单、单文件摘要、路径与展开大小均检查。只允许图片、文案与受支持的配置，不加载 DEX/JAR/SO/HTML/JavaScript 等执行内容。
5. 所有资源在新的 App 私有目录验证后才切换活动版本；失败保留旧资源。当前体验版支持 `strings.json` 中 commissions.subtitle、updates.resourceNote、announcement.footer，以及图片。商品价格、权限和资金规则不能通过资源包覆盖。

资源 ZIP 限 20 MiB、展开总量 100 MiB、最多 500 个文件，manifest.json ≤64 KiB；禁止清单外文件、重复路径、路径穿越和不支持类型。配置未提供生产服务地址时，体验 App 明确显示随包样例来源，不能把本机样例称为互联网更新。

后台上传二进制后由服务端提取真实包元信息和哈希；网页不能自行填写错误摘要来绕过检查。发布文件不可原地替换，撤回仅停止推荐。资源回退可将旧内容以递增资源版本重新发布；APK 不强制降级。

Android 实现依据：[安全共享文件](https://developer.android.com/training/secure-file-sharing/setup-sharing)、[安装来源设置](https://developer.android.com/reference/android/provider/Settings)、[动态代码加载风险与完整性检查](https://developer.android.com/privacy-and-security/risks/dynamic-code-loading)。机器契约遵循 [OpenAPI 3.0.3](https://spec.openapis.org/oas/v3.0.3.html)。

### 8.1 远程材质配置（2026-09-08 补充）

本次交付后端/客户端对接契约，新增路由均为 `contract_only`。07 APK 目前仍使用本机参数，底栏/浮层虽已有独立开关，但数值仍共用一组；后续 App 接入必须拆成独立状态和存储，不能把新增接口写成现有 APK 已能远程生效。

| 配置组 | 用途 | 独立字段 |
| --- | --- | --- |
| `bottomBarGlass` | 底部导航栏玻璃 | enabled、blurRadiusDp、opacity、refractionStrength、highlightStrength、dynamicHighlightEnabled、allowLocalTuning |
| `overlayGlass` | 弹窗、底部弹层、改密/转账/付款卡片等浮层玻璃 | 与底栏同类型，但独立存储、下发、调整、关闭和重置，绝不自动联动 |
| `headerGradient` | 顶部连续渐变 | blurRadiusDp、fadeHeightDp、allowLocalTuning；不隐式借用底栏/浮层参数 |

内置默认：两组玻璃分别为 enabled=true、blurRadiusDp=18、opacity=0.42、refractionStrength=9、highlightStrength=0.65、dynamicHighlightEnabled=true、allowLocalTuning=true；顶部 blurRadiusDp=18、fadeHeightDp=32、allowLocalTuning=true。它们初值相同不表示共享一份对象。模糊 0–36 dp、不透明度 0.08–1、折射 0–24、高光 0–1、顶部扩展 16–96 dp；服务端拒绝越界、非有限数、未知字段与执行内容，客户端也须校验。

**读取与后台修改：**

- App/网页使用 `GET /app/appearance`，提供 packageName、channel、schemaVersion=1；服务端根据登录会话识别玩家，返回完整三组配置、来源和 globalVersion/playerVersion/effectiveRevision。原生 dp 为密度无关单位；网页若复用外观可按逻辑像素映射，但不保证浏览器和 Android 模糊算法像素完全一致。
- 平台管理用 `GET/PATCH /admin/appearance/global` 和 `GET/PATCH /admin/appearance/players/{playerRef}`，查询参数指定包名/频道。只有平台 APPEARANCE_READ/APPEARANCE_MANAGE 权限且在对应管理范围内可用；普通玩家和商家不能自行授予或调用。每次修改记录操作者、范围、前后值、原因和版本到现有审计日志。
- 每组按 **玩家专属覆盖 → 全局覆盖 → 内置默认** 解析，三组各自计算来源，不交叉继承。全局调整只影响没有该组玩家专属覆盖的玩家；清除玩家覆盖后恢复当前全局值。
- PATCH 只修改出现的组：省略保留原值，完整对象覆盖该组，null 清除该范围的这一组。清除全局组恢复内置默认；清除玩家组恢复全局/内置。至少一组必须出现。使用 clientRequestId 幂等、expectedVersion 防覆盖；首次无记录 GET 返回 version=0。修改/重置均生成新版本，回退也按旧数值提交成新版本，不倒退版本号。

**生效与本机设置：**

1. 参数 allowLocalTuning=true 时，该组用户手动数值优先于远程默认；false 时以该组远程数值为准，保留但暂不应用用户自定义，以后恢复允许时可继续使用。只有没有本机自定义的字段使用远程默认。实际 enabled = 远程 enabled AND 用户本机开关；后台可关闭效果，但不能强行打开用户已经关闭的玻璃。动态追光同样受本机开关、减少动态效果和设备能力限制。
2. 客户端首次登录、切换账号、前台间隔到期、主动刷新或收到 appearance.updated 时重新读取；refreshAfterSeconds 默认 300，不要求后台常驻。下载资源包和 OSS 图片不覆盖这些设置，材质参数走独立配置接口。
3. 完整验证 schema 和数值后，在 UI 安全帧一次性应用全部三组；不要在付款动画中间切换正在使用的材质，可在本次动画结束后应用。过期并发响应按请求序号丢弃；effectiveRevision 只作相等比较，不按字符串大小排序。
4. 缓存按账号、包名、频道、schema 分开；网络失败保留上次有效配置，无缓存用内置默认。切换账号/退出时先停止使用上一账号专属配置。HTTP 使用 private, no-store，客户端只在私有存储显式保存已验证的最后有效配置；不使用共享 HTTP 缓存。
5. 客户端不认识 schema 或配置无效时整份不应用，保留旧配置并提示/记录错误；Android 不支持的渲染能力降级，但不能影响登录、订单或支付。未升级接入逻辑的旧 App 继续使用本机配置。管理端保存成功不等于所有玩家已应用；离线设备在恢复连接后读取。

只修改浮层、保持底栏与顶部不变的 PATCH 示例（具体响应和所有字段见后文新增端点）：

```json
{"clientRequestId":"appearance_patch_001","expectedVersion":0,"reason":"仅调整浮层参数","overlayGlass":{"enabled":true,"blurRadiusDp":24,"opacity":0.7,"refractionStrength":4,"highlightStrength":0.35,"dynamicHighlightEnabled":true,"allowLocalTuning":false}}
```

验收：只更新浮层时底栏逐字段不变；只关闭底栏时浮层保留；A 玩家覆盖不影响 B；清除 A 的浮层覆盖后恢复全局浮层，底栏不变；并发旧版本拒绝、幂等重试不增版本；非法参数不保存；断网、切换账号、旧客户端及减少动态效果按上述规则降级。客户端首次接入需要更新 APK，之后在 schema 支持范围内调参无需每次更新 APK。

## 9. 页面到接口映射

| App/网页功能 | 接口或本机行为 |
| --- | --- |
| 注册、登录、改密 | 原有 account code/register/login/password-reset/logout/me。 |
| 个人简介、头像、点头像资料 | assets、account/me/profile、players/{playerRef}。 |
| 联系人、在线状态、特别关心 | 原有 chat/player-directory、presence、online-players、follows。 |
| 钱包、转账、账单筛选/详情 | 原有 wallet + 新增 records/{recordId} 与筛选、冻结余额扩展。 |
| 官方商城、购物袋、结算 | store/*、checkout/quotes、store/orders、orders/*。 |
| 市场筛选、发布、上下架、多图 | market/categories、listings、me/listings、assets。 |
| 市场履约、施工完成、验收、退款理由 | orders/{id} 的 ship/start-work/complete-work/confirm/refunds。 |
| 委托大厅、发布/接取、完成/确认 | commissions、commissions/me、各状态动作和退款。 |
| 公共聊天、私聊、长按回复 | chat/ws 或 chat/messages；conversations；replyToMessageId。 |
| 长按复制 | 系统剪贴板，无业务 API。 |
| AI 聊天、套餐与额度 | 原有 ai/me/plans/messages/chat/stream/reset/purchases；引用字段增量。 |
| 公告 | announcements；后台 admin/announcements。 |
| 通知偏好、交易通知、推送设备 | notifications/preferences、inbox/read、devices。 |
| 软件更新与红点 | app/update-check、清单 HTTPS 文件；admin/releases 发布。 |
| 柔光玻璃两组开关、灵动视效、动态追光、材质参数 | app/appearance + admin/appearance/* 定义远程配置；当前 07 App 尚未接入。 |
| 模拟 Face ID 与圆环确认动画 | 本机渲染，无相机/人脸上传；正式成功条件来自对应资金接口。 |
| 平台介入 | interventions + admin/interventions；本轮 App 不做表单。 |

## 10. 端点详细参考

下列示例均为合成结构示例，`example.invalid`、示例 token、资产 ID 和摘要不是真实服务配置。跨字段业务约束仍按以上规则执行。

### 账号与资料

#### `POST /api/v1/account/login` — 登录

实现状态：`existing_v1`。权限：公开入口。

请求模型：`LoginRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "account": "example",
  "password": "example-"
}
```

响应 `200`：`AuthResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "token": "example",
    "user": {
      "userId": "example",
      "playerRef": "example",
      "gameId": "example",
      "qq": "example",
      "identityStatus": "bound",
      "bio": "example",
      "avatar": null,
      "profileVersion": 1
    }
  }
}
```

#### `POST /api/v1/account/logout` — 退出当前设备登录

实现状态：`existing_v1`。权限：已登录用户。

响应 `200`：`LogoutResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "loggedOut": true
  }
}
```

#### `GET /api/v1/account/me` — 获取当前用户资料

实现状态：`existing_v1`。权限：已登录用户。

响应 `200`：`UserProfileResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "user": {
      "userId": "example",
      "playerRef": "example",
      "gameId": "example",
      "qq": "example",
      "identityStatus": "bound",
      "bio": "example",
      "avatar": null,
      "profileVersion": 1
    }
  }
}
```

#### `PATCH /api/v1/account/me/profile` — 更新个人简介或头像

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

仅修改本人资料；头像资产必须属于本人且用途为 AVATAR。QQ 修改仍不在已确认的账号规则中。

请求模型：`ProfilePatchRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1,
  "bio": "example",
  "avatarAssetId": null
}
```

响应 `200`：`V2PublicPlayerProfileResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "playerRef": "player_mori",
    "gameId": "Mori",
    "qq": "1000101",
    "bio": "把每一次日落留在小花园里。",
    "avatar": {
      "assetId": "asset_cover_001",
      "purpose": "AVATAR",
      "status": "READY",
      "url": "https://cdn.example.invalid/assets/cover.png",
      "width": 1254,
      "height": 1254,
      "sizeBytes": 1819447,
      "sha256": null,
      "altText": "商品封面",
      "createdAt": "2026-09-08T02:00:00Z",
      "contentMd5": null,
      "urlExpiresAt": null
    },
    "online": true,
    "lastSeenAt": "2026-09-08T02:00:00Z",
    "followed": false,
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/account/password-reset` — 提交新密码

实现状态：`existing_v1`。权限：公开入口。

请求模型：`PasswordResetRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "verificationToken": "example",
  "code": "123456",
  "newPassword": "example-"
}
```

响应 `200`：`PasswordResetResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "passwordReset": true
  }
}
```

#### `POST /api/v1/account/password-reset-code` — 请求密码重设验证码

实现状态：`existing_v1`。权限：公开入口。

请求模型：`PasswordResetCodeRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "account": "example"
}
```

响应 `200`：`VerificationTokenResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "verificationToken": "example",
    "expiresAt": "2026-09-08T02:00:00Z",
    "resendAfterSeconds": 1
  }
}
```

#### `POST /api/v1/account/register` — 提交注册

实现状态：`existing_v1`。权限：公开入口。

请求模型：`RegisterRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "verificationToken": "example",
  "code": "123456",
  "password": "example-"
}
```

响应 `200`：`AuthResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "token": "example",
    "user": {
      "userId": "example",
      "playerRef": "example",
      "gameId": "example",
      "qq": "example",
      "identityStatus": "bound",
      "bio": "example",
      "avatar": null,
      "profileVersion": 1
    }
  }
}
```

#### `POST /api/v1/account/registration-code` — 请求注册验证码

实现状态：`existing_v1`。权限：公开入口。

请求模型：`RegistrationCodeRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "gameId": "example",
  "qq": "12345"
}
```

响应 `200`：`VerificationTokenResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "verificationToken": "example",
    "expiresAt": "2026-09-08T02:00:00Z",
    "resendAfterSeconds": 1
  }
}
```

#### `GET /api/v1/players/{playerRef}` — 获取玩家资料

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

供联系人、公共聊天、私聊头像统一使用。已登录才可读取 QQ；lastSeenAt 未知返回 null。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `playerRef` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

响应 `200`：`V2PublicPlayerProfileResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "playerRef": "player_mori",
    "gameId": "Mori",
    "qq": "1000101",
    "bio": "把每一次日落留在小花园里。",
    "avatar": {
      "assetId": "asset_cover_001",
      "purpose": "AVATAR",
      "status": "READY",
      "url": "https://cdn.example.invalid/assets/cover.png",
      "width": 1254,
      "height": 1254,
      "sizeBytes": 1819447,
      "sha256": null,
      "altText": "商品封面",
      "createdAt": "2026-09-08T02:00:00Z",
      "contentMd5": null,
      "urlExpiresAt": null
    },
    "online": true,
    "lastSeenAt": "2026-09-08T02:00:00Z",
    "followed": false,
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

### 钱包与转账

#### `GET /api/v1/wallet/balance` — 获取当前已知余额

实现状态：`existing_route_extension_pending`。权限：已登录用户。


新增 availableAmount、heldAmount、heldBreakdown 和 serverNow。amount 保留为可用余额兼容字段。

响应 `200`：`WalletBalanceResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "balance": {
      "currency": "CREDIT",
      "amount": "1820.50",
      "fresh": true,
      "refreshedAt": null,
      "availableAmount": "680.00",
      "heldAmount": "680.00",
      "heldBreakdown": {
        "market": "680.00",
        "commissions": "680.00"
      },
      "serverNow": "2026-09-08T02:00:00Z"
    }
  }
}
```

#### `POST /api/v1/wallet/balance/refresh` — 刷新信用点余额

实现状态：`existing_v1`。权限：已登录用户。

响应 `200`：`WalletBalanceResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "balance": {
      "currency": "CREDIT",
      "amount": "1820.50",
      "fresh": true,
      "refreshedAt": null,
      "availableAmount": "680.00",
      "heldAmount": "680.00",
      "heldBreakdown": {
        "market": "680.00",
        "commissions": "680.00"
      },
      "serverNow": "2026-09-08T02:00:00Z"
    }
  }
}
```

#### `GET /api/v1/wallet/recipients/search` — 搜索收款玩家

实现状态：`existing_v1`。权限：已登录用户。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `query` | query | 是 | string | 长度 1–—； |
| `type` | query | 否 | RecipientSearchType | ；默认 auto |

响应 `200`：`RecipientSearchResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "candidates": [
      {
        "playerRef": "player_mori",
        "gameId": "Mori",
        "qq": null,
        "online": true,
        "registered": true,
        "source": "game_id",
        "confirmedAt": "2026-09-08T02:00:00Z",
        "expiresAt": null
      }
    ]
  }
}
```

#### `GET /api/v1/wallet/records` — 获取基础信用点流水

实现状态：`existing_route_extension_pending`。权限：已登录用户。


新增筛选参数为待实现扩展；时间范围采用 [from,to)，最大跨度建议 366 天。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `direction` | query | 否 | string | `income` / `expense`； |
| `businessType` | query | 否 | string | `TRANSFER` / `STORE_PURCHASE` / `MARKET_RESERVE` / `MARKET_SETTLEMENT` / `COMMISSION_RESERVE` / `COMMISSION_SETTLEMENT` / `REFUND` / `ADJUSTMENT` / `AI_PURCHASE`； |
| `from` | query | 否 | string | date-time； |
| `to` | query | 否 | string | date-time； |

响应 `200`：`WalletRecordsResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "records": [
      {
        "recordId": "example",
        "direction": "income",
        "otherPlayer": {
          "playerRef": "player_mori",
          "gameId": "Mori",
          "qq": null,
          "online": true,
          "registered": true,
          "source": "game_id"
        },
        "amount": "680.00",
        "currency": "CREDIT",
        "status": "processing",
        "note": null,
        "occurredAt": "2026-09-08T02:00:00Z"
      }
    ]
  },
  "page": {
    "nextCursor": null
  }
}
```

#### `GET /api/v1/wallet/records/{recordId}` — 获取账单详情

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

冻结、结算和退款分别有不可变业务关联，不能用一条泛化转账掩盖订单资金操作。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `recordId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

响应 `200`：`V2WalletRecordDetailResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "recordId": "ref_demo_001",
    "direction": "income",
    "amount": "680.00",
    "currency": "CREDIT",
    "businessType": "TRANSFER",
    "businessRef": "ref_demo_001",
    "counterparty": {
      "kind": "PLAYER",
      "playerRef": "player_mori",
      "storeId": null,
      "displayName": "Mori",
      "contactQq": "1000101"
    },
    "status": "PROCESSING",
    "note": "example",
    "occurredAt": "2026-09-08T02:00:00Z"
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/wallet/transfers` — 提交信用点转账

实现状态：`existing_v1`。权限：已登录用户。

请求模型：`CreateTransferRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "example",
  "recipientPlayerRef": "example",
  "amount": "680.00",
  "note": null
}
```

响应 `200`：`TransferResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "transfer": {
      "transferId": "example",
      "clientRequestId": "example",
      "recipient": {
        "playerRef": "player_mori",
        "gameId": "Mori",
        "qq": null,
        "online": true,
        "registered": true,
        "source": "game_id"
      },
      "amount": "680.00",
      "currency": "CREDIT",
      "note": null,
      "status": "processing",
      "createdAt": "2026-09-08T02:00:00Z",
      "updatedAt": "2026-09-08T02:00:00Z"
    }
  }
}
```

处理中响应 202：查询 operationId；收到终态前不能把动画完成视为已付款。

#### `GET /api/v1/wallet/transfers/{transferId}` — 查询转账结果

实现状态：`existing_v1`。权限：已登录用户。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `transferId` | path | 是 | string | ； |

响应 `200`：`TransferResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "transfer": {
      "transferId": "example",
      "clientRequestId": "example",
      "recipient": {
        "playerRef": "player_mori",
        "gameId": "Mori",
        "qq": null,
        "online": true,
        "registered": true,
        "source": "game_id"
      },
      "amount": "680.00",
      "currency": "CREDIT",
      "note": null,
      "status": "processing",
      "createdAt": "2026-09-08T02:00:00Z",
      "updatedAt": "2026-09-08T02:00:00Z"
    }
  }
}
```

### 聊天与在线状态

#### `GET /api/v1/chat/conversations` — 获取私聊列表

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `cursor` | query | 否 | string | 长度 1–512； |
| `limit` | query | 否 | integer | 范围 1–100； |

响应 `200`：`V2ConversationViewListResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": [
    {
      "conversationId": "ref_demo_001",
      "otherPlayer": {
        "playerRef": "player_mori",
        "gameId": "Mori",
        "qq": "1000101",
        "bio": "把每一次日落留在小花园里。",
        "avatar": {
          "assetId": "asset_cover_001",
          "purpose": "AVATAR",
          "status": "READY",
          "url": "https://cdn.example.invalid/assets/cover.png",
          "width": 1254,
          "height": 1254,
          "sizeBytes": 1819447,
          "sha256": null,
          "altText": "商品封面",
          "createdAt": "2026-09-08T02:00:00Z",
          "contentMd5": null,
          "urlExpiresAt": null
        },
        "online": true,
        "lastSeenAt": "2026-09-08T02:00:00Z",
        "followed": false,
        "version": 1
      },
      "lastMessage": null,
      "unreadCount": 1,
      "updatedAt": "2026-09-08T02:00:00Z",
      "version": 1
    }
  ],
  "serverTime": "2026-09-08T02:00:00Z",
  "page": {
    "nextCursor": null
  }
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/chat/conversations` — 获取或创建与玩家的私聊

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

同一双方只有一个私聊会话；不能指定第三方发送者或读取不属于自己的会话。

请求模型：`CreateConversationRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "otherPlayerRef": "ref_demo_001"
}
```

响应 `201`：`V2ConversationViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "conversationId": "ref_demo_001",
    "otherPlayer": {
      "playerRef": "player_mori",
      "gameId": "Mori",
      "qq": "1000101",
      "bio": "把每一次日落留在小花园里。",
      "avatar": {
        "assetId": "asset_cover_001",
        "purpose": "AVATAR",
        "status": "READY",
        "url": "https://cdn.example.invalid/assets/cover.png",
        "width": 1254,
        "height": 1254,
        "sizeBytes": 1819447,
        "sha256": null,
        "altText": "商品封面",
        "createdAt": "2026-09-08T02:00:00Z",
        "contentMd5": null,
        "urlExpiresAt": null
      },
      "online": true,
      "lastSeenAt": "2026-09-08T02:00:00Z",
      "followed": false,
      "version": 1
    },
    "lastMessage": null,
    "unreadCount": 1,
    "updatedAt": "2026-09-08T02:00:00Z",
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/chat/conversations/{conversationId}/messages` — 分页读取私聊消息

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `conversationId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |
| `beforeMessageId` | query | 否 | string | 长度 1–128； |
| `cursor` | query | 否 | string | 长度 1–512； |
| `limit` | query | 否 | integer | 范围 1–100； |

响应 `200`：`V2ChatMessageListResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": [
    {
      "messageId": "message_reply",
      "sender": {
        "playerRef": "player_mori",
        "gameId": "Mori",
        "qq": null,
        "online": true,
        "registered": true,
        "source": "game_id"
      },
      "content": "收到，我上线后去看看。",
      "kind": "private_chat",
      "sentAt": "2026-09-08T02:00:00Z",
      "conversationId": "conversation_mori_xiwanzi",
      "reply": {
        "messageId": "message_original",
        "sender": {
          "playerRef": "player_mori",
          "gameId": "Mori",
          "qq": null,
          "online": true,
          "registered": true,
          "source": "game_id"
        },
        "content": "花园的灯笼已经摆好了。",
        "availability": "AVAILABLE"
      },
      "clientMessageId": "msg-client-002"
    }
  ],
  "serverTime": "2026-09-08T02:00:00Z",
  "page": {
    "nextCursor": null
  }
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/chat/conversations/{conversationId}/messages` — 发送私聊消息或回复

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

服务端填充发送者和引用快照；回复必须指向同一会话的可见消息。消息重试不创建重复行。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `conversationId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`SendChatRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientMessageId": "a472c3b4-d76a-4693-9121-62443e6e7c49",
  "content": "收到，我上线后去看看。",
  "replyToMessageId": "message_original",
  "mentionedPlayerRefs": []
}
```

响应 `200`：`V2ChatMessageResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "messageId": "message_reply",
    "sender": {
      "playerRef": "player_mori",
      "gameId": "Mori",
      "qq": null,
      "online": true,
      "registered": true,
      "source": "game_id"
    },
    "content": "收到，我上线后去看看。",
    "kind": "private_chat",
    "sentAt": "2026-09-08T02:00:00Z",
    "conversationId": "conversation_mori_xiwanzi",
    "reply": {
      "messageId": "message_original",
      "sender": {
        "playerRef": "player_mori",
        "gameId": "Mori",
        "qq": null,
        "online": true,
        "registered": true,
        "source": "game_id"
      },
      "content": "花园的灯笼已经摆好了。",
      "availability": "AVAILABLE"
    },
    "clientMessageId": "msg-client-002"
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/chat/conversations/{conversationId}/read` — 同步私聊已读位置

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

只允许把本人已读游标向前推进，不能修改对方游标。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `conversationId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`ConversationReadRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "lastReadMessageId": "ref_demo_001"
}
```

响应 `200`：`V2ConversationReadResultResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "unreadCount": 1,
    "lastReadMessageId": "ref_demo_001"
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/chat/follows` — 获取当前账号关心玩家

实现状态：`existing_v1`。权限：已登录用户。

响应 `200`：`FollowedPlayersResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "players": [
      {
        "playerRef": "example",
        "gameId": "example",
        "qq": null,
        "registered": true,
        "serverOnline": true,
        "appStatus": "online",
        "appConnected": true,
        "appForeground": true,
        "appLastSeenAt": null,
        "onlineSince": null,
        "followed": true,
        "self": true
      }
    ]
  }
}
```

#### `POST /api/v1/chat/follows` — 关心玩家

实现状态：`existing_v1`。权限：已登录用户。

请求模型：`PlayerFollowRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "playerRef": "example"
}
```

响应 `200`：`PlayerFollowResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "followed": true,
    "player": null
  }
}
```

#### `DELETE /api/v1/chat/follows/{playerRef}` — 取消关心玩家

实现状态：`existing_v1`。权限：已登录用户。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `playerRef` | path | 是 | string | ； |

响应 `200`：`PlayerFollowResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "followed": true,
    "player": null
  }
}
```

#### `GET /api/v1/chat/messages` — 获取最近公共聊天

实现状态：`existing_route_extension_pending`。权限：已登录用户。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `limit` | query | 否 | integer | 范围 1–100； |
| `before` | query | 否 | string | ； |

响应 `200`：`ChatMessagesResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "messages": [
      {
        "messageId": "message_reply",
        "sender": {
          "playerRef": "player_mori",
          "gameId": "Mori",
          "qq": null,
          "online": true,
          "registered": true,
          "source": "game_id"
        },
        "content": "收到，我上线后去看看。",
        "kind": "public_chat",
        "sentAt": "2026-09-08T02:00:00Z",
        "conversationId": "public",
        "reply": {
          "messageId": "message_original",
          "sender": {
            "playerRef": "player_mori",
            "gameId": "Mori",
            "qq": null,
            "online": true,
            "registered": true,
            "source": "game_id"
          },
          "content": "花园的灯笼已经摆好了。",
          "availability": "AVAILABLE"
        },
        "clientMessageId": "msg-client-002"
      }
    ]
  },
  "page": {
    "nextCursor": null
  }
}
```

#### `POST /api/v1/chat/messages` — 发送公共聊天或回复

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

HTTP 是待实现的同义入口；与 chat.send WebSocket 共享 clientMessageId 去重和回复校验。正式公共消息仍需插件桥成功转发，不能假装已到达服务器。

请求模型：`SendChatRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientMessageId": "a472c3b4-d76a-4693-9121-62443e6e7c49",
  "content": "收到，我上线后去看看。",
  "replyToMessageId": "message_original",
  "mentionedPlayerRefs": []
}
```

响应 `200`：`V2ChatMessageResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "messageId": "message_reply",
    "sender": {
      "playerRef": "player_mori",
      "gameId": "Mori",
      "qq": null,
      "online": true,
      "registered": true,
      "source": "game_id"
    },
    "content": "收到，我上线后去看看。",
    "kind": "public_chat",
    "sentAt": "2026-09-08T02:00:00Z",
    "conversationId": "public",
    "reply": {
      "messageId": "message_original",
      "sender": {
        "playerRef": "player_mori",
        "gameId": "Mori",
        "qq": null,
        "online": true,
        "registered": true,
        "source": "game_id"
      },
      "content": "花园的灯笼已经摆好了。",
      "availability": "AVAILABLE"
    },
    "clientMessageId": "msg-client-002"
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/chat/online-players` — 获取服务器在线玩家列表

实现状态：`existing_v1`。权限：已登录用户。

响应 `200`：`OnlinePlayersResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "players": [
      {
        "playerRef": "example",
        "gameId": "example",
        "qq": null,
        "registered": true,
        "onlineSince": null
      }
    ]
  }
}
```

#### `GET /api/v1/chat/player-directory` — 获取玩家目录

实现状态：`existing_v1`。权限：已登录用户。

服务器当前在线玩家优先，其后为已注册 App 玩家。

响应 `200`：`PlayerDirectoryResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "players": [
      {
        "playerRef": "example",
        "gameId": "example",
        "qq": null,
        "registered": true,
        "serverOnline": true,
        "appStatus": "online",
        "appConnected": true,
        "appForeground": true,
        "appLastSeenAt": null,
        "onlineSince": null,
        "followed": true,
        "self": true
      }
    ]
  }
}
```

#### `GET /api/v1/chat/presence` — 获取服务器在线概览

实现状态：`existing_v1`。权限：已登录用户。

响应 `200`：`PresenceResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "onlineCount": 1,
    "available": true,
    "updatedAt": null
  }
}
```

### AI 助手

#### `POST /api/v1/ai/chat/stream` — 流式发送 AI 消息

实现状态：`existing_route_extension_pending`。权限：已登录用户。

返回 text/event-stream。事件包括 meta、status、sources、delta、done、error； 等待模型或知识库时后端可发送 SSE comment heartbeat（例如 : keep-alive），客户端必须忽略 comment。 clientMessageId 是 AI 私聊发送幂等键，后端记录 pending/streaming/completed/failed； 重复提交不得重复扣额度，completed 复放既有回复，pending/streaming 提示稍后刷新。 同一用户同一时间只允许一个 pending/streaming 请求。后端会审计 provider 状态码、首个正文 delta 延迟、总耗时、retry 次数和是否已发送 delta。

新增 replyToMessageId；同一 AI 会话内校验，报价/购买仍使用原有 AI purchase 幂等接口。

请求模型：`AiChatStreamRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientMessageId": "example",
  "content": "example",
  "replyToMessageId": "example"
}
```

#### `POST /api/v1/ai/conversation/reset` — 重置当前 AI 会话

实现状态：`existing_v1`。权限：已登录用户。

响应 `200`：`AiConversationResetResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "conversation": {
      "conversationId": "example",
      "active": true,
      "startedAt": "2026-09-08T02:00:00Z",
      "updatedAt": "2026-09-08T02:00:00Z"
    }
  }
}
```

#### `GET /api/v1/ai/me` — 获取当前 AI 状态

实现状态：`existing_v1`。权限：已登录用户。

响应 `200`：`AiMeResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "assistantName": "example",
    "plan": {
      "planId": "example",
      "code": "example",
      "name": "example",
      "description": "example",
      "price": "680.00",
      "currency": "CREDIT",
      "quotaPerWindow": 1,
      "windowHours": 1,
      "durationDays": 1,
      "modelTier": "example",
      "active": true
    },
    "quota": {
      "used": 1,
      "limit": 1,
      "remaining": 1,
      "windowHours": 1,
      "resetsAt": "2026-09-08T02:00:00Z"
    },
    "conversation": {
      "conversationId": "example",
      "active": true,
      "startedAt": "2026-09-08T02:00:00Z",
      "updatedAt": "2026-09-08T02:00:00Z"
    }
  }
}
```

#### `GET /api/v1/ai/messages` — 获取当前 AI 会话消息

实现状态：`existing_v1`。权限：已登录用户。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `limit` | query | 否 | integer | 范围 1–100； |
| `before` | query | 否 | string | ； |

响应 `200`：`AiMessagesResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "messages": [
      {
        "messageId": "example",
        "conversationId": "example",
        "role": "user",
        "content": "example",
        "createdAt": "2026-09-08T02:00:00Z"
      }
    ]
  }
}
```

#### `GET /api/v1/ai/plans` — 获取可购买 AI 套餐

实现状态：`existing_v1`。权限：已登录用户。

响应 `200`：`AiPlansResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "plans": [
      {
        "planId": "example",
        "code": "example",
        "name": "example",
        "description": "example",
        "price": "680.00",
        "currency": "CREDIT",
        "quotaPerWindow": 1,
        "windowHours": 1,
        "durationDays": 1,
        "modelTier": "example",
        "active": true
      }
    ]
  }
}
```

#### `POST /api/v1/ai/purchases` — 购买 AI 套餐

实现状态：`existing_v1`。权限：已登录用户。

请求模型：`AiPurchaseRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "example",
  "planId": "example"
}
```

响应 `200`：`AiPurchaseResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "purchase": {
      "purchaseId": "example",
      "clientRequestId": "example",
      "plan": {
        "planId": "example",
        "code": "example",
        "name": "example",
        "description": "example",
        "price": "680.00",
        "currency": "CREDIT",
        "quotaPerWindow": 1,
        "windowHours": 1,
        "durationDays": 1,
        "modelTier": "example",
        "active": true
      },
      "amount": "680.00",
      "currency": "CREDIT",
      "status": "processing",
      "failureCode": null,
      "createdAt": "2026-09-08T02:00:00Z",
      "updatedAt": "2026-09-08T02:00:00Z"
    },
    "quota": null
  }
}
```

处理中响应 202：查询 operationId；收到终态前不能把动画完成视为已付款。

#### `GET /api/v1/ai/purchases/{purchaseId}` — 查询 AI 套餐购买结果

实现状态：`existing_v1`。权限：已登录用户。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `purchaseId` | path | 是 | string | ； |

响应 `200`：`AiPurchaseResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "purchase": {
      "purchaseId": "example",
      "clientRequestId": "example",
      "plan": {
        "planId": "example",
        "code": "example",
        "name": "example",
        "description": "example",
        "price": "680.00",
        "currency": "CREDIT",
        "quotaPerWindow": 1,
        "windowHours": 1,
        "durationDays": 1,
        "modelTier": "example",
        "active": true
      },
      "amount": "680.00",
      "currency": "CREDIT",
      "status": "processing",
      "failureCode": null,
      "createdAt": "2026-09-08T02:00:00Z",
      "updatedAt": "2026-09-08T02:00:00Z"
    },
    "quota": null
  }
}
```

处理中响应 202：查询 operationId；收到终态前不能把动画完成视为已付款。

### 媒体上传

#### `POST /api/v1/assets/uploads` — 申请客户端直传 OSS 的上传授权

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

只接受 JSON 元信息。验证用途、业务归属、限流/配额后生成私有暂存 key 和受限 POST Policy；二进制由客户端直接发 OSS。同键同正文返回同一会话，不重复创建 assetId；同键不同文件/用途拒绝。见 2.1。

请求模型：`OssUploadCreateRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "upload_request_001",
  "purpose": "AVATAR",
  "businessType": "PROFILE",
  "fileName": "avatar.png",
  "contentType": "image/png",
  "sizeBytes": 1819447,
  "contentMd5": "1B2M2Y8AsgTpgAmY7PhCfg==",
  "altText": "玩家头像"
}
```

响应 `201`：`V2OssUploadSessionResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "uploadId": "upload_demo_001",
    "assetId": "asset_cover_001",
    "purpose": "AVATAR",
    "status": "AUTHORIZED",
    "authorization": {
      "provider": "ALIYUN_OSS",
      "method": "POST",
      "uploadUrl": "https://deuterium-upload.oss-cn-hangzhou.aliyuncs.com/",
      "fileFieldName": "file",
      "formFields": {
        "key": "quarantine/upload_demo_001/random.png",
        "policy": "SYNTHETIC_POLICY_NOT_VALID",
        "x-oss-signature-version": "OSS4-HMAC-SHA256",
        "x-oss-credential": "SYNTHETIC_PUBLIC_CREDENTIAL",
        "x-oss-date": "20260908T020000Z",
        "x-oss-signature": "SYNTHETIC_SIGNATURE_NOT_VALID",
        "x-oss-object-acl": "private",
        "x-oss-forbid-overwrite": "true",
        "x-oss-content-type": "image/png",
        "success_action_status": "201"
      },
      "expiresAt": "2026-09-08T02:05:00Z",
      "sizeBytes": 1819447
    },
    "sessionExpiresAt": "2026-09-08T03:00:00Z",
    "asset": null,
    "rejectionCode": null,
    "retryAfterSeconds": 2
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/assets/uploads/{uploadId}` — 读取本人上传会话和验证结果

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

仅会话本人可见，其他账号 404。用于 OSS 超时后的结果恢复及 VERIFYING 轮询；READY 返回 asset，临时授权字段为 null。private, no-store。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `uploadId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

响应 `200`：`V2OssUploadSessionResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "uploadId": "upload_demo_001",
    "assetId": "asset_cover_001",
    "purpose": "AVATAR",
    "status": "READY",
    "authorization": null,
    "sessionExpiresAt": "2026-09-08T03:00:00Z",
    "asset": {
      "assetId": "asset_cover_001",
      "purpose": "AVATAR",
      "status": "READY",
      "url": "https://cdn.example.invalid/assets/cover.png",
      "width": 1254,
      "height": 1254,
      "sizeBytes": 1819447,
      "sha256": null,
      "altText": "商品封面",
      "createdAt": "2026-09-08T02:00:00Z",
      "contentMd5": "1B2M2Y8AsgTpgAmY7PhCfg==",
      "urlExpiresAt": null
    },
    "rejectionCode": null,
    "retryAfterSeconds": 2
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/assets/uploads/{uploadId}/complete` — 通知后端验证 OSS 对象并确认资产

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

后端读取已登记的 OSS key，冻结到客户端无写权限的最终对象后校验元信息/校验值/实际图片格式和尺寸，通过才写 READY。可返回 VERIFYING 并轮询会话；200 表示会话状态已返回，不等于素材已可发布。重复 complete 同一 uploadId 不产生第二个资产或重复绑定。网络依赖故障维持可重试状态，不直接判为校验失败。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `uploadId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`OssUploadCompleteRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "ossRequestId": "example"
}
```

响应 `200`：`V2OssUploadSessionResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "uploadId": "upload_demo_001",
    "assetId": "asset_cover_001",
    "purpose": "AVATAR",
    "status": "VERIFYING",
    "authorization": null,
    "sessionExpiresAt": "2026-09-08T03:00:00Z",
    "asset": {
      "assetId": "asset_cover_001",
      "purpose": "AVATAR",
      "status": "PROCESSING",
      "url": null,
      "width": null,
      "height": null,
      "sizeBytes": 1819447,
      "sha256": null,
      "altText": "商品封面",
      "createdAt": "2026-09-08T02:00:00Z",
      "contentMd5": null,
      "urlExpiresAt": null
    },
    "rejectionCode": null,
    "retryAfterSeconds": 2
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/assets/uploads/{uploadId}/renew` — 续签未完成上传的 OSS 授权

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

仅本人且仍 AUTHORIZED、会话未到期。沿用同一 key 和文件约束；旧授权无法即时撤销，只能等待 TTL。OSS 已存在对象时先 complete 验证，不能覆盖。READY/VERIFYING 不续签；会话过期返回 UPLOAD_EXPIRED 并重新申请。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `uploadId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`IdempotentRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001"
}
```

响应 `200`：`V2OssUploadSessionResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "uploadId": "upload_demo_001",
    "assetId": "asset_cover_001",
    "purpose": "AVATAR",
    "status": "AUTHORIZED",
    "authorization": {
      "provider": "ALIYUN_OSS",
      "method": "POST",
      "uploadUrl": "https://deuterium-upload.oss-cn-hangzhou.aliyuncs.com/",
      "fileFieldName": "file",
      "formFields": {
        "key": "quarantine/upload_demo_001/random.png",
        "policy": "SYNTHETIC_POLICY_NOT_VALID",
        "x-oss-signature-version": "OSS4-HMAC-SHA256",
        "x-oss-credential": "SYNTHETIC_PUBLIC_CREDENTIAL",
        "x-oss-date": "20260908T020000Z",
        "x-oss-signature": "SYNTHETIC_SIGNATURE_NOT_VALID",
        "x-oss-object-acl": "private",
        "x-oss-forbid-overwrite": "true",
        "x-oss-content-type": "image/png",
        "success_action_status": "201"
      },
      "expiresAt": "2026-09-08T02:05:00Z",
      "sizeBytes": 1819447
    },
    "sessionExpiresAt": "2026-09-08T03:00:00Z",
    "asset": null,
    "rejectionCode": null,
    "retryAfterSeconds": 2
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/assets/{assetId}` — 查询媒体处理结果

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

只返回调用方可见的资产；证据不是公共图片。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `assetId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

响应 `200`：`V2AssetViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "assetId": "asset_cover_001",
    "purpose": "AVATAR",
    "status": "READY",
    "url": "https://cdn.example.invalid/assets/cover.png",
    "width": 1254,
    "height": 1254,
    "sizeBytes": 1819447,
    "sha256": null,
    "altText": "商品封面",
    "createdAt": "2026-09-08T02:00:00Z",
    "contentMd5": null,
    "urlExpiresAt": null
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/assets/{assetId}/remove` — 删除尚未被业务引用的本人媒体

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

成交快照和介入证据已引用的图片不能经此接口删除，返回 ASSET_IN_USE。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `assetId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`IdempotentRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001"
}
```

响应 `200`：`V2RemovalResultResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "removed": true
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

### 官方商城

#### `GET /api/v1/store/brands` — 获取品牌筛选项

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `storeId` | query | 否 | string | 长度 1–128； |
| `cursor` | query | 否 | string | 长度 1–512； |
| `limit` | query | 否 | integer | 范围 1–100； |

响应 `200`：`V2StoreBrandListResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": [
    {
      "brandId": "ref_demo_001",
      "name": "Apple",
      "logoAssetId": null,
      "sortOrder": 1,
      "active": true,
      "version": 1
    }
  ],
  "serverTime": "2026-09-08T02:00:00Z",
  "page": {
    "nextCursor": null
  }
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/store/cart` — 获取购物袋

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

响应 `200`：`V2ShoppingBagResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "version": 1,
    "items": [
      {
        "productId": "ref_demo_001",
        "quantity": 1,
        "productVersion": 1
      }
    ],
    "updatedAt": "2026-09-08T02:00:00Z"
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `PUT /api/v1/store/cart/items/{productId}` — 添加商品或修改数量

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

数量受当前商品限购约束；操作不预留库存、不扣款。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `productId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`CartQuantityRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1,
  "quantity": 1
}
```

响应 `200`：`V2ShoppingBagResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "version": 1,
    "items": [
      {
        "productId": "ref_demo_001",
        "quantity": 1,
        "productVersion": 1
      }
    ],
    "updatedAt": "2026-09-08T02:00:00Z"
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/store/cart/items/{productId}/remove` — 移除购物袋商品

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `productId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`MutationRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1
}
```

响应 `200`：`V2ShoppingBagResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "version": 1,
    "items": [
      {
        "productId": "ref_demo_001",
        "quantity": 1,
        "productVersion": 1
      }
    ],
    "updatedAt": "2026-09-08T02:00:00Z"
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/store/categories` — 获取商城商品分类

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `storeId` | query | 否 | string | 长度 1–128； |
| `cursor` | query | 否 | string | 长度 1–512； |
| `limit` | query | 否 | integer | 范围 1–100； |

响应 `200`：`V2StoreCategoryListResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": [
    {
      "categoryId": "ref_demo_001",
      "name": "Mac",
      "sortOrder": 1,
      "active": true,
      "version": 1
    }
  ],
  "serverTime": "2026-09-08T02:00:00Z",
  "page": {
    "nextCursor": null
  }
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/store/orders` — 按报价创建官方商城订单并付款

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

quote 必须为 OFFICIAL_STORE，钱包扣款与邮箱发放关联同一业务操作。响应丢失用原 clientRequestId 查回，不再付款。

请求模型：`CreateOrderRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "a472c3b4-d76a-4693-9121-62443e6e7c49",
  "quoteId": "quote_store_001",
  "expectedQuoteVersion": 1
}
```

响应 `201`：`V2OrderCreationResultResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "operation": {
      "operationId": "operation_demo_001",
      "clientRequestId": "a472c3b4-d76a-4693-9121-62443e6e7c49",
      "kind": "STORE_PURCHASE",
      "status": "COMPLETED",
      "resourceType": "ORDER",
      "resourceId": "order_store_001",
      "amount": "1399.00",
      "createdAt": "2026-09-08T02:00:00Z",
      "updatedAt": "2026-09-08T02:00:00Z",
      "errorCode": null,
      "retryAfterSeconds": 2
    },
    "order": {
      "orderId": "order_store_001",
      "orderNo": "DT202609080001",
      "channel": "OFFICIAL_STORE",
      "status": "AWAITING_CLAIM",
      "construction": false,
      "buyer": {
        "kind": "PLAYER",
        "playerRef": "player_xiwanzi",
        "storeId": null,
        "displayName": "Xiwanzi",
        "contactQq": "1000202"
      },
      "seller": {
        "kind": "OFFICIAL_STORE",
        "playerRef": null,
        "storeId": "store_official",
        "displayName": "Deuterium 官方商城",
        "contactQq": "1000000"
      },
      "items": [
        {
          "productId": "product_macmini",
          "productVersion": 1,
          "title": "Mac mini",
          "subtitle": "小巧机身，为桌面留出更多空间。",
          "description": "用于商城页面和下单流程测试的示例商品。",
          "unitPrice": "1399.00",
          "quantity": 1,
          "photoAssetIds": [
            "asset_cover_001",
            "asset_detail_002"
          ],
          "categoryName": "Mac",
          "includedItems": [
            "Mac mini 主题物品 × 1",
            "产品说明 × 1"
          ],
          "contentBlocks": [
            {
              "blockId": "intro",
              "type": "PARAGRAPH",
              "text": "了解商品内容和游戏邮箱交付说明。"
            }
          ]
        }
      ],
      "amount": "1399.00",
      "currency": "CREDIT",
      "delivery": {
        "method": "MAILBOX",
        "location": "Xiwanzi",
        "projectName": ""
      },
      "snapshotId": "snapshot_order_001",
      "snapshotSha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      "fundsStatus": "PAID",
      "confirmationHours": 168,
      "serverNow": "2026-09-08T02:00:00Z",
      "autoConfirmAt": null,
      "pausedRemainingSeconds": null,
      "refund": null,
      "refundAttemptsUsed": 0,
      "availableActions": [
        "VIEW_MAILBOX",
        "REQUEST_REFUND"
      ],
      "createdAt": "2026-09-07T01:00:00Z",
      "shippedAt": "2026-09-08T02:00:00Z",
      "workCompletedAt": null,
      "confirmedAt": null,
      "automatic": false,
      "version": 1
    }
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

处理中响应 202：查询 operationId；收到终态前不能把动画完成视为已付款。

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/store/products` — 浏览和搜索已发布商品

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

仅返回 ACTIVE 已发布版本；编辑中的草稿不可见。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `storeId` | query | 否 | string | 长度 1–128； |
| `brandId` | query | 否 | string | 长度 1–128； |
| `categoryId` | query | 否 | string | 长度 1–128； |
| `q` | query | 否 | string | 长度 1–80； |
| `cursor` | query | 否 | string | 长度 1–512； |
| `limit` | query | 否 | integer | 范围 1–100； |

响应 `200`：`V2StoreProductListResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": [
    {
      "productId": "product_macmini",
      "storeId": "store_official",
      "version": 1,
      "content": {
        "title": "Mac mini",
        "subtitle": "小巧机身，为桌面留出更多空间。",
        "description": "用于商城页面和下单流程测试的示例商品。",
        "brandId": "brand_apple",
        "categoryId": "category_mac",
        "price": "1399.00",
        "coverAssetId": "asset_cover_001",
        "galleryAssetIds": [
          "asset_cover_001",
          "asset_detail_002"
        ],
        "galleryAltTexts": [
          "Mac mini 正面",
          "Mac mini 背面接口"
        ],
        "contentBlocks": [
          {
            "blockId": "intro",
            "type": "PARAGRAPH",
            "text": "了解商品内容和游戏邮箱交付说明。"
          }
        ],
        "includedItems": [
          "Mac mini 主题物品 × 1",
          "产品说明 × 1"
        ],
        "deliveryTemplateRef": "delivery_template_macmini",
        "deliverySummary": "发送至本人游戏内邮箱",
        "estimatedDelivery": "支付完成后 1 分钟内",
        "inventoryPolicy": "FINITE",
        "stock": 50,
        "limitPerOrder": 9,
        "posterTone": "LIGHT",
        "accentColor": "#BBD6EC",
        "badges": [
          "新品"
        ],
        "sortOrder": 10
      },
      "images": [
        {
          "assetId": "asset_cover_001",
          "purpose": "STORE_MEDIA",
          "status": "READY",
          "url": "https://cdn.example.invalid/assets/cover.png",
          "width": 1254,
          "height": 1254,
          "sizeBytes": 1819447,
          "sha256": null,
          "altText": "商品封面",
          "createdAt": "2026-09-08T02:00:00Z",
          "contentMd5": null,
          "urlExpiresAt": null
        },
        {
          "assetId": "asset_detail_002",
          "purpose": "STORE_MEDIA",
          "status": "READY",
          "url": "https://cdn.example.invalid/assets/detail.png",
          "width": 1254,
          "height": 1254,
          "sizeBytes": 1819447,
          "sha256": null,
          "altText": "商品封面",
          "createdAt": "2026-09-08T02:00:00Z",
          "contentMd5": null,
          "urlExpiresAt": null
        }
      ],
      "visibility": "ACTIVE",
      "availableStock": 50,
      "publishedAt": "2026-09-08T02:00:00Z",
      "updatedAt": "2026-09-08T02:00:00Z"
    }
  ],
  "serverTime": "2026-09-08T02:00:00Z",
  "page": {
    "nextCursor": null
  }
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/store/products/{productId}` — 获取商品所有展示信息

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

包含详情块、包含内容、图片、海报色调、价格、库存、限购与送达说明；不返回发放命令。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `productId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

响应 `200`：`V2StoreProductResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "productId": "product_macmini",
    "storeId": "store_official",
    "version": 1,
    "content": {
      "title": "Mac mini",
      "subtitle": "小巧机身，为桌面留出更多空间。",
      "description": "用于商城页面和下单流程测试的示例商品。",
      "brandId": "brand_apple",
      "categoryId": "category_mac",
      "price": "1399.00",
      "coverAssetId": "asset_cover_001",
      "galleryAssetIds": [
        "asset_cover_001",
        "asset_detail_002"
      ],
      "galleryAltTexts": [
        "Mac mini 正面",
        "Mac mini 背面接口"
      ],
      "contentBlocks": [
        {
          "blockId": "intro",
          "type": "PARAGRAPH",
          "text": "了解商品内容和游戏邮箱交付说明。"
        }
      ],
      "includedItems": [
        "Mac mini 主题物品 × 1",
        "产品说明 × 1"
      ],
      "deliveryTemplateRef": "delivery_template_macmini",
      "deliverySummary": "发送至本人游戏内邮箱",
      "estimatedDelivery": "支付完成后 1 分钟内",
      "inventoryPolicy": "FINITE",
      "stock": 50,
      "limitPerOrder": 9,
      "posterTone": "LIGHT",
      "accentColor": "#BBD6EC",
      "badges": [
        "新品"
      ],
      "sortOrder": 10
    },
    "images": [
      {
        "assetId": "asset_cover_001",
        "purpose": "STORE_MEDIA",
        "status": "READY",
        "url": "https://cdn.example.invalid/assets/cover.png",
        "width": 1254,
        "height": 1254,
        "sizeBytes": 1819447,
        "sha256": null,
        "altText": "商品封面",
        "createdAt": "2026-09-08T02:00:00Z",
        "contentMd5": null,
        "urlExpiresAt": null
      },
      {
        "assetId": "asset_detail_002",
        "purpose": "STORE_MEDIA",
        "status": "READY",
        "url": "https://cdn.example.invalid/assets/detail.png",
        "width": 1254,
        "height": 1254,
        "sizeBytes": 1819447,
        "sha256": null,
        "altText": "商品封面",
        "createdAt": "2026-09-08T02:00:00Z",
        "contentMd5": null,
        "urlExpiresAt": null
      }
    ],
    "visibility": "ACTIVE",
    "availableStock": 50,
    "publishedAt": "2026-09-08T02:00:00Z",
    "updatedAt": "2026-09-08T02:00:00Z"
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/store/stores` — 列出受控官方店铺

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

范围限 Deuterium 官方批准的店铺，不开放玩家自行注册商家。品牌分类不等于登录身份。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `cursor` | query | 否 | string | 长度 1–512； |
| `limit` | query | 否 | integer | 范围 1–100； |

响应 `200`：`V2StoreViewListResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": [
    {
      "storeId": "ref_demo_001",
      "name": "Deuterium 官方商城",
      "intro": "example",
      "logo": null,
      "cover": null,
      "contactQq": "1000000",
      "serviceHours": "example",
      "notice": "example",
      "version": 1
    }
  ],
  "serverTime": "2026-09-08T02:00:00Z",
  "page": {
    "nextCursor": null
  }
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/store/stores/{storeId}` — 获取官方店铺信息

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `storeId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

响应 `200`：`V2StoreViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "storeId": "ref_demo_001",
    "name": "Deuterium 官方商城",
    "intro": "example",
    "logo": null,
    "cover": null,
    "contactQq": "1000000",
    "serviceHours": "example",
    "notice": "example",
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/store/stores/{storeId}/homepage` — 获取商城首页区域和排序

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `storeId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

响应 `200`：`V2StoreHomepageResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "storeId": "ref_demo_001",
    "intro": "example",
    "sections": [
      {
        "sectionId": "ref_demo_001",
        "title": "example",
        "layout": "HERO_CAROUSEL",
        "productIds": [
          "ref_demo_001"
        ],
        "bannerAssetId": null,
        "sortOrder": 1
      }
    ],
    "brandIds": [
      "ref_demo_001"
    ],
    "categoryIds": [
      "ref_demo_001"
    ],
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

### 玩家市场

#### `GET /api/v1/market/categories` — 获取六个市场分类

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

仅建材、装备、补给、装饰、建筑服务、其他；再次点选当前分类是客户端清除 categoryCode，不新增反选 API。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `cursor` | query | 否 | string | 长度 1–512； |
| `limit` | query | 否 | integer | 范围 1–100； |

响应 `200`：`V2MarketCategoryListResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": [
    {
      "code": "MATERIALS",
      "name": "example",
      "description": "example",
      "sortOrder": 1
    }
  ],
  "serverTime": "2026-09-08T02:00:00Z",
  "page": {
    "nextCursor": null
  }
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/market/listings` — 浏览或搜索市场商品

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

不传分类即全部；仅展示 active=true。自身管理列表使用 /market/me/listings。

强制 active=true AND stock>0；stock 为扣除订单预留后的剩余可售数。售罄/下架移除；库存 1 被购买后隐藏，多库存剩余部分继续售卖。商品响应不包含已生成订单和买家履约数据；查询/搜索/翻页一致，private, no-store。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `categoryCode` | query | 否 | string | `MATERIALS` / `EQUIPMENT` / `SUPPLIES` / `DECORATION` / `CONSTRUCTION` / `OTHER`； |
| `q` | query | 否 | string | 长度 1–80； |
| `sellerPlayerRef` | query | 否 | string | 长度 1–128； |
| `cursor` | query | 否 | string | 长度 1–512； |
| `limit` | query | 否 | integer | 范围 1–100； |

响应 `200`：`V2MarketListingListResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": [
    {
      "title": "湖畔木屋代建",
      "subtitle": "设计、主体与基础内装",
      "description": "按约定图纸完成主体与内装，请在开工前确认材料和尺寸。",
      "categoryCode": "CONSTRUCTION",
      "price": "1800.00",
      "stock": 3,
      "photoAssetIds": [
        "asset_cover_001"
      ],
      "contactQq": "1000103",
      "deliveryMethods": [
        "WORKSITE"
      ],
      "pickupLocation": "",
      "workHours": 168,
      "listingId": "listing_house",
      "seller": {
        "playerRef": "player_aster",
        "gameId": "Aster",
        "qq": "1000103",
        "bio": "把每一次日落留在小花园里。",
        "avatar": {
          "assetId": "asset_cover_001",
          "purpose": "AVATAR",
          "status": "READY",
          "url": "https://cdn.example.invalid/assets/cover.png",
          "width": 1254,
          "height": 1254,
          "sizeBytes": 1819447,
          "sha256": null,
          "altText": "商品封面",
          "createdAt": "2026-09-08T02:00:00Z",
          "contentMd5": null,
          "urlExpiresAt": null
        },
        "online": true,
        "lastSeenAt": "2026-09-08T02:00:00Z",
        "followed": false,
        "version": 1
      },
      "photos": [
        {
          "assetId": "asset_cover_001",
          "purpose": "MARKET_PHOTO",
          "status": "READY",
          "url": "https://cdn.example.invalid/assets/cover.png",
          "width": 1254,
          "height": 1254,
          "sizeBytes": 1819447,
          "sha256": null,
          "altText": "商品封面",
          "createdAt": "2026-09-08T02:00:00Z",
          "contentMd5": null,
          "urlExpiresAt": null
        }
      ],
      "active": true,
      "version": 1,
      "createdAt": "2026-09-08T02:00:00Z",
      "updatedAt": "2026-09-08T02:00:00Z"
    }
  ],
  "serverTime": "2026-09-08T02:00:00Z",
  "page": {
    "nextCursor": null
  }
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/market/listings` — 发布带图片的商品或建筑服务

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

库存 1–999，1–5 张图片；首图封面。建筑服务 workHours 含验收预留，交付方式 WORKSITE；本人身份由会话决定。

请求模型：`ListingCreateRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "content": {
    "title": "湖畔木屋代建",
    "subtitle": "设计、主体与基础内装",
    "description": "按约定图纸完成主体与内装，请在开工前确认材料和尺寸。",
    "categoryCode": "CONSTRUCTION",
    "price": "1800.00",
    "stock": 3,
    "photoAssetIds": [
      "asset_cover_001"
    ],
    "contactQq": "1000103",
    "deliveryMethods": [
      "WORKSITE"
    ],
    "pickupLocation": "",
    "workHours": 168
  }
}
```

响应 `201`：`V2MarketListingResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "title": "湖畔木屋代建",
    "subtitle": "设计、主体与基础内装",
    "description": "按约定图纸完成主体与内装，请在开工前确认材料和尺寸。",
    "categoryCode": "CONSTRUCTION",
    "price": "1800.00",
    "stock": 3,
    "photoAssetIds": [
      "asset_cover_001"
    ],
    "contactQq": "1000103",
    "deliveryMethods": [
      "WORKSITE"
    ],
    "pickupLocation": "",
    "workHours": 168,
    "listingId": "listing_house",
    "seller": {
      "playerRef": "player_aster",
      "gameId": "Aster",
      "qq": "1000103",
      "bio": "把每一次日落留在小花园里。",
      "avatar": {
        "assetId": "asset_cover_001",
        "purpose": "AVATAR",
        "status": "READY",
        "url": "https://cdn.example.invalid/assets/cover.png",
        "width": 1254,
        "height": 1254,
        "sizeBytes": 1819447,
        "sha256": null,
        "altText": "商品封面",
        "createdAt": "2026-09-08T02:00:00Z",
        "contentMd5": null,
        "urlExpiresAt": null
      },
      "online": true,
      "lastSeenAt": "2026-09-08T02:00:00Z",
      "followed": false,
      "version": 1
    },
    "photos": [
      {
        "assetId": "asset_cover_001",
        "purpose": "MARKET_PHOTO",
        "status": "READY",
        "url": "https://cdn.example.invalid/assets/cover.png",
        "width": 1254,
        "height": 1254,
        "sizeBytes": 1819447,
        "sha256": null,
        "altText": "商品封面",
        "createdAt": "2026-09-08T02:00:00Z",
        "contentMd5": null,
        "urlExpiresAt": null
      }
    ],
    "active": true,
    "version": 1,
    "createdAt": "2026-09-08T02:00:00Z",
    "updatedAt": "2026-09-08T02:00:00Z"
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/market/listings/{listingId}` — 获取市场商品详情

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

普通玩家仅可读取 active=true 且 stock>0 的商品，否则 404。卖家及授权管理员可读取隐藏记录；买家查看成交内容使用自己的订单和 snapshot，不获得已下架商品的持续公开读取权。private, no-store。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `listingId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

响应 `200`：`V2MarketListingResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "title": "湖畔木屋代建",
    "subtitle": "设计、主体与基础内装",
    "description": "按约定图纸完成主体与内装，请在开工前确认材料和尺寸。",
    "categoryCode": "CONSTRUCTION",
    "price": "1800.00",
    "stock": 3,
    "photoAssetIds": [
      "asset_cover_001"
    ],
    "contactQq": "1000103",
    "deliveryMethods": [
      "WORKSITE"
    ],
    "pickupLocation": "",
    "workHours": 168,
    "listingId": "listing_house",
    "seller": {
      "playerRef": "player_aster",
      "gameId": "Aster",
      "qq": "1000103",
      "bio": "把每一次日落留在小花园里。",
      "avatar": {
        "assetId": "asset_cover_001",
        "purpose": "AVATAR",
        "status": "READY",
        "url": "https://cdn.example.invalid/assets/cover.png",
        "width": 1254,
        "height": 1254,
        "sizeBytes": 1819447,
        "sha256": null,
        "altText": "商品封面",
        "createdAt": "2026-09-08T02:00:00Z",
        "contentMd5": null,
        "urlExpiresAt": null
      },
      "online": true,
      "lastSeenAt": "2026-09-08T02:00:00Z",
      "followed": false,
      "version": 1
    },
    "photos": [
      {
        "assetId": "asset_cover_001",
        "purpose": "MARKET_PHOTO",
        "status": "READY",
        "url": "https://cdn.example.invalid/assets/cover.png",
        "width": 1254,
        "height": 1254,
        "sizeBytes": 1819447,
        "sha256": null,
        "altText": "商品封面",
        "createdAt": "2026-09-08T02:00:00Z",
        "contentMd5": null,
        "urlExpiresAt": null
      }
    ],
    "active": true,
    "version": 1,
    "createdAt": "2026-09-08T02:00:00Z",
    "updatedAt": "2026-09-08T02:00:00Z"
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `PUT /api/v1/market/listings/{listingId}` — 保存下架商品的编辑内容

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

只允许本人修改；建议先下架再编辑并重新发布。历史订单快照不跟随变化；库存调整不能占用未履约订单的已预留数量。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `listingId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`ListingEditRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1,
  "content": {
    "title": "湖畔木屋代建",
    "subtitle": "设计、主体与基础内装",
    "description": "按约定图纸完成主体与内装，请在开工前确认材料和尺寸。",
    "categoryCode": "CONSTRUCTION",
    "price": "1800.00",
    "stock": 3,
    "photoAssetIds": [
      "asset_cover_001"
    ],
    "contactQq": "1000103",
    "deliveryMethods": [
      "WORKSITE"
    ],
    "pickupLocation": "",
    "workHours": 168
  }
}
```

响应 `200`：`V2MarketListingResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "title": "湖畔木屋代建",
    "subtitle": "设计、主体与基础内装",
    "description": "按约定图纸完成主体与内装，请在开工前确认材料和尺寸。",
    "categoryCode": "CONSTRUCTION",
    "price": "1800.00",
    "stock": 3,
    "photoAssetIds": [
      "asset_cover_001"
    ],
    "contactQq": "1000103",
    "deliveryMethods": [
      "WORKSITE"
    ],
    "pickupLocation": "",
    "workHours": 168,
    "listingId": "listing_house",
    "seller": {
      "playerRef": "player_aster",
      "gameId": "Aster",
      "qq": "1000103",
      "bio": "把每一次日落留在小花园里。",
      "avatar": {
        "assetId": "asset_cover_001",
        "purpose": "AVATAR",
        "status": "READY",
        "url": "https://cdn.example.invalid/assets/cover.png",
        "width": 1254,
        "height": 1254,
        "sizeBytes": 1819447,
        "sha256": null,
        "altText": "商品封面",
        "createdAt": "2026-09-08T02:00:00Z",
        "contentMd5": null,
        "urlExpiresAt": null
      },
      "online": true,
      "lastSeenAt": "2026-09-08T02:00:00Z",
      "followed": false,
      "version": 1
    },
    "photos": [
      {
        "assetId": "asset_cover_001",
        "purpose": "MARKET_PHOTO",
        "status": "READY",
        "url": "https://cdn.example.invalid/assets/cover.png",
        "width": 1254,
        "height": 1254,
        "sizeBytes": 1819447,
        "sha256": null,
        "altText": "商品封面",
        "createdAt": "2026-09-08T02:00:00Z",
        "contentMd5": null,
        "urlExpiresAt": null
      }
    ],
    "active": true,
    "version": 1,
    "createdAt": "2026-09-08T02:00:00Z",
    "updatedAt": "2026-09-08T02:00:00Z"
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/market/listings/{listingId}/republish` — 重新校验完整表单并上架

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

UI 先读取原商品填入发布表单，提交新的完整 content、版本与幂等键。不得直接翻转 active 绕过校验。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `listingId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`ListingEditRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1,
  "content": {
    "title": "湖畔木屋代建",
    "subtitle": "设计、主体与基础内装",
    "description": "按约定图纸完成主体与内装，请在开工前确认材料和尺寸。",
    "categoryCode": "CONSTRUCTION",
    "price": "1800.00",
    "stock": 3,
    "photoAssetIds": [
      "asset_cover_001"
    ],
    "contactQq": "1000103",
    "deliveryMethods": [
      "WORKSITE"
    ],
    "pickupLocation": "",
    "workHours": 168
  }
}
```

响应 `200`：`V2MarketListingResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "title": "湖畔木屋代建",
    "subtitle": "设计、主体与基础内装",
    "description": "按约定图纸完成主体与内装，请在开工前确认材料和尺寸。",
    "categoryCode": "CONSTRUCTION",
    "price": "1800.00",
    "stock": 3,
    "photoAssetIds": [
      "asset_cover_001"
    ],
    "contactQq": "1000103",
    "deliveryMethods": [
      "WORKSITE"
    ],
    "pickupLocation": "",
    "workHours": 168,
    "listingId": "listing_house",
    "seller": {
      "playerRef": "player_aster",
      "gameId": "Aster",
      "qq": "1000103",
      "bio": "把每一次日落留在小花园里。",
      "avatar": {
        "assetId": "asset_cover_001",
        "purpose": "AVATAR",
        "status": "READY",
        "url": "https://cdn.example.invalid/assets/cover.png",
        "width": 1254,
        "height": 1254,
        "sizeBytes": 1819447,
        "sha256": null,
        "altText": "商品封面",
        "createdAt": "2026-09-08T02:00:00Z",
        "contentMd5": null,
        "urlExpiresAt": null
      },
      "online": true,
      "lastSeenAt": "2026-09-08T02:00:00Z",
      "followed": false,
      "version": 1
    },
    "photos": [
      {
        "assetId": "asset_cover_001",
        "purpose": "MARKET_PHOTO",
        "status": "READY",
        "url": "https://cdn.example.invalid/assets/cover.png",
        "width": 1254,
        "height": 1254,
        "sizeBytes": 1819447,
        "sha256": null,
        "altText": "商品封面",
        "createdAt": "2026-09-08T02:00:00Z",
        "contentMd5": null,
        "urlExpiresAt": null
      }
    ],
    "active": true,
    "version": 1,
    "createdAt": "2026-09-08T02:00:00Z",
    "updatedAt": "2026-09-08T02:00:00Z"
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/market/listings/{listingId}/unlist` — 确认下架商品

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

客户端先二次确认；只阻止新购买，不取消已有订单。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `listingId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`MutationRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1
}
```

响应 `200`：`V2MarketListingResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "title": "湖畔木屋代建",
    "subtitle": "设计、主体与基础内装",
    "description": "按约定图纸完成主体与内装，请在开工前确认材料和尺寸。",
    "categoryCode": "CONSTRUCTION",
    "price": "1800.00",
    "stock": 3,
    "photoAssetIds": [
      "asset_cover_001"
    ],
    "contactQq": "1000103",
    "deliveryMethods": [
      "WORKSITE"
    ],
    "pickupLocation": "",
    "workHours": 168,
    "listingId": "listing_house",
    "seller": {
      "playerRef": "player_aster",
      "gameId": "Aster",
      "qq": "1000103",
      "bio": "把每一次日落留在小花园里。",
      "avatar": {
        "assetId": "asset_cover_001",
        "purpose": "AVATAR",
        "status": "READY",
        "url": "https://cdn.example.invalid/assets/cover.png",
        "width": 1254,
        "height": 1254,
        "sizeBytes": 1819447,
        "sha256": null,
        "altText": "商品封面",
        "createdAt": "2026-09-08T02:00:00Z",
        "contentMd5": null,
        "urlExpiresAt": null
      },
      "online": true,
      "lastSeenAt": "2026-09-08T02:00:00Z",
      "followed": false,
      "version": 1
    },
    "photos": [
      {
        "assetId": "asset_cover_001",
        "purpose": "MARKET_PHOTO",
        "status": "READY",
        "url": "https://cdn.example.invalid/assets/cover.png",
        "width": 1254,
        "height": 1254,
        "sizeBytes": 1819447,
        "sha256": null,
        "altText": "商品封面",
        "createdAt": "2026-09-08T02:00:00Z",
        "contentMd5": null,
        "urlExpiresAt": null
      }
    ],
    "active": true,
    "version": 1,
    "createdAt": "2026-09-08T02:00:00Z",
    "updatedAt": "2026-09-08T02:00:00Z"
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/market/me/listings` — 查看我发布的商品

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

只返回会话本人发布的商品，包含售罄和下架；不接受查询他人管理列表。private, no-store。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `active` | query | 否 | boolean | ； |
| `cursor` | query | 否 | string | 长度 1–512； |
| `limit` | query | 否 | integer | 范围 1–100； |

响应 `200`：`V2MarketListingListResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": [
    {
      "title": "湖畔木屋代建",
      "subtitle": "设计、主体与基础内装",
      "description": "按约定图纸完成主体与内装，请在开工前确认材料和尺寸。",
      "categoryCode": "CONSTRUCTION",
      "price": "1800.00",
      "stock": 3,
      "photoAssetIds": [
        "asset_cover_001"
      ],
      "contactQq": "1000103",
      "deliveryMethods": [
        "WORKSITE"
      ],
      "pickupLocation": "",
      "workHours": 168,
      "listingId": "listing_house",
      "seller": {
        "playerRef": "player_aster",
        "gameId": "Aster",
        "qq": "1000103",
        "bio": "把每一次日落留在小花园里。",
        "avatar": {
          "assetId": "asset_cover_001",
          "purpose": "AVATAR",
          "status": "READY",
          "url": "https://cdn.example.invalid/assets/cover.png",
          "width": 1254,
          "height": 1254,
          "sizeBytes": 1819447,
          "sha256": null,
          "altText": "商品封面",
          "createdAt": "2026-09-08T02:00:00Z",
          "contentMd5": null,
          "urlExpiresAt": null
        },
        "online": true,
        "lastSeenAt": "2026-09-08T02:00:00Z",
        "followed": false,
        "version": 1
      },
      "photos": [
        {
          "assetId": "asset_cover_001",
          "purpose": "MARKET_PHOTO",
          "status": "READY",
          "url": "https://cdn.example.invalid/assets/cover.png",
          "width": 1254,
          "height": 1254,
          "sizeBytes": 1819447,
          "sha256": null,
          "altText": "商品封面",
          "createdAt": "2026-09-08T02:00:00Z",
          "contentMd5": null,
          "urlExpiresAt": null
        }
      ],
      "active": true,
      "version": 1,
      "createdAt": "2026-09-08T02:00:00Z",
      "updatedAt": "2026-09-08T02:00:00Z"
    }
  ],
  "serverTime": "2026-09-08T02:00:00Z",
  "page": {
    "nextCursor": null
  }
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/market/orders` — 按报价购买玩家商品并冻结货款

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

quote 必须为 PLAYER_MARKET；原子检查版本/库存并预付，不能购买本人商品。建筑服务项目名和工程地点必填。

生成的订单从创建起仅买卖双方及授权管理员可见；原子预留库存，最后一件被预留后移出公开市场，未知支付结果继续保留预留量。

请求模型：`CreateOrderRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "a472c3b4-d76a-4693-9121-62443e6e7c49",
  "quoteId": "quote_demo_001",
  "expectedQuoteVersion": 1
}
```

响应 `201`：`V2OrderCreationResultResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "operation": {
      "operationId": "operation_demo_001",
      "clientRequestId": "a472c3b4-d76a-4693-9121-62443e6e7c49",
      "kind": "MARKET_PURCHASE",
      "status": "COMPLETED",
      "resourceType": "ORDER",
      "resourceId": "order_demo_001",
      "amount": "1800.00",
      "createdAt": "2026-09-08T02:00:00Z",
      "updatedAt": "2026-09-08T02:00:00Z",
      "errorCode": null,
      "retryAfterSeconds": 2
    },
    "order": {
      "orderId": "order_demo_001",
      "orderNo": "DT202609080001",
      "channel": "PLAYER_MARKET",
      "status": "AWAITING_SHIPMENT",
      "construction": true,
      "buyer": {
        "kind": "PLAYER",
        "playerRef": "player_xiwanzi",
        "storeId": null,
        "displayName": "Xiwanzi",
        "contactQq": "1000202"
      },
      "seller": {
        "kind": "PLAYER",
        "playerRef": "player_aster",
        "storeId": null,
        "displayName": "Aster",
        "contactQq": "1000103"
      },
      "items": [
        {
          "productId": "listing_house",
          "productVersion": 1,
          "title": "湖畔木屋代建",
          "subtitle": "设计、主体与基础内装",
          "description": "按约定图纸完成主体与内装，请在开工前确认材料和尺寸。",
          "unitPrice": "1800.00",
          "quantity": 1,
          "photoAssetIds": [
            "asset_cover_001"
          ],
          "categoryName": "建筑服务",
          "includedItems": [],
          "contentBlocks": []
        }
      ],
      "amount": "1800.00",
      "currency": "CREDIT",
      "delivery": {
        "method": "WORKSITE",
        "location": "主世界 x=120 y=70 z=-80",
        "projectName": "湖畔木屋"
      },
      "snapshotId": "snapshot_order_001",
      "snapshotSha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      "fundsStatus": "HELD",
      "confirmationHours": 168,
      "serverNow": "2026-09-08T02:00:00Z",
      "autoConfirmAt": null,
      "pausedRemainingSeconds": null,
      "refund": null,
      "refundAttemptsUsed": 0,
      "availableActions": [
        "REQUEST_REFUND"
      ],
      "createdAt": "2026-09-08T02:00:00Z",
      "shippedAt": null,
      "workCompletedAt": null,
      "confirmedAt": null,
      "automatic": false,
      "version": 1
    }
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

处理中响应 202：查询 operationId；收到终态前不能把动画完成视为已付款。

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

### 订单与退款

#### `POST /api/v1/checkout/quotes` — 生成服务端结算报价

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

校验商品版本、当前可售库存、本人交付方式；返回最终金额与有效期。报价不扣款，不允许客户端传价格。跨官方店铺或多个玩家卖家的购物项必须拆成独立订单。

请求模型：`QuoteRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "channel": "PLAYER_MARKET",
  "items": [
    {
      "productId": "listing_house",
      "quantity": 1,
      "expectedProductVersion": 1
    }
  ],
  "delivery": {
    "method": "WORKSITE",
    "location": "主世界 x=120 y=70 z=-80",
    "projectName": "湖畔木屋"
  }
}
```

响应 `201`：`V2CheckoutQuoteResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "quoteId": "quote_demo_001",
    "channel": "PLAYER_MARKET",
    "items": [
      {
        "productId": "listing_house",
        "title": "湖畔木屋代建",
        "unitPrice": "1800.00",
        "quantity": 1,
        "subtotal": "1800.00",
        "productVersion": 1
      }
    ],
    "totalAmount": "1800.00",
    "currency": "CREDIT",
    "expiresAt": "2026-09-08T02:05:00Z",
    "delivery": {
      "method": "WORKSITE",
      "location": "主世界 x=120 y=70 z=-80",
      "projectName": "湖畔木屋"
    },
    "version": 1,
    "warnings": [
      "工期包含验收预留，从卖家开始施工起计时。"
    ]
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/interventions/{caseId}` — 查询本人参与的介入申请

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `caseId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

响应 `200`：`V2InterventionViewResponse`。

完整 JSON 示例见 `api-v2-examples.json` 中同 method/path 条目；模型字段全部在下文展开。

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/interventions/{caseId}/evidence` — 补充本人介入说明和证据

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

只能追加，不能改写已提交的证据与系统交易快照；不能引用第三方不可见的聊天消息。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `caseId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`InterventionEvidenceRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1,
  "description": "example",
  "evidenceAssetIds": [
    "ref_demo_001"
  ],
  "relatedMessageIds": [
    "ref_demo_001"
  ]
}
```

响应 `200`：`V2InterventionViewResponse`。

完整 JSON 示例见 `api-v2-examples.json` 中同 method/path 条目；模型字段全部在下文展开。

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/interventions/{caseId}/withdraw` — 撤回尚未裁决的介入申请

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

只允许申请人；资金是否恢复倒计时取决于当前阶段和剩余时间，不能重置为完整期限。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `caseId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`ReasonedMutationRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1,
  "reason": "已按双方确认的方案完成，请核对交付内容。"
}
```

响应 `200`：`V2InterventionViewResponse`。

完整 JSON 示例见 `api-v2-examples.json` 中同 method/path 条目；模型字段全部在下文展开。

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/orders` — 获取我的订单

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

仅返回调用方参与的订单；官方商家履约使用独立权限的 merchant/orders。

role 仅筛选本人 BUYER/SELLER；省略为本人关系并集。市场订单从创建起永不公开，终态仍可查历史；private, no-store。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `channel` | query | 否 | string | `OFFICIAL_STORE` / `PLAYER_MARKET`； |
| `role` | query | 否 | string | `BUYER` / `SELLER`； |
| `status` | query | 否 | string | `PAYMENT_PROCESSING` / `AWAITING_SHIPMENT` / `SHIPPED` / `WORK_COMPLETED` / `CONFIRMED` / `AWAITING_CLAIM` / `CLAIMED` / `REFUNDED` / `CANCELLED`； |
| `hasRefund` | query | 否 | boolean | ； |
| `cursor` | query | 否 | string | 长度 1–512； |
| `limit` | query | 否 | integer | 范围 1–100； |

响应 `200`：`V2OrderViewListResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": [
    {
      "orderId": "order_demo_001",
      "orderNo": "DT202609080001",
      "channel": "PLAYER_MARKET",
      "status": "WORK_COMPLETED",
      "construction": true,
      "buyer": {
        "kind": "PLAYER",
        "playerRef": "player_xiwanzi",
        "storeId": null,
        "displayName": "Xiwanzi",
        "contactQq": "1000202"
      },
      "seller": {
        "kind": "PLAYER",
        "playerRef": "player_aster",
        "storeId": null,
        "displayName": "Aster",
        "contactQq": "1000103"
      },
      "items": [
        {
          "productId": "listing_house",
          "productVersion": 1,
          "title": "湖畔木屋代建",
          "subtitle": "设计、主体与基础内装",
          "description": "按约定图纸完成主体与内装，请在开工前确认材料和尺寸。",
          "unitPrice": "1800.00",
          "quantity": 1,
          "photoAssetIds": [
            "asset_cover_001"
          ],
          "categoryName": "建筑服务",
          "includedItems": [],
          "contentBlocks": []
        }
      ],
      "amount": "1800.00",
      "currency": "CREDIT",
      "delivery": {
        "method": "WORKSITE",
        "location": "主世界 x=120 y=70 z=-80",
        "projectName": "湖畔木屋"
      },
      "snapshotId": "snapshot_order_001",
      "snapshotSha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      "fundsStatus": "HELD",
      "confirmationHours": 168,
      "serverNow": "2026-09-08T02:00:00Z",
      "autoConfirmAt": "2026-09-14T02:00:00Z",
      "pausedRemainingSeconds": null,
      "refund": null,
      "refundAttemptsUsed": 0,
      "availableActions": [
        "CONFIRM_ACCEPTANCE",
        "REQUEST_REFUND"
      ],
      "createdAt": "2026-09-07T01:00:00Z",
      "shippedAt": "2026-09-07T02:00:00Z",
      "workCompletedAt": "2026-09-08T02:00:00Z",
      "confirmedAt": null,
      "automatic": false,
      "version": 1
    }
  ],
  "serverTime": "2026-09-08T02:00:00Z",
  "page": {
    "nextCursor": null
  }
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/orders/{orderId}` — 获取订单详情和可用动作

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

serverNow、autoConfirmAt 与暂停剩余时间用于倒计时。前端本地归零只刷新状态，不能自己宣布正式结算成功。

对象级授权：只限该交易双方及具备对应案件/资源权限且留审计的管理员；按当前会话校验关系，普通商家无全局权限。非参与者统一 404 NOT_FOUND；嵌套退款、快照、介入和私有附件继承同一边界。见 4.1，可见性不随完成、取消、退款而扩大。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `orderId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

响应 `200`：`V2OrderViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "orderId": "order_demo_001",
    "orderNo": "DT202609080001",
    "channel": "PLAYER_MARKET",
    "status": "WORK_COMPLETED",
    "construction": true,
    "buyer": {
      "kind": "PLAYER",
      "playerRef": "player_xiwanzi",
      "storeId": null,
      "displayName": "Xiwanzi",
      "contactQq": "1000202"
    },
    "seller": {
      "kind": "PLAYER",
      "playerRef": "player_aster",
      "storeId": null,
      "displayName": "Aster",
      "contactQq": "1000103"
    },
    "items": [
      {
        "productId": "listing_house",
        "productVersion": 1,
        "title": "湖畔木屋代建",
        "subtitle": "设计、主体与基础内装",
        "description": "按约定图纸完成主体与内装，请在开工前确认材料和尺寸。",
        "unitPrice": "1800.00",
        "quantity": 1,
        "photoAssetIds": [
          "asset_cover_001"
        ],
        "categoryName": "建筑服务",
        "includedItems": [],
        "contentBlocks": []
      }
    ],
    "amount": "1800.00",
    "currency": "CREDIT",
    "delivery": {
      "method": "WORKSITE",
      "location": "主世界 x=120 y=70 z=-80",
      "projectName": "湖畔木屋"
    },
    "snapshotId": "snapshot_order_001",
    "snapshotSha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "fundsStatus": "HELD",
    "confirmationHours": 168,
    "serverNow": "2026-09-08T02:00:00Z",
    "autoConfirmAt": "2026-09-14T02:00:00Z",
    "pausedRemainingSeconds": null,
    "refund": null,
    "refundAttemptsUsed": 0,
    "availableActions": [
      "CONFIRM_ACCEPTANCE",
      "REQUEST_REFUND"
    ],
    "createdAt": "2026-09-07T01:00:00Z",
    "shippedAt": "2026-09-07T02:00:00Z",
    "workCompletedAt": "2026-09-08T02:00:00Z",
    "confirmedAt": null,
    "automatic": false,
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/orders/{orderId}/complete-work` — 卖家提交工程已完成

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

SHIPPED/施工中 → WORK_COMPLETED/已完成待验收。记录完成说明与时间，不重新开始倒计时、不结算。

对象级授权：只限该交易双方及具备对应案件/资源权限且留审计的管理员；按当前会话校验关系，普通商家无全局权限。非参与者统一 404 NOT_FOUND；嵌套退款、快照、介入和私有附件继承同一边界。见 4.1，可见性不随完成、取消、退款而扩大。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `orderId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`WorkCompletionRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "work-complete-001",
  "expectedVersion": 3,
  "description": "主体与内装已完成，请到约定地点验收。",
  "evidenceAssetIds": []
}
```

响应 `200`：`V2OrderViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "orderId": "order_demo_001",
    "orderNo": "DT202609080001",
    "channel": "PLAYER_MARKET",
    "status": "WORK_COMPLETED",
    "construction": true,
    "buyer": {
      "kind": "PLAYER",
      "playerRef": "player_xiwanzi",
      "storeId": null,
      "displayName": "Xiwanzi",
      "contactQq": "1000202"
    },
    "seller": {
      "kind": "PLAYER",
      "playerRef": "player_aster",
      "storeId": null,
      "displayName": "Aster",
      "contactQq": "1000103"
    },
    "items": [
      {
        "productId": "listing_house",
        "productVersion": 1,
        "title": "湖畔木屋代建",
        "subtitle": "设计、主体与基础内装",
        "description": "按约定图纸完成主体与内装，请在开工前确认材料和尺寸。",
        "unitPrice": "1800.00",
        "quantity": 1,
        "photoAssetIds": [
          "asset_cover_001"
        ],
        "categoryName": "建筑服务",
        "includedItems": [],
        "contentBlocks": []
      }
    ],
    "amount": "1800.00",
    "currency": "CREDIT",
    "delivery": {
      "method": "WORKSITE",
      "location": "主世界 x=120 y=70 z=-80",
      "projectName": "湖畔木屋"
    },
    "snapshotId": "snapshot_order_001",
    "snapshotSha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "fundsStatus": "HELD",
    "confirmationHours": 168,
    "serverNow": "2026-09-08T02:00:00Z",
    "autoConfirmAt": "2026-09-14T02:00:00Z",
    "pausedRemainingSeconds": null,
    "refund": null,
    "refundAttemptsUsed": 0,
    "availableActions": [
      "CONFIRM_ACCEPTANCE",
      "REQUEST_REFUND"
    ],
    "createdAt": "2026-09-07T01:00:00Z",
    "shippedAt": "2026-09-07T02:00:00Z",
    "workCompletedAt": "2026-09-08T02:00:00Z",
    "confirmedAt": null,
    "automatic": false,
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/orders/{orderId}/confirm` — 买家确认收货或工程验收

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

普通商品须已发货；工程手动验收须已提交完成。存在待处理退款或平台资金冻结时禁止确认；成功后只结算一次。

对象级授权：只限该交易双方及具备对应案件/资源权限且留审计的管理员；按当前会话校验关系，普通商家无全局权限。非参与者统一 404 NOT_FOUND；嵌套退款、快照、介入和私有附件继承同一边界。见 4.1，可见性不随完成、取消、退款而扩大。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `orderId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`MutationRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1
}
```

响应 `200`：`V2OrderCreationResultResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "operation": {
      "operationId": "operation_demo_001",
      "clientRequestId": "a472c3b4-d76a-4693-9121-62443e6e7c49",
      "kind": "SETTLEMENT",
      "status": "COMPLETED",
      "resourceType": "ORDER",
      "resourceId": "order_demo_001",
      "amount": "1800.00",
      "createdAt": "2026-09-08T02:00:00Z",
      "updatedAt": "2026-09-08T02:00:00Z",
      "errorCode": null,
      "retryAfterSeconds": 2
    },
    "order": {
      "orderId": "order_demo_001",
      "orderNo": "DT202609080001",
      "channel": "PLAYER_MARKET",
      "status": "CONFIRMED",
      "construction": true,
      "buyer": {
        "kind": "PLAYER",
        "playerRef": "player_xiwanzi",
        "storeId": null,
        "displayName": "Xiwanzi",
        "contactQq": "1000202"
      },
      "seller": {
        "kind": "PLAYER",
        "playerRef": "player_aster",
        "storeId": null,
        "displayName": "Aster",
        "contactQq": "1000103"
      },
      "items": [
        {
          "productId": "listing_house",
          "productVersion": 1,
          "title": "湖畔木屋代建",
          "subtitle": "设计、主体与基础内装",
          "description": "按约定图纸完成主体与内装，请在开工前确认材料和尺寸。",
          "unitPrice": "1800.00",
          "quantity": 1,
          "photoAssetIds": [
            "asset_cover_001"
          ],
          "categoryName": "建筑服务",
          "includedItems": [],
          "contentBlocks": []
        }
      ],
      "amount": "1800.00",
      "currency": "CREDIT",
      "delivery": {
        "method": "WORKSITE",
        "location": "主世界 x=120 y=70 z=-80",
        "projectName": "湖畔木屋"
      },
      "snapshotId": "snapshot_order_001",
      "snapshotSha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      "fundsStatus": "SETTLED",
      "confirmationHours": 168,
      "serverNow": "2026-09-08T02:00:00Z",
      "autoConfirmAt": null,
      "pausedRemainingSeconds": null,
      "refund": null,
      "refundAttemptsUsed": 0,
      "availableActions": [],
      "createdAt": "2026-09-07T01:00:00Z",
      "shippedAt": "2026-09-07T02:00:00Z",
      "workCompletedAt": "2026-09-08T02:00:00Z",
      "confirmedAt": "2026-09-08T02:00:00Z",
      "automatic": false,
      "version": 1
    }
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/orders/{orderId}/interventions` — 申请平台介入订单争议

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

只允许该交易参与方且退款已经被拒。服务端自行抓取成交快照、当前状态、付款/发货/退款日志，关联上传说明和证据。若担保款尚未释放，受理与暂停结算必须原子执行。

对象级授权：只限该交易双方及具备对应案件/资源权限且留审计的管理员；按当前会话校验关系，普通商家无全局权限。非参与者统一 404 NOT_FOUND；嵌套退款、快照、介入和私有附件继承同一边界。见 4.1，可见性不随完成、取消、退款而扩大。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `orderId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`InterventionRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "case-client-001",
  "expectedVersion": 4,
  "reasonCode": "REFUND_DISAGREEMENT",
  "description": "退款申请被拒，现提交约定方案与实际交付差异，请平台核对。",
  "desiredResolution": "PARTIAL_REFUND",
  "requestedRefundAmount": "300.00",
  "evidenceAssetIds": [
    "asset_evidence_001"
  ],
  "relatedMessageIds": [
    "message_original"
  ]
}
```

响应 `201`：`V2InterventionViewResponse`。

完整 JSON 示例见 `api-v2-examples.json` 中同 method/path 条目；模型字段全部在下文展开。

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/orders/{orderId}/mailbox` — 查询官方物品邮箱发放或领取状态

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

领取权威来自插件回执；App 不能通过 POST 伪造已领取。

对象级授权：只限该交易双方及具备对应案件/资源权限且留审计的管理员；按当前会话校验关系，普通商家无全局权限。非参与者统一 404 NOT_FOUND；嵌套退款、快照、介入和私有附件继承同一边界。见 4.1，可见性不随完成、取消、退款而扩大。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `orderId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

响应 `200`：`V2OrderViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "orderId": "order_store_001",
    "orderNo": "DT202609080001",
    "channel": "OFFICIAL_STORE",
    "status": "AWAITING_CLAIM",
    "construction": false,
    "buyer": {
      "kind": "PLAYER",
      "playerRef": "player_xiwanzi",
      "storeId": null,
      "displayName": "Xiwanzi",
      "contactQq": "1000202"
    },
    "seller": {
      "kind": "OFFICIAL_STORE",
      "playerRef": null,
      "storeId": "store_official",
      "displayName": "Deuterium 官方商城",
      "contactQq": "1000000"
    },
    "items": [
      {
        "productId": "product_macmini",
        "productVersion": 1,
        "title": "Mac mini",
        "subtitle": "小巧机身，为桌面留出更多空间。",
        "description": "用于商城页面和下单流程测试的示例商品。",
        "unitPrice": "1399.00",
        "quantity": 1,
        "photoAssetIds": [
          "asset_cover_001",
          "asset_detail_002"
        ],
        "categoryName": "Mac",
        "includedItems": [
          "Mac mini 主题物品 × 1",
          "产品说明 × 1"
        ],
        "contentBlocks": [
          {
            "blockId": "intro",
            "type": "PARAGRAPH",
            "text": "了解商品内容和游戏邮箱交付说明。"
          }
        ]
      }
    ],
    "amount": "1399.00",
    "currency": "CREDIT",
    "delivery": {
      "method": "MAILBOX",
      "location": "Xiwanzi",
      "projectName": ""
    },
    "snapshotId": "snapshot_order_001",
    "snapshotSha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "fundsStatus": "PAID",
    "confirmationHours": 168,
    "serverNow": "2026-09-08T02:00:00Z",
    "autoConfirmAt": null,
    "pausedRemainingSeconds": null,
    "refund": null,
    "refundAttemptsUsed": 0,
    "availableActions": [
      "VIEW_MAILBOX",
      "REQUEST_REFUND"
    ],
    "createdAt": "2026-09-07T01:00:00Z",
    "shippedAt": "2026-09-08T02:00:00Z",
    "workCompletedAt": null,
    "confirmedAt": null,
    "automatic": false,
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/orders/{orderId}/refunds` — 申请订单退款

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

未发货的市场订单可直接退；发货/开工后仅一次申请，暂停自动确认。官方商城仅未领取时自动撤回邮箱并退款，领取与撤回需互斥。

对象级授权：只限该交易双方及具备对应案件/资源权限且留审计的管理员；按当前会话校验关系，普通商家无全局权限。非参与者统一 404 NOT_FOUND；嵌套退款、快照、介入和私有附件继承同一边界。见 4.1，可见性不随完成、取消、退款而扩大。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `orderId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`RefundRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "refund-request-001",
  "expectedVersion": 3,
  "reasonCode": "NOT_AS_DESCRIBED",
  "description": "部分交付内容与双方确认的要求不符，请先协商处理。",
  "evidenceAssetIds": []
}
```

响应 `200`：`V2OrderViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "orderId": "order_demo_001",
    "orderNo": "DT202609080001",
    "channel": "PLAYER_MARKET",
    "status": "WORK_COMPLETED",
    "construction": true,
    "buyer": {
      "kind": "PLAYER",
      "playerRef": "player_xiwanzi",
      "storeId": null,
      "displayName": "Xiwanzi",
      "contactQq": "1000202"
    },
    "seller": {
      "kind": "PLAYER",
      "playerRef": "player_aster",
      "storeId": null,
      "displayName": "Aster",
      "contactQq": "1000103"
    },
    "items": [
      {
        "productId": "listing_house",
        "productVersion": 1,
        "title": "湖畔木屋代建",
        "subtitle": "设计、主体与基础内装",
        "description": "按约定图纸完成主体与内装，请在开工前确认材料和尺寸。",
        "unitPrice": "1800.00",
        "quantity": 1,
        "photoAssetIds": [
          "asset_cover_001"
        ],
        "categoryName": "建筑服务",
        "includedItems": [],
        "contentBlocks": []
      }
    ],
    "amount": "1800.00",
    "currency": "CREDIT",
    "delivery": {
      "method": "WORKSITE",
      "location": "主世界 x=120 y=70 z=-80",
      "projectName": "湖畔木屋"
    },
    "snapshotId": "snapshot_order_001",
    "snapshotSha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "fundsStatus": "HELD",
    "confirmationHours": 168,
    "serverNow": "2026-09-08T02:00:00Z",
    "autoConfirmAt": null,
    "pausedRemainingSeconds": 172800,
    "refund": {
      "refundId": "refund_demo_001",
      "status": "REQUESTED",
      "attempt": 1,
      "reason": "部分装饰与确认的方案不符。",
      "rejectionReason": "",
      "requestedAt": "2026-09-08T01:00:00Z",
      "resolvedAt": null,
      "amount": "1800.00",
      "version": 1
    },
    "refundAttemptsUsed": 1,
    "availableActions": [
      "WITHDRAW_REFUND"
    ],
    "createdAt": "2026-09-07T01:00:00Z",
    "shippedAt": "2026-09-07T02:00:00Z",
    "workCompletedAt": "2026-09-08T02:00:00Z",
    "confirmedAt": null,
    "automatic": false,
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/orders/{orderId}/refunds/{refundId}` — 获取退款详情

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

对象级授权：只限该交易双方及具备对应案件/资源权限且留审计的管理员；按当前会话校验关系，普通商家无全局权限。非参与者统一 404 NOT_FOUND；嵌套退款、快照、介入和私有附件继承同一边界。见 4.1，可见性不随完成、取消、退款而扩大。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `orderId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |
| `refundId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

响应 `200`：`V2OrderRefundResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "refundId": "refund_demo_001",
    "status": "REJECTED",
    "attempt": 1,
    "reason": "部分装饰与确认的方案不符。",
    "rejectionReason": "已按聊天中最终确认的方案完成，附图可核对。",
    "requestedAt": "2026-09-08T01:00:00Z",
    "resolvedAt": "2026-09-08T02:00:00Z",
    "amount": "1800.00",
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/orders/{orderId}/refunds/{refundId}/resolve` — 卖方同意或拒绝退款

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

仅卖方处理 REQUESTED 申请。REJECT 必须提供理由，并持久化/通知；APPROVE 完成实际退款后才标记终态。

对象级授权：只限该交易双方及具备对应案件/资源权限且留审计的管理员；按当前会话校验关系，普通商家无全局权限。非参与者统一 404 NOT_FOUND；嵌套退款、快照、介入和私有附件继承同一边界。见 4.1，可见性不随完成、取消、退款而扩大。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `orderId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |
| `refundId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`RefundResolutionRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "refund-decision-001",
  "expectedVersion": 2,
  "decision": "REJECT",
  "reason": "已按双方最终确认的方案完工，请核对提供的完成图片。"
}
```

响应 `200`：`V2OrderViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "orderId": "order_demo_001",
    "orderNo": "DT202609080001",
    "channel": "PLAYER_MARKET",
    "status": "WORK_COMPLETED",
    "construction": true,
    "buyer": {
      "kind": "PLAYER",
      "playerRef": "player_xiwanzi",
      "storeId": null,
      "displayName": "Xiwanzi",
      "contactQq": "1000202"
    },
    "seller": {
      "kind": "PLAYER",
      "playerRef": "player_aster",
      "storeId": null,
      "displayName": "Aster",
      "contactQq": "1000103"
    },
    "items": [
      {
        "productId": "listing_house",
        "productVersion": 1,
        "title": "湖畔木屋代建",
        "subtitle": "设计、主体与基础内装",
        "description": "按约定图纸完成主体与内装，请在开工前确认材料和尺寸。",
        "unitPrice": "1800.00",
        "quantity": 1,
        "photoAssetIds": [
          "asset_cover_001"
        ],
        "categoryName": "建筑服务",
        "includedItems": [],
        "contentBlocks": []
      }
    ],
    "amount": "1800.00",
    "currency": "CREDIT",
    "delivery": {
      "method": "WORKSITE",
      "location": "主世界 x=120 y=70 z=-80",
      "projectName": "湖畔木屋"
    },
    "snapshotId": "snapshot_order_001",
    "snapshotSha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "fundsStatus": "HELD",
    "confirmationHours": 168,
    "serverNow": "2026-09-08T02:00:00Z",
    "autoConfirmAt": "2026-09-10T02:00:00Z",
    "pausedRemainingSeconds": null,
    "refund": {
      "refundId": "refund_demo_001",
      "status": "REJECTED",
      "attempt": 1,
      "reason": "部分装饰与确认的方案不符。",
      "rejectionReason": "已按双方最终确认的方案完工，请核对提供的完成图片。",
      "requestedAt": "2026-09-08T01:00:00Z",
      "resolvedAt": "2026-09-08T02:00:00Z",
      "amount": "1800.00",
      "version": 1
    },
    "refundAttemptsUsed": 1,
    "availableActions": [
      "REQUEST_INTERVENTION"
    ],
    "createdAt": "2026-09-07T01:00:00Z",
    "shippedAt": "2026-09-07T02:00:00Z",
    "workCompletedAt": "2026-09-08T02:00:00Z",
    "confirmedAt": null,
    "automatic": false,
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/orders/{orderId}/refunds/{refundId}/withdraw` — 买方撤回退款申请

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

恢复剩余时间，但已使用的唯一机会不恢复。

对象级授权：只限该交易双方及具备对应案件/资源权限且留审计的管理员；按当前会话校验关系，普通商家无全局权限。非参与者统一 404 NOT_FOUND；嵌套退款、快照、介入和私有附件继承同一边界。见 4.1，可见性不随完成、取消、退款而扩大。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `orderId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |
| `refundId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`MutationRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1
}
```

响应 `200`：`V2OrderViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "orderId": "order_demo_001",
    "orderNo": "DT202609080001",
    "channel": "PLAYER_MARKET",
    "status": "WORK_COMPLETED",
    "construction": true,
    "buyer": {
      "kind": "PLAYER",
      "playerRef": "player_xiwanzi",
      "storeId": null,
      "displayName": "Xiwanzi",
      "contactQq": "1000202"
    },
    "seller": {
      "kind": "PLAYER",
      "playerRef": "player_aster",
      "storeId": null,
      "displayName": "Aster",
      "contactQq": "1000103"
    },
    "items": [
      {
        "productId": "listing_house",
        "productVersion": 1,
        "title": "湖畔木屋代建",
        "subtitle": "设计、主体与基础内装",
        "description": "按约定图纸完成主体与内装，请在开工前确认材料和尺寸。",
        "unitPrice": "1800.00",
        "quantity": 1,
        "photoAssetIds": [
          "asset_cover_001"
        ],
        "categoryName": "建筑服务",
        "includedItems": [],
        "contentBlocks": []
      }
    ],
    "amount": "1800.00",
    "currency": "CREDIT",
    "delivery": {
      "method": "WORKSITE",
      "location": "主世界 x=120 y=70 z=-80",
      "projectName": "湖畔木屋"
    },
    "snapshotId": "snapshot_order_001",
    "snapshotSha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "fundsStatus": "HELD",
    "confirmationHours": 168,
    "serverNow": "2026-09-08T02:00:00Z",
    "autoConfirmAt": "2026-09-10T02:00:00Z",
    "pausedRemainingSeconds": null,
    "refund": {
      "refundId": "refund_demo_001",
      "status": "WITHDRAWN",
      "attempt": 1,
      "reason": "部分装饰与确认的方案不符。",
      "rejectionReason": "",
      "requestedAt": "2026-09-08T01:00:00Z",
      "resolvedAt": "2026-09-08T02:00:00Z",
      "amount": "1800.00",
      "version": 1
    },
    "refundAttemptsUsed": 1,
    "availableActions": [],
    "createdAt": "2026-09-07T01:00:00Z",
    "shippedAt": "2026-09-07T02:00:00Z",
    "workCompletedAt": "2026-09-08T02:00:00Z",
    "confirmedAt": null,
    "automatic": false,
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/orders/{orderId}/ship` — 卖家确认普通商品已交付

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

仅普通市场商品卖家，AWAITING_SHIPMENT → SHIPPED；服务端记录发货时间并开始 72 小时自动确认。

对象级授权：只限该交易双方及具备对应案件/资源权限且留审计的管理员；按当前会话校验关系，普通商家无全局权限。非参与者统一 404 NOT_FOUND；嵌套退款、快照、介入和私有附件继承同一边界。见 4.1，可见性不随完成、取消、退款而扩大。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `orderId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`MutationRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1
}
```

响应 `200`：`V2OrderViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "orderId": "order_demo_001",
    "orderNo": "DT202609080001",
    "channel": "PLAYER_MARKET",
    "status": "SHIPPED",
    "construction": false,
    "buyer": {
      "kind": "PLAYER",
      "playerRef": "player_xiwanzi",
      "storeId": null,
      "displayName": "Xiwanzi",
      "contactQq": "1000202"
    },
    "seller": {
      "kind": "PLAYER",
      "playerRef": "player_aster",
      "storeId": null,
      "displayName": "Aster",
      "contactQq": "1000103"
    },
    "items": [
      {
        "productId": "listing_house",
        "productVersion": 1,
        "title": "湖畔木屋代建",
        "subtitle": "设计、主体与基础内装",
        "description": "按约定图纸完成主体与内装，请在开工前确认材料和尺寸。",
        "unitPrice": "1800.00",
        "quantity": 1,
        "photoAssetIds": [
          "asset_cover_001"
        ],
        "categoryName": "建筑服务",
        "includedItems": [],
        "contentBlocks": []
      }
    ],
    "amount": "1800.00",
    "currency": "CREDIT",
    "delivery": {
      "method": "PICKUP",
      "location": "主世界出生点",
      "projectName": ""
    },
    "snapshotId": "snapshot_order_001",
    "snapshotSha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "fundsStatus": "HELD",
    "confirmationHours": 72,
    "serverNow": "2026-09-08T02:00:00Z",
    "autoConfirmAt": "2026-09-11T02:00:00Z",
    "pausedRemainingSeconds": null,
    "refund": null,
    "refundAttemptsUsed": 0,
    "availableActions": [
      "CONFIRM_RECEIPT",
      "REQUEST_REFUND"
    ],
    "createdAt": "2026-09-07T01:00:00Z",
    "shippedAt": "2026-09-08T02:00:00Z",
    "workCompletedAt": null,
    "confirmedAt": null,
    "automatic": false,
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/orders/{orderId}/snapshot` — 获取不可变成交条款快照

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

只有订单参与者和获授权管理者可读；商品下架、编辑或删除后仍可读取成交时内容。

对象级授权：只限该交易双方及具备对应案件/资源权限且留审计的管理员；按当前会话校验关系，普通商家无全局权限。非参与者统一 404 NOT_FOUND；嵌套退款、快照、介入和私有附件继承同一边界。见 4.1，可见性不随完成、取消、退款而扩大。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `orderId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

响应 `200`：`V2OrderContractSnapshotResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "snapshotId": "ref_demo_001",
    "orderId": "ref_demo_001",
    "buyer": {
      "kind": "PLAYER",
      "playerRef": "player_mori",
      "storeId": null,
      "displayName": "Mori",
      "contactQq": "1000101"
    },
    "seller": {
      "kind": "PLAYER",
      "playerRef": "player_mori",
      "storeId": null,
      "displayName": "Mori",
      "contactQq": "1000101"
    },
    "items": [
      {
        "productId": "ref_demo_001",
        "productVersion": 1,
        "title": "example",
        "subtitle": "example",
        "description": "example",
        "unitPrice": "680.00",
        "quantity": 1,
        "photoAssetIds": [
          "ref_demo_001"
        ],
        "categoryName": "example",
        "includedItems": [
          "example"
        ],
        "contentBlocks": [
          {
            "blockId": "ref_demo_001",
            "type": "HEADING",
            "heading": "example",
            "text": "example",
            "assetId": null,
            "altText": "example",
            "rows": [
              {
                "label": "example",
                "value": "example"
              }
            ]
          }
        ]
      }
    ],
    "totalAmount": "680.00",
    "delivery": {
      "method": "DOOR",
      "location": "example",
      "projectName": "example"
    },
    "confirmationHours": 1,
    "capturedAt": "2026-09-08T02:00:00Z",
    "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/orders/{orderId}/start-work` — 卖家开始施工

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

仅建筑服务卖家；从此刻开始约定总工期，工期包含验收预留。

对象级授权：只限该交易双方及具备对应案件/资源权限且留审计的管理员；按当前会话校验关系，普通商家无全局权限。非参与者统一 404 NOT_FOUND；嵌套退款、快照、介入和私有附件继承同一边界。见 4.1，可见性不随完成、取消、退款而扩大。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `orderId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`MutationRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1
}
```

响应 `200`：`V2OrderViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "orderId": "order_demo_001",
    "orderNo": "DT202609080001",
    "channel": "PLAYER_MARKET",
    "status": "SHIPPED",
    "construction": true,
    "buyer": {
      "kind": "PLAYER",
      "playerRef": "player_xiwanzi",
      "storeId": null,
      "displayName": "Xiwanzi",
      "contactQq": "1000202"
    },
    "seller": {
      "kind": "PLAYER",
      "playerRef": "player_aster",
      "storeId": null,
      "displayName": "Aster",
      "contactQq": "1000103"
    },
    "items": [
      {
        "productId": "listing_house",
        "productVersion": 1,
        "title": "湖畔木屋代建",
        "subtitle": "设计、主体与基础内装",
        "description": "按约定图纸完成主体与内装，请在开工前确认材料和尺寸。",
        "unitPrice": "1800.00",
        "quantity": 1,
        "photoAssetIds": [
          "asset_cover_001"
        ],
        "categoryName": "建筑服务",
        "includedItems": [],
        "contentBlocks": []
      }
    ],
    "amount": "1800.00",
    "currency": "CREDIT",
    "delivery": {
      "method": "WORKSITE",
      "location": "主世界 x=120 y=70 z=-80",
      "projectName": "湖畔木屋"
    },
    "snapshotId": "snapshot_order_001",
    "snapshotSha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "fundsStatus": "HELD",
    "confirmationHours": 168,
    "serverNow": "2026-09-08T02:00:00Z",
    "autoConfirmAt": "2026-09-15T02:00:00Z",
    "pausedRemainingSeconds": null,
    "refund": null,
    "refundAttemptsUsed": 0,
    "availableActions": [
      "REQUEST_REFUND"
    ],
    "createdAt": "2026-09-07T01:00:00Z",
    "shippedAt": "2026-09-08T02:00:00Z",
    "workCompletedAt": null,
    "confirmedAt": null,
    "automatic": false,
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

### 委托

#### `GET /api/v1/commissions` — 浏览委托大厅

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

大厅仅返回预付成功的 OPEN 委托。

强制仅返回 status=OPEN 且 fundsStatus=HELD。status 省略等同 OPEN，其他值返回 400 INVALID_REQUEST；搜索/游标过滤一致。接取成功后立即移出大厅，后续仅双方及授权管理员可读。Cache-Control: private, no-store。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `status` | query | 否 | string | `OPEN`；默认且只允许 OPEN；其他状态返回 400 INVALID_REQUEST，服务端始终执行公开过滤。 |
| `urgency` | query | 否 | string | `NORMAL` / `SOON` / `URGENT`； |
| `q` | query | 否 | string | 长度 1–80； |
| `cursor` | query | 否 | string | 长度 1–512； |
| `limit` | query | 否 | integer | 范围 1–100； |

响应 `200`：`V2CommissionViewListResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": [
    {
      "commissionId": "commission_demo_001",
      "owner": {
        "kind": "PLAYER",
        "playerRef": "player_mori",
        "storeId": null,
        "displayName": "Mori",
        "contactQq": "1000101"
      },
      "worker": null,
      "content": {
        "title": "出生点花园补灯",
        "description": "为步道补齐灯笼，完成后说明布置位置。",
        "location": "主世界出生点花园",
        "urgency": "SOON",
        "reward": "420.00",
        "workHours": 48,
        "coverAssetId": "asset_cover_001"
      },
      "cover": {
        "assetId": "asset_cover_001",
        "purpose": "COMMISSION_COVER",
        "status": "READY",
        "url": "https://cdn.example.invalid/assets/cover.png",
        "width": 1254,
        "height": 1254,
        "sizeBytes": 1819447,
        "sha256": null,
        "altText": "商品封面",
        "createdAt": "2026-09-08T02:00:00Z",
        "contentMd5": null,
        "urlExpiresAt": null
      },
      "status": "OPEN",
      "fundsStatus": "HELD",
      "snapshotId": "snapshot_commission_001",
      "serverNow": "2026-09-08T02:00:00Z",
      "workDueAt": null,
      "acceptanceDueAt": null,
      "pausedRemainingSeconds": null,
      "completionDescription": "",
      "completionAssetIds": [],
      "refund": null,
      "refundAttemptsUsed": 0,
      "availableActions": [
        "ACCEPT"
      ],
      "createdAt": "2026-09-08T02:00:00Z",
      "acceptedAt": null,
      "completedAt": null,
      "confirmedAt": null,
      "automatic": false,
      "version": 1
    }
  ],
  "serverTime": "2026-09-08T02:00:00Z",
  "page": {
    "nextCursor": null
  }
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/commissions` — 预付报酬并发布委托

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

资金预付与发布使用同一幂等键；预付失败不能被接取，客户端没有指定付款人字段。

请求模型：`CommissionCreateRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "content": {
    "title": "出生点花园补灯",
    "description": "为步道补齐灯笼，完成后说明布置位置。",
    "location": "主世界出生点花园",
    "urgency": "SOON",
    "reward": "420.00",
    "workHours": 48,
    "coverAssetId": "asset_cover_001"
  }
}
```

响应 `201`：`V2CommissionCreationResultResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "operation": {
      "operationId": "operation_demo_001",
      "clientRequestId": "a472c3b4-d76a-4693-9121-62443e6e7c49",
      "kind": "COMMISSION_PUBLISH",
      "status": "COMPLETED",
      "resourceType": "COMMISSION",
      "resourceId": "commission_demo_001",
      "amount": "420.00",
      "createdAt": "2026-09-08T02:00:00Z",
      "updatedAt": "2026-09-08T02:00:00Z",
      "errorCode": null,
      "retryAfterSeconds": 2
    },
    "commission": {
      "commissionId": "commission_demo_001",
      "owner": {
        "kind": "PLAYER",
        "playerRef": "player_mori",
        "storeId": null,
        "displayName": "Mori",
        "contactQq": "1000101"
      },
      "worker": null,
      "content": {
        "title": "出生点花园补灯",
        "description": "为步道补齐灯笼，完成后说明布置位置。",
        "location": "主世界出生点花园",
        "urgency": "SOON",
        "reward": "420.00",
        "workHours": 48,
        "coverAssetId": "asset_cover_001"
      },
      "cover": {
        "assetId": "asset_cover_001",
        "purpose": "COMMISSION_COVER",
        "status": "READY",
        "url": "https://cdn.example.invalid/assets/cover.png",
        "width": 1254,
        "height": 1254,
        "sizeBytes": 1819447,
        "sha256": null,
        "altText": "商品封面",
        "createdAt": "2026-09-08T02:00:00Z",
        "contentMd5": null,
        "urlExpiresAt": null
      },
      "status": "OPEN",
      "fundsStatus": "HELD",
      "snapshotId": "snapshot_commission_001",
      "serverNow": "2026-09-08T02:00:00Z",
      "workDueAt": null,
      "acceptanceDueAt": null,
      "pausedRemainingSeconds": null,
      "completionDescription": "",
      "completionAssetIds": [],
      "refund": null,
      "refundAttemptsUsed": 0,
      "availableActions": [
        "CANCEL"
      ],
      "createdAt": "2026-09-08T02:00:00Z",
      "acceptedAt": null,
      "completedAt": null,
      "confirmedAt": null,
      "automatic": false,
      "version": 1
    }
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

处理中响应 202：查询 operationId；收到终态前不能把动画完成视为已付款。

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/commissions/me` — 查看我发布或接取的委托

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

只返回会话本人发布或接取的记录；role=PUBLISHER/WORKER 不接受他人身份替代。含已接取和终态历史，全部响应 private, no-store。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `role` | query | 是 | string | `PUBLISHER` / `WORKER`； |
| `status` | query | 否 | string | `FUNDING` / `OPEN` / `ACTIVE` / `COMPLETED` / `CONFIRMED` / `CANCELLED`； |
| `cursor` | query | 否 | string | 长度 1–512； |
| `limit` | query | 否 | integer | 范围 1–100； |

响应 `200`：`V2CommissionViewListResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": [
    {
      "commissionId": "commission_demo_001",
      "owner": {
        "kind": "PLAYER",
        "playerRef": "player_mori",
        "storeId": null,
        "displayName": "Mori",
        "contactQq": "1000101"
      },
      "worker": {
        "kind": "PLAYER",
        "playerRef": "player_xiwanzi",
        "storeId": null,
        "displayName": "Xiwanzi",
        "contactQq": "1000101"
      },
      "content": {
        "title": "出生点花园补灯",
        "description": "为步道补齐灯笼，完成后说明布置位置。",
        "location": "主世界出生点花园",
        "urgency": "SOON",
        "reward": "420.00",
        "workHours": 48,
        "coverAssetId": "asset_cover_001"
      },
      "cover": {
        "assetId": "asset_cover_001",
        "purpose": "COMMISSION_COVER",
        "status": "READY",
        "url": "https://cdn.example.invalid/assets/cover.png",
        "width": 1254,
        "height": 1254,
        "sizeBytes": 1819447,
        "sha256": null,
        "altText": "商品封面",
        "createdAt": "2026-09-08T02:00:00Z",
        "contentMd5": null,
        "urlExpiresAt": null
      },
      "status": "COMPLETED",
      "fundsStatus": "HELD",
      "snapshotId": "snapshot_commission_001",
      "serverNow": "2026-09-08T02:00:00Z",
      "workDueAt": "2026-09-09T02:00:00Z",
      "acceptanceDueAt": "2026-09-11T02:00:00Z",
      "pausedRemainingSeconds": null,
      "completionDescription": "步道补灯完成，已检查照明盲区。",
      "completionAssetIds": [],
      "refund": null,
      "refundAttemptsUsed": 0,
      "availableActions": [
        "CONFIRM",
        "REQUEST_REFUND"
      ],
      "createdAt": "2026-09-08T02:00:00Z",
      "acceptedAt": "2026-09-07T02:00:00Z",
      "completedAt": "2026-09-08T02:00:00Z",
      "confirmedAt": null,
      "automatic": false,
      "version": 1
    }
  ],
  "serverTime": "2026-09-08T02:00:00Z",
  "page": {
    "nextCursor": null
  }
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/commissions/{commissionId}` — 获取委托详情和进度

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

OPEN 且预付成功时允许已登录玩家读取公开内容；其他状态仅发布者、接取者及有对应资源权限的管理员可读。非参与者 404，不泄漏状态或参与者；private, no-store。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `commissionId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

响应 `200`：`V2CommissionViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "commissionId": "commission_demo_001",
    "owner": {
      "kind": "PLAYER",
      "playerRef": "player_mori",
      "storeId": null,
      "displayName": "Mori",
      "contactQq": "1000101"
    },
    "worker": {
      "kind": "PLAYER",
      "playerRef": "player_xiwanzi",
      "storeId": null,
      "displayName": "Xiwanzi",
      "contactQq": "1000101"
    },
    "content": {
      "title": "出生点花园补灯",
      "description": "为步道补齐灯笼，完成后说明布置位置。",
      "location": "主世界出生点花园",
      "urgency": "SOON",
      "reward": "420.00",
      "workHours": 48,
      "coverAssetId": "asset_cover_001"
    },
    "cover": {
      "assetId": "asset_cover_001",
      "purpose": "COMMISSION_COVER",
      "status": "READY",
      "url": "https://cdn.example.invalid/assets/cover.png",
      "width": 1254,
      "height": 1254,
      "sizeBytes": 1819447,
      "sha256": null,
      "altText": "商品封面",
      "createdAt": "2026-09-08T02:00:00Z",
      "contentMd5": null,
      "urlExpiresAt": null
    },
    "status": "COMPLETED",
    "fundsStatus": "HELD",
    "snapshotId": "snapshot_commission_001",
    "serverNow": "2026-09-08T02:00:00Z",
    "workDueAt": "2026-09-09T02:00:00Z",
    "acceptanceDueAt": "2026-09-11T02:00:00Z",
    "pausedRemainingSeconds": null,
    "completionDescription": "步道补灯完成，已检查照明盲区。",
    "completionAssetIds": [],
    "refund": null,
    "refundAttemptsUsed": 0,
    "availableActions": [
      "CONFIRM",
      "REQUEST_REFUND"
    ],
    "createdAt": "2026-09-08T02:00:00Z",
    "acceptedAt": "2026-09-07T02:00:00Z",
    "completedAt": "2026-09-08T02:00:00Z",
    "confirmedAt": null,
    "automatic": false,
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/commissions/{commissionId}/accept` — 接取委托

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

必须为其他玩家，状态 OPEN；一个委托只允许一位接取者，原子抢占，冲突返回 ALREADY_ACCEPTED。从接取时开始 workDueAt。

接取事务同步撤销公开可见性；接取者只能由本次会话绑定。其他玩家竞争接取仍按 ALREADY_ACCEPTED 处理，但不回显获胜者。此创建参与关系的动作允许有资格的非发布者在 OPEN 阶段调用；读取限制不禁止合法接取。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `commissionId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`MutationRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1
}
```

响应 `200`：`V2CommissionViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "commissionId": "commission_demo_001",
    "owner": {
      "kind": "PLAYER",
      "playerRef": "player_mori",
      "storeId": null,
      "displayName": "Mori",
      "contactQq": "1000101"
    },
    "worker": {
      "kind": "PLAYER",
      "playerRef": "player_xiwanzi",
      "storeId": null,
      "displayName": "Xiwanzi",
      "contactQq": "1000101"
    },
    "content": {
      "title": "出生点花园补灯",
      "description": "为步道补齐灯笼，完成后说明布置位置。",
      "location": "主世界出生点花园",
      "urgency": "SOON",
      "reward": "420.00",
      "workHours": 48,
      "coverAssetId": "asset_cover_001"
    },
    "cover": {
      "assetId": "asset_cover_001",
      "purpose": "COMMISSION_COVER",
      "status": "READY",
      "url": "https://cdn.example.invalid/assets/cover.png",
      "width": 1254,
      "height": 1254,
      "sizeBytes": 1819447,
      "sha256": null,
      "altText": "商品封面",
      "createdAt": "2026-09-08T02:00:00Z",
      "contentMd5": null,
      "urlExpiresAt": null
    },
    "status": "ACTIVE",
    "fundsStatus": "HELD",
    "snapshotId": "snapshot_commission_001",
    "serverNow": "2026-09-08T02:00:00Z",
    "workDueAt": "2026-09-10T02:00:00Z",
    "acceptanceDueAt": null,
    "pausedRemainingSeconds": null,
    "completionDescription": "",
    "completionAssetIds": [],
    "refund": null,
    "refundAttemptsUsed": 0,
    "availableActions": [
      "COMPLETE"
    ],
    "createdAt": "2026-09-08T02:00:00Z",
    "acceptedAt": "2026-09-08T02:00:00Z",
    "completedAt": null,
    "confirmedAt": null,
    "automatic": false,
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/commissions/{commissionId}/cancel` — 取消尚未接取的委托并退款

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

仅发布者，且仍为 OPEN。取消与接取互斥；接取成功后不得单方取消。

对象级授权：只限该交易双方及具备对应案件/资源权限且留审计的管理员；按当前会话校验关系，普通商家无全局权限。非参与者统一 404 NOT_FOUND；嵌套退款、快照、介入和私有附件继承同一边界。见 4.1，可见性不随完成、取消、退款而扩大。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `commissionId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`MutationRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1
}
```

响应 `200`：`V2CommissionCreationResultResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "operation": {
      "operationId": "operation_demo_001",
      "clientRequestId": "a472c3b4-d76a-4693-9121-62443e6e7c49",
      "kind": "REFUND",
      "status": "COMPLETED",
      "resourceType": "COMMISSION",
      "resourceId": "commission_demo_001",
      "amount": "420.00",
      "createdAt": "2026-09-08T02:00:00Z",
      "updatedAt": "2026-09-08T02:00:00Z",
      "errorCode": null,
      "retryAfterSeconds": 2
    },
    "commission": {
      "commissionId": "commission_demo_001",
      "owner": {
        "kind": "PLAYER",
        "playerRef": "player_mori",
        "storeId": null,
        "displayName": "Mori",
        "contactQq": "1000101"
      },
      "worker": null,
      "content": {
        "title": "出生点花园补灯",
        "description": "为步道补齐灯笼，完成后说明布置位置。",
        "location": "主世界出生点花园",
        "urgency": "SOON",
        "reward": "420.00",
        "workHours": 48,
        "coverAssetId": "asset_cover_001"
      },
      "cover": {
        "assetId": "asset_cover_001",
        "purpose": "COMMISSION_COVER",
        "status": "READY",
        "url": "https://cdn.example.invalid/assets/cover.png",
        "width": 1254,
        "height": 1254,
        "sizeBytes": 1819447,
        "sha256": null,
        "altText": "商品封面",
        "createdAt": "2026-09-08T02:00:00Z",
        "contentMd5": null,
        "urlExpiresAt": null
      },
      "status": "CANCELLED",
      "fundsStatus": "REFUNDED",
      "snapshotId": "snapshot_commission_001",
      "serverNow": "2026-09-08T02:00:00Z",
      "workDueAt": null,
      "acceptanceDueAt": null,
      "pausedRemainingSeconds": null,
      "completionDescription": "",
      "completionAssetIds": [],
      "refund": null,
      "refundAttemptsUsed": 0,
      "availableActions": [],
      "createdAt": "2026-09-08T02:00:00Z",
      "acceptedAt": null,
      "completedAt": null,
      "confirmedAt": null,
      "automatic": false,
      "version": 1
    }
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/commissions/{commissionId}/complete` — 接取者提交已完成

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

仅接取者，ACTIVE → COMPLETED，请求 description 必填，响应保存为 completionDescription。开始新的 72 小时确认期；履约期限逾期不阻止提交完成，也不自动付款。

对象级授权：只限该交易双方及具备对应案件/资源权限且留审计的管理员；按当前会话校验关系，普通商家无全局权限。非参与者统一 404 NOT_FOUND；嵌套退款、快照、介入和私有附件继承同一边界。见 4.1，可见性不随完成、取消、退款而扩大。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `commissionId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`WorkCompletionRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "work-complete-001",
  "expectedVersion": 3,
  "description": "步道补灯完成，已检查照明盲区。",
  "evidenceAssetIds": []
}
```

响应 `200`：`V2CommissionViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "commissionId": "commission_demo_001",
    "owner": {
      "kind": "PLAYER",
      "playerRef": "player_mori",
      "storeId": null,
      "displayName": "Mori",
      "contactQq": "1000101"
    },
    "worker": {
      "kind": "PLAYER",
      "playerRef": "player_xiwanzi",
      "storeId": null,
      "displayName": "Xiwanzi",
      "contactQq": "1000101"
    },
    "content": {
      "title": "出生点花园补灯",
      "description": "为步道补齐灯笼，完成后说明布置位置。",
      "location": "主世界出生点花园",
      "urgency": "SOON",
      "reward": "420.00",
      "workHours": 48,
      "coverAssetId": "asset_cover_001"
    },
    "cover": {
      "assetId": "asset_cover_001",
      "purpose": "COMMISSION_COVER",
      "status": "READY",
      "url": "https://cdn.example.invalid/assets/cover.png",
      "width": 1254,
      "height": 1254,
      "sizeBytes": 1819447,
      "sha256": null,
      "altText": "商品封面",
      "createdAt": "2026-09-08T02:00:00Z",
      "contentMd5": null,
      "urlExpiresAt": null
    },
    "status": "COMPLETED",
    "fundsStatus": "HELD",
    "snapshotId": "snapshot_commission_001",
    "serverNow": "2026-09-08T02:00:00Z",
    "workDueAt": "2026-09-09T02:00:00Z",
    "acceptanceDueAt": "2026-09-11T02:00:00Z",
    "pausedRemainingSeconds": null,
    "completionDescription": "步道补灯完成，已检查照明盲区。",
    "completionAssetIds": [],
    "refund": null,
    "refundAttemptsUsed": 0,
    "availableActions": [
      "CONFIRM",
      "REQUEST_REFUND"
    ],
    "createdAt": "2026-09-08T02:00:00Z",
    "acceptedAt": "2026-09-07T02:00:00Z",
    "completedAt": "2026-09-08T02:00:00Z",
    "confirmedAt": null,
    "automatic": false,
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/commissions/{commissionId}/confirm` — 发布者确认完成并结算报酬

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

仅发布者、COMPLETED 且无待处理退款/介入。自动确认和手动确认共享一次结算锁。

对象级授权：只限该交易双方及具备对应案件/资源权限且留审计的管理员；按当前会话校验关系，普通商家无全局权限。非参与者统一 404 NOT_FOUND；嵌套退款、快照、介入和私有附件继承同一边界。见 4.1，可见性不随完成、取消、退款而扩大。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `commissionId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`MutationRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1
}
```

响应 `200`：`V2CommissionCreationResultResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "operation": {
      "operationId": "operation_demo_001",
      "clientRequestId": "a472c3b4-d76a-4693-9121-62443e6e7c49",
      "kind": "SETTLEMENT",
      "status": "COMPLETED",
      "resourceType": "COMMISSION",
      "resourceId": "commission_demo_001",
      "amount": "420.00",
      "createdAt": "2026-09-08T02:00:00Z",
      "updatedAt": "2026-09-08T02:00:00Z",
      "errorCode": null,
      "retryAfterSeconds": 2
    },
    "commission": {
      "commissionId": "commission_demo_001",
      "owner": {
        "kind": "PLAYER",
        "playerRef": "player_mori",
        "storeId": null,
        "displayName": "Mori",
        "contactQq": "1000101"
      },
      "worker": {
        "kind": "PLAYER",
        "playerRef": "player_xiwanzi",
        "storeId": null,
        "displayName": "Xiwanzi",
        "contactQq": "1000101"
      },
      "content": {
        "title": "出生点花园补灯",
        "description": "为步道补齐灯笼，完成后说明布置位置。",
        "location": "主世界出生点花园",
        "urgency": "SOON",
        "reward": "420.00",
        "workHours": 48,
        "coverAssetId": "asset_cover_001"
      },
      "cover": {
        "assetId": "asset_cover_001",
        "purpose": "COMMISSION_COVER",
        "status": "READY",
        "url": "https://cdn.example.invalid/assets/cover.png",
        "width": 1254,
        "height": 1254,
        "sizeBytes": 1819447,
        "sha256": null,
        "altText": "商品封面",
        "createdAt": "2026-09-08T02:00:00Z",
        "contentMd5": null,
        "urlExpiresAt": null
      },
      "status": "CONFIRMED",
      "fundsStatus": "SETTLED",
      "snapshotId": "snapshot_commission_001",
      "serverNow": "2026-09-08T02:00:00Z",
      "workDueAt": "2026-09-09T02:00:00Z",
      "acceptanceDueAt": null,
      "pausedRemainingSeconds": null,
      "completionDescription": "步道补灯完成，已检查照明盲区。",
      "completionAssetIds": [],
      "refund": null,
      "refundAttemptsUsed": 0,
      "availableActions": [],
      "createdAt": "2026-09-08T02:00:00Z",
      "acceptedAt": "2026-09-07T02:00:00Z",
      "completedAt": "2026-09-08T02:00:00Z",
      "confirmedAt": "2026-09-08T02:00:00Z",
      "automatic": false,
      "version": 1
    }
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/commissions/{commissionId}/interventions` — 申请平台介入委托争议

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

身份、证据和资金冻结规则同订单介入；本轮 App 不实现表单或入口。

对象级授权：只限该交易双方及具备对应案件/资源权限且留审计的管理员；按当前会话校验关系，普通商家无全局权限。非参与者统一 404 NOT_FOUND；嵌套退款、快照、介入和私有附件继承同一边界。见 4.1，可见性不随完成、取消、退款而扩大。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `commissionId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`InterventionRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "case-client-001",
  "expectedVersion": 4,
  "reasonCode": "REFUND_DISAGREEMENT",
  "description": "退款申请被拒，现提交约定方案与实际交付差异，请平台核对。",
  "desiredResolution": "PARTIAL_REFUND",
  "requestedRefundAmount": "300.00",
  "evidenceAssetIds": [
    "asset_evidence_001"
  ],
  "relatedMessageIds": [
    "message_original"
  ]
}
```

响应 `201`：`V2InterventionViewResponse`。

完整 JSON 示例见 `api-v2-examples.json` 中同 method/path 条目；模型字段全部在下文展开。

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/commissions/{commissionId}/refunds` — 发布者申请委托退款

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

接取后仅可申请一次；暂停当前阶段剩余时间。接取者是该协商的处理方。

对象级授权：只限该交易双方及具备对应案件/资源权限且留审计的管理员；按当前会话校验关系，普通商家无全局权限。非参与者统一 404 NOT_FOUND；嵌套退款、快照、介入和私有附件继承同一边界。见 4.1，可见性不随完成、取消、退款而扩大。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `commissionId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`RefundRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "refund-request-001",
  "expectedVersion": 3,
  "reasonCode": "NOT_AS_DESCRIBED",
  "description": "部分交付内容与双方确认的要求不符，请先协商处理。",
  "evidenceAssetIds": []
}
```

响应 `200`：`V2CommissionViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "commissionId": "commission_demo_001",
    "owner": {
      "kind": "PLAYER",
      "playerRef": "player_mori",
      "storeId": null,
      "displayName": "Mori",
      "contactQq": "1000101"
    },
    "worker": {
      "kind": "PLAYER",
      "playerRef": "player_xiwanzi",
      "storeId": null,
      "displayName": "Xiwanzi",
      "contactQq": "1000101"
    },
    "content": {
      "title": "出生点花园补灯",
      "description": "为步道补齐灯笼，完成后说明布置位置。",
      "location": "主世界出生点花园",
      "urgency": "SOON",
      "reward": "420.00",
      "workHours": 48,
      "coverAssetId": "asset_cover_001"
    },
    "cover": {
      "assetId": "asset_cover_001",
      "purpose": "COMMISSION_COVER",
      "status": "READY",
      "url": "https://cdn.example.invalid/assets/cover.png",
      "width": 1254,
      "height": 1254,
      "sizeBytes": 1819447,
      "sha256": null,
      "altText": "商品封面",
      "createdAt": "2026-09-08T02:00:00Z",
      "contentMd5": null,
      "urlExpiresAt": null
    },
    "status": "COMPLETED",
    "fundsStatus": "HELD",
    "snapshotId": "snapshot_commission_001",
    "serverNow": "2026-09-08T02:00:00Z",
    "workDueAt": "2026-09-09T02:00:00Z",
    "acceptanceDueAt": null,
    "pausedRemainingSeconds": 172800,
    "completionDescription": "步道补灯完成，已检查照明盲区。",
    "completionAssetIds": [],
    "refund": {
      "refundId": "refund_demo_001",
      "status": "REQUESTED",
      "attempt": 1,
      "reason": "部分装饰与确认的方案不符。",
      "rejectionReason": "",
      "requestedAt": "2026-09-08T01:00:00Z",
      "resolvedAt": null,
      "amount": "420.00",
      "version": 1
    },
    "refundAttemptsUsed": 1,
    "availableActions": [
      "WITHDRAW_REFUND"
    ],
    "createdAt": "2026-09-08T02:00:00Z",
    "acceptedAt": "2026-09-07T02:00:00Z",
    "completedAt": "2026-09-08T02:00:00Z",
    "confirmedAt": null,
    "automatic": false,
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/commissions/{commissionId}/refunds/{refundId}/resolve` — 接取者处理委托退款

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

拒绝必填理由；同意后关闭委托并原路退回报酬，不再支付接取者。

对象级授权：只限该交易双方及具备对应案件/资源权限且留审计的管理员；按当前会话校验关系，普通商家无全局权限。非参与者统一 404 NOT_FOUND；嵌套退款、快照、介入和私有附件继承同一边界。见 4.1，可见性不随完成、取消、退款而扩大。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `commissionId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |
| `refundId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`RefundResolutionRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "refund-decision-001",
  "expectedVersion": 2,
  "decision": "REJECT",
  "reason": "已按双方最终确认的方案完工，请核对提供的完成图片。"
}
```

响应 `200`：`V2CommissionViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "commissionId": "commission_demo_001",
    "owner": {
      "kind": "PLAYER",
      "playerRef": "player_mori",
      "storeId": null,
      "displayName": "Mori",
      "contactQq": "1000101"
    },
    "worker": {
      "kind": "PLAYER",
      "playerRef": "player_xiwanzi",
      "storeId": null,
      "displayName": "Xiwanzi",
      "contactQq": "1000101"
    },
    "content": {
      "title": "出生点花园补灯",
      "description": "为步道补齐灯笼，完成后说明布置位置。",
      "location": "主世界出生点花园",
      "urgency": "SOON",
      "reward": "420.00",
      "workHours": 48,
      "coverAssetId": "asset_cover_001"
    },
    "cover": {
      "assetId": "asset_cover_001",
      "purpose": "COMMISSION_COVER",
      "status": "READY",
      "url": "https://cdn.example.invalid/assets/cover.png",
      "width": 1254,
      "height": 1254,
      "sizeBytes": 1819447,
      "sha256": null,
      "altText": "商品封面",
      "createdAt": "2026-09-08T02:00:00Z",
      "contentMd5": null,
      "urlExpiresAt": null
    },
    "status": "COMPLETED",
    "fundsStatus": "HELD",
    "snapshotId": "snapshot_commission_001",
    "serverNow": "2026-09-08T02:00:00Z",
    "workDueAt": "2026-09-09T02:00:00Z",
    "acceptanceDueAt": "2026-09-10T02:00:00Z",
    "pausedRemainingSeconds": null,
    "completionDescription": "步道补灯完成，已检查照明盲区。",
    "completionAssetIds": [],
    "refund": {
      "refundId": "refund_demo_001",
      "status": "REJECTED",
      "attempt": 1,
      "reason": "部分装饰与确认的方案不符。",
      "rejectionReason": "已按双方最终确认的方案完工，请核对提供的完成图片。",
      "requestedAt": "2026-09-08T01:00:00Z",
      "resolvedAt": "2026-09-08T02:00:00Z",
      "amount": "420.00",
      "version": 1
    },
    "refundAttemptsUsed": 1,
    "availableActions": [
      "REQUEST_INTERVENTION"
    ],
    "createdAt": "2026-09-08T02:00:00Z",
    "acceptedAt": "2026-09-07T02:00:00Z",
    "completedAt": "2026-09-08T02:00:00Z",
    "confirmedAt": null,
    "automatic": false,
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/commissions/{commissionId}/refunds/{refundId}/withdraw` — 撤回委托退款申请

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

恢复当前阶段计时，不恢复申请次数。

对象级授权：只限该交易双方及具备对应案件/资源权限且留审计的管理员；按当前会话校验关系，普通商家无全局权限。非参与者统一 404 NOT_FOUND；嵌套退款、快照、介入和私有附件继承同一边界。见 4.1，可见性不随完成、取消、退款而扩大。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `commissionId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |
| `refundId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`MutationRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1
}
```

响应 `200`：`V2CommissionViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "commissionId": "commission_demo_001",
    "owner": {
      "kind": "PLAYER",
      "playerRef": "player_mori",
      "storeId": null,
      "displayName": "Mori",
      "contactQq": "1000101"
    },
    "worker": {
      "kind": "PLAYER",
      "playerRef": "player_xiwanzi",
      "storeId": null,
      "displayName": "Xiwanzi",
      "contactQq": "1000101"
    },
    "content": {
      "title": "出生点花园补灯",
      "description": "为步道补齐灯笼，完成后说明布置位置。",
      "location": "主世界出生点花园",
      "urgency": "SOON",
      "reward": "420.00",
      "workHours": 48,
      "coverAssetId": "asset_cover_001"
    },
    "cover": {
      "assetId": "asset_cover_001",
      "purpose": "COMMISSION_COVER",
      "status": "READY",
      "url": "https://cdn.example.invalid/assets/cover.png",
      "width": 1254,
      "height": 1254,
      "sizeBytes": 1819447,
      "sha256": null,
      "altText": "商品封面",
      "createdAt": "2026-09-08T02:00:00Z",
      "contentMd5": null,
      "urlExpiresAt": null
    },
    "status": "COMPLETED",
    "fundsStatus": "HELD",
    "snapshotId": "snapshot_commission_001",
    "serverNow": "2026-09-08T02:00:00Z",
    "workDueAt": "2026-09-09T02:00:00Z",
    "acceptanceDueAt": "2026-09-10T02:00:00Z",
    "pausedRemainingSeconds": null,
    "completionDescription": "步道补灯完成，已检查照明盲区。",
    "completionAssetIds": [],
    "refund": {
      "refundId": "refund_demo_001",
      "status": "WITHDRAWN",
      "attempt": 1,
      "reason": "部分装饰与确认的方案不符。",
      "rejectionReason": "",
      "requestedAt": "2026-09-08T01:00:00Z",
      "resolvedAt": "2026-09-08T02:00:00Z",
      "amount": "420.00",
      "version": 1
    },
    "refundAttemptsUsed": 1,
    "availableActions": [],
    "createdAt": "2026-09-08T02:00:00Z",
    "acceptedAt": "2026-09-07T02:00:00Z",
    "completedAt": "2026-09-08T02:00:00Z",
    "confirmedAt": null,
    "automatic": false,
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/commissions/{commissionId}/snapshot` — 获取发布和接取时锁定的委托条款

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

对象级授权：只限该交易双方及具备对应案件/资源权限且留审计的管理员；按当前会话校验关系，普通商家无全局权限。非参与者统一 404 NOT_FOUND；嵌套退款、快照、介入和私有附件继承同一边界。见 4.1，可见性不随完成、取消、退款而扩大。

OPEN 阶段尚无接取者时只允许发布者及授权管理员；公开详情中的 snapshotId 不授予快照读取权。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `commissionId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

响应 `200`：`V2CommissionContractSnapshotResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "snapshotId": "ref_demo_001",
    "commissionId": "ref_demo_001",
    "owner": {
      "kind": "PLAYER",
      "playerRef": "player_mori",
      "storeId": null,
      "displayName": "Mori",
      "contactQq": "1000101"
    },
    "worker": null,
    "content": {
      "title": "出生点花园补灯",
      "description": "为步道补齐灯笼，完成后说明布置位置。",
      "location": "主世界出生点花园",
      "urgency": "SOON",
      "reward": "420.00",
      "workHours": 48,
      "coverAssetId": "asset_cover_001"
    },
    "acceptanceHours": 72,
    "capturedAt": "2026-09-08T02:00:00Z",
    "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

### 通知

#### `GET /api/v1/announcements` — 读取官方公告列表

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `cursor` | query | 否 | string | 长度 1–512； |
| `limit` | query | 否 | integer | 范围 1–100； |

响应 `200`：`V2AnnouncementViewListResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": [
    {
      "announcementId": "ref_demo_001",
      "title": "example",
      "summary": "example",
      "contentBlocks": [
        {
          "blockId": "ref_demo_001",
          "type": "HEADING",
          "heading": "example",
          "text": "example",
          "assetId": null,
          "altText": "example",
          "rows": [
            {
              "label": "example",
              "value": "example"
            }
          ]
        }
      ],
      "coverAssetId": null,
      "pinned": false,
      "priority": "NORMAL",
      "publishedAt": "2026-09-08T02:00:00Z",
      "updatedAt": "2026-09-08T02:00:00Z",
      "version": 1
    }
  ],
  "serverTime": "2026-09-08T02:00:00Z",
  "page": {
    "nextCursor": null
  }
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/announcements/{announcementId}` — 读取官方公告详情

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `announcementId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

响应 `200`：`V2AnnouncementViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "announcementId": "ref_demo_001",
    "title": "example",
    "summary": "example",
    "contentBlocks": [
      {
        "blockId": "ref_demo_001",
        "type": "HEADING",
        "heading": "example",
        "text": "example",
        "assetId": null,
        "altText": "example",
        "rows": [
          {
            "label": "example",
            "value": "example"
          }
        ]
      }
    ],
    "coverAssetId": null,
    "pinned": false,
    "priority": "NORMAL",
    "publishedAt": "2026-09-08T02:00:00Z",
    "updatedAt": "2026-09-08T02:00:00Z",
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/notifications` — 获取交易和消息收件箱

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `topic` | query | 否 | string | `DIRECT_MESSAGES` / `MENTIONS` / `FOLLOWED_PLAYERS` / `WALLET` / `MARKET_ORDERS` / `COMMISSIONS` / `ANNOUNCEMENTS` / `APP_UPDATES`； |
| `unreadOnly` | query | 否 | boolean | ； |
| `cursor` | query | 否 | string | 长度 1–512； |
| `limit` | query | 否 | integer | 范围 1–100； |

响应 `200`：`V2NotificationViewListResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": [
    {
      "notificationId": "ref_demo_001",
      "topic": "DIRECT_MESSAGES",
      "title": "example",
      "body": "example",
      "target": {
        "kind": "ORDER",
        "referenceId": "example",
        "stateVersion": 1
      },
      "createdAt": "2026-09-08T02:00:00Z",
      "readAt": null
    }
  ],
  "serverTime": "2026-09-08T02:00:00Z",
  "page": {
    "nextCursor": null
  }
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/notifications/devices` — 注册或轮换本机推送令牌

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

设备引用绑定当前会话用户；退出登录需注销当前用户在此设备的投递，令牌不出现在日志与响应。

请求模型：`PushDeviceRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "installationId": "ref_demo_001",
  "provider": "example",
  "pushToken": "example",
  "permissionGranted": true,
  "appVersionCode": 1,
  "locale": "zh-CN"
}
```

响应 `201`：`V2PushDeviceViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "deviceRef": "ref_demo_001",
    "installationId": "ref_demo_001",
    "provider": "example",
    "registeredAt": "2026-09-08T02:00:00Z",
    "permissionGranted": true
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/notifications/devices/{deviceRef}/unregister` — 注销本机推送登记

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

仅本人的设备，不影响其他设备会话。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `deviceRef` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`IdempotentRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001"
}
```

响应 `200`：`V2RemovalResultResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "removed": true
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/notifications/preferences` — 获取账号通知偏好

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

系统通知权限由 Android 管理；业务偏好不会绕过系统权限。

响应 `200`：`V2NotificationPreferencesViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "version": 1,
    "enabled": true,
    "showPreviews": true,
    "directMessages": true,
    "mentions": true,
    "followedPlayers": true,
    "wallet": true,
    "marketOrders": true,
    "commissions": true,
    "announcements": true,
    "appUpdates": true
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `PATCH /api/v1/notifications/preferences` — 修改通知分组开关

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

服务端推送和客户端接收都应用偏好；关闭推送不删除交易通知记录和账单。

请求模型：`NotificationPreferencesPatch`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1,
  "enabled": true,
  "showPreviews": true,
  "directMessages": true,
  "mentions": true,
  "followedPlayers": true,
  "wallet": true,
  "marketOrders": true,
  "commissions": true,
  "announcements": true,
  "appUpdates": true
}
```

响应 `200`：`V2NotificationPreferencesViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "version": 1,
    "enabled": true,
    "showPreviews": true,
    "directMessages": true,
    "mentions": true,
    "followedPlayers": true,
    "wallet": true,
    "marketOrders": true,
    "commissions": true,
    "announcements": true,
    "appUpdates": true
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/notifications/read` — 标记本人通知已读

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

请求模型：`ReadNotificationsRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "notificationIds": [
    "ref_demo_001"
  ]
}
```

响应 `200`：`V2NotificationReadResultResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "readCount": 1,
    "unreadCount": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

### 应用更新

#### `GET /api/v1/app/update-check` — 检查当前 App 版本

实现状态：`existing_route_extension_pending`。权限：公开入口。

兼容已有 latest/latestVersionCode/latestVersionName/message 字段。新增 APK 和资源更新对象；未发布或不兼容的更新为 null。latest 仅表示 APK 是否最新，hasUpdates 同时覆盖两种更新。无正式服务时客户端必须显示连接失败/本机样例，不能伪造版本。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `versionCode` | query | 是 | integer | 范围 0–—； |
| `versionName` | query | 否 | string | ； |
| `resourceVersion` | query | 否 | integer | 范围 0–2147483647；int64； |
| `packageName` | query | 否 | string | 长度 1–200； |
| `channel` | query | 否 | string | `stable` / `preview`； |

响应 `200`：`AppUpdateCheckResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "latest": false,
    "message": "有更新可用",
    "latestVersionCode": 8,
    "latestVersionName": "1.0.6",
    "serverNow": "2026-09-08T02:00:00Z",
    "hasUpdates": true,
    "apkUpdate": {
      "releaseId": "release_apk_8",
      "kind": "APK",
      "versionName": "1.0.6",
      "versionCode": 8,
      "resourceVersion": 0,
      "packageName": "com.deuterium.app",
      "releaseNotes": "优化委托履约与交易通知体验。",
      "downloadUrl": "https://cdn.example.invalid/releases/deuterium-1.0.6.apk",
      "sizeBytes": 30000000,
      "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      "minAppVersionCode": 7,
      "maxAppVersionCode": 99,
      "publishedAt": "2026-09-08T02:00:00Z",
      "channel": "stable"
    },
    "resourceUpdate": {
      "releaseId": "release_resources_2",
      "kind": "RESOURCES",
      "versionName": "资源版本 2",
      "versionCode": 0,
      "resourceVersion": 2,
      "packageName": "com.deuterium.app",
      "releaseNotes": "更新大厅文案和品牌图片。",
      "downloadUrl": "https://cdn.example.invalid/releases/resources-2.zip",
      "sizeBytes": 1817980,
      "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      "minAppVersionCode": 7,
      "maxAppVersionCode": 99,
      "publishedAt": "2026-09-08T02:00:00Z",
      "channel": "stable"
    }
  }
}
```

### 网页商家端

#### `GET /api/v1/merchant/me` — 获取商家店铺授权

实现状态：`contract_only`。权限：MERCHANT_LOGIN。

响应 `200`：`V2MerchantIdentityResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "playerRef": "ref_demo_001",
    "storeIds": [
      "ref_demo_001"
    ],
    "permissions": [
      "example"
    ]
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/merchant/orders` — 查询有权限店铺的订单

实现状态：`contract_only`。权限：ORDER_MANAGE。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `storeId` | query | 否 | string | 长度 1–128； |
| `status` | query | 否 | string | `PAYMENT_PROCESSING` / `AWAITING_SHIPMENT` / `SHIPPED` / `WORK_COMPLETED` / `CONFIRMED` / `AWAITING_CLAIM` / `CLAIMED` / `REFUNDED` / `CANCELLED`； |
| `cursor` | query | 否 | string | 长度 1–512； |
| `limit` | query | 否 | integer | 范围 1–100； |

响应 `200`：`V2OrderViewListResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": [
    {
      "orderId": "order_store_001",
      "orderNo": "DT202609080001",
      "channel": "OFFICIAL_STORE",
      "status": "AWAITING_CLAIM",
      "construction": false,
      "buyer": {
        "kind": "PLAYER",
        "playerRef": "player_xiwanzi",
        "storeId": null,
        "displayName": "Xiwanzi",
        "contactQq": "1000202"
      },
      "seller": {
        "kind": "OFFICIAL_STORE",
        "playerRef": null,
        "storeId": "store_official",
        "displayName": "Deuterium 官方商城",
        "contactQq": "1000000"
      },
      "items": [
        {
          "productId": "product_macmini",
          "productVersion": 1,
          "title": "Mac mini",
          "subtitle": "小巧机身，为桌面留出更多空间。",
          "description": "用于商城页面和下单流程测试的示例商品。",
          "unitPrice": "1399.00",
          "quantity": 1,
          "photoAssetIds": [
            "asset_cover_001",
            "asset_detail_002"
          ],
          "categoryName": "Mac",
          "includedItems": [
            "Mac mini 主题物品 × 1",
            "产品说明 × 1"
          ],
          "contentBlocks": [
            {
              "blockId": "intro",
              "type": "PARAGRAPH",
              "text": "了解商品内容和游戏邮箱交付说明。"
            }
          ]
        }
      ],
      "amount": "1399.00",
      "currency": "CREDIT",
      "delivery": {
        "method": "MAILBOX",
        "location": "Xiwanzi",
        "projectName": ""
      },
      "snapshotId": "snapshot_order_001",
      "snapshotSha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      "fundsStatus": "PAID",
      "confirmationHours": 168,
      "serverNow": "2026-09-08T02:00:00Z",
      "autoConfirmAt": null,
      "pausedRemainingSeconds": null,
      "refund": null,
      "refundAttemptsUsed": 0,
      "availableActions": [
        "VIEW_MAILBOX",
        "REQUEST_REFUND"
      ],
      "createdAt": "2026-09-07T01:00:00Z",
      "shippedAt": "2026-09-08T02:00:00Z",
      "workCompletedAt": null,
      "confirmedAt": null,
      "automatic": false,
      "version": 1
    }
  ],
  "serverTime": "2026-09-08T02:00:00Z",
  "page": {
    "nextCursor": null
  }
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/merchant/orders/{orderId}` — 查看官方订单与退款记录

实现状态：`contract_only`。权限：ORDER_MANAGE。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `orderId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

响应 `200`：`V2OrderViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "orderId": "order_store_001",
    "orderNo": "DT202609080001",
    "channel": "OFFICIAL_STORE",
    "status": "AWAITING_CLAIM",
    "construction": false,
    "buyer": {
      "kind": "PLAYER",
      "playerRef": "player_xiwanzi",
      "storeId": null,
      "displayName": "Xiwanzi",
      "contactQq": "1000202"
    },
    "seller": {
      "kind": "OFFICIAL_STORE",
      "playerRef": null,
      "storeId": "store_official",
      "displayName": "Deuterium 官方商城",
      "contactQq": "1000000"
    },
    "items": [
      {
        "productId": "product_macmini",
        "productVersion": 1,
        "title": "Mac mini",
        "subtitle": "小巧机身，为桌面留出更多空间。",
        "description": "用于商城页面和下单流程测试的示例商品。",
        "unitPrice": "1399.00",
        "quantity": 1,
        "photoAssetIds": [
          "asset_cover_001",
          "asset_detail_002"
        ],
        "categoryName": "Mac",
        "includedItems": [
          "Mac mini 主题物品 × 1",
          "产品说明 × 1"
        ],
        "contentBlocks": [
          {
            "blockId": "intro",
            "type": "PARAGRAPH",
            "text": "了解商品内容和游戏邮箱交付说明。"
          }
        ]
      }
    ],
    "amount": "1399.00",
    "currency": "CREDIT",
    "delivery": {
      "method": "MAILBOX",
      "location": "Xiwanzi",
      "projectName": ""
    },
    "snapshotId": "snapshot_order_001",
    "snapshotSha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "fundsStatus": "PAID",
    "confirmationHours": 168,
    "serverNow": "2026-09-08T02:00:00Z",
    "autoConfirmAt": null,
    "pausedRemainingSeconds": null,
    "refund": null,
    "refundAttemptsUsed": 0,
    "availableActions": [
      "VIEW_MAILBOX",
      "REQUEST_REFUND"
    ],
    "createdAt": "2026-09-07T01:00:00Z",
    "shippedAt": "2026-09-08T02:00:00Z",
    "workCompletedAt": null,
    "confirmedAt": null,
    "automatic": false,
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/merchant/orders/{orderId}/delivery-retry` — 重试失败的游戏邮箱发放

实现状态：`contract_only`。权限：ORDER_MANAGE。

仅发放失败/结果未知订单；先按原 deliveryId 查证，再幂等重试。禁止重新扣款或重复发物品。正常未领取退款仍自动处理，商家不能随意拒绝。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `orderId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`ReasonedMutationRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1,
  "reason": "已按双方确认的方案完成，请核对交付内容。"
}
```

响应 `200`：`V2OrderCreationResultResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "operation": {
      "operationId": "operation_demo_001",
      "clientRequestId": "a472c3b4-d76a-4693-9121-62443e6e7c49",
      "kind": "STORE_PURCHASE",
      "status": "COMPLETED",
      "resourceType": "ORDER",
      "resourceId": "order_store_001",
      "amount": "1399.00",
      "createdAt": "2026-09-08T02:00:00Z",
      "updatedAt": "2026-09-08T02:00:00Z",
      "errorCode": null,
      "retryAfterSeconds": 2
    },
    "order": {
      "orderId": "order_store_001",
      "orderNo": "DT202609080001",
      "channel": "OFFICIAL_STORE",
      "status": "AWAITING_CLAIM",
      "construction": false,
      "buyer": {
        "kind": "PLAYER",
        "playerRef": "player_xiwanzi",
        "storeId": null,
        "displayName": "Xiwanzi",
        "contactQq": "1000202"
      },
      "seller": {
        "kind": "OFFICIAL_STORE",
        "playerRef": null,
        "storeId": "store_official",
        "displayName": "Deuterium 官方商城",
        "contactQq": "1000000"
      },
      "items": [
        {
          "productId": "product_macmini",
          "productVersion": 1,
          "title": "Mac mini",
          "subtitle": "小巧机身，为桌面留出更多空间。",
          "description": "用于商城页面和下单流程测试的示例商品。",
          "unitPrice": "1399.00",
          "quantity": 1,
          "photoAssetIds": [
            "asset_cover_001",
            "asset_detail_002"
          ],
          "categoryName": "Mac",
          "includedItems": [
            "Mac mini 主题物品 × 1",
            "产品说明 × 1"
          ],
          "contentBlocks": [
            {
              "blockId": "intro",
              "type": "PARAGRAPH",
              "text": "了解商品内容和游戏邮箱交付说明。"
            }
          ]
        }
      ],
      "amount": "1399.00",
      "currency": "CREDIT",
      "delivery": {
        "method": "MAILBOX",
        "location": "Xiwanzi",
        "projectName": ""
      },
      "snapshotId": "snapshot_order_001",
      "snapshotSha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      "fundsStatus": "PAID",
      "confirmationHours": 168,
      "serverNow": "2026-09-08T02:00:00Z",
      "autoConfirmAt": null,
      "pausedRemainingSeconds": null,
      "refund": null,
      "refundAttemptsUsed": 0,
      "availableActions": [
        "VIEW_MAILBOX",
        "REQUEST_REFUND"
      ],
      "createdAt": "2026-09-07T01:00:00Z",
      "shippedAt": "2026-09-08T02:00:00Z",
      "workCompletedAt": null,
      "confirmedAt": null,
      "automatic": false,
      "version": 1
    }
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/merchant/products/{productId}` — 读取预填编辑表单及已发布版本

实现状态：`contract_only`。权限：PRODUCT_EDIT。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `productId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

响应 `200`：`V2MerchantProductResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "productId": "ref_demo_001",
    "storeId": "ref_demo_001",
    "version": 1,
    "draft": {
      "title": "Mac mini",
      "subtitle": "小巧机身，为桌面留出更多空间。",
      "description": "用于商城页面和下单流程测试的示例商品。",
      "brandId": "brand_apple",
      "categoryId": "category_mac",
      "price": "1399.00",
      "coverAssetId": "asset_cover_001",
      "galleryAssetIds": [
        "asset_cover_001",
        "asset_detail_002"
      ],
      "galleryAltTexts": [
        "Mac mini 正面",
        "Mac mini 背面接口"
      ],
      "contentBlocks": [
        {
          "blockId": "intro",
          "type": "PARAGRAPH",
          "text": "了解商品内容和游戏邮箱交付说明。"
        }
      ],
      "includedItems": [
        "Mac mini 主题物品 × 1",
        "产品说明 × 1"
      ],
      "deliveryTemplateRef": "delivery_template_macmini",
      "deliverySummary": "发送至本人游戏内邮箱",
      "estimatedDelivery": "支付完成后 1 分钟内",
      "inventoryPolicy": "FINITE",
      "stock": 50,
      "limitPerOrder": 9,
      "posterTone": "LIGHT",
      "accentColor": "#BBD6EC",
      "badges": [
        "新品"
      ],
      "sortOrder": 10
    },
    "published": null,
    "visibility": "DRAFT",
    "updatedAt": "2026-09-08T02:00:00Z"
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `PUT /api/v1/merchant/products/{productId}` — 更新所有商品内容字段

实现状态：`contract_only`。权限：PRODUCT_EDIT。

只更新草稿。价格、主图、画廊、详情块、包含内容、交付说明、限购、色调、排序均在 content 中。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `productId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`ProductWriteRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1,
  "content": {
    "title": "Mac mini",
    "subtitle": "小巧机身，为桌面留出更多空间。",
    "description": "用于商城页面和下单流程测试的示例商品。",
    "brandId": "brand_apple",
    "categoryId": "category_mac",
    "price": "1399.00",
    "coverAssetId": "asset_cover_001",
    "galleryAssetIds": [
      "asset_cover_001",
      "asset_detail_002"
    ],
    "galleryAltTexts": [
      "Mac mini 正面",
      "Mac mini 背面接口"
    ],
    "contentBlocks": [
      {
        "blockId": "intro",
        "type": "PARAGRAPH",
        "text": "了解商品内容和游戏邮箱交付说明。"
      }
    ],
    "includedItems": [
      "Mac mini 主题物品 × 1",
      "产品说明 × 1"
    ],
    "deliveryTemplateRef": "delivery_template_macmini",
    "deliverySummary": "发送至本人游戏内邮箱",
    "estimatedDelivery": "支付完成后 1 分钟内",
    "inventoryPolicy": "FINITE",
    "stock": 50,
    "limitPerOrder": 9,
    "posterTone": "LIGHT",
    "accentColor": "#BBD6EC",
    "badges": [
      "新品"
    ],
    "sortOrder": 10
  }
}
```

响应 `200`：`V2MerchantProductResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "productId": "ref_demo_001",
    "storeId": "ref_demo_001",
    "version": 1,
    "draft": {
      "title": "Mac mini",
      "subtitle": "小巧机身，为桌面留出更多空间。",
      "description": "用于商城页面和下单流程测试的示例商品。",
      "brandId": "brand_apple",
      "categoryId": "category_mac",
      "price": "1399.00",
      "coverAssetId": "asset_cover_001",
      "galleryAssetIds": [
        "asset_cover_001",
        "asset_detail_002"
      ],
      "galleryAltTexts": [
        "Mac mini 正面",
        "Mac mini 背面接口"
      ],
      "contentBlocks": [
        {
          "blockId": "intro",
          "type": "PARAGRAPH",
          "text": "了解商品内容和游戏邮箱交付说明。"
        }
      ],
      "includedItems": [
        "Mac mini 主题物品 × 1",
        "产品说明 × 1"
      ],
      "deliveryTemplateRef": "delivery_template_macmini",
      "deliverySummary": "发送至本人游戏内邮箱",
      "estimatedDelivery": "支付完成后 1 分钟内",
      "inventoryPolicy": "FINITE",
      "stock": 50,
      "limitPerOrder": 9,
      "posterTone": "LIGHT",
      "accentColor": "#BBD6EC",
      "badges": [
        "新品"
      ],
      "sortOrder": 10
    },
    "published": null,
    "visibility": "DRAFT",
    "updatedAt": "2026-09-08T02:00:00Z"
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/merchant/products/{productId}/archive` — 归档已下架商品

实现状态：`contract_only`。权限：PRODUCT_PUBLISH。

不删除历史订单和引用素材；仍须处理未完成履约。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `productId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`ReasonedMutationRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1,
  "reason": "已按双方确认的方案完成，请核对交付内容。"
}
```

响应 `200`：`V2MerchantProductResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "productId": "ref_demo_001",
    "storeId": "ref_demo_001",
    "version": 1,
    "draft": {
      "title": "Mac mini",
      "subtitle": "小巧机身，为桌面留出更多空间。",
      "description": "用于商城页面和下单流程测试的示例商品。",
      "brandId": "brand_apple",
      "categoryId": "category_mac",
      "price": "1399.00",
      "coverAssetId": "asset_cover_001",
      "galleryAssetIds": [
        "asset_cover_001",
        "asset_detail_002"
      ],
      "galleryAltTexts": [
        "Mac mini 正面",
        "Mac mini 背面接口"
      ],
      "contentBlocks": [
        {
          "blockId": "intro",
          "type": "PARAGRAPH",
          "text": "了解商品内容和游戏邮箱交付说明。"
        }
      ],
      "includedItems": [
        "Mac mini 主题物品 × 1",
        "产品说明 × 1"
      ],
      "deliveryTemplateRef": "delivery_template_macmini",
      "deliverySummary": "发送至本人游戏内邮箱",
      "estimatedDelivery": "支付完成后 1 分钟内",
      "inventoryPolicy": "FINITE",
      "stock": 50,
      "limitPerOrder": 9,
      "posterTone": "LIGHT",
      "accentColor": "#BBD6EC",
      "badges": [
        "新品"
      ],
      "sortOrder": 10
    },
    "published": null,
    "visibility": "DRAFT",
    "updatedAt": "2026-09-08T02:00:00Z"
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/merchant/products/{productId}/publish` — 校验并发布商品草稿

实现状态：`contract_only`。权限：PRODUCT_PUBLISH。

发布新版本，产生 catalog.changed 通知；原成交快照不变。服务端确认图片 READY、模板可用、价格和库存合法。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `productId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`MutationRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1
}
```

响应 `200`：`V2MerchantProductResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "productId": "ref_demo_001",
    "storeId": "ref_demo_001",
    "version": 1,
    "draft": {
      "title": "Mac mini",
      "subtitle": "小巧机身，为桌面留出更多空间。",
      "description": "用于商城页面和下单流程测试的示例商品。",
      "brandId": "brand_apple",
      "categoryId": "category_mac",
      "price": "1399.00",
      "coverAssetId": "asset_cover_001",
      "galleryAssetIds": [
        "asset_cover_001",
        "asset_detail_002"
      ],
      "galleryAltTexts": [
        "Mac mini 正面",
        "Mac mini 背面接口"
      ],
      "contentBlocks": [
        {
          "blockId": "intro",
          "type": "PARAGRAPH",
          "text": "了解商品内容和游戏邮箱交付说明。"
        }
      ],
      "includedItems": [
        "Mac mini 主题物品 × 1",
        "产品说明 × 1"
      ],
      "deliveryTemplateRef": "delivery_template_macmini",
      "deliverySummary": "发送至本人游戏内邮箱",
      "estimatedDelivery": "支付完成后 1 分钟内",
      "inventoryPolicy": "FINITE",
      "stock": 50,
      "limitPerOrder": 9,
      "posterTone": "LIGHT",
      "accentColor": "#BBD6EC",
      "badges": [
        "新品"
      ],
      "sortOrder": 10
    },
    "published": null,
    "visibility": "DRAFT",
    "updatedAt": "2026-09-08T02:00:00Z"
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/merchant/products/{productId}/stock-adjustments` — 调整实时可售库存并记录原因

实现状态：`contract_only`。权限：PRODUCT_EDIT。

不能侵占已预留库存；版本冲突先重新读取。调整不改旧订单金额。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `productId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`StockAdjustmentRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1,
  "delta": 1,
  "reason": "example"
}
```

响应 `200`：`V2MerchantProductResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "productId": "ref_demo_001",
    "storeId": "ref_demo_001",
    "version": 1,
    "draft": {
      "title": "Mac mini",
      "subtitle": "小巧机身，为桌面留出更多空间。",
      "description": "用于商城页面和下单流程测试的示例商品。",
      "brandId": "brand_apple",
      "categoryId": "category_mac",
      "price": "1399.00",
      "coverAssetId": "asset_cover_001",
      "galleryAssetIds": [
        "asset_cover_001",
        "asset_detail_002"
      ],
      "galleryAltTexts": [
        "Mac mini 正面",
        "Mac mini 背面接口"
      ],
      "contentBlocks": [
        {
          "blockId": "intro",
          "type": "PARAGRAPH",
          "text": "了解商品内容和游戏邮箱交付说明。"
        }
      ],
      "includedItems": [
        "Mac mini 主题物品 × 1",
        "产品说明 × 1"
      ],
      "deliveryTemplateRef": "delivery_template_macmini",
      "deliverySummary": "发送至本人游戏内邮箱",
      "estimatedDelivery": "支付完成后 1 分钟内",
      "inventoryPolicy": "FINITE",
      "stock": 50,
      "limitPerOrder": 9,
      "posterTone": "LIGHT",
      "accentColor": "#BBD6EC",
      "badges": [
        "新品"
      ],
      "sortOrder": 10
    },
    "published": null,
    "visibility": "DRAFT",
    "updatedAt": "2026-09-08T02:00:00Z"
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/merchant/products/{productId}/unlist` — 下架官方商品

实现状态：`contract_only`。权限：PRODUCT_PUBLISH。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `productId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`ReasonedMutationRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1,
  "reason": "已按双方确认的方案完成，请核对交付内容。"
}
```

响应 `200`：`V2MerchantProductResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "productId": "ref_demo_001",
    "storeId": "ref_demo_001",
    "version": 1,
    "draft": {
      "title": "Mac mini",
      "subtitle": "小巧机身，为桌面留出更多空间。",
      "description": "用于商城页面和下单流程测试的示例商品。",
      "brandId": "brand_apple",
      "categoryId": "category_mac",
      "price": "1399.00",
      "coverAssetId": "asset_cover_001",
      "galleryAssetIds": [
        "asset_cover_001",
        "asset_detail_002"
      ],
      "galleryAltTexts": [
        "Mac mini 正面",
        "Mac mini 背面接口"
      ],
      "contentBlocks": [
        {
          "blockId": "intro",
          "type": "PARAGRAPH",
          "text": "了解商品内容和游戏邮箱交付说明。"
        }
      ],
      "includedItems": [
        "Mac mini 主题物品 × 1",
        "产品说明 × 1"
      ],
      "deliveryTemplateRef": "delivery_template_macmini",
      "deliverySummary": "发送至本人游戏内邮箱",
      "estimatedDelivery": "支付完成后 1 分钟内",
      "inventoryPolicy": "FINITE",
      "stock": 50,
      "limitPerOrder": 9,
      "posterTone": "LIGHT",
      "accentColor": "#BBD6EC",
      "badges": [
        "新品"
      ],
      "sortOrder": 10
    },
    "published": null,
    "visibility": "DRAFT",
    "updatedAt": "2026-09-08T02:00:00Z"
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/merchant/stores/{storeId}` — 读取可编辑店铺资料

实现状态：`contract_only`。权限：STORE_EDIT。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `storeId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

响应 `200`：`V2StoreViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "storeId": "ref_demo_001",
    "name": "Deuterium 官方商城",
    "intro": "example",
    "logo": null,
    "cover": null,
    "contactQq": "1000000",
    "serviceHours": "example",
    "notice": "example",
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `PUT /api/v1/merchant/stores/{storeId}` — 编辑店铺全部展示资料

实现状态：`contract_only`。权限：STORE_EDIT。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `storeId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`StoreEditRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1,
  "name": "example",
  "intro": "example",
  "logoAssetId": null,
  "coverAssetId": null,
  "contactQq": "1000101",
  "serviceHours": "example",
  "notice": "example"
}
```

响应 `200`：`V2StoreViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "storeId": "ref_demo_001",
    "name": "Deuterium 官方商城",
    "intro": "example",
    "logo": null,
    "cover": null,
    "contactQq": "1000000",
    "serviceHours": "example",
    "notice": "example",
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/merchant/stores/{storeId}/brands` — 管理品牌列表

实现状态：`contract_only`。权限：PRODUCT_EDIT。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `storeId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |
| `cursor` | query | 否 | string | 长度 1–512； |
| `limit` | query | 否 | integer | 范围 1–100； |

响应 `200`：`V2StoreBrandListResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": [
    {
      "brandId": "ref_demo_001",
      "name": "Apple",
      "logoAssetId": null,
      "sortOrder": 1,
      "active": true,
      "version": 1
    }
  ],
  "serverTime": "2026-09-08T02:00:00Z",
  "page": {
    "nextCursor": null
  }
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/merchant/stores/{storeId}/brands` — 创建品牌

实现状态：`contract_only`。权限：PRODUCT_EDIT。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `storeId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`BrandCreateRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "name": "example",
  "logoAssetId": null,
  "sortOrder": 1,
  "active": true
}
```

响应 `201`：`V2StoreBrandResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "brandId": "ref_demo_001",
    "name": "Apple",
    "logoAssetId": null,
    "sortOrder": 1,
    "active": true,
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `PUT /api/v1/merchant/stores/{storeId}/brands/{entryId}` — 修改名称、图片或排序品牌

实现状态：`contract_only`。权限：PRODUCT_EDIT。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `storeId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |
| `entryId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`BrandWriteRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1,
  "name": "example",
  "logoAssetId": null,
  "sortOrder": 1,
  "active": true
}
```

响应 `200`：`V2StoreBrandResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "brandId": "ref_demo_001",
    "name": "Apple",
    "logoAssetId": null,
    "sortOrder": 1,
    "active": true,
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/merchant/stores/{storeId}/categories` — 管理商城分类列表

实现状态：`contract_only`。权限：PRODUCT_EDIT。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `storeId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |
| `cursor` | query | 否 | string | 长度 1–512； |
| `limit` | query | 否 | integer | 范围 1–100； |

响应 `200`：`V2StoreCategoryListResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": [
    {
      "categoryId": "ref_demo_001",
      "name": "Mac",
      "sortOrder": 1,
      "active": true,
      "version": 1
    }
  ],
  "serverTime": "2026-09-08T02:00:00Z",
  "page": {
    "nextCursor": null
  }
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/merchant/stores/{storeId}/categories` — 创建商城分类

实现状态：`contract_only`。权限：PRODUCT_EDIT。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `storeId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`CategoryCreateRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "name": "example",
  "sortOrder": 1,
  "active": true
}
```

响应 `201`：`V2StoreCategoryResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "categoryId": "ref_demo_001",
    "name": "Mac",
    "sortOrder": 1,
    "active": true,
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `PUT /api/v1/merchant/stores/{storeId}/categories/{entryId}` — 修改名称、图片或排序商城分类

实现状态：`contract_only`。权限：PRODUCT_EDIT。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `storeId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |
| `entryId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`CategoryWriteRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1,
  "name": "example",
  "sortOrder": 1,
  "active": true
}
```

响应 `200`：`V2StoreCategoryResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "categoryId": "ref_demo_001",
    "name": "Mac",
    "sortOrder": 1,
    "active": true,
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/merchant/stores/{storeId}/delivery-templates` — 列出可选游戏邮箱模板

实现状态：`contract_only`。权限：PRODUCT_EDIT。

只读服务端批准模板；商家网页不可提交任意服务器命令。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `storeId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |
| `cursor` | query | 否 | string | 长度 1–512； |
| `limit` | query | 否 | integer | 范围 1–100； |

响应 `200`：`V2DeliveryTemplateListResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": [
    {
      "templateRef": "ref_demo_001",
      "name": "example",
      "summary": "example",
      "active": true
    }
  ],
  "serverTime": "2026-09-08T02:00:00Z",
  "page": {
    "nextCursor": null
  }
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/merchant/stores/{storeId}/homepage` — 读取首页编排

实现状态：`contract_only`。权限：STORE_EDIT。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `storeId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

响应 `200`：`V2StoreHomepageResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "storeId": "ref_demo_001",
    "intro": "example",
    "sections": [
      {
        "sectionId": "ref_demo_001",
        "title": "example",
        "layout": "HERO_CAROUSEL",
        "productIds": [
          "ref_demo_001"
        ],
        "bannerAssetId": null,
        "sortOrder": 1
      }
    ],
    "brandIds": [
      "ref_demo_001"
    ],
    "categoryIds": [
      "ref_demo_001"
    ],
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `PUT /api/v1/merchant/stores/{storeId}/homepage` — 编辑轮播、网格、品牌和分类顺序

实现状态：`contract_only`。权限：STORE_EDIT。

只能引用同店铺已发布商品/有权媒体；纯结构化内容，不接受网页代码。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `storeId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`StoreHomepageWriteRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1,
  "intro": "example",
  "sections": [
    {
      "sectionId": "ref_demo_001",
      "title": "example",
      "layout": "HERO_CAROUSEL",
      "productIds": [
        "ref_demo_001"
      ],
      "bannerAssetId": null,
      "sortOrder": 1
    }
  ],
  "brandIds": [
    "ref_demo_001"
  ],
  "categoryIds": [
    "ref_demo_001"
  ]
}
```

响应 `200`：`V2StoreHomepageResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "storeId": "ref_demo_001",
    "intro": "example",
    "sections": [
      {
        "sectionId": "ref_demo_001",
        "title": "example",
        "layout": "HERO_CAROUSEL",
        "productIds": [
          "ref_demo_001"
        ],
        "bannerAssetId": null,
        "sortOrder": 1
      }
    ],
    "brandIds": [
      "ref_demo_001"
    ],
    "categoryIds": [
      "ref_demo_001"
    ],
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/merchant/stores/{storeId}/products` — 管理商品、草稿和上下架状态

实现状态：`contract_only`。权限：PRODUCT_EDIT。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `storeId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |
| `visibility` | query | 否 | string | `DRAFT` / `ACTIVE` / `UNLISTED` / `ARCHIVED`； |
| `q` | query | 否 | string | 长度 1–80； |
| `cursor` | query | 否 | string | 长度 1–512； |
| `limit` | query | 否 | integer | 范围 1–100； |

响应 `200`：`V2MerchantProductListResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": [
    {
      "productId": "ref_demo_001",
      "storeId": "ref_demo_001",
      "version": 1,
      "draft": {
        "title": "Mac mini",
        "subtitle": "小巧机身，为桌面留出更多空间。",
        "description": "用于商城页面和下单流程测试的示例商品。",
        "brandId": "brand_apple",
        "categoryId": "category_mac",
        "price": "1399.00",
        "coverAssetId": "asset_cover_001",
        "galleryAssetIds": [
          "asset_cover_001",
          "asset_detail_002"
        ],
        "galleryAltTexts": [
          "Mac mini 正面",
          "Mac mini 背面接口"
        ],
        "contentBlocks": [
          {
            "blockId": "intro",
            "type": "PARAGRAPH",
            "text": "了解商品内容和游戏邮箱交付说明。"
          }
        ],
        "includedItems": [
          "Mac mini 主题物品 × 1",
          "产品说明 × 1"
        ],
        "deliveryTemplateRef": "delivery_template_macmini",
        "deliverySummary": "发送至本人游戏内邮箱",
        "estimatedDelivery": "支付完成后 1 分钟内",
        "inventoryPolicy": "FINITE",
        "stock": 50,
        "limitPerOrder": 9,
        "posterTone": "LIGHT",
        "accentColor": "#BBD6EC",
        "badges": [
          "新品"
        ],
        "sortOrder": 10
      },
      "published": null,
      "visibility": "DRAFT",
      "updatedAt": "2026-09-08T02:00:00Z"
    }
  ],
  "serverTime": "2026-09-08T02:00:00Z",
  "page": {
    "nextCursor": null
  }
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/merchant/stores/{storeId}/products` — 新建商品完整草稿

实现状态：`contract_only`。权限：PRODUCT_EDIT。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `storeId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`ProductCreateRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "content": {
    "title": "Mac mini",
    "subtitle": "小巧机身，为桌面留出更多空间。",
    "description": "用于商城页面和下单流程测试的示例商品。",
    "brandId": "brand_apple",
    "categoryId": "category_mac",
    "price": "1399.00",
    "coverAssetId": "asset_cover_001",
    "galleryAssetIds": [
      "asset_cover_001",
      "asset_detail_002"
    ],
    "galleryAltTexts": [
      "Mac mini 正面",
      "Mac mini 背面接口"
    ],
    "contentBlocks": [
      {
        "blockId": "intro",
        "type": "PARAGRAPH",
        "text": "了解商品内容和游戏邮箱交付说明。"
      }
    ],
    "includedItems": [
      "Mac mini 主题物品 × 1",
      "产品说明 × 1"
    ],
    "deliveryTemplateRef": "delivery_template_macmini",
    "deliverySummary": "发送至本人游戏内邮箱",
    "estimatedDelivery": "支付完成后 1 分钟内",
    "inventoryPolicy": "FINITE",
    "stock": 50,
    "limitPerOrder": 9,
    "posterTone": "LIGHT",
    "accentColor": "#BBD6EC",
    "badges": [
      "新品"
    ],
    "sortOrder": 10
  }
}
```

响应 `201`：`V2MerchantProductResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "productId": "ref_demo_001",
    "storeId": "ref_demo_001",
    "version": 1,
    "draft": {
      "title": "Mac mini",
      "subtitle": "小巧机身，为桌面留出更多空间。",
      "description": "用于商城页面和下单流程测试的示例商品。",
      "brandId": "brand_apple",
      "categoryId": "category_mac",
      "price": "1399.00",
      "coverAssetId": "asset_cover_001",
      "galleryAssetIds": [
        "asset_cover_001",
        "asset_detail_002"
      ],
      "galleryAltTexts": [
        "Mac mini 正面",
        "Mac mini 背面接口"
      ],
      "contentBlocks": [
        {
          "blockId": "intro",
          "type": "PARAGRAPH",
          "text": "了解商品内容和游戏邮箱交付说明。"
        }
      ],
      "includedItems": [
        "Mac mini 主题物品 × 1",
        "产品说明 × 1"
      ],
      "deliveryTemplateRef": "delivery_template_macmini",
      "deliverySummary": "发送至本人游戏内邮箱",
      "estimatedDelivery": "支付完成后 1 分钟内",
      "inventoryPolicy": "FINITE",
      "stock": 50,
      "limitPerOrder": 9,
      "posterTone": "LIGHT",
      "accentColor": "#BBD6EC",
      "badges": [
        "新品"
      ],
      "sortOrder": 10
    },
    "published": null,
    "visibility": "DRAFT",
    "updatedAt": "2026-09-08T02:00:00Z"
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

### 平台管理端

#### `GET /api/v1/admin/announcements` — 管理公告草稿和已发布公告

实现状态：`contract_only`。权限：ANNOUNCEMENT_EDIT。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `cursor` | query | 否 | string | 长度 1–512； |
| `limit` | query | 否 | integer | 范围 1–100； |

响应 `200`：`V2AnnouncementViewListResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": [
    {
      "announcementId": "ref_demo_001",
      "title": "example",
      "summary": "example",
      "contentBlocks": [
        {
          "blockId": "ref_demo_001",
          "type": "HEADING",
          "heading": "example",
          "text": "example",
          "assetId": null,
          "altText": "example",
          "rows": [
            {
              "label": "example",
              "value": "example"
            }
          ]
        }
      ],
      "coverAssetId": null,
      "pinned": false,
      "priority": "NORMAL",
      "publishedAt": "2026-09-08T02:00:00Z",
      "updatedAt": "2026-09-08T02:00:00Z",
      "version": 1
    }
  ],
  "serverTime": "2026-09-08T02:00:00Z",
  "page": {
    "nextCursor": null
  }
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/admin/announcements` — 创建官方公告草稿

实现状态：`contract_only`。权限：ANNOUNCEMENT_EDIT。

请求模型：`AnnouncementCreateRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "title": "example",
  "summary": "example",
  "contentBlocks": [
    {
      "blockId": "ref_demo_001",
      "type": "HEADING",
      "heading": "example",
      "text": "example",
      "assetId": null,
      "altText": "example",
      "rows": [
        {
          "label": "example",
          "value": "example"
        }
      ]
    }
  ],
  "coverAssetId": null,
  "pinned": false,
  "priority": "NORMAL"
}
```

响应 `201`：`V2AnnouncementViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "announcementId": "ref_demo_001",
    "title": "example",
    "summary": "example",
    "contentBlocks": [
      {
        "blockId": "ref_demo_001",
        "type": "HEADING",
        "heading": "example",
        "text": "example",
        "assetId": null,
        "altText": "example",
        "rows": [
          {
            "label": "example",
            "value": "example"
          }
        ]
      }
    ],
    "coverAssetId": null,
    "pinned": false,
    "priority": "NORMAL",
    "publishedAt": "2026-09-08T02:00:00Z",
    "updatedAt": "2026-09-08T02:00:00Z",
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `PUT /api/v1/admin/announcements/{announcementId}` — 编辑公告标题、摘要、封面和正文

实现状态：`contract_only`。权限：ANNOUNCEMENT_EDIT。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `announcementId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`AnnouncementWriteRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1,
  "title": "example",
  "summary": "example",
  "contentBlocks": [
    {
      "blockId": "ref_demo_001",
      "type": "HEADING",
      "heading": "example",
      "text": "example",
      "assetId": null,
      "altText": "example",
      "rows": [
        {
          "label": "example",
          "value": "example"
        }
      ]
    }
  ],
  "coverAssetId": null,
  "pinned": false,
  "priority": "NORMAL"
}
```

响应 `200`：`V2AnnouncementViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "announcementId": "ref_demo_001",
    "title": "example",
    "summary": "example",
    "contentBlocks": [
      {
        "blockId": "ref_demo_001",
        "type": "HEADING",
        "heading": "example",
        "text": "example",
        "assetId": null,
        "altText": "example",
        "rows": [
          {
            "label": "example",
            "value": "example"
          }
        ]
      }
    ],
    "coverAssetId": null,
    "pinned": false,
    "priority": "NORMAL",
    "publishedAt": "2026-09-08T02:00:00Z",
    "updatedAt": "2026-09-08T02:00:00Z",
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/admin/announcements/{announcementId}/publish` — 发布公告并发送业务通知

实现状态：`contract_only`。权限：ANNOUNCEMENT_PUBLISH。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `announcementId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`MutationRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1
}
```

响应 `200`：`V2AnnouncementViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "announcementId": "ref_demo_001",
    "title": "example",
    "summary": "example",
    "contentBlocks": [
      {
        "blockId": "ref_demo_001",
        "type": "HEADING",
        "heading": "example",
        "text": "example",
        "assetId": null,
        "altText": "example",
        "rows": [
          {
            "label": "example",
            "value": "example"
          }
        ]
      }
    ],
    "coverAssetId": null,
    "pinned": false,
    "priority": "NORMAL",
    "publishedAt": "2026-09-08T02:00:00Z",
    "updatedAt": "2026-09-08T02:00:00Z",
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/admin/announcements/{announcementId}/unpublish` — 撤下公告

实现状态：`contract_only`。权限：ANNOUNCEMENT_PUBLISH。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `announcementId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`ReasonedMutationRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1,
  "reason": "已按双方确认的方案完成，请核对交付内容。"
}
```

响应 `200`：`V2AnnouncementViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "announcementId": "ref_demo_001",
    "title": "example",
    "summary": "example",
    "contentBlocks": [
      {
        "blockId": "ref_demo_001",
        "type": "HEADING",
        "heading": "example",
        "text": "example",
        "assetId": null,
        "altText": "example",
        "rows": [
          {
            "label": "example",
            "value": "example"
          }
        ]
      }
    ],
    "coverAssetId": null,
    "pinned": false,
    "priority": "NORMAL",
    "publishedAt": "2026-09-08T02:00:00Z",
    "updatedAt": "2026-09-08T02:00:00Z",
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/admin/appearance/global` — 读取全局外观配置管理记录

实现状态：`contract_only`。权限：APPEARANCE_READ。

平台管理员按包名/频道授权读取。记录不存在时 version=0，三组覆盖均 null，使用内置默认。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `packageName` | query | 是 | string | 长度 1–200； |
| `channel` | query | 是 | string | `stable` / `preview`； |

响应 `200`：`V2AppearanceAdminRecordResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "scope": "GLOBAL",
    "playerRef": null,
    "packageName": "com.deuterium.app.uilab",
    "channel": "stable",
    "version": 0,
    "overrides": {
      "bottomBarGlass": null,
      "overlayGlass": null,
      "headerGradient": null
    },
    "updatedAt": null
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `PATCH /api/v1/admin/appearance/global` — 修改或重置全局外观参数

实现状态：`contract_only`。权限：APPEARANCE_MANAGE。

仅更新正文出现的组；expectedVersion 对应全局记录，省略不变、null 重置到内置默认。原子校验并写入新版本和审计，200 只代表配置已保存；不能声称在线/离线设备均已生效。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `packageName` | query | 是 | string | 长度 1–200； |
| `channel` | query | 是 | string | `stable` / `preview`； |

请求模型：`AppearancePatchRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "appearance_patch_001",
  "expectedVersion": 0,
  "reason": "仅调整浮层参数，底栏保持原值",
  "overlayGlass": {
    "enabled": true,
    "blurRadiusDp": 24,
    "opacity": 0.7,
    "refractionStrength": 4,
    "highlightStrength": 0.35,
    "dynamicHighlightEnabled": true,
    "allowLocalTuning": false
  }
}
```

响应 `200`：`V2AppearanceAdminRecordResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "scope": "GLOBAL",
    "playerRef": null,
    "packageName": "com.deuterium.app.uilab",
    "channel": "stable",
    "version": 1,
    "overrides": {
      "bottomBarGlass": null,
      "overlayGlass": {
        "enabled": true,
        "blurRadiusDp": 24,
        "opacity": 0.7,
        "refractionStrength": 4,
        "highlightStrength": 0.35,
        "dynamicHighlightEnabled": true,
        "allowLocalTuning": false
      },
      "headerGradient": null
    },
    "updatedAt": "2026-09-08T02:00:00Z"
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/admin/appearance/players/{playerRef}` — 读取指定玩家的外观覆盖记录

实现状态：`contract_only`。权限：APPEARANCE_READ。

目标玩家须存在，且管理员有包名/频道和玩家配置权限；不存在的配置记录返回 version=0 与全 null 覆盖，不等于玩家不存在。普通商家不可调用。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `playerRef` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |
| `packageName` | query | 是 | string | 长度 1–200； |
| `channel` | query | 是 | string | `stable` / `preview`； |

响应 `200`：`V2AppearanceAdminRecordResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "scope": "PLAYER",
    "playerRef": "player_xiwanzi",
    "packageName": "com.deuterium.app.uilab",
    "channel": "stable",
    "version": 0,
    "overrides": {
      "bottomBarGlass": null,
      "overlayGlass": null,
      "headerGradient": null
    },
    "updatedAt": null
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `PATCH /api/v1/admin/appearance/players/{playerRef}` — 修改或重置指定玩家的外观参数

实现状态：`contract_only`。权限：APPEARANCE_MANAGE。

每组独立覆盖；expectedVersion 对应目标玩家管理记录。只传 overlayGlass 不改变其 bottomBarGlass/headerGradient；null 清除此组覆盖，恢复全局/内置继承。重置后保留递增版本防止旧写入复活。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `playerRef` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |
| `packageName` | query | 是 | string | 长度 1–200； |
| `channel` | query | 是 | string | `stable` / `preview`； |

请求模型：`AppearancePatchRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "appearance_patch_001",
  "expectedVersion": 0,
  "reason": "仅调整浮层参数，底栏保持原值",
  "overlayGlass": {
    "enabled": true,
    "blurRadiusDp": 24,
    "opacity": 0.7,
    "refractionStrength": 4,
    "highlightStrength": 0.35,
    "dynamicHighlightEnabled": true,
    "allowLocalTuning": false
  }
}
```

响应 `200`：`V2AppearanceAdminRecordResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "scope": "PLAYER",
    "playerRef": "player_xiwanzi",
    "packageName": "com.deuterium.app.uilab",
    "channel": "stable",
    "version": 1,
    "overrides": {
      "bottomBarGlass": null,
      "overlayGlass": {
        "enabled": true,
        "blurRadiusDp": 24,
        "opacity": 0.7,
        "refractionStrength": 4,
        "highlightStrength": 0.35,
        "dynamicHighlightEnabled": true,
        "allowLocalTuning": false
      },
      "headerGradient": null
    },
    "updatedAt": "2026-09-08T02:00:00Z"
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/admin/audit-events` — 查询后台重要操作审计

实现状态：`contract_only`。权限：AUDIT_READ。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `resourceRef` | query | 否 | string | 长度 1–128； |
| `from` | query | 否 | string | date-time； |
| `to` | query | 否 | string | date-time； |
| `cursor` | query | 否 | string | 长度 1–512； |
| `limit` | query | 否 | integer | 范围 1–100； |

响应 `200`：`V2AuditEventListResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": [
    {
      "auditId": "ref_demo_001",
      "actorRef": "ref_demo_001",
      "permissionUsed": "example",
      "action": "example",
      "resourceRef": "ref_demo_001",
      "requestId": "ref_demo_001",
      "occurredAt": "2026-09-08T02:00:00Z",
      "summary": "example"
    }
  ],
  "serverTime": "2026-09-08T02:00:00Z",
  "page": {
    "nextCursor": null
  }
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/admin/interventions` — 管理员查询待处理争议

实现状态：`contract_only`。权限：DISPUTE_READ。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `status` | query | 否 | string | `SUBMITTED` / `IN_REVIEW` / `WAITING_EVIDENCE` / `RESOLVING` / `RESOLVED` / `WITHDRAWN`； |
| `assignedToMe` | query | 否 | boolean | ； |
| `transactionId` | query | 否 | string | 长度 1–128； |
| `cursor` | query | 否 | string | 长度 1–512； |
| `limit` | query | 否 | integer | 范围 1–100； |

响应 `200`：`V2InterventionViewListResponse`。

完整 JSON 示例见 `api-v2-examples.json` 中同 method/path 条目；模型字段全部在下文展开。

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/admin/interventions/{caseId}` — 读取争议详情与权威交易快照

实现状态：`contract_only`。权限：DISPUTE_READ。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `caseId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

响应 `200`：`V2InterventionViewResponse`。

完整 JSON 示例见 `api-v2-examples.json` 中同 method/path 条目；模型字段全部在下文展开。

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/admin/interventions/{caseId}/assign` — 领取争议处理任务

实现状态：`contract_only`。权限：DISPUTE_HANDLE。

服务端将当前管理员设为处理人，不能由请求伪造管理员身份。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `caseId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`MutationRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1
}
```

响应 `200`：`V2InterventionViewResponse`。

完整 JSON 示例见 `api-v2-examples.json` 中同 method/path 条目；模型字段全部在下文展开。

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/admin/interventions/{caseId}/request-evidence` — 要求参与方补充说明

实现状态：`contract_only`。权限：DISPUTE_HANDLE。

产生定向通知，保留审计；不因索要证据自动退款或自动结算。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `caseId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`ReasonedMutationRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1,
  "reason": "已按双方确认的方案完成，请核对交付内容。"
}
```

响应 `200`：`V2InterventionViewResponse`。

完整 JSON 示例见 `api-v2-examples.json` 中同 method/path 条目；模型字段全部在下文展开。

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/admin/interventions/{caseId}/resolve` — 作出裁决并编排资金处理

实现状态：`contract_only`。权限：DISPUTE_RESOLVE。

需要分配到本人且版本一致。资金动作先进入 RESOLVING，实际回执完成后 RESOLVED。裁决不能凭网页传来的金额直接写余额。已释放的款项只能人工追偿，不强制透支。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `caseId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`InterventionResolutionRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1,
  "decision": "FULL_REFUND",
  "refundAmount": "680.00",
  "reason": "example-va"
}
```

响应 `200`：`V2InterventionViewResponse`。

完整 JSON 示例见 `api-v2-examples.json` 中同 method/path 条目；模型字段全部在下文展开。

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/admin/me` — 读取平台管理授权

实现状态：`contract_only`。权限：ADMIN_LOGIN。

响应 `200`：`V2AdminIdentityResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "playerRef": "ref_demo_001",
    "permissions": [
      "example"
    ]
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/admin/releases` — 查询应用和资源发布记录

实现状态：`contract_only`。权限：RELEASE_MANAGE。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `kind` | query | 否 | string | `APK` / `RESOURCES`； |
| `channel` | query | 否 | string | `stable` / `preview`； |
| `cursor` | query | 否 | string | 长度 1–512； |
| `limit` | query | 否 | integer | 范围 1–100； |

响应 `200`：`V2ReleaseAdminViewListResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": [
    {
      "releaseId": "ref_demo_001",
      "version": 1,
      "status": "DRAFT",
      "artifact": {
        "artifactRef": "ref_demo_001",
        "kind": "APK",
        "status": "PROCESSING",
        "sizeBytes": 1,
        "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
        "packageName": "example",
        "versionCode": 1,
        "resourceVersion": 1,
        "validationMessages": [
          "example"
        ]
      },
      "release": {
        "releaseId": "release_apk_8",
        "kind": "APK",
        "versionName": "1.0.6",
        "versionCode": 8,
        "resourceVersion": 0,
        "packageName": "com.deuterium.app",
        "releaseNotes": "优化委托履约与交易通知体验。",
        "downloadUrl": "https://cdn.example.invalid/releases/deuterium-1.0.6.apk",
        "sizeBytes": 30000000,
        "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
        "minAppVersionCode": 7,
        "maxAppVersionCode": 99,
        "publishedAt": "2026-09-08T02:00:00Z",
        "channel": "stable"
      },
      "createdAt": "2026-09-08T02:00:00Z",
      "updatedAt": "2026-09-08T02:00:00Z"
    }
  ],
  "serverTime": "2026-09-08T02:00:00Z",
  "page": {
    "nextCursor": null
  }
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/admin/releases` — 创建更新发布草稿

实现状态：`contract_only`。权限：RELEASE_MANAGE。

请求模型：`ReleaseCreateRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "artifactRef": "ref_demo_001",
  "kind": "APK",
  "versionName": "example",
  "releaseNotes": "example",
  "channel": "stable",
  "minAppVersionCode": 1,
  "maxAppVersionCode": 1
}
```

响应 `201`：`V2ReleaseAdminViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "releaseId": "ref_demo_001",
    "version": 1,
    "status": "DRAFT",
    "artifact": {
      "artifactRef": "ref_demo_001",
      "kind": "APK",
      "status": "PROCESSING",
      "sizeBytes": 1,
      "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      "packageName": "example",
      "versionCode": 1,
      "resourceVersion": 1,
      "validationMessages": [
        "example"
      ]
    },
    "release": {
      "releaseId": "release_apk_8",
      "kind": "APK",
      "versionName": "1.0.6",
      "versionCode": 8,
      "resourceVersion": 0,
      "packageName": "com.deuterium.app",
      "releaseNotes": "优化委托履约与交易通知体验。",
      "downloadUrl": "https://cdn.example.invalid/releases/deuterium-1.0.6.apk",
      "sizeBytes": 30000000,
      "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      "minAppVersionCode": 7,
      "maxAppVersionCode": 99,
      "publishedAt": "2026-09-08T02:00:00Z",
      "channel": "stable"
    },
    "createdAt": "2026-09-08T02:00:00Z",
    "updatedAt": "2026-09-08T02:00:00Z"
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/admin/releases/artifacts` — 上传并检查 APK 或资源更新包

实现状态：`contract_only`。权限：RELEASE_MANAGE。

核验包名/签名/版本、资源清单、哈希、路径和容量后才成为 READY。普通商家上传接口不能上传执行文件。

请求模型：`object`，Content-Type：`multipart/form-data`。字段见模型参考。

按 multipart/form-data 上传实际 file；不要把文件路径或 Base64 当成普通 JSON 上传。

| 表单字段 | 类型 | 必填 | 约束/说明 |
| --- | --- | --- | --- |
| `kind` | string | 是 | `APK` / `RESOURCES`； |
| `file` | string | 是 | binary；APK 最大 200 MiB；资源 ZIP 最大 20 MiB。服务器计算摘要并检查格式。 |

响应 `201`：`V2ReleaseArtifactResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "artifactRef": "ref_demo_001",
    "kind": "APK",
    "status": "PROCESSING",
    "sizeBytes": 1,
    "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "packageName": "example",
    "versionCode": 1,
    "resourceVersion": 1,
    "validationMessages": [
      "example"
    ]
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/admin/releases/artifacts/{artifactRef}` — 查询更新包验证结果

实现状态：`contract_only`。权限：RELEASE_MANAGE。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `artifactRef` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

响应 `200`：`V2ReleaseArtifactResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "artifactRef": "ref_demo_001",
    "kind": "APK",
    "status": "PROCESSING",
    "sizeBytes": 1,
    "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "packageName": "example",
    "versionCode": 1,
    "resourceVersion": 1,
    "validationMessages": [
      "example"
    ]
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `PUT /api/v1/admin/releases/{releaseId}` — 修改更新说明与兼容范围

实现状态：`contract_only`。权限：RELEASE_MANAGE。

已发布文件不可原地替换；修改二进制必须新建 artifactRef 和递增版本。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `releaseId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`ReleaseWriteRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1,
  "artifactRef": "ref_demo_001",
  "kind": "APK",
  "versionName": "example",
  "releaseNotes": "example",
  "channel": "stable",
  "minAppVersionCode": 1,
  "maxAppVersionCode": 1
}
```

响应 `200`：`V2ReleaseAdminViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "releaseId": "ref_demo_001",
    "version": 1,
    "status": "DRAFT",
    "artifact": {
      "artifactRef": "ref_demo_001",
      "kind": "APK",
      "status": "PROCESSING",
      "sizeBytes": 1,
      "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      "packageName": "example",
      "versionCode": 1,
      "resourceVersion": 1,
      "validationMessages": [
        "example"
      ]
    },
    "release": {
      "releaseId": "release_apk_8",
      "kind": "APK",
      "versionName": "1.0.6",
      "versionCode": 8,
      "resourceVersion": 0,
      "packageName": "com.deuterium.app",
      "releaseNotes": "优化委托履约与交易通知体验。",
      "downloadUrl": "https://cdn.example.invalid/releases/deuterium-1.0.6.apk",
      "sizeBytes": 30000000,
      "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      "minAppVersionCode": 7,
      "maxAppVersionCode": 99,
      "publishedAt": "2026-09-08T02:00:00Z",
      "channel": "stable"
    },
    "createdAt": "2026-09-08T02:00:00Z",
    "updatedAt": "2026-09-08T02:00:00Z"
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/admin/releases/{releaseId}/publish` — 发布可供 App 检查的更新

实现状态：`contract_only`。权限：RELEASE_PUBLISH。

原子更新版本索引；版本不回退，哈希由验证过程产生。兼容范围、包名和频道隔离。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `releaseId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`MutationRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1
}
```

响应 `200`：`V2ReleaseAdminViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "releaseId": "ref_demo_001",
    "version": 1,
    "status": "DRAFT",
    "artifact": {
      "artifactRef": "ref_demo_001",
      "kind": "APK",
      "status": "PROCESSING",
      "sizeBytes": 1,
      "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      "packageName": "example",
      "versionCode": 1,
      "resourceVersion": 1,
      "validationMessages": [
        "example"
      ]
    },
    "release": {
      "releaseId": "release_apk_8",
      "kind": "APK",
      "versionName": "1.0.6",
      "versionCode": 8,
      "resourceVersion": 0,
      "packageName": "com.deuterium.app",
      "releaseNotes": "优化委托履约与交易通知体验。",
      "downloadUrl": "https://cdn.example.invalid/releases/deuterium-1.0.6.apk",
      "sizeBytes": 30000000,
      "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      "minAppVersionCode": 7,
      "maxAppVersionCode": 99,
      "publishedAt": "2026-09-08T02:00:00Z",
      "channel": "stable"
    },
    "createdAt": "2026-09-08T02:00:00Z",
    "updatedAt": "2026-09-08T02:00:00Z"
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/admin/releases/{releaseId}/withdraw` — 撤回问题更新的推荐

实现状态：`contract_only`。权限：RELEASE_PUBLISH。

仅停止推荐，不强制降级用户 APK。资源回退应把旧内容发布成更高资源版本，用户端仍保留上一有效版本作失败恢复。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `releaseId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`ReasonedMutationRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1,
  "reason": "已按双方确认的方案完成，请核对交付内容。"
}
```

响应 `200`：`V2ReleaseAdminViewResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "releaseId": "ref_demo_001",
    "version": 1,
    "status": "DRAFT",
    "artifact": {
      "artifactRef": "ref_demo_001",
      "kind": "APK",
      "status": "PROCESSING",
      "sizeBytes": 1,
      "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      "packageName": "example",
      "versionCode": 1,
      "resourceVersion": 1,
      "validationMessages": [
        "example"
      ]
    },
    "release": {
      "releaseId": "release_apk_8",
      "kind": "APK",
      "versionName": "1.0.6",
      "versionCode": 8,
      "resourceVersion": 0,
      "packageName": "com.deuterium.app",
      "releaseNotes": "优化委托履约与交易通知体验。",
      "downloadUrl": "https://cdn.example.invalid/releases/deuterium-1.0.6.apk",
      "sizeBytes": 30000000,
      "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      "minAppVersionCode": 7,
      "maxAppVersionCode": 99,
      "publishedAt": "2026-09-08T02:00:00Z",
      "channel": "stable"
    },
    "createdAt": "2026-09-08T02:00:00Z",
    "updatedAt": "2026-09-08T02:00:00Z"
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/admin/stores/{storeId}/members` — 管理店铺成员授权

实现状态：`contract_only`。权限：MERCHANT_MANAGE。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `storeId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |
| `cursor` | query | 否 | string | 长度 1–512； |
| `limit` | query | 否 | integer | 范围 1–100； |

响应 `200`：`V2StoreMemberListResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": [
    {
      "playerRef": "ref_demo_001",
      "permissions": [
        "STORE_EDIT"
      ],
      "version": 1
    }
  ],
  "serverTime": "2026-09-08T02:00:00Z",
  "page": {
    "nextCursor": null
  }
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/admin/stores/{storeId}/members` — 授予或更新店铺成员权限

实现状态：`contract_only`。权限：MERCHANT_MANAGE。

只允许管理员。普通登录接口不接受 role/permissions 字段来提升权限。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `storeId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`StoreMemberWriteRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1,
  "playerRef": "ref_demo_001",
  "permissions": [
    "STORE_EDIT"
  ]
}
```

响应 `200`：`V2StoreMemberResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "playerRef": "ref_demo_001",
    "permissions": [
      "STORE_EDIT"
    ],
    "version": 1
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `POST /api/v1/admin/stores/{storeId}/members/{playerRef}/revoke` — 撤销店铺成员权限

实现状态：`contract_only`。权限：MERCHANT_MANAGE。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `storeId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |
| `playerRef` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

请求模型：`ReasonedMutationRequest`，Content-Type：`application/json`。字段见模型参考。

```json
{
  "clientRequestId": "ref_demo_001",
  "expectedVersion": 1,
  "reason": "已按双方确认的方案完成，请核对交付内容。"
}
```

响应 `200`：`V2RemovalResultResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "removed": true
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

### 资金操作查询

#### `GET /api/v1/operations/by-client-request` — 按幂等键恢复未收到编号的资金操作

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

用于请求响应丢失。查询范围自动绑定当前账号；同键重试也必须返回原操作。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | query | 是 | string | 长度 1–128； |
| `kind` | query | 是 | string | `STORE_PURCHASE` / `MARKET_PURCHASE` / `COMMISSION_PUBLISH` / `REFUND` / `SETTLEMENT` / `TRANSFER` / `AI_PURCHASE`； |

响应 `200`：`V2OperationLookupResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "operationId": "operation_demo_001",
    "clientRequestId": "a472c3b4-d76a-4693-9121-62443e6e7c49",
    "kind": "MARKET_PURCHASE",
    "status": "COMPLETED",
    "resourceType": "ORDER",
    "resourceId": "order_demo_001",
    "amount": "1800.00",
    "createdAt": "2026-09-08T02:00:00Z",
    "updatedAt": "2026-09-08T02:00:00Z",
    "errorCode": null,
    "retryAfterSeconds": 2
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

#### `GET /api/v1/operations/{operationId}` — 查询资金操作状态

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

只允许参与方查询。PROCESSING/UNKNOWN 时保持等待或查账，不得显示为已成功或新建第二笔操作。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `operationId` | path | 是 | string | 长度 1–128；服务端生成的不透明引用 |

响应 `200`：`V2OperationLookupResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "operationId": "operation_demo_001",
    "clientRequestId": "a472c3b4-d76a-4693-9121-62443e6e7c49",
    "kind": "MARKET_PURCHASE",
    "status": "COMPLETED",
    "resourceType": "ORDER",
    "resourceId": "order_demo_001",
    "amount": "1800.00",
    "createdAt": "2026-09-08T02:00:00Z",
    "updatedAt": "2026-09-08T02:00:00Z",
    "errorCode": null,
    "retryAfterSeconds": 2
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

### 远程外观配置

#### `GET /api/v1/app/appearance` — 读取当前玩家生效的远程外观配置

实现状态：`contract_only`。权限：AUTHENTICATED_PLAYER。

服务端按当前会话合成内置默认、全局配置、玩家专属覆盖；三组独立，不接受任意 playerRef。Cache-Control: private, no-store。客户端生效和离线规则见 8.1。当前 07 App 尚未接入。

| 参数 | 位置 | 必填 | 类型 | 约束/说明 |
| --- | --- | --- | --- | --- |
| `packageName` | query | 是 | string | 长度 1–200； |
| `channel` | query | 是 | string | `stable` / `preview`； |
| `schemaVersion` | query | 是 | integer | `1`； |

响应 `200`：`V2EffectiveAppearanceResponse`。

```json
{
  "requestId": "req_01HV0000000000000000000000",
  "data": {
    "schemaVersion": 1,
    "packageName": "com.deuterium.app.uilab",
    "channel": "stable",
    "globalVersion": 1,
    "playerVersion": 0,
    "effectiveRevision": "appearance_demo_g1_p0_s1",
    "settings": {
      "bottomBarGlass": {
        "enabled": true,
        "blurRadiusDp": 18,
        "opacity": 0.42,
        "refractionStrength": 9,
        "highlightStrength": 0.65,
        "dynamicHighlightEnabled": true,
        "allowLocalTuning": true
      },
      "overlayGlass": {
        "enabled": true,
        "blurRadiusDp": 18,
        "opacity": 0.42,
        "refractionStrength": 9,
        "highlightStrength": 0.65,
        "dynamicHighlightEnabled": true,
        "allowLocalTuning": true
      },
      "headerGradient": {
        "blurRadiusDp": 18,
        "fadeHeightDp": 32,
        "allowLocalTuning": true
      }
    },
    "sources": {
      "bottomBarGlass": "GLOBAL",
      "overlayGlass": "GLOBAL",
      "headerGradient": "GLOBAL"
    },
    "refreshAfterSeconds": 300,
    "updatedAt": "2026-09-08T02:00:00Z"
  },
  "serverTime": "2026-09-08T02:00:00Z"
}
```

错误分支：`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `STATE_VERSION_CONFLICT`, `INVALID_STATE_TRANSITION`, `RATE_LIMITED`, `SERVER_UNAVAILABLE`；另执行端点说明中的专属业务约束。

## 11. 数据模型完整参考

`必填` 指该模型中的 required 字段；标注 null 的字段可为空。请求中未定义的业务身份或权限字段应拒绝。基础 v1 模型保留已有字段，增量模型见对应实现状态。

### `RequestId`

类型：`string`。

### `ApiResponseBase`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |

### `AppUpdateCheckData`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `latest` | boolean | 是 |  |  |
| `message` | string | 是 | 长度 1–200 | 人类可读状态；不可据此判断版本 |
| `latestVersionCode` | integer | 是 |  |  |
| `latestVersionName` | string | 是 |  |  |
| `serverNow` | string | 否 | date-time |  |
| `hasUpdates` | boolean | 否 |  |  |
| `apkUpdate` | AppRelease / null | 否 |  |  |
| `resourceUpdate` | AppRelease / null | 否 |  |  |

### `AppUpdateCheckResponse`

组合模型：`ApiResponseBase`, `object`。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `data` | AppUpdateCheckData | 是 |  |  |

### `Page`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `nextCursor` | string / null | 是 |  |  |

### `ErrorResponse`

组合模型：`ApiResponseBase`, `object`。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `error` | ErrorBody | 是 |  |  |

### `ErrorBody`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `code` | string | 是 | `INVALID_REQUEST` / `UNAUTHORIZED` / `FORBIDDEN` / `NOT_FOUND` / `CONFLICT` / `RATE_LIMITED` / `SERVER_UNAVAILABLE` / `PLUGIN_BRIDGE_UNAVAILABLE` / `PLAYER_NOT_ONLINE` / `PLAYER_NOT_FOUND` / `PLAYER_IDENTITY_CONFLICT` / `QQ_ALREADY_USED` / `UUID_ALREADY_REGISTERED` / `VERIFICATION_COOLDOWN` / `VERIFICATION_INVALID` / `VERIFICATION_EXPIRED` / `VERIFICATION_ATTEMPTS_EXCEEDED` / `PASSWORD_INVALID` / `ACCOUNT_PASSWORD_INVALID` / `LOGIN_LOCKED` / `AMOUNT_INVALID` / `RECIPIENT_NOT_FOUND` / `RECIPIENT_IDENTITY_UNCONFIRMED` / `BALANCE_INSUFFICIENT` / `TRANSFER_DUPLICATE` / `TRANSFER_FAILED` / `TRANSFER_RESULT_UNKNOWN` / `CHAT_CONNECTION_UNAVAILABLE` / `CHAT_MESSAGE_EMPTY` / `CHAT_MESSAGE_TOO_LONG` / `CHAT_SEND_FAILED` / `SENDER_IDENTITY_UNCONFIRMED` / `AI_DISABLED` / `AI_MESSAGE_EMPTY` / `AI_MESSAGE_TOO_LONG` / `AI_QUOTA_EXCEEDED` / `AI_PROVIDER_UNAVAILABLE` / `AI_PROVIDER_TIMEOUT` / `AI_PROMPT_RISK_BLOCKED` / `AI_PLAN_NOT_FOUND` / `AI_PURCHASE_DUPLICATE` / `AI_PURCHASE_FAILED` / `AI_PURCHASE_RESULT_UNKNOWN` |  |
| `message` | string | 是 |  |  |
| `details` | object | 否 |  |  |
| `retryAfterSeconds` | integer | 否 | 范围 0–— |  |

### `RegistrationCodeRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `gameId` | string | 是 | 长度 1–32 |  |
| `qq` | string | 是 | 5–20 位数字 |  |
| `password` | string | 否 | 已废弃 | 兼容旧客户端，取码时忽略；密码在创建账号时校验 |

### `RegisterRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `verificationToken` | string | 是 |  |  |
| `code` | string | 是 | 正则 `^\d{6}$` |  |
| `password` | string | 是 | 长度 8–64；password |  |

### `LoginRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `account` | string | 是 | 长度 1–— |  |
| `password` | string | 是 | 长度 8–64；password |  |

### `PasswordResetCodeRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `account` | string | 是 | 长度 1–— |  |

### `PasswordResetRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `verificationToken` | string | 是 |  |  |
| `code` | string | 是 | 正则 `^\d{6}$` |  |
| `newPassword` | string | 是 | 长度 8–64；password |  |

### `VerificationTokenData`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `verificationToken` | string | 是 |  |  |
| `expiresAt` | string | 是 | date-time |  |
| `resendAfterSeconds` | integer | 是 | 范围 0–— |  |

### `VerificationTokenResponse`

组合模型：`ApiResponseBase`, `object`。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `data` | VerificationTokenData | 是 |  |  |

### `AuthData`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `token` | string | 是 |  |  |
| `user` | UserProfile | 是 |  |  |

### `AuthResponse`

组合模型：`ApiResponseBase`, `object`。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `data` | AuthData | 是 |  |  |

### `PasswordResetData`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `passwordReset` | boolean | 是 |  |  |

### `PasswordResetResponse`

组合模型：`ApiResponseBase`, `object`。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `data` | PasswordResetData | 是 |  |  |

### `LogoutData`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `loggedOut` | boolean | 是 |  |  |

### `LogoutResponse`

组合模型：`ApiResponseBase`, `object`。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `data` | LogoutData | 是 |  |  |

### `UserProfile`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `userId` | string | 是 |  |  |
| `playerRef` | string | 是 |  | Opaque player reference returned by backend. |
| `gameId` | string | 是 |  |  |
| `qq` | string | 是 |  |  |
| `identityStatus` | string | 是 | `bound` |  |
| `bio` | string | 否 | 长度 0–200 | 个人简介 |
| `avatar` | AssetView / null | 否 |  |  |
| `profileVersion` | integer | 否 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |

### `UserProfileData`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `user` | UserProfile | 是 |  |  |

### `UserProfileResponse`

组合模型：`ApiResponseBase`, `object`。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `data` | UserProfileData | 是 |  |  |

### `PlayerSummary`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `playerRef` | string | 是 |  | Opaque player reference. Android must not parse it. |
| `gameId` | string | 是 |  |  |
| `qq` | string / null | 否 |  |  |
| `online` | boolean | 是 |  |  |
| `registered` | boolean | 是 |  |  |
| `source` | string | 是 | `game_id` / `qq` / `search` / `online_list` |  |

### `WalletBalance`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `currency` | string | 是 | `CREDIT` |  |
| `amount` | string | 是 | 正则 `^\d+(\.\d{1,2})?$` |  |
| `fresh` | boolean | 是 |  |  |
| `refreshedAt` | string / null | 是 | date-time |  |
| `availableAmount` | CreditAmount | 否 |  |  |
| `heldAmount` | CreditAmount | 否 |  |  |
| `heldBreakdown` | object | 否 |  |  |
| `heldBreakdown.market` | CreditAmount | 是 |  |  |
| `heldBreakdown.commissions` | CreditAmount | 是 |  |  |
| `serverNow` | string | 否 | date-time |  |

### `WalletBalanceData`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `balance` | WalletBalance | 是 |  |  |

### `WalletBalanceResponse`

组合模型：`ApiResponseBase`, `object`。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `data` | WalletBalanceData | 是 |  |  |

### `RecipientSearchType`

类型：`string`。`auto` / `game_id` / `qq` / `search`

### `RecipientSearchData`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `candidates` | array<ResolvedPlayerRef> | 是 |  |  |

### `RecipientSearchResponse`

组合模型：`ApiResponseBase`, `object`。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `data` | RecipientSearchData | 是 |  |  |

### `ResolvedPlayerRef`

组合模型：`PlayerSummary`, `object`。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `confirmedAt` | string | 是 | date-time |  |
| `expiresAt` | string / null | 否 | date-time |  |

### `CreateTransferRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 |  |
| `recipientPlayerRef` | string | 是 |  |  |
| `amount` | string | 是 | 正则 `^\d+(\.\d{1,2})?$` |  |
| `note` | string / null | 否 | 长度 —–80 |  |

### `TransferStatus`

类型：`string`。`processing` / `success` / `failed` / `unknown`

### `Transfer`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `transferId` | string | 是 |  |  |
| `clientRequestId` | string | 是 |  |  |
| `recipient` | PlayerSummary | 是 |  |  |
| `amount` | string | 是 | 正则 `^\d+(\.\d{1,2})?$` |  |
| `currency` | string | 是 | `CREDIT` |  |
| `note` | string / null | 否 |  |  |
| `status` | TransferStatus | 是 |  |  |
| `createdAt` | string | 是 | date-time |  |
| `updatedAt` | string | 是 | date-time |  |

### `TransferData`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `transfer` | Transfer | 是 |  |  |

### `TransferResponse`

组合模型：`ApiResponseBase`, `object`。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `data` | TransferData | 是 |  |  |

### `WalletRecordDirection`

类型：`string`。`income` / `expense`

### `WalletRecord`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `recordId` | string | 是 |  |  |
| `direction` | WalletRecordDirection | 是 |  |  |
| `otherPlayer` | PlayerSummary | 是 |  |  |
| `amount` | string | 是 | 正则 `^\d+(\.\d{1,2})?$` |  |
| `currency` | string | 是 | `CREDIT` |  |
| `status` | TransferStatus | 是 |  |  |
| `note` | string / null | 否 |  |  |
| `occurredAt` | string | 是 | date-time |  |

### `WalletRecordsData`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `records` | array<WalletRecord> | 是 |  |  |

### `WalletRecordsResponse`

组合模型：`ApiResponseBase`, `object`。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `data` | WalletRecordsData | 是 |  |  |
| `page` | Page | 是 |  |  |

### `ChatMessage`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `messageId` | string | 是 |  |  |
| `sender` | PlayerSummary | 是 |  |  |
| `content` | string | 是 | 长度 1–256 |  |
| `kind` | string | 是 | `public_chat` / `private_chat` |  |
| `sentAt` | string | 是 | date-time |  |
| `conversationId` | string | 否 | 长度 0–128 | 私聊时的会话 ID，公共频道可为空 |
| `reply` | ReplySnapshot / null | 否 |  |  |
| `clientMessageId` | string | 否 | 长度 1–128 | 客户端幂等发送 ID |

### `ChatMessagesData`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `messages` | array<ChatMessage> | 是 |  |  |

### `ChatMessagesResponse`

组合模型：`ApiResponseBase`, `object`。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `data` | ChatMessagesData | 是 |  |  |
| `page` | Page | 是 |  |  |

### `PresenceData`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `onlineCount` | integer | 是 | 范围 0–— |  |
| `available` | boolean | 是 |  |  |
| `updatedAt` | string / null | 是 | date-time |  |

### `PresenceResponse`

组合模型：`ApiResponseBase`, `object`。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `data` | PresenceData | 是 |  |  |

### `OnlinePlayer`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `playerRef` | string | 是 |  |  |
| `gameId` | string | 是 |  |  |
| `qq` | string / null | 否 |  |  |
| `registered` | boolean | 是 |  |  |
| `onlineSince` | string / null | 否 | date-time |  |

### `OnlinePlayersData`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `players` | array<OnlinePlayer> | 是 |  |  |

### `OnlinePlayersResponse`

组合模型：`ApiResponseBase`, `object`。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `data` | OnlinePlayersData | 是 |  |  |

### `AppStatus`

类型：`string`。`online` / `just_online` / `recent_online` / `recently_online` / `long_offline`

### `PlayerDirectoryItem`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `playerRef` | string | 是 |  |  |
| `gameId` | string | 是 |  |  |
| `qq` | string / null | 否 |  |  |
| `registered` | boolean | 是 |  |  |
| `serverOnline` | boolean | 是 |  |  |
| `appStatus` | AppStatus | 是 |  |  |
| `appConnected` | boolean | 否 |  |  |
| `appForeground` | boolean | 否 |  |  |
| `appLastSeenAt` | string / null | 否 | date-time |  |
| `onlineSince` | string / null | 否 | date-time |  |
| `followed` | boolean | 是 |  |  |
| `self` | boolean | 是 |  |  |

### `PlayerDirectoryData`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `players` | array<PlayerDirectoryItem> | 是 |  |  |

### `PlayerDirectoryResponse`

组合模型：`ApiResponseBase`, `object`。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `data` | PlayerDirectoryData | 是 |  |  |

### `PlayerFollowRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `playerRef` | string | 是 |  |  |

### `PlayerFollowData`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `followed` | boolean | 是 |  |  |
| `player` | PlayerDirectoryItem / null | 否 |  |  |

### `PlayerFollowResponse`

组合模型：`ApiResponseBase`, `object`。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `data` | PlayerFollowData | 是 |  |  |

### `FollowedPlayersData`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `players` | array<PlayerDirectoryItem> | 是 |  |  |

### `FollowedPlayersResponse`

组合模型：`ApiResponseBase`, `object`。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `data` | FollowedPlayersData | 是 |  |  |

### `AiQuota`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `used` | integer | 是 | 范围 0–— |  |
| `limit` | integer | 是 | 范围 0–— |  |
| `remaining` | integer | 是 | 范围 0–— |  |
| `windowHours` | integer | 是 | 范围 1–— |  |
| `resetsAt` | string | 是 | date-time |  |

### `AiPlan`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `planId` | string | 是 |  |  |
| `code` | string | 是 |  |  |
| `name` | string | 是 |  |  |
| `description` | string | 是 |  |  |
| `price` | string | 是 | 正则 `^\d+(\.\d{1,2})?$` |  |
| `currency` | string | 是 | `CREDIT` |  |
| `quotaPerWindow` | integer | 是 | 范围 0–— |  |
| `windowHours` | integer | 是 | 范围 1–— |  |
| `durationDays` | integer | 是 | 范围 0–— |  |
| `modelTier` | string | 是 |  |  |
| `active` | boolean | 是 |  |  |

### `AiConversationState`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `conversationId` | string | 是 |  |  |
| `active` | boolean | 是 |  |  |
| `startedAt` | string | 是 | date-time |  |
| `updatedAt` | string | 是 | date-time |  |

### `AiMessage`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `messageId` | string | 是 |  |  |
| `conversationId` | string | 是 |  |  |
| `role` | string | 是 | `user` / `assistant` |  |
| `content` | string | 是 |  |  |
| `createdAt` | string | 是 | date-time |  |

### `AiMeData`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `assistantName` | string | 是 |  |  |
| `plan` | AiPlan | 是 |  |  |
| `quota` | AiQuota | 是 |  |  |
| `conversation` | AiConversationState | 是 |  |  |

### `AiMeResponse`

组合模型：`ApiResponseBase`, `object`。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `data` | AiMeData | 是 |  |  |

### `AiPlansData`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `plans` | array<AiPlan> | 是 |  |  |

### `AiPlansResponse`

组合模型：`ApiResponseBase`, `object`。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `data` | AiPlansData | 是 |  |  |

### `AiMessagesData`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `messages` | array<AiMessage> | 是 |  |  |

### `AiMessagesResponse`

组合模型：`ApiResponseBase`, `object`。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `data` | AiMessagesData | 是 |  |  |

### `AiChatStreamRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientMessageId` | string | 是 | 长度 1–128 |  |
| `content` | string | 是 | 长度 1–2000 |  |
| `replyToMessageId` | string | 否 | 长度 1–128 | 同一 AI 会话内引用的消息 ID |

### `AiConversationResetData`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `conversation` | AiConversationState | 是 |  |  |

### `AiConversationResetResponse`

组合模型：`ApiResponseBase`, `object`。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `data` | AiConversationResetData | 是 |  |  |

### `AiPurchaseRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 |  |
| `planId` | string | 是 |  |  |

### `AiPurchase`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `purchaseId` | string | 是 |  |  |
| `clientRequestId` | string | 是 |  |  |
| `plan` | AiPlan | 是 |  |  |
| `amount` | string | 是 | 正则 `^\d+(\.\d{1,2})?$` |  |
| `currency` | string | 是 | `CREDIT` |  |
| `status` | string | 是 | `processing` / `success` / `failed` / `unknown` |  |
| `failureCode` | string / null | 否 |  |  |
| `createdAt` | string | 是 | date-time |  |
| `updatedAt` | string | 是 | date-time |  |

### `AiPurchaseData`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `purchase` | AiPurchase | 是 |  |  |
| `quota` | AiQuota / null | 否 |  |  |

### `AiPurchaseResponse`

组合模型：`ApiResponseBase`, `object`。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `data` | AiPurchaseData | 是 |  |  |

### `CreditAmount`

信用点十进制字符串，最多两位小数。付款字段必须大于零；不得使用浮点数计算。

类型：`string`。长度 1–20；正则 `^(0\|[1-9][0-9]*)(\.[0-9]{1,2})?$`

### `MutationRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `expectedVersion` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |

### `ReasonedMutationRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `expectedVersion` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `reason` | string | 是 | 长度 2–500 | 具体说明，不能只有空白 |

### `OperationLookup`

UNKNOWN 不等于失败；必须查询或使用同幂等键重试，不能创建第二笔扣款。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `operationId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `kind` | string | 是 | `STORE_PURCHASE` / `MARKET_PURCHASE` / `COMMISSION_PUBLISH` / `REFUND` / `SETTLEMENT` / `TRANSFER` / `AI_PURCHASE` |  |
| `status` | string | 是 | `PROCESSING` / `COMPLETED` / `FAILED` / `UNKNOWN` |  |
| `resourceType` | string | 是 | `ORDER` / `COMMISSION` / `REFUND` / `TRANSFER` / `AI_PURCHASE` |  |
| `resourceId` | string / null | 是 | 长度 0–128 | 完成后关联的业务资源 ID；处理中可能为空 |
| `amount` | CreditAmount | 是 |  |  |
| `createdAt` | string | 是 | date-time |  |
| `updatedAt` | string | 是 | date-time |  |
| `errorCode` | string / null | 是 | 长度 0–80 | 失败时的稳定错误码 |
| `retryAfterSeconds` | integer | 是 | 范围 1–60；int64 | 建议查询间隔 |

### `PublicPlayerProfile`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `playerRef` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `gameId` | string | 是 | 长度 1–32 | 游戏内 ID |
| `qq` | string | 是 | 长度 5–12；正则 `^\d{5,12}$` | 仅对已登录用户返回；不是可修改的登录绑定入口 |
| `bio` | string | 是 | 长度 0–200 | 个人简介 |
| `avatar` | AssetView / null | 是 |  |  |
| `online` | boolean | 是 |  |  |
| `lastSeenAt` | string / null | 是 | date-time | 未知时为 null，不能显示成刚刚在线 |
| `followed` | boolean | 是 |  |  |
| `version` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |

### `ProfilePatchRequest`

至少修改 bio 或 avatarAssetId 一项；不支持用该接口修改 QQ、游戏身份或权限。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `expectedVersion` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `bio` | string | 否 | 长度 0–200 | 个人简介；空字符串用于清空 |
| `avatarAssetId` | string / null | 否 | 长度 0–128 | 头像图片 assetId；null 恢复默认头像 |

### `AssetView`

READY 仅由后端核验真实 OSS 对象后设置。业务只存 assetId，不能存临时上传 URL、客户端 objectKey 或自报下载 URL。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `assetId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `purpose` | string | 是 | `AVATAR` / `STORE_MEDIA` / `MARKET_PHOTO` / `COMMISSION_COVER` / `DISPUTE_EVIDENCE` |  |
| `status` | string | 是 | `PROCESSING` / `READY` / `REJECTED` |  |
| `url` | string / null | 是 | 长度 1–4096；uri | 服务端提供的 HTTPS 图片地址；私密证据使用短期授权 URL |
| `width` | integer / null | 是 | 范围 1–8192；int64 | 像素宽 |
| `height` | integer / null | 是 | 范围 1–8192；int64 | 像素高 |
| `sizeBytes` | integer | 是 | 范围 1–20971520；int64 | 文件字节数 |
| `sha256` | string / null | 是 | 长度 64–64；正则 `^[a-fA-F0-9]{64}$` | 可信云端处理实际计算的 SHA-256；未计算时 null，禁止把客户端声明/自定义 OSS 元数据冒充服务端校验摘要。 |
| `altText` | string | 是 | 长度 0–200 | 替代文字 |
| `createdAt` | string | 是 | date-time |  |
| `contentMd5` | string / null | 是 | 长度 24–24；正则 `^[A-Za-z0-9+/]{22}==$` | 由后端读取 OSS 实际对象元数据取得的 Base64 MD5；尚未验证时 null。只作传输完整性校验，不授予资源权限。 |
| `urlExpiresAt` | string / null | 是 | date-time | 私有签名下载地址到期时间；无 URL 或公开 CDN 地址时 null |

### `StoreBrand`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `brandId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `name` | string | 是 | 长度 1–60 | 品牌名称 |
| `logoAssetId` | string / null | 是 | 长度 0–128 | 品牌图标引用，可为空 |
| `sortOrder` | integer | 是 | 范围 0–9999；int64 | 排序权重 |
| `active` | boolean | 是 |  |  |
| `version` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |

### `StoreCategory`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `categoryId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `name` | string | 是 | 长度 1–40 | 商城展示分类 |
| `sortOrder` | integer | 是 | 范围 0–2147483647；int64 |  |
| `active` | boolean | 是 |  |  |
| `version` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |

### `StoreView`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `storeId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `name` | string | 是 | 长度 1–60 | 官方店铺名称 |
| `intro` | string | 是 | 长度 0–300 | 店铺简介 |
| `logo` | AssetView / null | 是 |  |  |
| `cover` | AssetView / null | 是 |  |  |
| `contactQq` | string | 是 | 长度 5–12；正则 `^\d{5,12}$` | 客服 QQ |
| `serviceHours` | string | 是 | 长度 0–100 | 客服时间 |
| `notice` | string | 是 | 长度 0–1000 | 店铺公告，纯文本 |
| `version` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |

### `ContentBlock`

按 type 校验相应字段。顺序即商品详情展示顺序。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `blockId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `type` | string | 是 | `HEADING` / `PARAGRAPH` / `IMAGE` / `KEY_VALUE_LIST` |  |
| `heading` | string | 否 | 长度 0–100 | 标题 |
| `text` | string | 否 | 长度 0–3000 | 正文；不接收 HTML/脚本 |
| `assetId` | string / null | 否 | 长度 0–128 | IMAGE 时必填的图片引用 |
| `altText` | string | 否 | 长度 0–200 | 图片说明 |
| `rows` | array<object> | 否 | 数量 0–30 |  |
| `rows[].label` | string | 是 | 长度 1–80 | 标签 |
| `rows[].value` | string | 是 | 长度 1–500 | 内容 |

### `ProductContent`

官方商城所有可编辑信息。交易动作文字、支付规则和服务端权限不是可编辑商品内容。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `title` | string | 是 | 长度 1–80 | 商品标题 |
| `subtitle` | string | 是 | 长度 1–200 | 简介 |
| `description` | string | 是 | 长度 1–10000 | 完整描述 |
| `brandId` | string | 是 | 长度 1–128 | 品牌 ID |
| `categoryId` | string | 是 | 长度 1–128 | 商城分类 ID |
| `price` | CreditAmount | 是 |  |  |
| `coverAssetId` | string | 是 | 长度 1–128 | 首页封面；必须等于 galleryAssetIds 第一项 |
| `galleryAssetIds` | array<string> | 是 | 数量 1–20 |  |
| `galleryAltTexts` | array<string> | 是 | 数量 1–20 |  |
| `contentBlocks` | array<ContentBlock> | 是 | 数量 0–80 |  |
| `includedItems` | array<string> | 是 | 数量 1–40 |  |
| `deliveryTemplateRef` | string | 是 | 长度 1–128 | 服务端已批准的游戏邮箱发放模板，不能提交任意命令 |
| `deliverySummary` | string | 是 | 长度 1–500 | 展示的发放方式和内容 |
| `estimatedDelivery` | string | 是 | 长度 1–150 | 预计送达说明 |
| `inventoryPolicy` | string | 是 | `FINITE` / `UNLIMITED` |  |
| `stock` | integer | 是 | 范围 0–999999；int64 | 有限库存的可售数量；无限库存时应为 0 且不参与扣减 |
| `limitPerOrder` | integer | 是 | 范围 1–999；int64 | 单笔限购，当前体验版默认 9 |
| `posterTone` | string | 是 | `LIGHT` / `DARK` |  |
| `accentColor` | string | 是 | 长度 7–7；正则 `^#[0-9A-Fa-f]{6}$` | 海报装饰色，不改变支付和系统主题 |
| `badges` | array<string> | 是 | 数量 0–4 |  |
| `sortOrder` | integer | 是 | 范围 0–99999；int64 | 展示排序 |

### `StoreProduct`

公开端仅返回 ACTIVE 商品和已发布版本。quote 与订单冻结该版本，后台编辑不改已成交内容。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `productId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `storeId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `version` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `content` | ProductContent | 是 |  |  |
| `images` | array<AssetView> | 是 | 数量 1–20 |  |
| `visibility` | string | 是 | `ACTIVE` / `UNLISTED` / `ARCHIVED` |  |
| `availableStock` | integer / null | 是 | 范围 0–999999；int64 | 实时可售数量；不限量时为 null |
| `publishedAt` | string | 是 | date-time |  |
| `updatedAt` | string | 是 | date-time |  |

### `MerchantProduct`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `productId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `storeId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `version` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `draft` | ProductContent | 是 |  |  |
| `published` | StoreProduct / null | 是 |  |  |
| `visibility` | string | 是 | `DRAFT` / `ACTIVE` / `UNLISTED` / `ARCHIVED` |  |
| `updatedAt` | string | 是 | date-time |  |

### `ProductWriteRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `expectedVersion` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `content` | ProductContent | 是 |  |  |

### `ProductCreateRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `content` | ProductContent | 是 |  |  |

### `StoreEditRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `expectedVersion` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `name` | string | 是 | 长度 1–60 | 店铺名称 |
| `intro` | string | 是 | 长度 0–300 | 简介 |
| `logoAssetId` | string / null | 是 | 长度 0–128 | Logo 引用 |
| `coverAssetId` | string / null | 是 | 长度 0–128 | 封面引用 |
| `contactQq` | string | 是 | 长度 5–12；正则 `^\d{5,12}$` | 客服 QQ |
| `serviceHours` | string | 是 | 长度 0–100 | 客服时间 |
| `notice` | string | 是 | 长度 0–1000 | 店铺公告 |

### `BrandWriteRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `expectedVersion` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `name` | string | 是 | 长度 1–60 | 品牌名称 |
| `logoAssetId` | string / null | 是 | 长度 0–128 | Logo 引用 |
| `sortOrder` | integer | 是 | 范围 0–2147483647；int64 |  |
| `active` | boolean | 是 |  |  |

### `CategoryWriteRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `expectedVersion` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `name` | string | 是 | 长度 1–40 | 商城分类名称 |
| `sortOrder` | integer | 是 | 范围 0–2147483647；int64 |  |
| `active` | boolean | 是 |  |  |

### `DeliveryTemplate`

只提供受控模板；商家不能提交或读取任意服务器命令、插件密钥。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `templateRef` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `name` | string | 是 | 长度 1–80 | 发放模板名称 |
| `summary` | string | 是 | 长度 1–1000 | 具体包含物品和数量 |
| `active` | boolean | 是 |  |  |

### `CartItem`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `productId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `quantity` | integer | 是 | 范围 1–999；int64 | 购买数量 |
| `productVersion` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |

### `ShoppingBag`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `version` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `items` | array<CartItem> | 是 | 数量 0–100 |  |
| `updatedAt` | string | 是 | date-time |  |

### `CartQuantityRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `expectedVersion` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `quantity` | integer | 是 | 范围 1–999；int64 | 受商品单笔限购约束 |

### `MarketCategory`

当前固定六类，不返回多级细分类；不传分类表示全部。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `code` | string | 是 | `MATERIALS` / `EQUIPMENT` / `SUPPLIES` / `DECORATION` / `CONSTRUCTION` / `OTHER` |  |
| `name` | string | 是 | 长度 1–20 | 分类展示名称 |
| `description` | string | 是 | 长度 1–120 | 范围说明 |
| `sortOrder` | integer | 是 | 范围 0–2147483647；int64 |  |

### `ListingContent`

普通商品 deliveryMethods 为 DOOR/PICKUP；建筑服务只能 WORKSITE，并要求 workHours。第一张照片为首页封面。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `title` | string | 是 | 长度 1–80 | 标题 |
| `subtitle` | string | 是 | 长度 1–200 | 简介 |
| `description` | string | 是 | 长度 1–10000 | 详细要求 |
| `categoryCode` | string | 是 | `MATERIALS` / `EQUIPMENT` / `SUPPLIES` / `DECORATION` / `CONSTRUCTION` / `OTHER` |  |
| `price` | CreditAmount | 是 |  |  |
| `stock` | integer | 是 | 范围 1–999；int64 | 上架时可售库存 1–999 |
| `photoAssetIds` | array<string> | 是 | 数量 1–5 |  |
| `contactQq` | string | 是 | 长度 5–12；正则 `^\d{5,12}$` | 商品联系 QQ，不修改账号登录 QQ |
| `deliveryMethods` | array<string> | 是 | 数量 1–2 |  |
| `pickupLocation` | string | 是 | 长度 0–120 | 支持自取时必填 |
| `workHours` | integer | 是 | 范围 1–8760；int64 | 建筑服务总工期，含验收预留；普通商品忽略 |

### `MarketListing`

可见性必须遵循正文 4.1 和对应端点对象授权；结构字段不赋予读取权限。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `title` | string | 是 | 长度 1–80 | 标题 |
| `subtitle` | string | 是 | 长度 1–200 | 简介 |
| `description` | string | 是 | 长度 1–10000 | 详细要求 |
| `categoryCode` | string | 是 | `MATERIALS` / `EQUIPMENT` / `SUPPLIES` / `DECORATION` / `CONSTRUCTION` / `OTHER` |  |
| `price` | CreditAmount | 是 |  |  |
| `stock` | integer | 是 | 范围 0–999；int64 | 实时可售库存；售罄为 0 |
| `photoAssetIds` | array<string> | 是 | 数量 1–5 |  |
| `contactQq` | string | 是 | 长度 5–12；正则 `^\d{5,12}$` | 商品联系 QQ，不修改账号登录 QQ |
| `deliveryMethods` | array<string> | 是 | 数量 1–2 |  |
| `pickupLocation` | string | 是 | 长度 0–120 | 支持自取时必填 |
| `workHours` | integer | 是 | 范围 1–8760；int64 | 建筑服务总工期，含验收预留；普通商品忽略 |
| `listingId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `seller` | PublicPlayerProfile | 是 |  |  |
| `photos` | array<AssetView> | 是 | 数量 1–5 |  |
| `active` | boolean | 是 |  |  |
| `version` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `createdAt` | string | 是 | date-time |  |
| `updatedAt` | string | 是 | date-time |  |

### `ListingCreateRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `content` | ListingContent | 是 |  |  |

### `ListingEditRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `expectedVersion` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `content` | ListingContent | 是 |  |  |

### `DeliveryAddress`

不允许通过填写邮箱账户代替当前登录玩家；官方商城固定发到本人游戏邮箱。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `method` | string | 是 | `DOOR` / `PICKUP` / `WORKSITE` / `MAILBOX` |  |
| `location` | string | 否 | 长度 0–120 | 交付地点；邮箱交付时可不传，由服务端绑定当前玩家 |
| `projectName` | string | 否 | 长度 0–80 | 建筑项目名，WORKSITE 必填 |

### `QuoteRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `channel` | string | 是 | `OFFICIAL_STORE` / `PLAYER_MARKET` |  |
| `items` | array<object> | 是 | 数量 1–100 |  |
| `items[].productId` | string | 是 | 长度 1–128 | 对应商品或上架物品 ID |
| `items[].quantity` | integer | 是 | 范围 1–999；int64 | 数量 |
| `items[].expectedProductVersion` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `delivery` | DeliveryAddress | 是 |  |  |

### `QuoteLine`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `productId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `title` | string | 是 | 长度 1–80 | 成交快照标题 |
| `unitPrice` | CreditAmount | 是 |  |  |
| `quantity` | integer | 是 | 范围 1–999；int64 | 数量 |
| `subtotal` | CreditAmount | 是 |  |  |
| `productVersion` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |

### `CheckoutQuote`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `quoteId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `channel` | string | 是 | `OFFICIAL_STORE` / `PLAYER_MARKET` |  |
| `items` | array<QuoteLine> | 是 | 数量 1–100 |  |
| `totalAmount` | CreditAmount | 是 |  |  |
| `currency` | string | 是 | `CREDIT` |  |
| `expiresAt` | string | 是 | date-time | 默认建议 5 分钟，实际以返回值为准 |
| `delivery` | DeliveryAddress | 是 |  |  |
| `version` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `warnings` | array<string> | 是 | 数量 0–10 |  |

### `CreateOrderRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `quoteId` | string | 是 | 长度 1–128 | 刚确认过、尚未过期的服务端报价 |
| `expectedQuoteVersion` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |

### `OrderParty`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `kind` | string | 是 | `PLAYER` / `OFFICIAL_STORE` |  |
| `playerRef` | string / null | 是 | 长度 0–128 | 玩家参与者引用，官方店铺可为空 |
| `storeId` | string / null | 是 | 长度 0–128 | 官方店铺 ID，玩家可为空 |
| `displayName` | string | 是 | 长度 1–80 | 显示名称 |
| `contactQq` | string / null | 是 | 长度 0–12 | 对订单参与方可见的联系 QQ |

### `OrderItemSnapshot`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `productId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `productVersion` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `title` | string | 是 | 长度 1–80 | 成交时标题 |
| `subtitle` | string | 是 | 长度 0–200 | 成交时简介 |
| `description` | string | 是 | 长度 0–10000 | 成交时详情 |
| `unitPrice` | CreditAmount | 是 |  |  |
| `quantity` | integer | 是 | 范围 1–999；int64 | 成交数量 |
| `photoAssetIds` | array<string> | 是 | 数量 1–20 |  |
| `categoryName` | string | 是 | 长度 0–60 | 成交时分类 |
| `includedItems` | array<string> | 是 | 数量 0–40 |  |
| `contentBlocks` | array<ContentBlock> | 是 | 数量 0–80 |  |

### `OrderRefund`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `refundId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `status` | string | 是 | `REQUESTED` / `REJECTED` / `APPROVED` / `WITHDRAWN` / `PROCESSING` |  |
| `attempt` | integer | 是 | 范围 1–1；int64 | 发货后的唯一申请次数 |
| `reason` | string | 是 | 长度 2–500 | 买方申请原因 |
| `rejectionReason` | string | 是 | 长度 0–500 | 拒绝时必填的具体理由 |
| `requestedAt` | string | 是 | date-time |  |
| `resolvedAt` | string / null | 是 | date-time |  |
| `amount` | CreditAmount | 是 |  |  |
| `version` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |

### `OrderView`

可见性必须遵循正文 4.1 和对应端点对象授权；结构字段不赋予读取权限。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `orderId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `orderNo` | string | 是 | 长度 1–64 | 可复制展示订单号 |
| `channel` | string | 是 | `OFFICIAL_STORE` / `PLAYER_MARKET` |  |
| `status` | string | 是 | `PAYMENT_PROCESSING` / `AWAITING_SHIPMENT` / `SHIPPED` / `WORK_COMPLETED` / `CONFIRMED` / `AWAITING_CLAIM` / `CLAIMED` / `REFUNDED` / `CANCELLED` |  |
| `construction` | boolean | 是 |  |  |
| `buyer` | OrderParty | 是 |  |  |
| `seller` | OrderParty | 是 |  |  |
| `items` | array<OrderItemSnapshot> | 是 | 数量 1–100 |  |
| `amount` | CreditAmount | 是 |  |  |
| `currency` | string | 是 | `CREDIT` |  |
| `delivery` | DeliveryAddress | 是 |  |  |
| `snapshotId` | string | 是 | 长度 1–128 | 服务端保存的不可变成交快照 |
| `snapshotSha256` | string | 是 | 长度 64–64 | 快照摘要 |
| `fundsStatus` | string | 是 | `PROCESSING` / `PAID` / `HELD` / `SETTLING` / `SETTLED` / `REFUNDING` / `REFUNDED` / `INTERVENTION_HOLD` / `UNKNOWN` |  |
| `confirmationHours` | integer | 是 | 范围 1–8760；int64 | 普通商品 72；工程为约定总工期 |
| `serverNow` | string | 是 | date-time |  |
| `autoConfirmAt` | string / null | 是 | date-time | 绝对时间；退款/介入暂停时为 null |
| `pausedRemainingSeconds` | integer / null | 是 | 范围 0–31536000；int64 | 暂停时剩余秒数，未暂停时为 null |
| `refund` | OrderRefund / null | 是 |  |  |
| `refundAttemptsUsed` | integer | 是 | 范围 0–1；int64 | 已消耗次数 |
| `availableActions` | array<string> | 是 | 数量 0–10 |  |
| `createdAt` | string | 是 | date-time |  |
| `shippedAt` | string / null | 是 | date-time |  |
| `workCompletedAt` | string / null | 是 | date-time |  |
| `confirmedAt` | string / null | 是 | date-time |  |
| `automatic` | boolean | 是 |  |  |
| `version` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |

### `OrderCreationResult`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `operation` | OperationLookup | 是 |  |  |
| `order` | OrderView / null | 是 |  |  |

### `RefundRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `expectedVersion` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `reasonCode` | string | 是 | `NO_LONGER_NEEDED` / `DELIVERY_DELAY` / `NOT_AS_DESCRIBED` / `OTHER` |  |
| `description` | string | 是 | 长度 2–500 | 申请说明；工程须描述分歧 |
| `evidenceAssetIds` | array<string> | 是 | 数量 0–5 |  |

### `RefundResolutionRequest`

REJECT 不得为空；理由进入退款记录、通知和平台介入快照。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `expectedVersion` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `decision` | string | 是 | `APPROVE` / `REJECT` |  |
| `reason` | string | 是 | 长度 0–500 | REJECT 时必填 2–500 字，APPROVE 可为空 |

### `WorkCompletionRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `expectedVersion` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `description` | string | 是 | 长度 2–500 | 完成说明 |
| `evidenceAssetIds` | array<string> | 是 | 数量 0–5 |  |

### `CommissionContent`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `title` | string | 是 | 长度 2–60 | 委托标题 |
| `description` | string | 是 | 长度 5–2000 | 具体任务与完成标准 |
| `location` | string | 是 | 长度 2–120 | 地点或交付方式 |
| `urgency` | string | 是 | `NORMAL` / `SOON` / `URGENT` |  |
| `reward` | CreditAmount | 是 |  |  |
| `workHours` | integer | 是 | 范围 1–8760；int64 | 接取后的履约时限；1 小时至 365 天 |
| `coverAssetId` | string | 是 | 长度 1–128 | 一张 READY 委托封面 |

### `CommissionCreateRequest`

创建并预付报酬是同一个幂等业务操作；预付未完成不能进入大厅或被接取。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `content` | CommissionContent | 是 |  |  |

### `CommissionView`

可见性必须遵循正文 4.1 和对应端点对象授权；结构字段不赋予读取权限。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `commissionId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `owner` | OrderParty | 是 |  |  |
| `worker` | OrderParty / null | 是 |  |  |
| `content` | CommissionContent | 是 |  |  |
| `cover` | AssetView | 是 |  |  |
| `status` | string | 是 | `FUNDING` / `OPEN` / `ACTIVE` / `COMPLETED` / `CONFIRMED` / `CANCELLED` |  |
| `fundsStatus` | string | 是 | `PROCESSING` / `HELD` / `SETTLING` / `SETTLED` / `REFUNDING` / `REFUNDED` / `INTERVENTION_HOLD` / `UNKNOWN` |  |
| `snapshotId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `serverNow` | string | 是 | date-time |  |
| `workDueAt` | string / null | 是 | date-time | 从接取起计算；逾期只标记，不结算 |
| `acceptanceDueAt` | string / null | 是 | date-time | 完成提交后 72 小时；暂停时 null |
| `pausedRemainingSeconds` | integer / null | 是 | 范围 0–31536000；int64 | 当前暂停阶段剩余秒数 |
| `completionDescription` | string | 是 | 长度 0–500 | 完成说明 |
| `completionAssetIds` | array<string> | 是 | 数量 0–5 |  |
| `refund` | OrderRefund / null | 是 |  |  |
| `refundAttemptsUsed` | integer | 是 | 范围 0–1；int64 | 接取后唯一退款申请次数 |
| `availableActions` | array<string> | 是 | 数量 0–8 |  |
| `createdAt` | string | 是 | date-time |  |
| `acceptedAt` | string / null | 是 | date-time |  |
| `completedAt` | string / null | 是 | date-time |  |
| `confirmedAt` | string / null | 是 | date-time |  |
| `automatic` | boolean | 是 |  |  |
| `version` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |

### `CommissionCreationResult`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `operation` | OperationLookup | 是 |  |  |
| `commission` | CommissionView / null | 是 |  |  |

### `NotificationPreferencesView`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `version` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `enabled` | boolean | 是 |  | 账号推送总开关 |
| `showPreviews` | boolean | 是 |  | 通知中显示业务摘要 |
| `directMessages` | boolean | 是 |  |  |
| `mentions` | boolean | 是 |  |  |
| `followedPlayers` | boolean | 是 |  |  |
| `wallet` | boolean | 是 |  |  |
| `marketOrders` | boolean | 是 |  |  |
| `commissions` | boolean | 是 |  |  |
| `announcements` | boolean | 是 |  |  |
| `appUpdates` | boolean | 是 |  |  |

### `NotificationPreferencesPatch`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `expectedVersion` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `enabled` | boolean | 否 |  | 账号推送总开关 |
| `showPreviews` | boolean | 否 |  | 通知中显示业务摘要 |
| `directMessages` | boolean | 否 |  |  |
| `mentions` | boolean | 否 |  |  |
| `followedPlayers` | boolean | 否 |  |  |
| `wallet` | boolean | 否 |  |  |
| `marketOrders` | boolean | 否 |  |  |
| `commissions` | boolean | 否 |  |  |
| `announcements` | boolean | 否 |  |  |
| `appUpdates` | boolean | 否 |  |  |

### `NotificationTarget`

目标为受控业务路由，不接收任意外部 Intent/URL。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `kind` | string | 是 | `ORDER` / `REFUND` / `COMMISSION` / `CONVERSATION` / `PUBLIC_CHAT` / `WALLET` / `ANNOUNCEMENT` / `APP_UPDATE` |  |
| `referenceId` | string | 是 | 长度 0–128 | 关联资源引用，钱包/公共频道可为空 |
| `stateVersion` | integer | 是 | 范围 0–2147483647；int64 | 事件关联的业务版本 |

### `NotificationView`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `notificationId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `topic` | string | 是 | `DIRECT_MESSAGES` / `MENTIONS` / `FOLLOWED_PLAYERS` / `WALLET` / `MARKET_ORDERS` / `COMMISSIONS` / `ANNOUNCEMENTS` / `APP_UPDATES` |  |
| `title` | string | 是 | 长度 1–100 | 标题 |
| `body` | string | 是 | 长度 1–500 | 摘要 |
| `target` | NotificationTarget | 是 |  |  |
| `createdAt` | string | 是 | date-time |  |
| `readAt` | string / null | 是 | date-time |  |

### `ReadNotificationsRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `notificationIds` | array<string> | 是 | 数量 1–100 |  |

### `NotificationReadResult`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `readCount` | integer | 是 | 范围 0–2147483647；int64 |  |
| `unreadCount` | integer | 是 | 范围 0–2147483647；int64 |  |

### `PushDeviceRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `installationId` | string | 是 | 长度 1–128 | 本次安装生成的随机标识，不使用硬件设备 ID |
| `provider` | string | 是 | 长度 1–60 | 服务端配置的 Android 推送适配器标识，本契约不锁定厂商 |
| `pushToken` | string | 是 | 长度 1–4096；仅写入，不回显 | 厂商设备令牌，仅写入、不回显 |
| `permissionGranted` | boolean | 是 |  | 客户端报告的系统权限状态，仅作投递参考 |
| `appVersionCode` | integer | 是 | 范围 1–2147483647；int64 | 安装版本 |
| `locale` | string | 是 | 长度 2–30 | 语言区域 |

### `PushDeviceView`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `deviceRef` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `installationId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `provider` | string | 是 | 长度 1–60 | 推送适配器 |
| `registeredAt` | string | 是 | date-time |  |
| `permissionGranted` | boolean | 是 |  |  |

### `AnnouncementView`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `announcementId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `title` | string | 是 | 长度 1–100 | 公告标题 |
| `summary` | string | 是 | 长度 1–300 | 列表摘要 |
| `contentBlocks` | array<ContentBlock> | 是 | 数量 1–100 |  |
| `coverAssetId` | string / null | 是 | 长度 0–128 | 可选封面 |
| `pinned` | boolean | 是 |  |  |
| `priority` | string | 是 | `NORMAL` / `IMPORTANT` |  |
| `publishedAt` | string | 是 | date-time |  |
| `updatedAt` | string | 是 | date-time |  |
| `version` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |

### `AnnouncementWriteRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `expectedVersion` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `title` | string | 是 | 长度 1–100 | 公告标题 |
| `summary` | string | 是 | 长度 1–300 | 摘要 |
| `contentBlocks` | array<ContentBlock> | 是 | 数量 1–100 |  |
| `coverAssetId` | string / null | 是 | 长度 0–128 | 可选封面 |
| `pinned` | boolean | 是 |  |  |
| `priority` | string | 是 | `NORMAL` / `IMPORTANT` |  |

### `ReplySnapshot`

客户端只提交 replyToMessageId。服务端校验同会话可见性，生成引用快照；不能信任客户端自报原作者或内容。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `messageId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `sender` | PlayerSummary | 是 |  |  |
| `content` | string | 是 | 长度 0–256 | 服务端取得的原消息引用内容 |
| `availability` | string | 是 | `AVAILABLE` / `UNAVAILABLE` |  |

### `SendChatRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientMessageId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `content` | string | 是 | 长度 1–256 | 纯文本消息 |
| `replyToMessageId` | string | 否 | 长度 1–128 | 可选被回复消息 ID |
| `mentionedPlayerRefs` | array<string> | 否 | 数量 0–20 |  |

### `ConversationView`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `conversationId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `otherPlayer` | PublicPlayerProfile | 是 |  |  |
| `lastMessage` | ChatMessage / null | 是 |  |  |
| `unreadCount` | integer | 是 | 范围 0–2147483647；int64 |  |
| `updatedAt` | string | 是 | date-time |  |
| `version` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |

### `CreateConversationRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `otherPlayerRef` | string | 是 | 长度 1–128 | 对方玩家，不能指定发送者 |

### `ConversationReadRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `lastReadMessageId` | string | 是 | 长度 1–128 | 服务端验证属于该会话 |

### `ConversationReadResult`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `unreadCount` | integer | 是 | 范围 0–2147483647；int64 |  |
| `lastReadMessageId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |

### `WalletRecordDetail`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `recordId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `direction` | string | 是 | `income` / `expense` |  |
| `amount` | CreditAmount | 是 |  |  |
| `currency` | string | 是 | `CREDIT` |  |
| `businessType` | string | 是 | `TRANSFER` / `STORE_PURCHASE` / `MARKET_RESERVE` / `MARKET_SETTLEMENT` / `COMMISSION_RESERVE` / `COMMISSION_SETTLEMENT` / `REFUND` / `ADJUSTMENT` / `AI_PURCHASE` |  |
| `businessRef` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `counterparty` | OrderParty | 是 |  |  |
| `status` | string | 是 | `PROCESSING` / `SUCCESS` / `FAILED` / `UNKNOWN` |  |
| `note` | string | 是 | 长度 0–500 | 业务备注 |
| `occurredAt` | string | 是 | date-time |  |

### `AppRelease`

AP​​K versionCode 必须大于当前版本且与包内版本一致；资源版本必须递增且兼容当前 APK。资源更新不加载代码。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `releaseId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `kind` | string | 是 | `APK` / `RESOURCES` |  |
| `versionName` | string | 是 | 长度 1–80 | 用户可读的版本名称 |
| `versionCode` | integer | 是 | 范围 0–2147483647；int64 | APK 版本号，资源包可为 0 |
| `resourceVersion` | integer | 是 | 范围 0–2147483647；int64 | 资源版本号，APK 可为 0 |
| `packageName` | string | 是 | 长度 1–200 | 目标 Android 包名；客户端仍核对实际 APK |
| `releaseNotes` | string | 是 | 长度 1–10000 | 纯文本更新说明 |
| `downloadUrl` | string | 是 | 长度 1–4096；uri | HTTPS 文件地址，不使用 bearer token；可使用短期签名 URL |
| `sizeBytes` | integer | 是 | 范围 1–209715200；int64 | 文件大小，APK ≤200 MiB，资源包 ≤20 MiB |
| `sha256` | string | 是 | 长度 64–64；正则 `^[a-fA-F0-9]{64}$` | 整个文件 SHA-256 |
| `minAppVersionCode` | integer | 是 | 范围 1–2147483647；int64 | 资源包兼容范围下界 |
| `maxAppVersionCode` | integer | 是 | 范围 1–2147483647；int64 | 资源包兼容范围上界 |
| `publishedAt` | string | 是 | date-time |  |
| `channel` | string | 是 | `stable` / `preview` |  |

### `ResourceManifest`

ZIP 根目录 manifest.json ≤64 KiB。仅 JSON/PNG/JPEG/WebP，总解压量≤100 MiB，禁止额外文件和重复路径。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `schemaVersion` | integer | 是 | 范围 1–1；int64 | 当前仅 1 |
| `resourceVersion` | integer | 是 | 范围 1–2147483647；int64 | 资源版本 |
| `minAppVersionCode` | integer | 是 | 范围 1–2147483647；int64 | 最低 APK 版本 |
| `maxAppVersionCode` | integer | 是 | 范围 1–2147483647；int64 | 最高 APK 版本 |
| `entries` | array<object> | 是 | 数量 1–500 |  |
| `entries[].path` | string | 是 | 长度 1–180 | 包内相对路径；不能有 ..、绝对路径、反斜线或冒号 |
| `entries[].sizeBytes` | integer | 是 | 范围 1–104857600；int64 | 解压后的大小 |
| `entries[].sha256` | string | 是 | 长度 64–64 | 单文件摘要 |

### `ReleaseArtifact`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `artifactRef` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `kind` | string | 是 | `APK` / `RESOURCES` |  |
| `status` | string | 是 | `PROCESSING` / `READY` / `REJECTED` |  |
| `sizeBytes` | integer | 是 | 范围 0–2147483647；int64 |  |
| `sha256` | string | 是 | 长度 64–64 | 服务端计算的摘要 |
| `packageName` | string | 是 | 长度 0–200 | 从 APK 读取 |
| `versionCode` | integer | 是 | 范围 0–2147483647；int64 |  |
| `resourceVersion` | integer | 是 | 范围 0–2147483647；int64 |  |
| `validationMessages` | array<string> | 是 | 数量 0–20 |  |

### `ReleaseWriteRequest`

文件字节数、SHA、实际包名/版本由服务端读取，不接受后台网页自报后直接信任。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `expectedVersion` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `artifactRef` | string | 是 | 长度 1–128 | 服务端已检查为 READY 的上传文件 |
| `kind` | string | 是 | `APK` / `RESOURCES` |  |
| `versionName` | string | 是 | 长度 1–80 | 展示版本名称 |
| `releaseNotes` | string | 是 | 长度 1–10000 | 更新内容 |
| `channel` | string | 是 | `stable` / `preview` |  |
| `minAppVersionCode` | integer | 是 | 范围 1–2147483647；int64 | 兼容下限 |
| `maxAppVersionCode` | integer | 是 | 范围 1–2147483647；int64 | 兼容上限 |

### `ReleaseAdminView`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `releaseId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `version` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `status` | string | 是 | `DRAFT` / `VALIDATING` / `PUBLISHED` / `WITHDRAWN` |  |
| `artifact` | ReleaseArtifact | 是 |  |  |
| `release` | AppRelease | 是 |  |  |
| `createdAt` | string | 是 | date-time |  |
| `updatedAt` | string | 是 | date-time |  |

### `MerchantIdentity`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `playerRef` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `storeIds` | array<string> | 是 | 数量 1–20 |  |
| `permissions` | array<string> | 是 | 数量 1–50 |  |

### `AdminIdentity`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `playerRef` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `permissions` | array<string> | 是 | 数量 1–100 |  |

### `StoreMember`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `playerRef` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `permissions` | array<string> | 是 | 数量 1–4 |  |
| `version` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |

### `StoreMemberWriteRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `expectedVersion` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `playerRef` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `permissions` | array<string> | 是 | 数量 1–4 |  |

### `TransactionEvidenceSnapshot`

服务端从成交快照、账本和审计日志取数据；客户端不能上传一个 JSON 冒充交易事实。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `snapshotId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `transactionKind` | string | 是 | `ORDER` / `COMMISSION` |  |
| `transactionId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `transactionVersion` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `capturedAt` | string | 是 | date-time |  |
| `sha256` | string | 是 | 长度 64–64 | 不可变快照摘要 |
| `order` | OrderView / null | 是 |  |  |
| `commission` | CommissionView / null | 是 |  |  |
| `eventLog` | array<object> | 是 | 数量 0–500 |  |
| `eventLog[].eventId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `eventLog[].type` | string | 是 | 长度 1–80 | 业务事件 |
| `eventLog[].actorPlayerRef` | string / null | 是 | 长度 0–128 | 系统事件可为空 |
| `eventLog[].occurredAt` | string | 是 | date-time |  |
| `eventLog[].summary` | string | 是 | 长度 1–1000 | 不包含密钥或密码 |

### `InterventionRequest`

requestedRefundAmount 仅表示诉求，不是付款指令。平台介入表单与网页管理端属于后续实现，本轮 App 不新增入口。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `expectedVersion` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `reasonCode` | string | 是 | `NOT_DELIVERED` / `NOT_AS_DESCRIBED` / `REFUND_DISAGREEMENT` / `OTHER` |  |
| `description` | string | 是 | 长度 10–3000 | 申请人描述事实和争议 |
| `desiredResolution` | string | 是 | `FULL_REFUND` / `PARTIAL_REFUND` / `CONTINUE_FULFILLMENT` / `OTHER` |  |
| `requestedRefundAmount` | CreditAmount | 否 |  |  |
| `evidenceAssetIds` | array<string> | 是 | 数量 0–10 |  |
| `relatedMessageIds` | array<string> | 否 | 数量 0–30 |  |

### `InterventionView`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `caseId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `transactionKind` | string | 是 | `ORDER` / `COMMISSION` |  |
| `transactionId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `applicant` | OrderParty | 是 |  |  |
| `respondent` | OrderParty | 是 |  |  |
| `status` | string | 是 | `SUBMITTED` / `IN_REVIEW` / `WAITING_EVIDENCE` / `RESOLVING` / `RESOLVED` / `WITHDRAWN` |  |
| `description` | string | 是 | 长度 10–3000 | 申请说明 |
| `desiredResolution` | string | 是 | `FULL_REFUND` / `PARTIAL_REFUND` / `CONTINUE_FULFILLMENT` / `OTHER` |  |
| `requestedRefundAmount` | CreditAmount | 是 |  |  |
| `evidenceAssetIds` | array<string> | 是 | 数量 0–10 |  |
| `snapshot` | TransactionEvidenceSnapshot | 是 |  |  |
| `fundsHeldForReview` | boolean | 是 |  | 是否实际冻结了尚未释放的担保款 |
| `assignedAdminRef` | string / null | 是 | 长度 0–128 | 管理员引用，未分配时 null |
| `resolution` | string | 是 | 长度 0–3000 | 处理结论 |
| `createdAt` | string | 是 | date-time |  |
| `updatedAt` | string | 是 | date-time |  |
| `version` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |

### `InterventionEvidenceRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `expectedVersion` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `description` | string | 是 | 长度 2–3000 | 补充说明 |
| `evidenceAssetIds` | array<string> | 是 | 数量 1–10 |  |
| `relatedMessageIds` | array<string> | 否 | 数量 0–30 |  |

### `InterventionResolutionRequest`

金额必须不超过实际冻结款。已结算订单不能伪造冻结款或强制扣成负数，只能进入人工追偿/协商处理。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `expectedVersion` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `decision` | string | 是 | `FULL_REFUND` / `PARTIAL_REFUND` / `RELEASE_TO_PAYEE` / `CONTINUE_FULFILLMENT` / `REQUIRE_MANUAL_RECOVERY` / `NO_ACTION` |  |
| `refundAmount` | CreditAmount | 否 |  |  |
| `reason` | string | 是 | 长度 10–3000 | 必须写明依据，形成审计记录 |

### `AuditEvent`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `auditId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `actorRef` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `permissionUsed` | string | 是 | 长度 1–80 | 权限码 |
| `action` | string | 是 | 长度 1–100 | 动作 |
| `resourceRef` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `requestId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `occurredAt` | string | 是 | date-time |  |
| `summary` | string | 是 | 长度 1–2000 | 脱敏说明 |

### `IdempotentRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |

### `RemovalResult`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `removed` | boolean | 是 |  |  |

### `StoreHomeSection`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `sectionId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `title` | string | 是 | 长度 1–80 | 区域标题 |
| `layout` | string | 是 | `HERO_CAROUSEL` / `PRODUCT_GRID` / `BANNER` |  |
| `productIds` | array<string> | 是 | 数量 0–40 |  |
| `bannerAssetId` | string / null | 是 | 长度 0–128 | BANNER 图片引用 |
| `sortOrder` | integer | 是 | 范围 0–2147483647；int64 |  |

### `StoreHomepage`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `storeId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `intro` | string | 是 | 长度 0–300 | 首页引导文案 |
| `sections` | array<StoreHomeSection> | 是 | 数量 1–20 |  |
| `brandIds` | array<string> | 是 | 数量 0–30 |  |
| `categoryIds` | array<string> | 是 | 数量 0–30 |  |
| `version` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |

### `StoreHomepageWriteRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `expectedVersion` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `intro` | string | 是 | 长度 0–300 | 首页引导文案 |
| `sections` | array<StoreHomeSection> | 是 | 数量 1–20 |  |
| `brandIds` | array<string> | 是 | 数量 0–30 |  |
| `categoryIds` | array<string> | 是 | 数量 0–30 |  |

### `StockAdjustmentRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `expectedVersion` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `delta` | integer | 是 | 范围 -999999–999999；int64 | 可售库存变更量；不得使库存为负或动用已预留数量 |
| `reason` | string | 是 | 长度 2–500 | 库存调整原因 |

### `OrderContractSnapshot`

可见性必须遵循正文 4.1 和对应端点对象授权；结构字段不赋予读取权限。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `snapshotId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `orderId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `buyer` | OrderParty | 是 |  |  |
| `seller` | OrderParty | 是 |  |  |
| `items` | array<OrderItemSnapshot> | 是 | 数量 1–100 |  |
| `totalAmount` | CreditAmount | 是 |  |  |
| `delivery` | DeliveryAddress | 是 |  |  |
| `confirmationHours` | integer | 是 | 范围 1–8760；int64 | 成交时约定 |
| `capturedAt` | string | 是 | date-time |  |
| `sha256` | string | 是 | 长度 64–64 | 不可变条款摘要 |

### `CommissionContractSnapshot`

可见性必须遵循正文 4.1 和对应端点对象授权；结构字段不赋予读取权限。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `snapshotId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `commissionId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `owner` | OrderParty | 是 |  |  |
| `worker` | OrderParty / null | 是 |  |  |
| `content` | CommissionContent | 是 |  |  |
| `acceptanceHours` | integer | 是 | 范围 72–72；int64 | 完成后的验收期限固定 72 小时 |
| `capturedAt` | string | 是 | date-time |  |
| `sha256` | string | 是 | 长度 64–64 | 不可变条款摘要 |

### `BrandCreateRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `name` | string | 是 | 长度 1–60 | 品牌名称 |
| `logoAssetId` | string / null | 是 | 长度 0–128 | Logo 引用 |
| `sortOrder` | integer | 是 | 范围 0–2147483647；int64 |  |
| `active` | boolean | 是 |  |  |

### `CategoryCreateRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `name` | string | 是 | 长度 1–40 | 商城分类名称 |
| `sortOrder` | integer | 是 | 范围 0–2147483647；int64 |  |
| `active` | boolean | 是 |  |  |

### `AnnouncementCreateRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `title` | string | 是 | 长度 1–100 | 公告标题 |
| `summary` | string | 是 | 长度 1–300 | 摘要 |
| `contentBlocks` | array<ContentBlock> | 是 | 数量 1–100 |  |
| `coverAssetId` | string / null | 是 | 长度 0–128 | 可选封面 |
| `pinned` | boolean | 是 |  |  |
| `priority` | string | 是 | `NORMAL` / `IMPORTANT` |  |

### `ReleaseCreateRequest`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `artifactRef` | string | 是 | 长度 1–128 | 服务端已检查为 READY 的上传文件 |
| `kind` | string | 是 | `APK` / `RESOURCES` |  |
| `versionName` | string | 是 | 长度 1–80 | 展示版本名称 |
| `releaseNotes` | string | 是 | 长度 1–10000 | 更新内容 |
| `channel` | string | 是 | `stable` / `preview` |  |
| `minAppVersionCode` | integer | 是 | 范围 1–2147483647；int64 | 兼容下限 |
| `maxAppVersionCode` | integer | 是 | 范围 1–2147483647；int64 | 兼容上限 |

### `V2PublicPlayerProfileResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | PublicPlayerProfile | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `OssUploadCreateRequest`

只提交 JSON 元信息，不包含 file/Base64/对象 URL。AVATAR 对应 PROFILE；STORE_MEDIA 对应 STORE 且 businessRef=有权店铺；DISPUTE_EVIDENCE 必须绑定本人参与的 ORDER/COMMISSION/INTERVENTION。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `purpose` | string | 是 | `AVATAR` / `STORE_MEDIA` / `MARKET_PHOTO` / `COMMISSION_COVER` / `DISPUTE_EVIDENCE` |  |
| `businessType` | string | 是 | `PROFILE` / `MARKET_LISTING` / `COMMISSION` / `ORDER` / `INTERVENTION` / `STORE` |  |
| `businessRef` | string | 否 | 长度 1–128 | 已存在业务的引用；证据/商家媒体必填并检查权限，新头像/商品/委托草稿可不填 |
| `fileName` | string | 是 | 长度 1–160 | 仅展示/诊断，不拼接对象路径 |
| `contentType` | string | 是 | `image/jpeg` / `image/png` / `image/webp` |  |
| `sizeBytes` | integer | 是 | 范围 1–20971520；int64 | 裁切/压缩之后的实际文件字节数 |
| `contentMd5` | string | 是 | 长度 24–24；正则 `^[A-Za-z0-9+/]{22}==$` | 同一最终文件的 Base64 MD5，后端将与 OSS 实际校验值比对 |
| `altText` | string | 否 | 长度 0–200 | 替代文字 |

### `OssPostAuthorization`

采用 OSS POST Policy 直传，policy 绑定唯一暂存 key、文件精确大小、类型与 private ACL；使用官方 SDK 签名，不自行实现加密算法。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `provider` | string | 是 | `ALIYUN_OSS` |  |
| `method` | string | 是 | `POST` |  |
| `uploadUrl` | string | 是 | 长度 1–4096；uri | 可信 OSS Bucket HTTPS 根地址；客户端向此地址传文件，不向业务 API 传文件 |
| `fileFieldName` | string | 是 | `file` |  |
| `formFields` | object | 是 |  | OSS SDK 签发的完整 POST Policy 表单字段，包含精确 key、policy、签名和必要限制；客户端逐项原样添加，最后添加 file。没有 AccessKeySecret。 |
| `expiresAt` | string | 是 | date-time | 上传授权过期时间，默认 5 分钟，且不超过会话期限 |
| `sizeBytes` | integer | 是 | 范围 1–20971520；int64 | Policy 限制的精确文件大小，最大 20 MiB |

### `OssUploadSession`

仅创建授权的当前账号可读/续签/提交验证。AUTHORIZED 不代表上传完成，OSS 201 不代表 READY；验证成功后仍需调用对应资料/商品/委托接口绑定 assetId。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `uploadId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `assetId` | string | 是 | 长度 1–128 | 授权时预分配，只有 READY 后才能用于业务写入 |
| `purpose` | string | 是 | `AVATAR` / `STORE_MEDIA` / `MARKET_PHOTO` / `COMMISSION_COVER` / `DISPUTE_EVIDENCE` |  |
| `status` | string | 是 | `AUTHORIZED` / `VERIFYING` / `READY` / `REJECTED` / `EXPIRED` |  |
| `authorization` | OssPostAuthorization / null | 是 |  |  |
| `sessionExpiresAt` | string | 是 | date-time | 会话最长 1 小时；续签不无限延长，过期重新申请 |
| `asset` | AssetView / null | 是 |  |  |
| `rejectionCode` | string / null | 是 | 长度 1–80 | 确定失败的业务码；无拒绝时 null |
| `retryAfterSeconds` | integer | 是 | 范围 1–30；int64 | VERIFYING 时建议查询间隔 |

### `OssUploadCompleteRequest`

只通知后端开始验证；不接收文件、任意 URL、objectKey、bucket、assetId 或客户端指定 READY。对象位置从 uploadId 的服务端记录读取。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `ossRequestId` | string | 否 | 长度 1–128 | OSS 返回的请求号，可用于排障，不能作为成功或归属凭证 |

### `V2OssUploadSessionResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | OssUploadSession | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2AssetViewResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | AssetView | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2RemovalResultResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | RemovalResult | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2OperationLookupResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | OperationLookup | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2WalletRecordDetailResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | WalletRecordDetail | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2StoreViewListResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | array<StoreView> | 是 | 数量 0–100 |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |
| `page` | Page | 是 |  |  |

### `V2StoreViewResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | StoreView | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2StoreHomepageResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | StoreHomepage | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2StoreBrandListResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | array<StoreBrand> | 是 | 数量 0–100 |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |
| `page` | Page | 是 |  |  |

### `V2StoreCategoryListResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | array<StoreCategory> | 是 | 数量 0–100 |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |
| `page` | Page | 是 |  |  |

### `V2StoreProductListResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | array<StoreProduct> | 是 | 数量 0–100 |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |
| `page` | Page | 是 |  |  |

### `V2StoreProductResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | StoreProduct | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2ShoppingBagResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | ShoppingBag | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2CheckoutQuoteResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | CheckoutQuote | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2OrderCreationResultResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | OrderCreationResult | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2MarketCategoryListResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | array<MarketCategory> | 是 | 数量 0–100 |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |
| `page` | Page | 是 |  |  |

### `V2MarketListingListResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | array<MarketListing> | 是 | 数量 0–100 |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |
| `page` | Page | 是 |  |  |

### `V2MarketListingResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | MarketListing | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2OrderViewListResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | array<OrderView> | 是 | 数量 0–100 |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |
| `page` | Page | 是 |  |  |

### `V2OrderViewResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | OrderView | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2OrderContractSnapshotResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | OrderContractSnapshot | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2OrderRefundResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | OrderRefund | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2CommissionViewListResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | array<CommissionView> | 是 | 数量 0–100 |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |
| `page` | Page | 是 |  |  |

### `V2CommissionCreationResultResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | CommissionCreationResult | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2CommissionViewResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | CommissionView | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2CommissionContractSnapshotResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | CommissionContractSnapshot | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2ChatMessageResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | ChatMessage | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2ConversationViewListResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | array<ConversationView> | 是 | 数量 0–100 |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |
| `page` | Page | 是 |  |  |

### `V2ConversationViewResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | ConversationView | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2ChatMessageListResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | array<ChatMessage> | 是 | 数量 0–100 |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |
| `page` | Page | 是 |  |  |

### `V2ConversationReadResultResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | ConversationReadResult | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2AnnouncementViewListResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | array<AnnouncementView> | 是 | 数量 0–100 |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |
| `page` | Page | 是 |  |  |

### `V2AnnouncementViewResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | AnnouncementView | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2NotificationPreferencesViewResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | NotificationPreferencesView | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2PushDeviceViewResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | PushDeviceView | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2NotificationViewListResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | array<NotificationView> | 是 | 数量 0–100 |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |
| `page` | Page | 是 |  |  |

### `V2NotificationReadResultResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | NotificationReadResult | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2MerchantIdentityResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | MerchantIdentity | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2StoreBrandResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | StoreBrand | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2StoreCategoryResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | StoreCategory | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2DeliveryTemplateListResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | array<DeliveryTemplate> | 是 | 数量 0–100 |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |
| `page` | Page | 是 |  |  |

### `V2MerchantProductListResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | array<MerchantProduct> | 是 | 数量 0–100 |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |
| `page` | Page | 是 |  |  |

### `V2MerchantProductResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | MerchantProduct | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2InterventionViewResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | InterventionView | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2AdminIdentityResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | AdminIdentity | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2InterventionViewListResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | array<InterventionView> | 是 | 数量 0–100 |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |
| `page` | Page | 是 |  |  |

### `V2AuditEventListResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | array<AuditEvent> | 是 | 数量 0–100 |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |
| `page` | Page | 是 |  |  |

### `V2StoreMemberListResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | array<StoreMember> | 是 | 数量 0–100 |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |
| `page` | Page | 是 |  |  |

### `V2StoreMemberResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | StoreMember | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2ReleaseArtifactResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | ReleaseArtifact | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2ReleaseAdminViewListResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | array<ReleaseAdminView> | 是 | 数量 0–100 |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |
| `page` | Page | 是 |  |  |

### `V2ReleaseAdminViewResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | ReleaseAdminView | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `GlassMaterialConfig`

bottomBarGlass 和 overlayGlass 各保存独立完整对象，禁止共享可变参数或自动复制到另一组。允许值以本文范围为准，NaN/Infinity/未知字段拒绝。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `enabled` | boolean | 是 |  | 远程材质总许可；false 必须关闭本组；true 仍尊重用户本机关闭开关 |
| `blurRadiusDp` | number | 是 | 范围 0–36；float | 模糊半径 dp |
| `opacity` | number | 是 | 范围 0.08–1；float | 表面不透明度，0.08 至 1 |
| `refractionStrength` | number | 是 | 范围 0–24；float | 折射强度，无量纲 |
| `highlightStrength` | number | 是 | 范围 0–1；float | 边缘高光强度 |
| `dynamicHighlightEnabled` | boolean | 是 |  | 本组是否允许动态追光；仍尊重设备能力和用户减少动态效果 |
| `allowLocalTuning` | boolean | 是 |  | true 时用户本组自定义数值优先；false 时远程数值生效，但仍允许本机关闭玻璃 |

### `HeaderGradientConfig`

独立顶部渐变参数，不从底栏或浮层隐式继承；固定采用平滑渐变算法，不能远程下发 shader/脚本。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `blurRadiusDp` | number | 是 | 范围 0–36；float | 顶部渐变模糊半径 dp，独立于两组玻璃 |
| `fadeHeightDp` | number | 是 | 范围 16–96；float | 顶部渐变扩展参数，对应现有 headerFade；不代表完整工具栏高度 |
| `allowLocalTuning` | boolean | 是 |  | 是否保留用户顶部参数自定义 |

### `AppearanceSettings`

三组必须完整返回；修改一组不会影响其他组。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `bottomBarGlass` | GlassMaterialConfig | 是 |  |  |
| `overlayGlass` | GlassMaterialConfig | 是 |  |  |
| `headerGradient` | HeaderGradientConfig | 是 |  |  |

### `AppearanceOverrides`

管理记录的三组覆盖项；null 表示本范围不覆盖，PLAYER 继承 GLOBAL，GLOBAL 继承内置默认。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `bottomBarGlass` | GlassMaterialConfig / null | 是 |  |  |
| `overlayGlass` | GlassMaterialConfig / null | 是 |  |  |
| `headerGradient` | HeaderGradientConfig / null | 是 |  |  |

### `AppearanceSources`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `bottomBarGlass` | string | 是 | `BUILTIN` / `GLOBAL` / `PLAYER` |  |
| `overlayGlass` | string | 是 | `BUILTIN` / `GLOBAL` / `PLAYER` |  |
| `headerGradient` | string | 是 | `BUILTIN` / `GLOBAL` / `PLAYER` |  |

### `EffectiveAppearance`

只读取当前登录玩家最终配置，不接受 playerRef 替代身份。合成结果不是修改设备成功的回执。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `schemaVersion` | integer | 是 | `1` |  |
| `packageName` | string | 是 | 长度 1–200 | 已登记的应用包名 |
| `channel` | string | 是 | `stable` / `preview` |  |
| `globalVersion` | integer | 是 | 范围 0–2147483647；int64 | 全局管理记录版本；未建立时为 0 |
| `playerVersion` | integer | 是 | 范围 0–2147483647；int64 | 当前玩家管理记录版本；未建立时为 0 |
| `effectiveRevision` | string | 是 | 长度 1–200 | 服务端生成的版本标识，绑定账号/包名/频道/schema；客户端只作相等比较 |
| `settings` | AppearanceSettings | 是 |  |  |
| `sources` | AppearanceSources | 是 |  |  |
| `refreshAfterSeconds` | integer | 是 | 范围 60–3600；int64 | 前台建议拉取间隔，不要求后台常驻 |
| `updatedAt` | string | 是 | date-time | 当前合成配置最近变更时间 |

### `AppearanceAdminRecord`

每个 packageName + channel + scope + playerRef 独立一条记录；不存在时 GET 返回 version=0、全 null 覆盖项，便于第一次 PATCH。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `scope` | string | 是 | `GLOBAL` / `PLAYER` |  |
| `playerRef` | string / null | 是 |  |  |
| `packageName` | string | 是 | 长度 1–200 | 包名作用域 |
| `channel` | string | 是 | `stable` / `preview` |  |
| `version` | integer | 是 | 范围 0–2147483647；int64 | 本范围乐观锁版本；首次读取为 0，更新后递增且重置也不归零 |
| `overrides` | AppearanceOverrides | 是 |  |  |
| `updatedAt` | string / null | 是 | date-time | 尚未写入时为 null |

### `AppearancePatchRequest`

至少提供一组。省略组保持原样；对象必须包含本组所有字段；null 清除本组覆盖并恢复继承。不接受同时修改金额、权限、主题或执行代码。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `clientRequestId` | string | 是 | 长度 1–128 | 客户端业务幂等键；重试复用，同键不同正文返回冲突 |
| `expectedVersion` | integer | 是 | 范围 0–2147483647；int64 | 目标范围当前 version；首次写入填 0 |
| `reason` | string | 是 | 长度 2–500 | 配置变更或回退原因，用于管理员审计 |
| `bottomBarGlass` | GlassMaterialConfig / null | 否 |  |  |
| `overlayGlass` | GlassMaterialConfig / null | 否 |  |  |
| `headerGradient` | HeaderGradientConfig / null | 否 |  |  |

### `V2EffectiveAppearanceResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | EffectiveAppearance | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `V2AppearanceAdminRecordResponse`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `requestId` | RequestId | 是 |  |  |
| `data` | AppearanceAdminRecord | 是 |  |  |
| `serverTime` | string | 是 | date-time | 服务端当前 UTC 时间 |

### `AppearanceChangedEvent`

仅提示重新 GET /app/appearance，不能直接覆盖 UI。GLOBAL 发给该包名/频道受影响的已登录会话，PLAYER 仅目标玩家的匹配会话；不向其他玩家公开目标 ID 或专属配置。事件丢失靠前台重新拉取恢复。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `type` | string | 是 | `appearance.updated` |  |
| `sentAt` | string | 是 | date-time |  |
| `payload` | object | 是 |  |  |
| `payload.eventId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `payload.packageName` | string | 是 | 长度 1–200 | 包名范围 |
| `payload.channel` | string | 是 | `stable` / `preview` |  |
| `payload.scope` | string | 是 | `GLOBAL` / `PLAYER` |  |
| `payload.recordVersion` | integer | 是 | 范围 1–2147483647；int64 | 发生变化的记录版本 |

### `ResourceChangedPayload`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `eventId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `resourceId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `resourceVersion` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `occurredAt` | string | 是 | date-time |  |
| `status` | string | 是 | 长度 1–80 | 对应订单或委托状态 |

### `BusinessResourceEvent`

可见性必须遵循正文 4.1 和对应端点对象授权；结构字段不赋予读取权限。

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `type` | string | 是 | `order.updated` / `commission.updated` |  |
| `requestId` | string | 否 | 长度 1–128 | 服务端生成的不透明引用 |
| `sentAt` | string | 是 | date-time |  |
| `payload` | ResourceChangedPayload | 是 |  |  |

### `NotificationCreatedEvent`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `type` | string | 是 | `notification.created` |  |
| `sentAt` | string | 是 | date-time |  |
| `payload` | object | 是 |  |  |
| `payload.eventId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `payload.notification` | NotificationView | 是 |  |  |

### `CatalogChangedEvent`

| 字段 | 类型 | 必填 | 约束 | 说明 |
| --- | --- | --- | --- | --- |
| `type` | string | 是 | `catalog.changed` |  |
| `sentAt` | string | 是 | date-time |  |
| `payload` | object | 是 |  |  |
| `payload.eventId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `payload.storeId` | string | 是 | 长度 1–128 | 服务端生成的不透明引用 |
| `payload.productIds` | array<string> | 是 | 数量 1–100 |  |
| `payload.resourceVersion` | integer | 是 | 范围 1–2147483647；int64 | 乐观锁版本。修改必须提交最近读取的版本 |
| `payload.occurredAt` | string | 是 | date-time |  |

## 12. 对接验收清单

- 任意非本人订单、私聊、推送设备、证据 ID 均不能越权读取或修改；商家必须受店铺 ACL 限制。
- 同一幂等键重试只扣/退/发一次；同键不同金额、商品或委托内容返回冲突。
- 报价后改价、库存被他人购买、并发接取/取消、退款/自动确认竞争均有确定结果。
- 拒绝理由不能空白；撤回/拒绝不恢复退款机会；平台介入快照由服务端生成。
- 工程完成不重置总工期；委托未完成逾期不结算，完成后 72 小时才自动结算。
- 断网和进程恢复要查服务端终态，不能用手机倒计时修改正式余额。
- 通知按参与方和偏好投递，消息复制不联网，回复不能跨会话引用不可见消息。
- APK 错包名、旧版本、错签名、大小/哈希错误均阻止安装；无来源许可交给系统授权。
- 资源包不兼容、重复/越界路径、清单缺失、多余文件、错误哈希、超大图片/配置均阻止切换，旧资源仍可使用。
- 本机外观设置和模拟扫脸不产生伪造的服务器认证证据。

正式接入前还需确认：部署域名与 CORS、推送适配器、商城邮箱模板、担保经济桥实现、管理人员权限及人工争议处理流程。上述待配置项不影响本轮本机体验，但不能用演示数据代替线上服务。
