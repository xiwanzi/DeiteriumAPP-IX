# 聊天玻璃与贴底面板 App 2.0.24 / Web 2.0.33 上线

2026-10-10 **00:47（UTC+8）**，App **2.0.24（22400）**、Web **2.0.33** 已上线。[PR #89](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/89)，源码 `ab3893a935bdb3d0577903d753572fb8c916673f`，合并 `73db6f9ab9b71f77106691353198019dd5f56437`。

聊天玻璃改为取样实际消息与页面底色，输入栏与表情面板采用不同模糊/遮罩层次。表情面板**仅顶部两个角圆润，左右贴边、底部直铺到屏幕底边，覆盖手势导航区**；控件在内部避让手势。已有贴纸圆角与媒体行为保留。

上线后用户要求继续提高质感与可读性，并独立调整输入栏/面板参数；后续调校应先完成本机验证，不将此记录当作后续参数已发布的证据。

## 产物与验证

[下载 2.0.24](https://47.103.99.34/downloads/Deuterium-2.0.24-22400-test.apk)，包名 `com.deuterium.app.uilab`，9,055,762 字节，SHA-256 `0a597b1f33576edff5f52e7610daa1c6196632f7f52c033344f3bcb7f88bd9b5`。证书 SHA-256 `04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`，与原版一致。

- Android 172 项 JVM、performance 构建/R8/资源压缩/lint，以及浅深色、玻璃关闭、实际内容取样与屏幕边界检查通过；正式 APK 模拟器覆盖安装与启动通过。
- Web 81 项单元、13 项媒体回归、19 项动效/布局和材质专项通过，见[验收](../qa/chat-glass-depth-20261010.md)。不把模拟器结果等同于真机手感。
- 公网 APK 完整字节及哈希一致；20004/21500/22200/22300/22399 返回 22400，22400/22401 无重复更新或降级，原 24 个清单条目保留。没有另发账号通知。
- 两个 Web 域名的版本、JS/CSS、Worker 和两份 WASM 与本机产物一致，WASM MIME 正确。
- 服务 active/ready，贴纸 API 与商城首批 40 件读取正常；三常驻游戏节点在线，MEK 保持停服。临时只读验收会话已撤销，无生产聊天或游戏投递。

## 数据与回退

切换前后 **58 条账号记录、44 笔订单、5 个委托、3,085 件在售商品、12 个工单、2 条注销记录**及既有业务 ID 集合保留。Go 二进制、迁移、配置、官网/客服中心及游戏插件不变，只重启原 Go 加载更新清单。

备份 `/var/backups/deuterium/chat-glass-v224-22400-ab3893a935bd`；Web `/var/www/deuterium/releases/web-2.0.33-ab3893a935bd/dist`；Go 继续为 `/opt/deuterium/releases/chat-media-v222-60b0cb525842`。

回退可恢复 Web 2.0.32 链接及旧更新清单并重启现有后端，不恢复整库覆盖新增业务数据。

本机交付 `C:/DeuteriumAPP/delivery/Deuterium-2.0.24/`，维护工作区 `.worktrees/chat-media`。

[构建清单](artifacts/v224/build-manifest.json) · [切换记录](artifacts/v224/deployment.json) · [公网验证](artifacts/v224/verification.json)
