# App 2.0.12 部署记录

2026-09-11 **22:18:29（UTC+8）**，App **2.0.12（21200）** 已发布在线更新，配套 Go 于 **22:17:58** 上线。代码通过 [PR #23](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/23) 合入 main，合并源码 `a0ed4446d074294062b9f7b1a6482a36b1880859`。

[下载 App 2.0.12](https://47.103.99.34/downloads/Deuterium-2.0.12-21200-test.apk)。维护工作区 `.worktrees/auth-registration-glass`，发布记录分支 `xiwanzi/release-v212-live`。

## 发布内容

更新说明：

> 注册验证码无需先填写密码；修复忘记密码页面玻璃背景过于透明的问题，统一浮层显示效果。

验证码请求只依赖游戏 ID、QQ，密码在创建账号时校验。保留旧客户端附带密码的兼容、服务器身份校验、绑定占用检查、发送限次和一次消费；字段错误显示具体提示。登录页接入原有浮层背景采样，沿用全局玻璃参数和开关。

APK 继续使用同签名、非 debuggable 的 `performance` 构建，保留 R8、资源压缩和 Baseline Profile。Web 继续为 2.0.10，游戏插件未更换，数据库迁移及业务配置未改。本轮没有处理 `luoyinwuchen` 旧号注销。

## 产物

| 项目 | 值 |
| --- | --- |
| 包名 | `com.deuterium.app.uilab` |
| App 版本 | `2.0.12` / `21200` |
| APK 大小 | 8,875,538 字节 |
| APK SHA-256 | `7e8bf5ff4a9bc313cb70430545e832294443be5136565a8b79742ba6d03129a1` |
| 签名证书 SHA-256 | `04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`，与原版一致 |
| 构建源码 | `b804685109e52484074197ddcc0cfde8bc919171` |
| 合并源码 | `a0ed4446d074294062b9f7b1a6482a36b1880859` |
| Android 源码树 | `01b3391ee3305fcf61bb93a41e619cb90d98f840`，构建与合并提交一致 |
| Go 源码树 | `97309d5134a038b1cf6fce3f9739a8ce167d3e6c`，构建与合并提交一致 |
| Go 二进制 SHA-256 | `2078dcaf614e7b9a049915dd6af724544c45ff17d08d363abbd656c0f2aeef51` |

本机交付目录 `delivery/Deuterium-2.0.12/` 保存 APK、Go Linux amd64 程序、构建清单和 R8 mapping 压缩包。

## 验证与部署

- 版本递增后重新执行 `assemblePerformance`、149 项 JVM 测试和 `lintDebug`，全部通过；lint 0 errors、24 warnings。检查包名、版本、证书、非 debuggable 标志和包内 Baseline Profile。
- 注册/验证码的 MariaDB 集成测试本轮增加 `-race` 执行，通过无密码取码、旧密码字段兼容、发送限次、密码校验、一次消费及改密撤销会话。Go 全量单元测试及 vet 在同一功能源码的实现阶段已通过，未冒充本次重新执行的全量集成测试。
- 先前同一功能代码的模拟器验证覆盖取码请求仅含游戏 ID/QQ、注册密码提示，以及忘记密码浅色/深色/关闭玻璃/键盘/返回，见[实现验收](../qa/account-registration-glass.md)。本轮没有把先前 21100 Debug 的界面检查写为 21200 真机安装验收。
- 先在服务端备份配置、发布清单与数据库，上传新命名 APK 和 Go；服务端摘要与公网 APK 完整下载摘要一致，使用正常 HTTPS 证书校验。
- 确认没有进行中的商城资金操作后停止 Go，停稳后再次确认，原子切换 Go 路径并启动。健康检查 ready 后，无密码与旧版空密码请求均正确进入账号占用检查并返回 `ACCOUNT_ALREADY_EXISTS`；无效 QQ 返回 `QQ_INVALID`。验证使用已存在的账号触发拒绝，不向真实玩家发送验证码或创建账号。
- 然后原子替换发布清单，再次受控重启 Go 加载清单。保留原来 12 个发布条目；新增 21200 的适用范围为 1–21199。
- 公网检查 20004、20101、20200、20300、20401、20500、20600、20700、20800、20901、21000、21100、21199 都能获取 21200；21200、21201 不重复更新或降级。下载 URL、大小、摘要和更新说明一致。
- `xiwanzi` 账号提醒于 **22:18:32（UTC+8）** 创建，唯一事件键 `app.release:21200`，计数 1，标题“2.0.12 更新已就绪”，已通过账号通知 API 验证可见；没有公共聊天广播。
- 核对运行进程实际执行新 Go 二进制。原有账号 ID 和订单/委托 ID 全部保留，最终为 **71 个账号、27 笔订单、5 个委托**。旧号仍为 active；业务配置和全部迁移摘要一致。
- Login、Amiya、Odyssey、MEK **四节点当前均已连接**。本轮只操作 App 后端，没有启动、停止或重启游戏服；早期“MEK 离线”的部署快照不再作为本次状态。
- 临时验证会话已移除，并确认再次使用返回 401。

## 备份与回退

服务端备份：`/var/backups/deuterium/release-2.0.12-20260911-a0ed4446d074/`，目录权限 0700、配置和数据库备份权限 0600。数据库压缩备份 2,274,010 字节，SHA-256 `42a87664b2103225c14aa83234507e66335dfb42725e29794a22584e21ca8bf1`。

- 当前 Go：`/opt/deuterium/releases/release-2.0.12-a0ed4446d074`。
- 原 Go：`/opt/deuterium/releases/release-2.0.10-65c08a290896`，摘要 `1fa7b3dc691bc80cf7a579be832cec1465724225b8afbfb845b66b03d2666ee3`。
- Web 保持 `/var/www/deuterium/releases/web-2.0.10-65c08a290896/dist`。
- 原清单摘要 `9799085d0275a5040df14088767f4eb61a1e62541c9e009ae00da4975b0a9e8a`；新清单摘要 `4cdfe6b4ce55ff96944732d8908b36dc7a630118f2342d9e3060c0e71d8380af`。

若只需撤回 App 分发，恢复备份的发布清单并重启 Go，保留修正后的兼容后端。若必须回退 Go，应同时撤回 21200 分发，并处理已安装新版客户端无密码取码的兼容影响；旧后端仍要求取码密码。不要恢复数据库或删除新业务记录。已安装 21200 的设备通过更高版本修正，不自动降级。

[构建](artifacts/v212/build-manifest.json) · [发布前](artifacts/v212/preflight.json) · [备份及上传](artifacts/v212/stage.json) · [后端上线](artifacts/v212/activated.json) · [App 发布](artifacts/v212/published.json) · [公网验证](artifacts/v212/public-verification.json) · [提醒](artifacts/v212/update-notice.json) · [节点及会话](artifacts/v212/final-verification.json) · [最终不变量](artifacts/v212/final-state.json)
