# 聊天输入区动效 App 2.0.23 / Web 2.0.32 上线

2026-10-09 **23:40（UTC+8）**，App **2.0.23（22300）**、Web **2.0.32** 已上线。[PR #87](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/87)，源码 `e21b38b45e8ad81662ebb3cce860fa51b84ddfd7`，合并 `34edf7733680dca7cc13c6b48ad806e977f3ab66`。Go 二进制、数据库结构、游戏插件、官网与客服中心沿用原版。

## 用户可见行为

- 按参考录屏调整胶囊输入栏、左侧表情入口、附件与圆形发送按钮。
- 键盘与表情面板共用底部空间，连续展开/收起，最新消息随输入栏移动；分类、最近使用、完整 Emoji 删除与底部表情/贴纸切换。
- 紧凑贴纸网格及可拖拽图片确认抽屉；长截图可按宽度滚动查看。Android 相册选择继续使用系统单图 Photo Picker。
- 静态/GIF 贴纸在消息、面板、导入预览及查看器中自动圆角，透明背景与动图时序保留。按实际图像边缘裁切，已有贴纸无需重新上传，不改变源字节或去重指纹。

## 下载与验证

[下载 2.0.23](https://47.103.99.34/downloads/Deuterium-2.0.23-22300-test.apk)，包名 `com.deuterium.app.uilab`，9,055,762 字节，SHA-256 `d38be0d03eee65ce5c73818c1db32d56fdef24b1e1f9d2600beb9627db8986f4`。证书 SHA-256 `04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`，与原版一致。

- 发布版重新完成 performance 构建、R8/资源压缩、lint、版本/签名及凭据扫描；正式 APK 模拟器覆盖安装及桌面别名启动成功。
- 实现阶段 Android 172 项 JVM、Web 81 项单元、47 项浏览器检查及原生浅深色/减少动效/手势/圆角像素检查通过，见[验收记录](../qa/chat-composer-motion-20261009.md)。真机键盘差异及触控手感仍以用户实测为准。
- 公网 APK 完整大小/哈希一致。20004、21500、22100、22200、22299 返回 22300；22300/22301 不重复更新或降级。原 23 个发布条目保留，没有另发账号消息。
- 两个 Web 域名版本正确，主 JS/CSS、媒体 Worker 和两份 WASM 与本机构建字节一致，WASM MIME 正确。
- 服务 active / ready，贴纸 API 与商城首批 40 件读取正常；Login、Amiya、Odyssey 在线，MEK 保持停服。临时只读验收会话已撤销，没有创建线上聊天消息或游戏投递。

## 数据与回退

切换前后均为 **58 条账号记录、44 笔订单、5 个委托、3,085 件在售商品、12 个工单、2 条注销记录**；既有业务 ID 集合保留。Go 的 SHA-256 仍为 `2a2168aac81e565b5f3a9790ae12a501585e26f6af781373e740970b43bdc3b4`，全部迁移及配置校验值保持。

备份 `/var/backups/deuterium/chat-motion-v223-22300-e21b38b45e8a`，包含一致性数据库、旧程序、配置和更新清单。只切换 Web 与 APK 清单，重启现有 Go 加载清单；没有迁移或数据清理。

Web `/var/www/deuterium/releases/web-2.0.32-e21b38b45e8a/dist`；Go 继续为 `/opt/deuterium/releases/chat-media-v222-60b0cb525842`。

如需回退，可恢复 Web 2.0.31 链接与旧更新清单并重启现有后端。不得用整库备份覆盖新业务记录；已安装的新 App 可继续使用原有兼容 API。

维护 `.worktrees/chat-media` / `xiwanzi/release-v223-live`。本机交付 `C:/DeuteriumAPP/delivery/Deuterium-2.0.23/`，包含 APK、mapping、Web 产物和证据。

[构建清单](artifacts/v223/build-manifest.json) · [切换记录](artifacts/v223/deployment.json) · [公网验证](artifacts/v223/verification.json)
