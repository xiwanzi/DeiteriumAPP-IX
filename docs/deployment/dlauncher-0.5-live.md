# DLauncher 0.5.0 / Web 2.0.14

2026-09-12 后续维护：本记录的 App Web/Go 源码 `b5b06c8` 已随 [PR #26](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/26) 合入 main，配合[账号注销发布](deuterium-2.0.13-live.md)上线至 Web 2.0.15。启动器内容版本 5、配置和索引保留，Windows 启动器未重新发布。下方“尚未合并”是本轮首次交付时的状态。

2026-09-12，桌面启动器改为整包发布后仅通过 McPatch 更新。用户本轮要求只修改启动器与打包脚本，不生成完整游戏包。

## 实现

- 删除首次安装配方接口 `/api/v1/launcher/bootstrap`，现返回 404。APP 的首次安装页面改为“整包发布”，移除 Java/NeoForge 下载配方及在线安装开关。
- 保留数据库兼容键 bootstrap，仅保存实例、MC/NeoForge、整包资源版本和 McPatch 地址；写入时清除 java、installer、enabled 等旧字段。
- 内容 schemaVersion=2，游戏入口为 Deuterium IX，D9 蓝红版本标与 DSA 素材；明日方舟和泡姆泡姆通过 visible=false 隐藏。
- 基于 origin/main `4a9c249`，完整保留 `9051de0` 的同步超时独立上下文终态修复。本次该修复随新 Go 上线。
- McPatch 原生管理页和同步流程继续使用，没有改成另一套上传系统。

## 部署

- Go：`/opt/deuterium/releases/dlauncher-0.5-20260912`。
- Go SHA-256：`d1d2c106a5c341bc33f6825eb233c6d638a4cb0acd59bed8f150b538d2891465`。
- Web：`/var/www/deuterium/releases/web-2.0.14-dlauncher/dist`。
- 内容发布版本：5。
- 备份：`/var/backups/deuterium/launcher-20260912-v05`，含本轮数据库一致性备份及原配置。未新增迁移。
- 切换前后账号及业务 ID 摘要一致：71 个账号、27 笔订单、5 个委托。Android 发布清单和业务配置保留，未操作游戏服务。
- 回退可切回 Go `dlauncher-0.4-20260912-r3` 与 Web `web-2.0.13-dlauncher-final/dist`，并恢复本轮启动器内容备份；不要恢复整库覆盖新业务数据。

## 验证

Go 启动器集成/竞态测试、全量单元测试及 vet 通过；Web 构建和测试通过。实际浏览器确认单一 Deuterium IX 入口、DSA 素材与整包发布页面，旧安装开关消失。原同步失败/超时修复保持。

桌面端测试覆盖：没有完整包时不发起更新网络请求，出厂版本与移动实例，先恢复事务再检查游戏，打包排除个人数据、保留模组设置、禁止覆盖；Qt 真实点击验证悬停显示/移开隐藏、暂停/继续、无独立下载弹窗，支持 65% 小画布。原生增量修改、坏包拒绝和无网络本地恢复通过。中文路径独立程序在 100/125/150% 缩放验证通过。

整包打包脚本仅对已有完整测试实例执行检查计划，不生成游戏 ZIP；真实游戏进服未在本轮执行。维护分支为 `feat/dlauncher-bundled`，本轮变更只本地提交，尚未合并主分支。
