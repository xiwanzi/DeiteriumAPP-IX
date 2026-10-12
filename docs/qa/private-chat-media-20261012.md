# App 私聊表情与媒体验收

2026-10-12。用户要求私聊与公屏一致，完成后直接发布。基于 `origin/main 416f8d1`，维护 `.worktrees/chat-media` / `xiwanzi/private-chat-media`。本轮已于 23:50 随 App 2.0.26 上线，见[部署记录](../deployment/deuterium-2.0.26-live.md)。

## 实现

- 私聊复用公屏输入栏、Emoji 分类/最近使用、光标插入/删除、个人贴纸库、单张系统选图与发送确认抽屉。保留浅深色独立玻璃、面板顶部圆角和贴底手势区布局。
- 图片、GIF、静态/动态贴纸复用压缩、圆角、播放、缩放预览、长按收藏及本机缓存；私聊消息历史和联系人摘要仍使用已有私聊接口。
- 待发送队列保存会话 ID、稳定玩家引用和原请求 ID，仅显示在对应私聊；公屏排除私聊任务。切换联系人、重启、失败重试不会改投其他会话。旧公屏待发送记录仍兼容。
- 私聊在三天内使用后端已有持久幂等回执恢复；超过三天仅分页查询当前用户的原会话历史核对，不重新发送未提交的旧图片/永久贴纸。异常接收人元数据拒绝恢复，不降级到公屏。
- AI 会话和 Web 无变更。后端已随游戏聊天提供私聊媒体、成员授权与回收，本次无需更换 Go、插件或迁移数据库。

## 本机验证

- Android 172 项 JVM 通过；QA/performance 构建、R8、资源压缩和 lint 通过。发布 APK 包名 `com.deuterium.app.uilab`，版本 2.0.26 / 22600，沿用原签名。
- 独立 `.chatmediaqa` 应用浅深色私聊验收均通过：真实 Emoji 点击与私聊发送、接收图片/贴纸展示及查看器、共享贴纸点击发送、系统 Photo Picker 选图→确认预览→发送。
- 两个私聊的待发送隔离、公屏草稿不变、只调用私聊发送路径；回执丢失后重启、原收件人/请求/引用保持，已提交超期消息只读分页恢复，未提交超期贴纸不发送，未超期失败请求原样重试通过。
- 公屏浅深色回归：图片/GIF、播放与关闭动效、贴纸圆角、Emoji、键盘/面板、查看器、附件抽屉、500MiB 缓存、两种清理及未知结果恢复通过。
- 本机 MariaDB 专项 `TestPrivateAndNearbyMediaCannotEscapeRecipientGrant`、`TestChatMediaDedupPermissionsQuotaAndExpiry`、`TestChatMediaHTTPAuthorizationAndSignedDeadline` 在 `-race -tags integration` 下通过。验证接收人可读、第三方不可读、去重/名额/期限/授权；未重复完整 Go 测试。

日志位于 `C:/DeuteriumAPP/.tools/private-media-{release-build,backend,native-light,native-dark,public-light,public-dark}.log`；截图位于 `C:/DeuteriumAPP/output/private-chat-media/`。原生验证使用本机假账号、回环 HTTP 和合成图片，未向真实玩家发私聊，不等同真机手感或真实弱网验收。
