# YSM 网站身份接入上线

2026-10-10 19:17（UTC+8），网站身份接口与配套 Cloud、客户端 0.0.34-dev 已上线。网页保持 2.0.35，Android 保持 2.0.25；游戏服与 Velocity 未重启，MEK 保持停止。

源码 cfab4420a1f6638e7042dc49207e6f737737e22e 经 [PR #94](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/94) 合入主线 43417c84e258302e3c565d1128782280b3b0ee10。Go 程序位于 /opt/deuterium/releases/ysm-identity-cfab4420a1f6，SHA-256 fe498970b73177ebae77e06299270b1c435fa88af7da600b2fcd2f1470def24d。

部署前备份位于 /var/backups/deuterium/ysm-identity-cfab4420a1f6，包含数据库、原配置、Nginx 和原 release 指向。数据库备份 167,740,710 字节，SHA-256 1ccafb74a218028871797f49b9cea1ab4324543244934222499140ca6bb3721f。上线后核对原有 ID：59 账号、49 订单/委托记录、9,328 目录记录、12 工单、2 注销记录、61 聊天消息全部保留。

本轮无数据库迁移、账号创建或绑定变更。Go 配置仅新增 ysmIdentityTokenSha256；原 backend.env 和网页 release 不变。Nginx 增加 /internal/v1/ysm/identities/ 专用代理，凭据由 Go 校验，不扩大其他内部路由。独立服务令牌仅保存在私有服务配置。

验证：Go/MariaDB TestYSM 测试通过；生产两个既有 Cloud 正版身份均获得网站资格，一个具备管理员角色；匿名身份请求 401、不存在账号 403。Go→Velocity→Cloud 管理链路正常，283 模型、3 NPC、三个常驻节点在线，匿名管理请求拒绝。隔离完整游戏客户端以正版账号进入正式服，不打开菜单即可恢复现有模型/贴图并复用 283/283 缓存，原版菜单与网站管理员增强入口通过，测试后已退出。

配套模型源码与客户端见 [DeuteriumYSM PR #3](https://github.com/Deuterium-IX/DeuteriumYSM/pull/3)。客户端增量包只更新 Sparkle JAR 与开屏版本配置；发布任务 ls_dd91c56eb8d24383d732f696b3b4ca7c 成功，公开完整下载与 Range 206 校验通过。

如需回退 Go，恢复备份中的原 current 目标和配置后重启 deuterium.service；应与 Cloud 网站身份开关配套处理。不要用旧数据库覆盖上线后的新业务写入。
