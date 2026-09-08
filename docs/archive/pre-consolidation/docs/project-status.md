# 项目状态与文档索引

## 2.0.2 当前发布（2026-09-08）

当前 App 为 **2.0.2 (20200)**，源码 4299ce37d0bf，已推送在线更新与账号提醒。已确认的付款请求在 1–3 秒模拟识别期间开始，删除原额外 470ms/100ms 等待；识别结束且后端成功才显示对勾。转账固定请求 ID 和收款人引用，保留防重复与未知结果恢复。此前“付款时序未改”属于 2.0.1 历史说明。

[2.0.2 部署记录](../../../deployment/deuterium-2.0.2-live.md)。Android 工作区 ui-motion-lab / codex/payment-overlap-v202；Go 服务仍为 1b5d9d1c63f8，缓存、联系人和 OSS 生命周期沿用 2.0.1，网页与游戏插件未更换。


## 2.0.1 当前发布（2026-09-08）

当前 App 为 **2.0.1 (20101)**，Android 源码 d23bd79；Go 图片生命周期服务源码 1b5d9d1c63f8 已部署。新增共享图片缓存、“我的 → 存储空间”、按私聊历史和新消息排列的联系人，特别关心置顶、搜索可查询全部玩家。图片没有当前业务用途后才进入 deuterium-test/gc/，雨云按 30 天生命周期清理，订单记录保留。当前 APK 与旧版同签名，在线更新及账号更新提醒已发布。

入口见 [2.0.1 部署记录](../../../deployment/deuterium-2.0.1-live.md)。下方 2.0.0、09 和此前“未实现”说明属于历史快照。Android 维护工作区仍为 ui-motion-lab，联系人分支 codex/recent-contacts-v201；后端为 test-release-v2，分支 codex/assets-lifecycle-2.0.1。网页和游戏插件未在本轮更换。支付时序仅定位到模拟识别后固定等待 470 ms 再提交请求，本轮未改资金提交流程。


核对日期：2026-09-08（Asia/Singapore）。本页属于 `codex/test-release-v2` 集成分支，当前依据为[全栈部署任务](codex://threads/01a07e94-3ac4-7ab2-9d25-2a8698f58cf1)的最新明确要求。下表区分已发布、已验证与仍待验收，不用历史文件名推断现状。

## 1. 当前 2.0.0 发布组合

网页/API 使用 [https://47.103.99.34](https://47.103.99.34)，域名仍在备案。详细下载、签名、部署目录与完整校验值见[实际部署页](../../../deployment/deuterium-2.0.0-live.md)。

| 对象 | 实际状态 |
| --- | --- |
| 产品 | 已确认 Deuterium IX；原生 App 商城/市场/信息/我的四入口，钱包在我的、AI 与委托在信息；本轮保持现有设计并修正启动圆徽标起始遮罩 |
| Android | **已发布 `2.0.0 (20004)`**，包名 `com.deuterium.app.uilab`、Android 8.0+；APK 22,684,005 bytes，SHA-256 `bb48ece21019d8ba25e554e2e6b13f94876545bf9019de5874afc127a2c5f7a3`，延用原测试签名 |
| 网页 | **已发布静态源码 `4de1ee7`**，实际入口脚本 `index-tMukH1-X.js`；归档 SHA-256 `c92bc90edfab96a76138bf416c1c82fc65c9c00111d97a9b8014a9b60dddbb76`，仅上传 dist，无 Node 常驻服务 |
| 当前业务后端 | **已部署 Go `8744f5b`**，包含社交、资产、目录、AI、013 交易/介入和 017 只读审计；不是旧 Ktor 运行包 |
| 实服已执行证据 | 一笔真实 1.00 CREDIT 转账及原键重放/改额拒绝；一条用户授权公共消息在 App/Web/常驻节点间互通；原交易和原消息已由最终客户端读取核对 |
| DIMA / DaoYu | 已初始化为 XConomy 官方收款/冻结系统专户，初始各 0.00、hidden=1；不创建 App 身份，不以补款冒充实际托管 |
| 内容与权限快照 | **13:47 验收时**经营目录、订单、委托和案件为空，xiwanzi 有平台权限但无店铺，`STORE_NOT_CONFIGURED` 与界面一致；**14:18 用户已确认新增记录为本人真实测试**，不再把空态写成当前事实，不回滚/删除后续业务 |
| 最终 MC 配套 | Core 1.0.0（SHA `6be8b022e6a360b11455100150d59f1032087edd34c6d75fa8e4b81adf9c5137`）+ Sync `0.7-deuterium.3` + 独立 Mail `0.6.1 / API 2` **14:17 已完成四服安装，随后运行能力核对通过**；仍不代表全部真实客户端业务交互已验收 |

最终运行核对中，Amiya/Odyssey 的 `playerDataReady=true`、`playerDataProvider=youermodsync-transaction-v1`，Mail API 2 的 `commerceReady`、`uncertainClaimProtection`、`missingDeliveryCancellation` 均为 true。三常驻节点启动后检查通过；MEK 保留开始前的停止状态。Amiya/Odyssey/MEK 使用 survival 库存域，MEK 继续禁领并保留未验证兼容档案；Login 不启用玩家同步集成和领取。详细节点、备份及后续业务核对由部署页更新。

## 2. 证据范围

| 证据 | 结果与不能外推的范围 |
| --- | --- |
| [实际部署与下载](../../../deployment/deuterium-2.0.0-live.md) | 公网完整下载 APK，核对大小、哈希、签名与更新版本；当前 20004 不再提示自身更新 |
| [实服资金](../../../qa/live-funds-2026-09-08.md) | 真实 1.00 转账、单个操作和两条账目、幂等与身份；该页是当时资金基线记录，不代表最终 MC 新包已安装 |
| [交易状态机](../../../qa/commerce-v2-2026-09-08.md) | Go 最终全量 race/integration 110 项通过，人工浏览器夹具因未显式启用跳过；随机回环 MariaDB、可控 Core 执行器、并发/回滚/UNKNOWN、9 类 HTTP 响应契约。不能代替实服非空商城/委托交易和游戏领取 |
| [目录与发布](../../../qa/catalog-v2-2026-09-08.md) | 发布快照、图片绑定、权限、购物袋与报价；整单数量/跨店/跨域/共同领取节点检查已补永久回归，见交易验收页 |
| Android 最终验收 | 91 项测试通过、Lint 0 errors；源码与原生只读验收由 App 源码清单（历史引用 `C:/DeuteriumAPP/delivery/Deuterium-2.0.0/source-handoff/Android-2.0.0-source-manifest.json`，未纳入当前仓库）及最终 QA 清单（历史引用 `C:/DeuteriumAPP/delivery/Deuterium-2.0.0/source-handoff/Android-2.0.0-final-readonly-qa-manifest.json`，未纳入当前仓库）定位；模拟器不替代真机手感 |
| 网页最终验收 | 54 项 Node 测试中含 23 项历史领域回归，其余为当前协议/恢复等；**13:47** 的 26 项公网只读检查直接加载已部署 `4de1ee7`，0 业务 HTTP 写入、0 JS 异常。该时刻的空态不覆盖后续新增业务。见[最终网页记录](../../../../web-app/docs/qa-v2-live-final.md) |
| [Core / Sync / Mail](../../../qa/core-sync-mail-2026-09-08.md) | Core 29 项、Sync 9 项真实 MariaDB 测试，隔离 Youer 的背包保存/恢复和 Mail 领取/未知保存证明恢复；不是完整玩家 MOD 客户端或代理切服验收 |
| [最终 TrChat 保护](../../../qa/core-trchat-2026-09-08.md) | Core `6be8b022…` 的真实最终事件类型、过滤/取消/私聊/暗禁言隔离验证；该摘要覆盖此前 `388534…` 候选，并用于本轮最终换包 |
| [最终 MC 预检](../../../qa/minecraft-final-precheck-2026-09-08.md) | 原 JAR、配置范围、三库/世界目录和客户端最小清单的只读事实；预检不等于停服备份或换包已经完成 |

图片直传已实际验证 Android/浏览器上传、READY 校验、读取和删除；AI 已验证真实 DeepSeek V4 Flash SSE、按需搜索及同请求恢复。免费 20 次/24 小时，平台管理员免次数，其他计划 `9999999.00` 且禁购。账号封禁、完整第三方 OIDC Provider、所有真实手机/游戏客户端交互不在本轮已通过声明中。

## 3. 工作区与源码交接

| 位置 | 当前用途 |
| --- | --- |
| `C:/DeuteriumAPP/.worktrees/test-release-v2` | `codex/test-release-v2`：Go、Core 接入、部署契约和验收的集成分支；运行中 Go 保持冻结 `00fbdac`，后续 Core/Sync 源码可继续交接，不能凭 HEAD 推断已部署二进制 |
| `C:/DeuteriumAPP/.worktrees/ui-motion-lab` | 原生 `android-app/ui-lab`；2.0.0 源码 `ff0dcc3`，最终只读 QA `215d9f2`，发布 APK 为 20003 |
| `C:/DeuteriumAPP/.worktrees/web-player-v1` | 网页源码 `4de1ee7`，公网 QA 文档 `2455abb`；文档提交不改变已部署静态包 |
| `C:/DeuteriumAPP/.worktrees/backend-rewrite` | Core、Sync 与受控游戏能力上游；按明确提交交接到集成分支，独立 Mail 只通过公开 API 对接 |
| `C:/DeuteriumAPP/delivery/Deuterium-2.0.0/source-handoff` | 精确源码清单与窄补丁，含 Android 和相对既有 0.6+JDBC 基线的 Mail .6.1 交接；不是生产配置或 QA 会话归档 |
| `C:/DeuteriumAPP` 及旧优化/硬化工作区 | 根入口和历史工作线；不得自动合并、清理未提交内容或把根残留源码当新版。根入口由主任务另行同步 |

完整交接关系见[部署页源码与证据](../../../deployment/deuterium-2.0.0-live.md#源码与证据)与 Mail 源码清单（历史引用 `C:/DeuteriumAPP/delivery/Deuterium-2.0.0/source-handoff/Mail-0.6.1-api-v2-source-manifest.json`，未纳入当前仓库）。本次没有自动推送 main；APK/ZIP/私有备份不因提交文档而混入源码。

## 4. 当前文档入口

| 文档 | 适用范围 |
| --- | --- |
| [AGENTS](../AGENTS.md) / [CONTEXT](../CONTEXT.md) / [产品基线](prd/deuterium-current.md) | 协作、领域与需求；当前发布状态以本页和部署页为准 |
| [ADR 0008](../../../adr/0008-current-product-and-visual-baseline.md) / [ADR 0009](../../../adr/0009-deuterium-backend-core.md) | 原生与视觉方向；Go/Linux、Java 21/Youer、多节点和独立 Mail 边界 |
| [完整 v2 契约](../../../contracts/README-api-v2.md) / [OpenAPI](../../../contracts/openapi-app-v2.yaml) | 跨端模型与接口；结合实际路由和本轮增量判断，不把早期 existing 标注当成所有新服务都已支持 |
| [交易 HTTP 对接](../../../contracts/commerce-http-seam-v2.md) / [严格邮箱取消](../../../contracts/mailbox-cancellation-core-v2.md) | 金额、版本、原操作恢复、平台证据、领取/撤回证明 |
| [AI 流协议](../../../contracts/ai-streaming-v2.md) / [图片协议](../../../contracts/assets-rainyun-s3-v2.md) | 真实 SSE、额度与来源；授权直传、完整校验和资源绑定 |
| [后端入口](../../../../backend-next/README.md) / [Core 入口](../../../../deuterium-core/README.md) | 上游模块的构建和职责；完整集成发布状态以部署页覆盖其早期工作线措辞 |
| [账号 v1](../../../prd/account.md) / [钱包 v1](../../../prd/wallet-transfer.md) / [聊天 v1](../../../prd/chat.md) | 旧协议与迁移参考，冲突处由当前 PRD、ADR 和本轮明确决定覆盖 |

## 5. 历史快照与下一步

07、09 为同日早期体验版：当时的示例余额、模拟身份、契约先行、网页/新业务服务尚未接入和生产更新地址为空均是**历史状态**。当前 20003/4de1ee7/00fbdac 已覆盖这些状态，仍适用的视觉及业务需求保留。01–03 旧底栏、动态取色、彩色主题和 Mastercard 动画不恢复；早期计划和 QA 在原工作区及 Git 历史中继续供追溯。旧 Ktor、Foundation 0.1 和其他硬化/优化分支的测试不重复计入本次结果。

最终 MC 已完成受控备份、安装和运行能力检查；后续补真实手机、实际游戏客户端领取/切服与非空经营流程验收。14:18 起观察到的真实业务由主任务只读核对，未经明确要求不恢复空库或撤销交易。阶段结果应先更新部署页，再同步本页；某时刻目录为空、功能未实际操作和操作失败应分别记录，不能靠示例数据或安装成功替代业务验收。
