# 玩家社区域名上线

2026-09-19 **21:12:28（UTC+8）**，`https://chat.deuteriumix.com` 已接入现有玩家社区。用户本轮授权配置域名、HTTPS 和配套来源兼容。

## 变更

- DNS A 记录已由用户配置，公网解析核对为 `47.103.99.34`，TTL 300。
- 新增 Nginx `/www/server/panel/vhost/nginx/deuterium-chat.conf`：80 跳转 HTTPS，保留 ACME 验证路径；443 使用独立证书，沿用当前网页目录、API / WebSocket / SSE / 下载 / 启动器和申请路径代理。
- Let's Encrypt 证书覆盖 `chat.deuteriumix.com`，初次有效期至 2026-12-18；复用已启用的 `deuterium-certbot-renew.timer` 和校验后重载 Nginx 的 deploy hook。指定此证书的续期 dry-run 成功。
- Go 新增 `additionalPublicOrigins` 精确白名单，生产增加 `https://chat.deuteriumix.com`。原 `publicOrigin`、申请来源与申请链接仍保留 IP，原生 App、旧网页和启动器入口无需改变。
- 没有更换 Web 2.0.19、App 2.0.13、启动器、申请站或游戏插件，没有数据库迁移。

## 产物与备份

| 项目 | 值 |
| --- | --- |
| Go 源码 | `a1064819ac0a00466352c78b66805ec32eb3aec7` |
| Go 目录 | `/opt/deuterium/releases/chat-domain-a1064819ac0a` |
| Go SHA-256 | `3020ba909cf019da3ca458ff54c6f75fd6d3462ff56c5f144805f305f8190aaf` |
| 原 Go | `/opt/deuterium/releases/backend-concurrency-a824088cb518` |
| 备份 | `/var/backups/deuterium/chat-domain-20260919`，含数据库一致性备份、旧配置、旧 IP 站点、新站点 HTTP 配置及切换前后数量 |

发布前无活动数据库事务、无尚未到期的 pending/streaming AI 请求。后端正常停启并就绪后启用新站点；原 IP Nginx 文件未修改。切换前后 72 条身份记录、34 条 commerce_resources、25 条钱包转账和 27 条迁移记录数量一致。数量核对不等于全库逐字段比较。

## 本次验证与限制

- `go test ./...` 全部通过，包括新旧登录来源通过、未知/空/null/近似域名拒绝，以及来源配置的 HTTPS、路径、凭据和通配符校验。
- Go 1.27.1 构建 Linux amd64 / CGO=0，生产可执行文件 SHA-256 与发布包一致，服务 active、ready。
- 公网新域名 TLS 验证通过，主页 200，HTTP 301 到 HTTPS。新旧域名的网页配置、申请页和申请配置 API 均为 200。
- 对实际 `/api/v1/web/session` 提交畸形 JSON：新旧受信来源进入请求校验返回 400，外站来源返回 403；没有提交真实账号密码。
- Playwright 在正常证书校验下显示“登录 · Deuterium ID”，页面图片加载成功、无 HTTP 混合资源。匿名会话探测 401 为未登录状态；没有声称控制台零错误。
- 证书 dry-run 成功，续期 timer enabled / active。
- 本次没有使用真实账号登录、创建测试会话、发送消息或进行真实交易；真实账号登录后的聊天收发未作生产验收。WebSocket 白名单改动与 HTTP 使用相同配置。

## 回退与维护

后续部署必须保留 `additionalPublicOrigins` 支持，避免旧程序严格解析配置失败。回退时停止 Go，恢复备份 `config.json`（owner deuterium、mode 600），原子切回旧目录并启动；同时恢复新域名站点为备份的 HTTP-only 配置或禁用该站点，再校验及重载 Nginx。旧 IP 站点原文件始终保留。不要恢复数据库覆盖上线后的用户业务。

证书续期依赖公网 80 端口和 `/.well-known/acme-challenge/` 路径可达，后续更改重定向时应保留此例外。更换主域名入口后，浏览器在新域名首次使用需重新登录。
