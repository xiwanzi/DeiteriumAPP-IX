# App 2.0.11 部署记录

2026-09-11 **10:40:27（UTC+8）**，App **2.0.11（21100）** 已发布在线更新。代码通过 [PR #21](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/21) 合入远端 main，合并源码 `46d4c5e76eb57722a59bbe8a2e36c8633359d3ab`。

用户指定更新说明已原样写入更新清单和账号提醒：

> 大幅提升流畅度，提升系统稳定性。

[下载 App 2.0.11](https://47.103.99.34/downloads/Deuterium-2.0.11-21100-test.apk)。维护工作区 `C:/DeiteriumAPP-IX/.worktrees/performance-audit-v210`，发布记录分支 `xiwanzi/release-v211-live`。

## 发布范围

包含两轮 App 性能优化：启动徽标预热、列表与历史记录计算、动画状态读取范围、图片授权解析、AI 固定元数据、请求身份后台持久化与防重入，以及全局静态背景缓存。玻璃继续使用原始采样源和实时前景，保留原素材、渐变、动画参数及业务契约。

APK 使用已验证的非 debuggable `performance` 构建，保留 R8、资源压缩和 Baseline Profile。Web 与 Go 沿用 2.0.10 配套文件；游戏插件与数据库结构未在本轮更换。现有 Go 启动时加载发布清单，因此重启一次 `deuterium.service` 使新清单生效。

## 产物

| 项目 | 值 |
| --- | --- |
| 包名 | `com.deuterium.app.uilab` |
| 版本 | `2.0.11` / `21100` |
| APK 大小 | 8,875,538 字节 |
| APK SHA-256 | `4adec4a7e3e0e361c30fdcab93127a810d492262ed002f87c07ac0e545b7a40d` |
| 证书 SHA-256 | `04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`，与原版相同 |
| 构建源码 | `6a0a68767cd4da0795a3bf11b88d810e7b4efdf0` |
| 合并源码 | `46d4c5e76eb57722a59bbe8a2e36c8633359d3ab` |
| Android 源码树 | 两者均为 `1901a426b8c037f466050d185bd48aec15873247` |

本机交付位于 `delivery/Deuterium-2.0.11/`，含 APK、构建清单和 R8 mapping 压缩文件。公开清单继续沿用 `stable` 频道，保留原来的 11 个发布条目；新条目版本范围为 1–21099，21100 及更高版本不会重复安装或降级。

## 本次验证

- 版本递增后的 `assemblePerformance`、149 项 JVM 测试和 `lintDebug` 通过；0 测试失败/错误，lint 0 errors、24 warnings。APK 版本、包名、签名、非 debuggable 状态及包内 Profile 已检查。
- 构建源码与合并源码的 Android 树一致。此前对相同功能代码完成的 68 张固定画面、4 组真机背景/玻璃对照、96 次控件动画采样和离线业务检查，分别见[性能验收](../qa/app-performance-equivalence-v210.md)及[过度绘制验收](../qa/app-global-overdraw-v210.md)。版本递增后未重复宣称这些真机/像素检查为新执行结果。
- 先上传全新命名 APK，服务器与公网完整下载大小/SHA-256 一致，再原子替换发布清单。默认 TLS 验证通过，未绕过证书校验。
- 20004、20101、20200、20300、20401、20500、20600、20700、20800、20901、21000、21099 均返回 21100；21100、21101 不返回自身更新或降级。更新说明、下载 URL、大小、摘要与发布条目完全一致。
- 切换前无进行中的商城资金操作；重启后健康检查 ready，Go 二进制、网站目录、业务配置与环境摘要保持原值。检查保留 **27 笔订单、4 个委托、70 个账号**。
- Login、Amiya、Odyssey 已连接，MEK 仍离线；本轮没有重启游戏服。
- `xiwanzi` 账号更新提醒已于 **10:43:49（UTC+8）** 创建，唯一事件键 `app.release:21100`，计数为 1；标题“2.0.11 更新已就绪”，正文为用户指定原文，目标为 App 更新页。已通过账号通知 API 核对可见，没有公共聊天广播。
- 为读取节点状态和通知创建的临时验证会话已撤销，并验证撤销后返回 401。

本轮为 App 发布，未重跑 Go/Web/游戏插件业务测试。实际 2.0.11 安装由客户端更新流程执行；本记录不将此前 21000 功能候选的覆盖安装写成 21100 的新真机安装验收。

## 备份与回退

清单备份位于 `/var/backups/deuterium/app-2.0.11-20260911-21100/`，目录权限 0700、备份文件权限 0600，保存 `releases-before.json` 和发布前不变量摘要。

- 原清单 SHA-256：`7151ae6287e1822c64f91f6afa9f94b8c06f591e1b8b037a9ff0221452371847`。
- 新清单 SHA-256：`9799085d0275a5040df14088767f4eb61a1e62541c9e009ae00da4975b0a9e8a`。
- Go 仍指向 `/opt/deuterium/releases/release-2.0.10-65c08a290896`，二进制 SHA-256 `1fa7b3dc691bc80cf7a579be832cec1465724225b8afbfb845b66b03d2666ee3`。
- Web 仍为 `/var/www/deuterium/releases/web-2.0.10-65c08a290896/dist`。

如需撤回分发，仅恢复备份清单并重启现有 Go；保留旧 APK 和全部业务数据，不恢复数据库。已经安装 21100 的设备须发布更高版本修正，不自动降级。

[构建](artifacts/v211/build-manifest.json) · [发布前检查](artifacts/v211/preflight.json) · [上传与备份](artifacts/v211/stage.json) · [发布](artifacts/v211/published.json) · [公网检查](artifacts/v211/public-verification.json) · [账号提醒](artifacts/v211/update-notice.json) · [最终状态](artifacts/v211/final-verification.json)
