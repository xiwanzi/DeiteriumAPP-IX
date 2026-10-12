# App 私聊表情与媒体 2.0.26 上线

2026-10-12 **23:50（UTC+8）**，App **2.0.26（22600）** 已上线。[PR #119](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/119)，源码 `01cdac40bbc468fe15d9c889ca72b7c8f75cd042`，合并 `4a8aaedd53e9e70579e33acffa48475c5484fddc`。

私聊输入栏、Emoji、图片/GIF、个人贴纸及预览/收藏/失败重试与公屏共用组件。待发送项固定原会话与玩家引用，切换联系人、重启和重试不改变收件人；超过三天仅查询原会话历史确认，不重新发出未提交的旧媒体。详见[验收记录](../qa/private-chat-media-20261012.md)。

## 产物与验证

[下载 2.0.26](https://47.103.99.34/downloads/Deuterium-2.0.26-22600-test.apk)，包名 `com.deuterium.app.uilab`，9,072,142 字节，SHA-256 `494225a20ef70eb2dc97438b9285674886c409eafa032aa71bd3b7ece9cd03ec`。证书 SHA-256 `04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`，与原版一致。

- 172 项 JVM、QA/performance、R8、资源压缩与 lint 通过；浅深色私聊专项和公屏回归通过。真实系统选图、发送预览、两会话隔离、未知结果、重启/超期恢复均在隔离应用验证。
- 3 项既有后端私聊/共享媒体权限专项 `-race` 集成通过，包括第三方不可读。没有向真实玩家发送测试私信。
- 正式 APK 从 22500 覆盖安装成功，冷启动 2036ms；模拟器不等同真机手感验收。
- 公网 APK 完整字节与哈希一致。20004/21500/22200/22500/22599 获取 22600；22600/22601 不重复更新、不降级；原 26 个发布条目保留，没有额外发账号通知。
- 两个社区域名仍为 Web 2.0.38，首页及 5 份 JS/CSS/Worker/WASM 资源与现有服务器文件一致。服务 active/ready，贴纸 API、商城首批 40 件及三常驻节点读取正常；MEK 保持停止。临时只读验收会话已撤销。

## 保留与回退

本次仅更新 APK 与发布清单，短暂重启既有主 Go 以加载清单。Web **2.0.38**、主 Go `/opt/deuterium/releases/nameplates-610ca57`、数据库迁移及配置、官网/客服站、游戏插件和图片审核独立服务保持；图片审核 PID `2776531` 未变。

切换前后 **61 账号、47 订单、5 委托、3,085 件在售商品、12 工单、2 条注销记录**及既有业务 ID 集保留。

备份 `/var/backups/deuterium/private-media-v226-22600-01cdac40bbc4`；暂存 `/opt/deuterium/releases/private-media-app-2.0.26-01cdac40bbc4`。回退仅恢复旧更新清单并重启现有主 Go，不恢复整库覆盖新增业务数据。

本机交付 `C:/DeuteriumAPP/delivery/Deuterium-2.0.26/`；维护 `.worktrees/chat-media` / `xiwanzi/release-v226-live`。

[构建清单](artifacts/v226/build-manifest.json) · [切换记录](artifacts/v226/deployment.json) · [公网验收](artifacts/v226/verification.json)
