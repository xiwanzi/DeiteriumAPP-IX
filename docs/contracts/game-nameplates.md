# 游戏铭牌与头衔（2026-10-12，本机候选）

已按用户授权于 2026-10-12 上线，见[发布记录](../deployment/nameplates-0.0.51-live.md)。

- 头衔使用稳定 ID；不可修改、不可复用的内部 ID 组成 `deuterium.title.<internalId>`。管理员修改显示名后，佩戴记录仍引用同一 ID，游戏动态取最新定义。
- 每个头衔增加 `description`（介绍及获取方式，最多 1000 字，支持换行）与 `public`（公开展示）。已拥有列表仅按权限显示，允许私有头衔；全部列表只显示公开且已发布的头衔，即使佩戴者本人拥有私有头衔也不在全部列表中展示。公开不等于授予权限，修改这两项不改变头衔 ID 或发布顺序。
- 显示名支持内联 `#RRGGBB`，颜色持续至下一个色码；纯文本默认为 `#A9B3B8`。最多 32 个可见字符，不接受控制符或其他富文本指令。
- 状态为草稿、已发布、停用、已删除。删除保留内部 ID 墓碑。默认 `title_default` / `default` 为“溯行者”，不可停用或删除，无需 LP 权限。
- 排序：权重降序，同权重发布序号降序。首次发布及重新发布分配新序号，修改文字不改变发布顺序。
- Core 在游戏主线程直接读取 LuckPerms 当前上下文的权限缓存；只接受 TRUE，不使用 Bukkit 的默认 OP 权限。客户端不能提交已拥有权限列表。选择前重新检查；在线轮询处理权限过期，正常网络下数秒内传播。
- 每次主动选择记录历史。当前头衔失效时向更早的佩戴历史回退到最近仍有效的头衔；都无效则默认。恢复权限不自动重新佩戴已回退的头衔。
- Go 保存 UUID、当前选择和历史；Core 请求必须来自认证节点，并匹配在线玩家所在节点与会话。选择使用版本和请求 ID 防止并发覆盖与重试重复。
- 后台仅平台管理员可写，使用已有鉴权、审计、幂等和版本冲突机制。游戏目录只公布已发布项；默认项始终可用。
- 选择及权限不依赖 App 注册。玩家跨服使用相同 UUID 状态。停用、删除、重命名都通过目录和玩家状态同步到游戏。

验收覆盖：管理权限、内部 ID 唯一、重命名引用、渐变解析、排序、选择幂等与并发、撤权/过期回退及恢复权限不反弹。Web 预览使用明确的本地示例状态，不冒充已部署或真实授权。

## 接口

后台 `GET /api/v1/admin/titles` 返回目录；`POST /api/v1/admin/titles` 创建，`POST /api/v1/admin/titles/{titleId}` 修改。写入包含 `clientRequestId`、`expectedVersion`、`internalId`、`displayName`、`weight`、`status`。停用/删除属于版本化状态修改。

Core 专用 POST 前缀 `/bridge/v1/game-chat/titles/`：`catalog` 返回已发布目录与佩戴投影，`state` 进行授权对账，`select` 主动选择。后两项必须匹配在线 UUID、所在节点、`sessionEpoch` 和 `catalogRevision`；`allowedIds` 仅接受经认证 Core 的计算结果。`select` 另带 `titleId`、`clientRequestId`、`expectedVersion`。

游戏铭牌通道为 `deuterium:titles`，只在物理客户端注册 optional 原始 UTF-8 JSON codec；专用服务器由 Core Bukkit 通道独占。Youer 会丢弃与模组服务端同名注册冲突的 Bukkit 通道，故不在 MMO 服务端注册此 ID。聊天的 `player-title` 展示投影仍沿用 `deuterium:chat`。

客户端 `titles-list` 含 `requestId`、`offset`、`ownedOnly`（默认 true）、`query`；Core 先做权限/公开过滤，再搜索、计数、分页。每页最多 24 项且目录 JSON 小于 20 KiB，返回 `nextOffset`、`revision`、`items`、`ownedCount`、`total` 和 `current`。每项含 `description`、`public`、`owned`。中途目录 revision 变化时客户端重读首页。

`title-select` 含 `titleId`、`requestId`、`expectedVersion`，结果 `title-selected`；自己的佩戴变化同步至铭牌通道。未知结果保留原请求 ID；收到新权威版本后新的明确选择才使用新请求。页面关闭不取消已经提交的选择。拒绝/断线不得把本机选择显示为佩戴成功。

目录桥全量同步限制 1 MiB；桥失联超过 15 秒不接受新选择，在线展示退到默认，持久历史保留。原生页面固定 640×360 参考画布（1920×1080 / GUI 3），由现有 Transform2D 连续适配。说明区内部滚动；长头衔按真实字体测量省略，预览括号保留。字距与省略由原生 Label 扩展实现，圆角为 SDF，模糊复用菜单六遍半径 22 与 WavePoint。页面自有 Shader 生命周期不使用 LDLib2 TextureValue 的十秒共享缓存。
