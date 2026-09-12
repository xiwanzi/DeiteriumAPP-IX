# App 2.0.13 / Web 2.0.15 部署记录

2026-09-12 **19:15:57（UTC+8）**，配套 Go 与 Web **2.0.15** 上线；**19:15:58** 开放 App **2.0.13（21300）** 更新，**19:16:02** 向 `xiwanzi` 推送唯一账号更新提醒。

代码经 [PR #26](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/26) 合入 main，合并源码 `15690c138288746188ea242379695be1f3e6622a`。维护工作区 `.worktrees/account-erasure`，记录分支 `xiwanzi/release-v213-live`。

[下载 App 2.0.13](https://47.103.99.34/downloads/Deuterium-2.0.13-21300-test.apk) · [账号与权限](https://47.103.99.34/admin?section=accounts)

## 发布范围

管理台增加当前管理员密码确认的“永久注销”；App/Web 清理注销账号的联系人、主页、消息、商品和关联缓存，历史交易匿名保留。未完成交易、未知转账、本人及最后有效管理员保护生效。具体行为见[契约](../contracts/account-permanent-deletion.md)与[实现验收](../qa/account-permanent-deletion.md)。

本次部署只发布功能，**没有执行真实账号注销**。19:18 验收时注销记录为 0，`luoyinwuchen` 仍为 active。实际注销由管理员在工作台输入自己的当前密码后操作；离线或旧版客户端需要更新并联网同步后清理缓存。

发布前发现启动器已单独升级至 0.5：Go 与 Web 2.0.14 的已部署源码 `b5b06c80d62cef65efbc219edf296d5c1827a648` 尚未合入主线。本次在独立发布工作区整合该提交和最新 main，保留整包发布、D9/DSA 品牌及同步超时终态修复；没有修改原启动器工作区或重新发布 Windows 启动器。

保留已执行的 `027_desktop_launcher.sql` 原文/摘要，注销迁移使用 **`028_account_deletion.sql`**。迁移只建立门控、注销记录表和状态索引，不删除用户。发布前 Web 的实际文件属于 2.0.14，但 `web-config.json` 留着 2.0.10；本次统一修正为 2.0.15。

## 构建与产物

| 产物 | 大小 | SHA-256 |
| --- | --- | --- |
| App 2.0.13（21300） | 8,875,538 字节 | `2d063c93d865c374be426e9aa263658869c96a6aeb8f4c4de029f990f5fd398b` |
| Go Linux amd64 | 13,156,512 字节 | `3ffce2b5552de15005064310b5318bccf9812e56db413b928c64effd6558ebcf` |
| Web 2.0.15 ZIP | 33,703,632 字节 | `c1b370b4db8d4f62cd8e8ad7ade87d19211cf4e938850119464b573efb3964ca` |

App 包名 `com.deuterium.app.uilab`，采用非 debuggable 的 `performance` 构建，保留 R8、资源压缩及 Baseline Profile。签名证书 SHA-256 为 `04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`，与线上 App 一致。

构建源码 `6231ac2c833656ec259747dbeaa1c484a4a2904a` 与合并源码的 Android、Go、Web 三个源码树分别一致：

- Android：`4fbc9624866b84d89bcce9d7bfd5be2e46d455fc`。
- Go：`750ff7d8f9cf316147626dedf5560515c26e5885`。
- Web：`9a922f663cfb2ea3f7617328b1204ae76f9a4353`。

本机产物位于 `delivery/Deuterium-2.0.13/`，含三端发布文件、R8 mapping 和构建清单。

## 本次验证

- 整合启动器已部署源码、调整迁移编号后，重新运行 `go test -race -tags integration ./... -timeout 9m` 和 `go vet ./...`，全部通过。
- `assemblePerformance`、152 项 Android JVM 测试、`lintDebug` 通过；0 测试失败/错误，lint 0 errors、24 warnings。检查 APK 包名、版本、签名、非 debuggable 状态、ZIP 完整性与 Profile。
- Web 68 项测试与生产构建通过；约 516KB JS 主包产生大小提示，没有编译错误。未进行无关拆包重构。
- 上一实现阶段的浏览器和模拟器功能检查仍见原验收；本次不将其写为 21300 的新真机安装验收。
- 服务器先备份数据库、配置、systemd/启动器环境文件，再上传全新命名产物。服务端与公网 APK 的大小/摘要一致，使用正常 HTTPS 证书验证。
- 切换前及停止 Go 后均确认无进行中的商城资金操作和启动器同步任务。执行 028 迁移后切换 Go/Web，健康检查 ready，再更新 App 清单并受控重启 Go。
- 公网版本矩阵：20004、20101、20200、20300、20401、20500、20600、20700、20800、20901、21000、21100、21200、21299 都可获得 21300；21300、21301 不重复更新或降级。原 13 个发布条目保留。
- 真实账号管理列表和注销增量接口可读取。使用“管理员本人目标 + 空密码”的非破坏探针，确认返回 400 `ADMIN_PASSWORD_INVALID`，没有执行注销。
- 已发布网页包含“永久注销”“整包发布”和 2.0.15；启动器内容接口正常，已删除的 `/launcher/bootstrap` 仍返回 404。
- 更新提醒事件键 `app.release:21300`，收件人为 `xiwanzi`，计数 1，已通过账号通知 API 核对可见；无公共聊天广播。
- 核对进程实际运行的新 Go 文件。停稳前的全部账号 ID 和订单/委托 ID 保留：**71 个账号、27 笔订单、5 个委托**。注销记录为 0，旧号仍 active。
- 启动器草稿、发布内容及索引摘要完全一致，发布版本仍为 5。业务配置、systemd 主文件、启动器 drop-in 和环境文件摘要不变。
- Login、Amiya、Odyssey、MEK 四个节点重新连接，本轮没有重启游戏服务或 McPatch 原生服务。
- 两次临时验收会话均已撤销，并验证返回 401。

## 备份与回退

备份位于 `/var/backups/deuterium/release-2.0.13-20260912-15690c138288/`，目录 0700，敏感备份文件 0600。数据库一致性压缩备份 2,983,640 字节，SHA-256 `32b5bef3d940a2935fd4d6bd0db142eaa1a96e6bba415eff7aa096932c9d56cb`；保存发布前及停稳后的业务 ID 集合。

| 项目 | 原值 | 当前值 |
| --- | --- | --- |
| Go | `/opt/deuterium/releases/dlauncher-0.5-20260912` | `/opt/deuterium/releases/release-2.0.13-15690c138288` |
| Web | `/var/www/deuterium/releases/web-2.0.14-dlauncher/dist` | `/var/www/deuterium/releases/web-2.0.15-15690c138288/dist` |
| 清单摘要 | `4cdfe6b4ce55ff96944732d8908b36dc7a630118f2342d9e3060c0e71d8380af` | `725d226faf0b67d38ab97d35cc066a7f2c05c6580ed8afef30e1ea7b4f0904c3` |

需要撤回 App 分发时，仅恢复清单并重启当前 Go。若启动失败且还未执行真实注销，可受控切回上述旧 Go/Web，保留新增表及全部数据；**不得恢复整库来覆盖新业务或复活已注销账号**。一旦开始使用永久注销，应优先前向修复，不能退回缺少注销处理的代码继续管理这些记录。已安装 21300 的客户端需更高版本修正，不自动降级。

[构建](artifacts/v213/manifest.json) · [发布前](artifacts/v213/preflight.json) · [备份上传](artifacts/v213/stage.json) · [迁移与切换](artifacts/v213/activated.json) · [App 发布](artifacts/v213/published.json) · [公网验证](artifacts/v213/public-verification.json) · [账号验证](artifacts/v213/account-verification.json) · [更新提醒](artifacts/v213/update-notice.json) · [最终数据核对](artifacts/v213/final-state.json)
