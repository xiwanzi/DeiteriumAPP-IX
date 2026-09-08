# Deuterium Core 远程协议 v1

2026-09-08。阶段 A 的已实现消息为节点握手、公共聊天和物品版本发布；邮件／经济命令在独立章节定义为待接入，不可按消息名推断已运行。邮箱保留独立插件，由用户修改；Core 只调用邮箱 API。

## 1. 连接与节点权限（已实现）

Core 主动连接 `wss://<api>/bridge/v1/connect`。请求头：

```http
Authorization: Bearer <独立节点的随机密钥>
X-Deuterium-Node-ID: amiya
```

不允许 Cookie、Origin 或查询参数。后端从配置把凭据绑定到 serverId；负载中的其他 serverId 不作为来源权威。每节点一个活动连接，第二条连接拒绝，不踢走已认证连接。密钥泄露时移除／更换该节点配置并重启服务撤销连接，不能共用四服密钥。

握手后 5 秒内发送：

```json
{"type":"core.hello","requestId":"hello-1","payload":{"protocolVersion":1}}
```

响应 `core.welcome`，payload 含 protocolVersion、serverId、chat、itemPrefix、heartbeatSeconds=20、maxFrameBytes=32768。只有已配置 chat=true 的节点能上传聊天；只有 itemPrefix 非空的节点能发布其命名空间物品。能力升级先改配置与契约，不接受节点自行宣称财务权限。

双方支持 WebSocket ping/pong；后端每 20 秒 ping，5 秒无响应关闭。JSON 文本帧最大 32 KiB，压缩关闭。重连采用带随机抖动的指数退避，建议 1 秒至 30 秒；稳定连接后再归零。普通网络断线不丢弃待确认事件。

## 2. 持久化事件信封（已实现）

```json
{
  "type":"chat.public.event",
  "eventId":"amiya-chat-01",
  "requestId":"transport-attempt-01",
  "payload":{
    "playerUuid":"d97161f9-2a7c-4abd-a8e7-6fd64a64c001",
    "gameId":"Alice",
    "content":"晚上一起建房子"
  }
}
```

eventId 是节点内稳定业务事件身份；requestId 是本次传输关联 ID，可重试时变。Core 先持久化事件，再发送。后端在一个事务中写事件去重记录和业务结果，提交后响应：

```json
{"type":"core.event.result","payload":{"eventId":"amiya-chat-01","status":"committed","sequence":1,"replayed":false}}
```

响应还包含标准 sentAt、原 requestId。相同 serverId+eventId+内容重投返回同 sequence、replayed=true；相同 ID 不同内容返回 rejected / IDEMPOTENCY_CONFLICT。数据库错误返回 unknown，Core 保留原事件并重投，不能换 eventId；只有 committed 才可以从出站日志清理。格式／能力错误为 rejected，进入本地错误队列，不能无限快速重发。

后端时间为记录时间；不信任游戏节点自报时间决定交易期限。事件保留到迁移／归档明确完成，不能清除去重记录后又接收旧事件。

## 3. 多服公共聊天（已实现服务端；Core/TrChat 适配待实施）

`chat.public.event` 只包含本服玩家原始、审核通过的公共消息。玩家 UUID 必须取代理转发后的实际玩家身份；禁止按游戏名重新生成离线 UUID。内容 trim 后 1–256 字符，无控制字符。不得采集私聊、管理频道、验证码或已桥接消息。

四服上报都写入同一公共消息流，App/网页登录后接收原有 `chat.message`。**游戏消息不再由后端回发其他游戏服**，避免和现有 TrChat 的游戏服互通重复。

App 的原有 `chat.send` 使用会话身份，不能携带 sender／UUID。后端先持久化消息及每个配置目标服的投递记录，再返回 accepted；至少一个聊天节点在线才接收新 App 消息。accepted 指后端已记录，不是四服全部显示成功。相同 clientMessageId 的重试即使当前离线，也可查询原接收结果。

各 Core 收到：

```json
{"type":"chat.app.delivery","payload":{"messageId":"msg_opaque","senderUuid":"d97161f9-2a7c-4abd-a8e7-6fd64a64c001","gameId":"Alice","content":"晚上一起建房子","origin":"app","expiresAt":"2026-09-08T06:10:00Z","suppressRebroadcast":true}}
```

Core 按 messageId 持久去重，使用不会被 TrChat 再次传播或 Core 再次采集的显示入口。安全转义颜色／富文本控制，不把内容当命令或 MiniMessage 执行。接受到持久化投递日志后发送 `chat.delivery.ack`，payload `{messageId}`；后端按当前认证节点确认，只能确认本节点的投递。

ACK 表示 Core 已持久接收，不证明玩家已经阅读。聊天显示本身不与数据库处于同一事务；崩溃发生在实际显示前后时必须保留可诊断状态，不能承诺严格 exactly-once 的屏幕副作用。正常重连不重复显示，10 分钟后过时聊天不补刷。后端每 5 秒补查未 ACK 项，单批 50。

App 目前已接入基础公共聊天；带提及或引用的消息明确返回 CAPABILITY_UNAVAILABLE，后续阶段实现后开放，不静默丢掉这些语义。私聊绝不进入该通道。

## 4. 物品版本同步（已实现服务端；物品库适配待实施）

只有配置发布命名空间的节点发送 `item.version.published`：

```json
{
  "type":"item.version.published","eventId":"item-battery-v3",
  "payload":{
    "itemRef":"deuterium:battery","revision":3,
    "payloadSha256":"a4dc378aedfae66eb2a44c5f80c7b034147465a542a634ef2a6165da90a4c8a7f",
    "displayName":"储能电池","description":"发货时按第 3 版物品快照解析",
    "maxQuantity":64,"compatibleServerIds":["amiya","odyssey"],"requiredMods":["mekanism"]
  }
}
```

摘要示例仅演示格式，不对应真实物品。itemRef+revision 不可变；换 payload、名称、数量上限或兼容范围都要新 revision。后端不接收原始 NBT／Bukkit 对象／文件路径；原始内容由物品库保留并计算 SHA-256，Core 发货时按相同版本读取、验证摘要。

metadata 是展示信息，前端按纯文本处理。compatibleServerIds 必须是已登记节点；它不是最终邮件领取许可，仍要与管理员选择、本服禁领开关和同步条件取交集。

管理端可通过 `GET /api/v1/admin/core/nodes` 和 `GET /api/v1/admin/core/items` 查询。当前需要 `core.read` 权限，由本机 CLI 显式授予，不从旧账号或游戏 OP 自动推导。items 分页使用 afterItemRef+afterRevision，单批 50；只返回元数据和摘要，不返回任何节点密钥。

## 5. 独立邮箱与经济（待接入）

具体调用时序见 [独立邮箱交接](mailbox-integration-v1.md)。Core 在本地通过邮箱公开 API 调用，不访问其数据库。缺少关键能力时上报 unavailable，后端禁止真实商城付款，不排队等恢复后暗中扣款。

旧 `bridge-commerce-v2.md` 的 wallet.escrow reserve/release/refund/query 语义仍适用，但 Vault 的 withdraw/deposit 本身不提供原子担保。需要受控经济适配及可恢复执行证据；本服务版本未实现或启用任何真实扣款／退款消息。

未来新增命令保持 operationId、deliveryId、orderId 与快照绑定，结果为 PROCESSING／COMPLETED／FAILED／UNKNOWN。UNKNOWN 只能查原身份恢复，禁止换 ID 重发。正式开启前完成 Linux 后端到 Windows Core、独立邮箱、同步插件和经济插件的联合故障测试。
