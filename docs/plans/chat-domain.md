# 网页域名接入（2026-09-19）

用户已授权将现有网页版接入 `https://chat.deuteriumix.com`，保留 Android 和启动器使用的旧 IP 入口。本轮不变更业务数据、数据库结构、App 或网页版本。

Nginx 新增独立域名站点，沿用现有网页目录和代理规则，使用独立 Let's Encrypt 证书及现有续期任务。后端新增 `additionalPublicOrigins` 精确来源列表，主来源保持不变；HTTP 登录、会话 CSRF、WebSocket 和申请来源校验使用相同的已配置来源，不允许通配符、任意来源或省略网页 Origin。原生 App 无 Origin 的既有行为保持。

验收：配置拒绝不安全来源；新旧网页来源可进入登录校验，外站被拒绝；WebSocket 来源兼容；新域名证书、静态资源、API、旧 IP 和申请站可达；证书续期 dry-run。保留旧程序、配置和 Nginx 备份，切换失败恢复程序与配置，不恢复数据库。
