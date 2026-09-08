# 社交与内容服务 v2

2026-09-08。实现于 `internal/httpapi/social_v2.go`、`internal/store/social_v2.go`、`010_social_v2.sql`。与 Go Foundation 共用账号、Cookie/CSRF、权限和 MariaDB，不创建演示账号或示例业务内容。

## 已确认的边界

- 资料只能修改本人；QQ、UUID、登录标识与权限不由资料接口改动。查看资料必须登录。简介上限 200 字；头像绑定本人完成验证的 AVATAR 资产。
- 私聊只存在于两个已注册玩家之间；同一双方只有一个会话。请求不能指定发送者。每一次请求重新验证登录、成员与权限。
- 回复只引用当前会话的可见原消息，由数据库生成原作者和正文快照。转发可把本人有权读取的公共消息/私聊转入本人有权参与的私聊；不得传入自称的原作者或正文。
- 幂等记录与业务结果同一事务提交；同一用户、动作、请求标识换正文返回冲突。不同会话复用同一私聊发送标识也冲突。
- 私聊通知只给另一位参与者；提及第三人不能绕过成员隔离。公共提醒只基于已经提交的公共消息。发布公告才产生通知，编辑草稿不发布。
- 资料、公告与通知偏好使用版本校验；旧版本写入返回冲突。已读位置只能前进。

## 已实现接口

| 接口 | 结果 |
| --- | --- |
| GET `/players/{playerRef}` | PublicPlayerProfile；未知最近在线时间为 null，不伪造在线状态 |
| PATCH `/account/me/profile` | `{clientRequestId,expectedVersion,bio?,avatarAssetId?}`；头像 null 恢复默认 |
| GET `/chat/player-directory` | 兼容 `{data:{players:[]}}`，支持 query，最多 100 项 |
| GET/POST `/chat/follows`、DELETE `/chat/follows/{playerRef}` | 关注数据持久化、按登录用户隔离，POST 使用既有 `{playerRef}` |
| GET/POST `/chat/conversations` | v2 列表/会话；创建使用 `{clientRequestId,otherPlayerRef}` |
| GET/POST `/chat/conversations/{id}/messages` | 列表倒序，支持 cursor 或 beforeMessageId；发送复用 SendChatRequest |
| POST `/chat/conversations/{id}/read` | `{clientRequestId,lastReadMessageId}`，不允许标记第三方消息 |
| GET `/announcements`、GET `/announcements/{id}` | 只显示已发布版本；保存新草稿期间公开内容不变 |
| GET/POST `/admin/announcements`、GET/PUT `/admin/announcements/{id}` | 真实公告草稿与编辑；权限 announcements.manage 或 platform.admin |
| POST `/admin/announcements/{id}/publish`、`/unpublish` | clientRequestId + expectedVersion；撤下需要 reason |
| GET `/notifications`、POST `/notifications/read` | 通知数组及按接收人校验的已读动作；支持 unreadOnly 和 cursor |
| GET/PATCH `/notifications/preferences` | 完整业务偏好与版本；不把业务开关冒充系统推送权限 |

以上除特别注明的 v1 形状外，使用 v2 `{requestId,data,serverTime,page?}`。所有列表为空时返回空数组。列表游标携带资源/用户作用域，私聊和通知不能交叉使用另一人的游标。

## 本轮转发与管理扩展

新增 `POST /chat/conversations/{目标会话id}/forwards`：

```json
{"clientMessageId":"本次发送的稳定标识","sourceMessageId":"服务器消息引用","sourceConversationId":"原私聊会话引用"}
```

公共源省略 sourceConversationId 或传 public。私聊源必须填写原会话引用；服务端同时检查源可读与目标可发。返回普通 ChatMessage，并增加 `forwarded`，其形状与 ReplySnapshot 一致：messageId、sender、content、availability。不存在或无权读取的源统一 404。私聊/公共源都不能自报正文或原作者。该接口是原 v2 文档未覆盖的增量，不是另一套聊天协议。

管理公告响应在 AnnouncementView 上增加 status、publishedVersion、hasUnpublishedChanges，区分草稿、已发布和已撤下。公开响应不返回这些管理字段。创建草稿的 publishedAt 为 null，公开公告发布时间来自真正 publish 操作。

公告读取另增加可选 `cover`（AssetView）与 `media`（AssetView 数组）。通过已鉴权的正文引用和业务绑定生成短期下载 URL，客户端用 contentBlocks[].assetId 匹配 media；公开端仅解析已发布快照中的引用，不把新草稿图片暴露给普通玩家。这些字段是明确的本轮契约扩展，旧版 additionalProperties=false 的 schema 需要应用对应补丁后再进行完整符合性校验。

## 图片与事务

头像存储资产引用，每次资料读取重新生成受控下载链接，不永久缓存会过期的签名 URL。资料更新在同一业务事务调用 SetAssetBindingsV2，避免并发删除打坏头像。

公告媒体使用本轮新增 ANNOUNCEMENT_MEDIA。保存草稿保留“新草稿 ∪ 当前公开版本”的资产绑定；发布新版本才释放被替换的公开资产。其他有权限的管理员可以继续使用该公告已经绑定的媒体，但新增媒体仍需属于本人。系统不将任意外部 URL 当成可发布资产。

## 公共消息提醒接点

`Store.PublicSocialNotificationsV2(ctx, sourceID, clientMessageID)` 只能在原公共消息事务提交后调用。App/Web 源是 `app:<userId>`，游戏源是 `core:<nodeId>`。方法从已提交消息读取作者和正文，生成提及/特别关心通知；同一消息、接收人只生成一条，提及优先于特别关心。

本轮保留 Core 的普通聊天投递内容，并在 App/Core 公共消息提交后接入即时提醒 hook；公共 WS payload 增量保持兼容。现阶段私聊与通知可以通过 HTTP 轮询跨设备同步。没有接入系统 Push 供应商时，不将通知入库描述为设备离线推送成功。公共聊天的原 Core 投递/失败语义继续由原链路决定。

新增 `014_social_reconcile.sql` 保存补扫游标，维护任务调用 `ReconcilePublicSocialNotificationsV2(ctx)`，每次最多 50 条、每条通知提交成功后单独推进游标。原消息已提交但进程在即时 hook 前退出，也会在重启后补齐；游标提交前退出则重扫并依靠通知唯一事件键去重。特别关心只匹配关注时间不晚于消息时间的关系，不因新关注回溯提醒全部历史。

`015_public_chat_metadata.sql` 保存公共引用快照和显式 mentionedPlayerRefs，与公共消息及 Core 投递队列同事务提交。原 `chat.send` 支持 replyToMessageId/mentionedPlayerRefs；同标识更改引用或提及集合冲突，旧无元数据消息仍兼容原指纹。公开回复只能引用公开消息，私聊 ID 不可用于此入口；显式提及必须是已注册身份或可信公共记录里的玩家引用。HTTP 历史/WS ChatMessage 返回元数据，游戏投递仍只携带普通 content，不携带私聊正文。即时 App/Core hook 和补扫都读取数据库里的显式提及，提及通知不依赖文本里一定存在 @名字。

## 本机验证

8 组真实 MariaDB HTTP 集成 + 4 项纯单元通过：资料版本/幂等、空数据、关注、私聊成员/引用/转发隔离、通知隔离、12 并发重复发送、分页作用域、已读不回退、公告草稿/发布/撤下、管理员权限、偏好、头像归属、图片删除互斥、跨管理员编辑、公开提醒去重、公共源转发、重启补扫及公共引用/显式提及的原子校验。

测试仅使用明确给出的回环 MariaDB 测试连接，随机创建 `deuterium_test_*` 库并清理自己的库；未修改生产经济或账号。执行：`go test -tags integration ./internal/httpapi ./internal/store -run Social -count=1`。`DEUTERIUM_TEST_DSN` 必须显式设置，测试不会读取生产 DSN。

尚需主任务核实：实际部署、Core 公共提醒挂接、权威在线状态、网页/App 最终联调与系统推送供应商。010 迁移部署后不修改校验值；新增结构另起迁移。
