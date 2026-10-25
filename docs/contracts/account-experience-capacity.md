# 账号会话、身份冲突与查询容量

2026-10-14，本机候选。产品决定见[实施计划](../plans/account-experience-capacity.md)，证据见[本机验收](../qa/account-experience-capacity-20261014.md)。没有上线或改变现有发布版本。

## 会话与设备

- `app` 原生 token 无定时到期，数据库 `expires_at=NULL`；可主动退出、改密撤销、封禁撤销和远程退出。
- `web` Cookie 会话为签发起 30 天；恢复网页会话仅将 Cookie 对齐数据库的原到期时间，不滑动续期。启动器内嵌社区也使用网页会话。启动器微软登录沿用自身刷新机制，不在本次平台会话策略内。
- `gamechat` 继续为十分钟、受在线 Core 和接口白名单约束的专用令牌，不展示为平台登录设备，也不因“退出其他设备”被撤销。既有改密/封禁的处理保持。
- 登录、注册统一调用 `identity.IssueSession` 的同一签发策略。客户端 `deviceName` 仅为展示标签，限制长度并去掉控制字符；不收集硬件序列号，不用设备名称代替凭据或作为互踢依据。网页从 User-Agent 提取浏览器/系统标签。
- 每个有效设备最多每五分钟更新一次最近活动；不增加客户端轮询。鉴权仍读取数据库中的撤销和账号状态，不依赖事件先送达。

| 接口 | 行为 |
| --- | --- |
| `GET /api/v1/account/sessions` | 当前账号最近 200 个有效 App/Web 会话；返回 `sessionId/clientKind/deviceName/createdAt/lastSeenAt/expiresAt/current`，不返回 token、token hash、CSRF 或其他账号设备 |
| `DELETE /api/v1/account/sessions/{sessionId}` | 只退出本账号指定设备；重复退出成功，跨账号返回 404；退出当前网页同时清 Cookie |
| `POST /api/v1/account/sessions/revoke-others` | 空 JSON；退出本账号全部其他 App/Web 会话，保留当前会话；不受列表 200 条上限影响 |

未验证账号可以管理自己的设备。网页写操作继续验证 Origin/CSRF；游戏聊天 token 不得进入以上接口。撤销在事务中复查当前会话与账号状态，落审计后唤醒既有用户事件通道。

## 未验证占用处理

所有接口要求现有认证规则；管理接口只允许 `platform.admin`，工单管理员不能直接解除登记。

| 接口 | 输入和结果 |
| --- | --- |
| `POST /api/v1/admin/accounts/{userId}/profile-release-code` | 空 JSON；从目标登记 UUID 定位 Core 可信在线会话，向该游戏玩家投递验证码。独立用途 `profile_claim`，十分钟、最多五次错误、一次性消费；Core 线上协议仍使用现有 `register` 投递文案 |
| `POST /api/v1/admin/accounts/{userId}/profile-release` | `clientRequestId/expectedVersion/claimantEmail/evidenceType/evidenceReference/reason/ownershipConfirmed/password`；游戏证明另带 `verificationToken/code`。`evidenceType` 为 `GAME_CODE` 或 `SUPPORT_REVIEW`；返回 `caseId/userId/released/version` |
| `GET /api/v1/admin/accounts/{userId}/profile-resolutions` | 最近 50 条处理记录，含操作者、时间、证据引用、原名、原认领邮箱/当前认领邮箱、注册结果和最多 200 条邮箱更正记录 |
| `POST /api/v1/admin/profile-reservations/{caseId}` | `clientRequestId/expectedEmail/claimantEmail/reason/password`；仅更正尚未完成注册的持有人邮箱，保留前后邮箱和理由，不能移交已创建账号 |
| `POST /api/v1/account/game-profile` | `clientRequestId/gameId`；仅允许已登录且原登记已解除的未验证账号重新登记。已有登记或已验证者不能在此换绑。成功重试直接返回持久结果，不依赖正版接口再次可用 |

解除事务保留原 userId/playerRef、邮箱、密码和业务历史；清除公开 UUID/游戏登录别名，原账号以邮箱登录并重新登记。原会话与旧游戏挑战失效。已验证身份、有 `server_uuid` 的异常记录、已删除账号、缺少邮箱恢复入口、管理员本人，均不得走此快捷流程。

原登记者不能立即抢回已释放 UUID：它暂时预留给工单核实的邮箱，该邮箱仍需验证码注册。登记成功只消耗预留，不授予游戏资产、白名单或 `game_verified`。正常未验证账号没有到期、自动释放或清理。

预留邮箱不匹配时，在邮件投递前拒绝，最终注册事务再次加锁校验。旧客户端对已解除登记账号请求游戏验证码时返回 `GAME_PROFILE_REQUIRED`，提示通过最新版 App 或网页重新登记，不向 Core 发出空名字查询。

游戏验证码记录与处理记录关联；人工核验记录必须说明可靠依据，公开资料、名字和截图不单独构成证明。密码/验证码不保存到幂等请求、处理结果或审计正文。管理员密码及当前会话在事务中复核，重复请求返回原结果，变更请求正文须使用新请求键。

## 容量与失败恢复

- 名字/UUID 查询统一为 8 个外部请求、最多 128 个待执行查询、256 个等待调用者；等待最多 3 秒，单个共享查询总期限 8 秒。同一在途请求按规范化名字/UUID 合并。一个调用者取消不会取消其他调用者，全部取消则停止工作。
- 官方查询池只合并在途请求，共享响应携带实际查询开始时间，继续执行迟到证明保护。2026-10-25 登录候选改为本地平台别名＋正确密码优先，成功后后台按 UUID 刷新；未知名字/本地凭据不匹配才同步查询。名字复用的暂时别名滞后与事务复核边界见[实施计划](../plans/login-fast.md)，不宣称本地别名与官方即时持有人始终等价。注册沿用既有十分钟资料缓存及邮箱验证码中保存的 UUID，最终提交通常不重复查询。
- 密码计算仍最多 2 个（既有 Argon2 内存参数保持），等待队列 128、等待最多 5 秒。容量不足返回 503 与 `Retry-After`，不计为密码错误。
- 登录来源流量为每 IP 每分钟 240 次；账号别名和稳定 userId 的失败保护为各 8 次/15 分钟，包含计算中的预留。成功、外部故障、取消和容量不足只释放本次、同一时间窗口的预留，不清除并发错误记录。
- 快速路径补充：已经完成本地密码计算且明确错误时，后续官方故障不释放这次实际错误；若成功定位到名字当前持有人并验证其正确密码，则整体视为成功。尚未判错密码的外部失败/容量不足继续不计密码错误。
- 登录/注册/改密请求使用独立 128 槽 HTTP 预算。游戏通行检查/心跳、Core 连接、实时消息和普通 HTTP 保留各自预算；不在游戏通行检查中执行正版查询或密码计算。
- 邮件投递与游戏验证码的用途、冷却和投递额度保留。本次百人测试针对账号登录和游戏通行响应，不宣称百人同 IP 同时发验证码注册。
- 超级管理员既有 `/api/v1/admin/runtime/capacity` 增加 `minecraftProfiles` 和 `authentication`：在途/合并/排队/拒绝数、累计查询时间、密码计算与等待、登录与会话写入耗时；不记录玩家密码、原始 token 或请求正文。

## 迁移与回退

新增 `043_account_sessions.sql`、`044_profile_conflicts.sql`；旧迁移内容保持。043 只延长 active 账号尚未到期且未撤销的 App/Web 会话，不复活旧 token；044 仅建表。发布前应备份数据库与旧程序，并核对各类账号、业务记录及现有迁移校验。

**不能直接切回旧 Go。** 旧程序不认识永久会话的 NULL 到期时间，也不会填写新 `session_id` 必填列。若必须回退，先停止新写入，在备份及维护窗口内：

1. 将 NULL 到期的 App 会话改为短期到期或撤销，并填充有限到期时间；确认无 NULL 后恢复 `expires_at NOT NULL`。
2. 将 `session_id` 临时改为允许 NULL/default NULL，使旧程序能签发会话；保留新增表与所有处理记录，不恢复旧整库。
3. **已有身份解除或未完成认领时，禁止回退到不识别预留表的旧注册代码。** 应使用保留预留校验的回退构件，或在维护窗口关闭注册/身份编辑入口直至恢复新程序，避免绕过已确认的归属处理。
4. 再升级时，先补齐回退期间新建会话的 `session_id`（沿用 043 的确定性哈希生成式）、恢复必填约束，并重新核对会话策略。不得以重新执行/篡改历史迁移校验代替此核对。

2026-10-14 14:19 已执行 043/044 生产迁移并上线，原账号和业务全字段校验保持；没有执行回退。见[上线记录](../deployment/account-experience-2.0.29-live.md)。
