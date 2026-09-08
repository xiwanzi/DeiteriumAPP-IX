# App 2.0.3 接入说明

2026-09-09。Android 2.0.3 与同批 Go 配套，沿用原账号、资金和游戏插件协议。

## 记录显示

`POST /orders/{orderId}/hide`、`POST /commissions/{commissionId}/hide`、`POST /market/listings/{listingId}/hide` 接收 `clientRequestId` 与 `expectedVersion`，返回 `data.hidden=true`。App Bearer、Web Cookie/CSRF 继续使用同一鉴权。相同请求重试幂等，不接受其他玩家的资源身份。

交易仅在 CONFIRMED / CLAIMED / REFUNDED / CANCELLED，且资金 SETTLED / REFUNDED / UNPAID、无待处理操作、退款或介入时可隐藏；市场商品仅发布者可以隐藏下架/售罄记录。拒绝版本过期和仍活跃的记录。

`personal_record_visibility_v203` 按用户/资源类型/资源编号保存显示偏好。个人列表过滤，详情追加 `canHideRecord`、`hiddenFromHistory`；交易源数据、快照、账本、通知、对方历史及商家管理视图保留。后续资源重新活跃或资金未知时重新显示，不能用隐藏偏好掩盖未处理义务。客户端抑制删除前已经在途的列表响应，避免刷新把旧记录加回来。

公开委托列表原本已限定 OPEN / HELD / 无待处理操作；App 本次修正合并个人历史后的展示过滤，已接取/取消/结算记录不回到大厅。历史仍从 `/commissions/me` 读取。

## 公共聊天

`POST /chat/messages` 已实现，输入沿用 `SendChatRequest`，与 WebSocket `chat.send` 共用持久 `clientMessageId`、鉴权、频控、回复/提及权限和游戏节点可用性检查。

HTTP 200 的 `data.status` 为 accepted / failed / unknown；accepted 带 `messageId`，正常情况下同时带规范化 `message`，便于立即显示；不能将 HTTP 200 本身当作成功。accepted 只表示已持久接受、进入游戏投递队列，不证明全部游戏节点已 ACK。业务拒绝放在 `data.error`；格式/鉴权等失败仍返回 HTTP 错误。

App 发送不等待 WebSocket 握手或历史 GET；接受后清除待发送状态，历史在后台补齐。断网未知结果保留原正文、回复、提及和编号，重试不换键、不制造假成功。

联系人使用现有 `unreadCount`，前台定期更新摘要；进入可见私聊后才推进服务端读游标。已读回执成功再清红点，后台同步与历史翻页不替代已读。

## 付款与 Markdown

付款保留权威商品/金额校验与原幂等订单创建；按用户最新要求，界面只显示“是否支付 ×× 信用点？”及取消/确认付款，不展示商品明细和内部报价警告。

公告和 AI 正文使用 [Markwon 4.6.2](https://noties.io/Markwon/docs/v4/install.html) 的原生文本跨度支持标题、强调、列表、引用、代码、表格、删除线和链接；不使用 WebView。链接只允许 HTTP/HTTPS，长按消息操作保留。当前未增加 Markdown 远程图片加载或 LaTeX 插件。

完整结构见 [OpenAPI](openapi-app-v2.yaml) 的新增响应与显示字段。
