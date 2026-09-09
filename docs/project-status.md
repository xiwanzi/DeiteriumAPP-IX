# 当前项目状态

本机候选：2.0.4 管理/SMTP/统一流水和 App 体验修复已完成，见 [候选交付](deployment/deuterium-2.0.4-candidate.md)与[本次 QA](qa/admin-ledger-v204-2026-09-09.md)。尚未推送、合并或部署，下方线上版本仍为 2.0.3。

基准：2026-09-09。App 2.0.3 (20300) 与配套 Go 已发布，账号更新提醒已推送；本次新增显示偏好表，保留真实业务数据。

| 部分 | 当前状态 | 依据 |
| --- | --- | --- |
| App | 2.0.3 (20300)，修复加购/搜索，简洁付款确认、历史删除、未读红点、Markdown 和公共发送优化 | [2.0.3](deployment/deuterium-2.0.3-live.md) |
| 网站 | 2.0.0 静态 React/Vite，玩家、商家、平台管理及只读审计 | [网页部署](../web-app/docs/deployment-v2.md) |
| 后端 | Go `15bb45dc6625`，新增账号级隐藏与公共消息 HTTP 入口，既有完整业务保留 | [2.0.0](deployment/deuterium-2.0.0-live.md)、[2.0.1](deployment/deuterium-2.0.1-live.md) |
| Core / Mail / Sync / XConomy | 四服安装同批配套；三常驻服已启动验证，MEK 停止禁领 | [插件记录](deployment/deuterium-2.0.0-live.md) |
| 源码维护 | 新主仓库汇总 App/Web/Go/Core/XConomy；Mail、Sync 独立 PR | [归档交接](repository-handoff.md) |

## 本次验证与历史证据

2.0.3：Android 119 项单测、Lint 0 错误，模拟器交互/Markdown 浅深色/付款时序通过；Go 122 项 race/integration 通过（既有人工夹具跳过），两个实际响应通过契约校验。公网 APK 摘要、更新检查、账号提醒、生产只读接口和三常驻节点在线均已核对。见 [本次 QA](qa/app-v203-2026-09-09.md)。

## 已有验证和仍需验收

历史发布证据包括：App 113 项单元测试及付款四时序模拟器检查；Go 119 项 race/integration 通过（既有人工浏览器夹具跳过）；网站 54 项 Node 测试；Core/Sync/Mail 真实隔离 Youer、保存证明恢复与三常驻服启动验证。这些数目不是本次新跑的测试，整理验证单列于 [交接记录](repository-handoff.md)。

真实手机手感、完整游戏客户端领取/跨服流程以及 MEK 模组兼容仍不能用自动化测试代替。完整第三方 OIDC Provider、账号封禁管理没有被本轮发布证据确认为完成。AI 当前免费 20 次/24 小时、管理员豁免，其余套餐禁购。

## 数据和版本规则

已存在的用户订单、委托、转账与图片是真实业务，不清空恢复演示。新增迁移必须保留既有 SQL 摘要；版本号、Git 提交、APK/JAR 摘要和线上状态分别记录。外部组件以 [固定版本清单](../components.lock.json) 为准，不跟随默认分支自动升级。

旧入口中的 VIII、Ktor、未实现网站/后端和“付款时序未改”等说明已归入历史语境；不再作为当前结论。
