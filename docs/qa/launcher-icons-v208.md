# 2.0.8 图标切换验收

2026-09-10。验证对象为 `feat/launcher-icons-v208` 的 App 20800、Web 2.0.8 与配套 Go。所有配置发布测试使用独立本机数据库；未向生产用户下发图标切换。

## 结果

- Go：`go test -race -tags=integration ./...` 全部通过。新测试覆盖默认值、持久化、权限、Cookie Origin / CSRF、非法 ID、幂等重放、同值不重复发布、并发保存、版本冲突，以及 WebSocket 新能力声明、即时通知和重连补同步。
- Android：129 项 JVM 单测通过，`assembleDebug`、`assembleDebugAndroidTest`、`lintDebug` 通过，Lint 0 errors。包含新协议标识 / 整数版本 / 兼容性验证，以及新实时事件只触发图标补同步、不混入聊天消息的测试。
- 原生 API 35：批量切换与旧版逐项切换路径往返通过；切换时 Activity / 进程未被结束；本机保存、重新创建控制器、离线、未知图标、过期响应和同版本矛盾配置均通过；活动入口能打开同一实际 Activity。
- 20700 → 20800：同签名覆盖安装通过，私有文件探针保留。旧桌面图标先通过拖放固定，再安装新包，位置保持 `[314,861][519,1134]`。默认图标切至活动图标后仍在相同位置，点击进入活动 alias。
- 网页：60 项单测和生产构建通过。真实 Go 接口完成切换；离线失败保留选择，恢复联网后复用相同 `clientRequestId` 成功。另一管理操作已发布同一目标时，冲突恢复显示无需重复发布。普通账号无菜单入口，直接管理 API 返回 403。
- 1440px 桌面与 390px 手机，浅 / 深色预览通过；无横向溢出；单选卡可用方向键切换，图片均成功加载。
- 两个真实 HTTP 响应通过新增 OpenAPI JSON Schema 验证。

## 证据

[汇总](artifacts/v208/summary.json)、[原生检查](artifacts/v208/launcher-check.txt)、[覆盖安装位置](artifacts/v208/launcher-upgrade.json)、[活动桌面](artifacts/v208/launcher-hot.json)、[契约响应](artifacts/v208/contracts.json)。

| 对象 | 预览 |
| --- | --- |
| 管理页 | [浅色](artifacts/v208/admin-final-light.png) / [深色](artifacts/v208/admin-final-dark.png) |
| 手机管理页 | [浅色](artifacts/v208/admin-final-mobile-light.png) / [深色](artifacts/v208/admin-final-mobile-dark.png) |
| 原生图标 | [默认](artifacts/v208/launcher-default-light.png) / [活动浅色](artifacts/v208/launcher-anniversary-light.png) / [活动深色](artifacts/v208/launcher-anniversary-dark.png) |
| 桌面覆盖安装与热切 | [20700 固定位置](artifacts/v208/launcher-207-pinned.png) / [20800 默认](artifacts/v208/launcher-208-pinned.png) / [活动图标](artifacts/v208/launcher-hot-anniversary.png) |

桌面截图的 Dock 可能同时出现系统“预测应用”，它不是第二个启用的启动组件；原生检查确认始终只保留一个实际 LAUNCHER 入口。

## 范围与限制

本轮实际设备为独立 API 35 Google APIs 模拟器。26–32 的逐项切换调用路径在该设备上验证，未声称跑过所有旧系统或所有厂商桌面。不同启动器的缓存 / 位置行为仍需真机确认；系统主题图标可能隐藏活动横幅并使用单色 D。

默认图标是原生矢量。活动前景从用户确认的 SVG 导出，中央区域保留原构图；透明 PNG 与白底 SVG 合成后存在极少量抗锯齿边缘舍入差异，平均通道差异约 `0.0016 / 255`，未修改图形或文字。

没有清理、初始化或覆盖生产账号、订单、委托、图片、经济数据。游戏插件没有改动。此次验收与候选构建不代表已经发布上线。
