# 聊天玻璃独立参数 App 2.0.25 / Web 2.0.34 上线

2026-10-10 **01:55（UTC+8）**，App **2.0.25（22500）**、Web **2.0.34** 已上线。[PR #91](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/91)，源码 `984ceee1163dd567de1f15ba7d9a2fa4bdeae11f`，合并 `22df11463f3b5804424fb37e07ed47f39e25ed10`。

输入栏和表情贴纸面板分别使用独立的浅深色材质，面板增加遮挡与模糊、降低反光，提高复杂消息背景下的可读性。App “材质细节”支持分别调整、保存和还原；旧设置保留。面板仍仅顶部圆角，左右贴边并铺满底部手势区。默认参数及本轮交互验证见[验收记录](../qa/chat-glass-materials-20261010.md)。

## 产物与验证

[下载 2.0.25](https://47.103.99.34/downloads/Deuterium-2.0.25-22500-test.apk)，包名 `com.deuterium.app.uilab`，9,055,762 字节，SHA-256 `68b784105a1461c7dbecbbd72fdc05b267160867d416d9114b2fc8015f09975a`。证书 SHA-256 `04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`，与原版一致。

- 本轮候选 Android 172 项 JVM、浅深色原生交互与参数保存检查通过；Web 81 项单元、13 项媒体与 19 项动效回归通过。
- 升版后重新执行 `:ui-lab:assemblePerformance :ui-lab:lintPerformance`，R8/资源压缩与 lint 通过；Web 生产构建通过。正式 APK 模拟器覆盖安装与启动通过（2464ms），不等同真机手感验收。
- 公网 APK 完整字节及哈希一致；20004/21500/22200/22400/22499 返回 22500，22500/22501 无重复更新或降级，原 25 个清单条目保留。没有另发账号通知。
- 两个 Web 域名版本、JS/CSS、Worker 及两份 WASM 与本机产物一致，WASM MIME 正确。
- 服务 active/ready，贴纸 API、商城首批 40 件读取正常；三常驻游戏节点在线，MEK 保持停服。临时只读验收会话已撤销，无生产聊天或游戏投递。

## 数据与回退

切换前后 **58 条账号记录、44 笔订单、5 个委托、3,085 件在售商品、12 个工单、2 条注销记录**及既有业务 ID 集合保留。Go 二进制、迁移、配置、官网/客服中心及游戏插件不变；仅重启现有 Go 加载更新清单。本轮未重复 Go 业务测试。

备份 `/var/backups/deuterium/chat-tuning-v225-22500-984ceee1163d`；Web `/var/www/deuterium/releases/web-2.0.34-984ceee1163d/dist`；Go 继续为 `/opt/deuterium/releases/chat-media-v222-60b0cb525842`。

回退可恢复 Web 2.0.33 链接及旧更新清单并重启现有后端，不恢复整库覆盖新增业务数据。

本机交付 `C:/DeuteriumAPP/delivery/Deuterium-2.0.25/`，维护工作区 `.worktrees/chat-media` / `xiwanzi/release-v225-live`。

[构建清单](artifacts/v225/build-manifest.json) · [切换记录](artifacts/v225/deployment.json) · [公网验证](artifacts/v225/verification.json)
