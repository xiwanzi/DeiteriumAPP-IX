# App 2.0.18 输入框可读性更新上线

2026-10-05 **23:53（UTC+8）**，App **2.0.18（21800）** 已发布在线更新。[PR #76](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/76)，源码 `2e671cc98d3272d953650189edce5a246b5d9bbc`，合并 `1198bbdb7cd513a2bf5dd1547863e0e2a11ba569`。两者 Android 源码树均为 `ea95f7704254d49d3c2ae6468b682fa990ffe04d`。

## 已生效

找回密码等共用表单输入框使用不透明主题底色、常驻细边框及清晰的聚焦边框；标签加深并调整为 14sp 中等字重。用户确认的浮层玻璃保持，业务流程不变。更新说明：

> 提升找回密码等表单输入框的可读性，优化浅深色背景、边框和字段标签。

[下载 App 2.0.18](https://47.103.99.34/downloads/Deuterium-2.0.18-21800-test.apk)。包名 `com.deuterium.app.uilab`，APK 8,908,306 字节，SHA-256 `ca6d55f43040110c935bb53806ce5813fbd3c28837d8cecbe41c25cc41ccda8b`；证书 SHA-256 `04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`，与原版相同。

## 验证和范围

- 非 debuggable `performance` 构建成功，R8、资源压缩和 Baseline Profile 保留；152 项 JVM 测试通过，lint 0 errors、23 warnings。APK 已扫描已知部署敏感值，未发现匹配。
- 本轮样式候选已完成 8 组原生账号回归和浅深色、玻璃关闭、键盘截图，见[验收记录](../qa/input-readability-20261005.md)。正式 21800 APK 在 Android 35 模拟器覆盖安装成功，经已导出的 MainActivity 桌面别名启动正常。没有将候选测试描述成发布包重新执行的原生全量回归；未冒充真机验收。
- 公网完整 APK 大小/SHA-256 一致。20004、21300、21500、21600、21700、21799 返回 21800；21800、21801 无重复更新或降级。最低适用版本为 1，原 18 个发布条目完整保留。
- 仅更新 APK 和更新清单。Go 在启动时读取清单，因此短暂重启原 `deuterium.service`；程序、业务配置、环境文件、Web/官网/客服中心目录均保持原值，Web 仍为 2.0.27，游戏插件未更换。
- 切换前无进行中的商城资金操作，切换后服务 active / ready；Login、Amiya、Odyssey、MEK 四节点在线。读取节点状态的短期管理员会话已撤销。
- 发布前后均为 **55 个账号、27 笔订单、5 个委托**。未迁移或清理业务数据；本轮未逐表统计工单和注销记录。未另发账号通知，用户在 App 内检查更新即可获取。

## 备份与交付

清单备份 `/var/backups/deuterium/input-readability-21800-2e671cc98d32/releases.json`。若需撤回，可恢复旧清单并重启原服务；保留 APK 和用户新数据，无需还原数据库。

本机交付 `C:/DeuteriumAPP/delivery/Deuterium-2.0.18/`，含 APK、构建清单、R8 mapping 和发布验证记录。

[构建清单](artifacts/v218/build-manifest.json) · [发布记录](artifacts/v218/published.json) · [公网检查](artifacts/v218/verification.json) · [节点与服务](artifacts/v218/final-verification.json)
