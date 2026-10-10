# YSM 网站身份查询

网站继续持有账号与角色权威。Cloud 完成正版会话证明后，以正版 UUID 查询既有网站身份；玩家无需重复注册、绑定或确认。此接口只读，不新增业务账号、身份绑定或数据库表。

## 请求与资格

`GET /internal/v1/ysm/identities/{uuid}`，UUID 为小写带连字符格式，无查询参数。请求头为 `Authorization: Bearer <service-token>`。

Go 配置 `ysmIdentityTokenSha256` 保存独立随机服务令牌的 SHA-256 十六进制摘要；Cloud 配置 `SPM_SITE_IDENTITY_ORIGIN` 和 `SPM_SITE_IDENTITY_TOKEN` 保存网站地址与原始令牌。客户端不持有此令牌。未配置时该接口拒绝请求。

响应沿用标准成功信封，`data` 包含：

| 字段 | 含义 |
| --- | --- |
| userId | 既有网站账号 ID |
| profileUuid | identities.server_uuid |
| modelAccess | 已验证且有效账号的模型资格 |
| administrator | 是否持有 platform.admin |

账号必须 UUID 精确匹配、`game_verified=TRUE` 且 `status='active'`。`support.manage` 不授予 YSM 管理员权限。错误服务凭据返回 401，UUID 错误返回 400，无有效身份返回 403，数据库故障返回 503。网站会话、游戏节点凭据、客户端角色字段均不能替代服务凭据。

## 配套行为与验证

Cloud 复用已有内部身份和模型/贴图/参数；普通资格缓存不超过 30 秒，打开增强选单时强制刷新，查询失败不使用过期权限。网站角色不转换为 bootstrap 服务令牌；资源与 NPC 管理沿用现有 Go→Velocity→Cloud 链路及网站管理鉴权。

2026-10-10 本机 MariaDB 集成及 Go `TestYSM` 测试通过，覆盖已验证用户、未验证/禁用/不存在账号、管理员与普通角色、管理接口拒绝等情况。隔离游戏测试使用两个明确标记的测试 UUID：普通用户无法打开增强菜单，撤销管理员角色后增强菜单关闭；查询未创建网站账号。它不代表生产 Mojang 登录或线上部署完成。

2026-10-10 已通过 PR #94 合入并上线，配套客户端为 0.0.34-dev。生产现有两个 Cloud 正版身份均匹配有效网站账号，一个具有管理员角色；实际正版客户端自动恢复外观与菜单检查通过。服务令牌通过私有运行配置交付，不写入文档、源码或客户端。详见[上线记录](../deployment/ysm-website-identity-20261010-live.md)。
