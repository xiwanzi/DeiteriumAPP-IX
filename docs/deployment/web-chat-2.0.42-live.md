# 社区 Web 2.0.42 聊天更新

2026-11-18 02:46（UTC+8），[PR #139](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/139) 合入主线。功能提交为 `f42b61d45ac6`，合并提交为 `40d7a0a0b275`，两者的 Web 源码树一致。02:47:40 原子切换社区静态站点。具体交互及真机验证边界见[三轮验收](../qa/web-chat-workspace-20261117.md)。

- 发布版构建、91 项 Node 测试、57 项本机浏览器检查通过。发布 ZIP 共 34,241,614 字节，SHA-256 为 `de6dbe26b4611cd60cf1a0aedecfbae703b769502c72cd9d03f767f4950f0b06`。
- 服务器解包后逐一核对 287 个构建文件的 SHA-256，原媒体文件内容全部核对一致。新目录额外保留上版的两个哈希命名 JS/CSS 文件，供旧页面继续请求。
- 社区域名和原 IP 的 HTTPS 入口各核对 16 项，包括首页、`/information`、配置、入口 JS/CSS、壁纸、媒体工作线程及表情样本，响应内容均匹配发布构建。
- 从正式域名加载生产 JS/CSS 和表情资源后，通过 57 项浏览器检查，259 个本地表情均可加载。HTTP 业务接口与 WebSocket 使用模拟数据，未发送真实玩家消息。手机交互验证范围为 Chromium 触屏模拟。
- 此次发布仅切换 `/var/www/deuterium/current`。Go 进程切换前后 PID 均为 `3602880`。没有数据库迁移或游戏服务器操作。

当前目标为 `/var/www/deuterium/releases/web-2.0.42-f42b61d45ac6/dist`。如需回退，可将社区入口链接原子切回保留的 `/var/www/deuterium/releases/web-2.0.41-bec12a61ada8/dist`。
