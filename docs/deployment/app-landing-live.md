# App 宣传页上线（2026-09-19）

21:43 后续修正：用户指出旧图标，现已将页面五处图标及 favicon 替换为当前 App 正式白底黑色分体弧线 D，直接复用 `web-app/src/assets/launcher-icons/default.svg`。线上图标配置只读确认为 `default` / version 3。资源更名为 `app-icon-default-v208.svg` 避免旧缓存；页面保持圆角裁切。当前目录为 `/var/www/deuterium-app/releases/app-landing-icon-e352b6d`，前一静态版本仍保留用于回退。浏览器验证五处加载、favicon 新地址、期待弹窗及无 APK 链接通过；图标文件 SHA-256 为 `aca22a58f9740d02b7aa4bb192ba5a1372f7cac3ff9e05cae91688e2e77ae4ec`。下方为首次部署记录。

**21:36:58（UTC+8）**，用户确认的第七版宣传页已部署到 https://app.deuteriumix.com 。用户本轮要求删除“尚未连接正式服务”等旧说明，App 下载和网页版仍为“敬请期待”。

## 范围与入口

- 保留原版布局、社区图片、带壳 App 截图、历史支付录屏及交互。
- 移除本机示例数据、未接正式服务、0.9.0 下载包和已开放下载说明；历史支付录屏仍标注其真实版本。
- 全页 14 个 App / 玩家网页 / 商家按钮均弹出“敬请期待”，没有跳转到 chat 域名。
- 发布包只包含当前引用的 32 个静态文件，不含 APK、下载目录、旧版预览页或本地工具。
- 此域名独立静态站点，`/downloads/` 与 `.apk` 返回 404。原 chat 域名、IP 入口、现有 App 下载服务和后端均保持；本轮未重启 Go 或操作数据库。

## 部署

| 项目 | 值 |
| --- | --- |
| 源码 | `c32a08d`，仓库 `app-landing/` |
| 站点目录 | `/var/www/deuterium-app/releases/app-landing-c32a08d` |
| 当前链接 | `/var/www/deuterium-app/current` |
| Nginx | `/www/server/panel/vhost/nginx/deuterium-app.conf` |
| 备份及发布包 | `/var/backups/deuterium/app-landing-20260919` |
| 发布包 SHA-256 | `c12039d8d75e397be5fac9f28784c9ea7a0a874bf624a84dc21c4e193cffbb6e` |
| 证书 | Let's Encrypt `deuterium-app`，首次有效期至 2026-12-18 |

用户添加 A 记录后公网确认指向 `47.103.99.34`、TTL 300。80 保留 ACME 验证路径，其余跳转 HTTPS；证书复用现有续期 timer 与 Nginx deploy hook，指定证书 dry-run 成功。

## 本次验证

- JS 语法检查通过；本机及公网 Playwright 的下载和网页版弹窗检查通过。
- 390 / 1440 宽度无横向溢出、无已完成但加载失败的图片，首屏截图已人工查看；控制台零错误与警告。
- 公网 14 个入口均打开期待弹窗，APK 链接数为 0，旧说明文本不存在。
- 历史视频 Range 请求返回 206；正常浏览器证书校验通过。
- `/downloads/Deuterium-UI-Lab-0.9.0.apk`、`/downloads/`、旧预览路径与 `/api/v1/web/session` 均返回 404；服务器上线首页与发布源文件逐字节一致。
- 原 chat 和 IP 入口 `/web-config.json` 仍为 200 / Web 2.0.19。

## 回退

这是独立静态站，回退只需恢复备份 `http-only.conf` 到本次新增的 Nginx 站点配置并校验、重载。后续静态版本可切换 `current` 软链接。不要修改 chat/IP 站点、后端或数据库。
