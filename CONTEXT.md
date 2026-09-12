# Deuterium IX 项目上下文

2026-09-12：**Web 2.0.13 与启动器配套 Go 已上线**，管理导航新增“启动器”，连接同机 McPatch 原生 Web 与 OSS 同步，初始基线已发布。Android 沿用 2.0.12；71 个账号、27 笔订单、5 个委托保留，游戏服务未重启。本轮维护 `.worktrees/dlauncher-admin` / `feat/dlauncher-admin`，代码仅本地提交。见[部署与验证](docs/deployment/dlauncher-0.4-live.md)。下方为较早记录。

2026-09-11：**App 2.0.12（21200）与配套 Go 已于 22:18 上线**，源码 `a0ed4446d074`（PR #23），xiwanzi 更新提醒已推送。注册验证码不再依赖密码，忘记密码浮层接入统一背景模糊；Web 沿用 2.0.10，游戏插件未更换。71 个账号、27 笔订单和 5 个委托全部保留，四个游戏节点当前在线，旧号未注销。维护工作区 `.worktrees/auth-registration-glass`，分支 `xiwanzi/release-v212-live`。见[部署记录](docs/deployment/deuterium-2.0.12-live.md)。下方为较早记录。

2026-09-11：**App 2.0.11（21100）已发布**，合并源码 `46d4c5e76eb5`，更新说明“大幅提升流畅度，提升系统稳定性。”；账号提醒已推送。Web/Go 沿用 2.0.10，游戏组件沿用既有部署。当前维护工作区 `.worktrees/performance-audit-v210`、分支 `xiwanzi/release-v211-live`，见[部署记录](docs/deployment/deuterium-2.0.11-live.md)。

2026-09-11：全局过度绘制专项已在连接真机上完成覆盖验收，维护分支 `xiwanzi/global-overdraw-v210`，工作区 `.worktrees/performance-audit-v210`。保留玻璃原采样，只缓存静态屏幕背景；App 版本仍为 2.0.10（21000），未线上发布。见[真机结果与边界](docs/qa/app-global-overdraw-v210.md)。

2026-09-11：本机新增 App 性能优化候选，工作区 `.worktrees/performance-audit-v210`、分支 `xiwanzi/performance-equivalence-v210`；保持原视觉和动画参数，版本仍为 2.0.10（21000），未发布。见[同效验收与限制](docs/qa/app-performance-equivalence-v210.md)。下方为线上版本记录。

2026-09-11：**App 2.0.10（21000）、Web 2.0.10 与配套 Go 已于 02:38 上线**，源码 `65c08a290896`（PR #18、#19），已向 xiwanzi 推送更新提醒。优惠券统一应用内通知、停用券/商品可删除、全额退款返券及启动绘制优化已生效；迁移释放 1 条历史退款券占用，26 笔订单与 4 个委托保留。三常驻服在线，MEK 继续停止；游戏插件未更换。维护工作区 `.worktrees/commerce-refunds-performance`，分支 `xiwanzi/release-v210-live`。见[部署记录](docs/deployment/deuterium-2.0.10-live.md)。下方为较早记录。

2026-09-11：优惠券生命周期、商品删除及启动绘制修正已通过 [PR #18](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/18) 合入 main，源码 `303db4ee57f9`。App/Web 2.0.10（App 21000）准备测试交付；线上部署状态见[2.0.10 交付记录](docs/deployment/deuterium-2.0.10-handoff.md)。下方“尚未推送、合并”是开发阶段记录。

2026-09-11：本机已完成优惠券应用内通知、停用/草稿券删除、商店商品删除、全额退款返券与启动/加载绘制优化。维护工作区 `.worktrees/commerce-refunds-performance`，分支 `xiwanzi/commerce-refunds-performance`；[验收记录](docs/qa/commerce-refunds-performance.md)。Go 全量竞态/集成、Android 134、Web 65 及原生通知检查通过；尚未推送、合并或部署，线上仍为下述 2.0.9。

2026-09-10：**App 2.0.9（20901）、Web 2.0.9、配套 Go/Core 1.0.2、Mail 0.6.2/API3、Bridge 0.6.1 与 XConomy .3 已于 16:36 上线**。主源码 `e4e4c4b7791c`（PR #16），Mail `67db93749bf9`（PR #2）。支持优惠券草稿与批量发放、同批提醒去重、折扣/周期限购及组合交付；22 笔订单、4 个委托和原配置保留，三常驻服恢复，MEK 继续停止禁领。维护工作区 `.worktrees/commerce-v209`，分支 `xiwanzi/release-v209-live`。见[部署记录](docs/deployment/deuterium-2.0.9-live.md)。下方为较早记录。

2026-09-10：**App 2.0.8（20800）、Web 2.0.8 与配套 Go 已于 11:13 上线**，源码 `6566bb617f8d`（PR #14）。默认弧线 D 与双子节图标已内置，管理员可远程切换；线上默认无横幅。22 笔订单、4 个委托及原配置保留。本轮未重跑测试。当前维护工作区 `.worktrees/launcher-icons-v208`，分支 `release/v208-live`。下方为较早记录。 见 [2.0.8 部署记录](docs/deployment/deuterium-2.0.8-live.md)。

2026-09-10：**App 2.0.7（20700）、Web 2.0.7 与配套 Go 已于 03:22 上线**，源码 `4262edd84c03`（PR #12）。Saki 支持逐日递减、最低 5 天差价的升级，严格保留原到期时间；购买确认展示升级实付与淡色删除线原价，不显示公式。空会话才显示输入框上方套餐入口。用户已开启的套餐配置及 11 笔订单、4 个委托保留，三常驻服在线，MEK 离线。当前维护工作区 `.worktrees/saki-upgrade-checkout`，分支 `release/v207-live`。下方为较早发布记录。 见 [2.0.7 部署记录](docs/deployment/deuterium-2.0.7-live.md)。

2026-09-09：**App 2.0.6（20600）、Web 2.0.6 与配套 Go 已上线**。App/Web 源码 `4d9a001ebd0e`（PR #9），Go 最终源码 `515c87fd4936`（PR #10）。账单、Saki/账号后台、头像裁切、购买订单与通知优化见 [2.0.6 部署记录](docs/deployment/deuterium-2.0.6-live.md)。Saki 售卖仍关闭；本轮工作区 `.worktrees/saki-admin-experience`，分支 `release/v206-live`。下方 2.0.5 及更早状态为历史。

2026-09-09，当前 App 2.0.5（20500），见 [2.0.5](docs/deployment/deuterium-2.0.5-live.md)；Web 2.0.5 与配套 Go（`0386f1bd5664`）已部署，见 [网页 2.0.5](docs/deployment/web-2.0.5-live.md)。Core 1.0.1 与 XConomy .2 沿用 [2.0.4](docs/deployment/deuterium-2.0.4-live.md) 配套版本。原生 Android、同源网站与统一后端连接真实账号、信用点、聊天、商城、市场及委托；游戏内收支与 App 交易使用同一份已提交账本。

## 架构

```text
Android (Kotlin / Compose)    Web (React / Vite 静态站)
             \                 /
              HTTPS / API v1、WebSocket、SSE
                        |
                  Go 后端 + MariaDB
                        |
                   Core 主动 WSS 连接
                        |
               Deuterium Core (Java 21 / Youer 1.21.1)
                 |          |          |
          XConomy资金API  独立Mail API3  Sync保存屏障
```

`backend-next` 的 Go 模块路径仍为旧仓库名，用于保持现有导入和构建兼容；Git 远程位置已经独立，不据此导入旧 Ktor 实现。网站生产只发布静态 `dist`，Node 网关不是生产前置依赖。

## 领域与边界

| 术语 | 含义 |
| --- | --- |
| Deuterium ID | 第一方账号体系；游戏服务解析的 UUID 是资产身份权威，QQ/游戏名是登录或展示标识 |
| 信用点 | XConomy 自有数据库和受控事务为权威；后端保存业务状态，不直接改游戏经济表 |
| DIMA / DaoYu | 官方收入 / 预付托管专户；不是可登录的普通玩家账号 |
| 物品版本 | Core 保存不可变 ItemStack 快照，商品与邮件锁定版本及摘要 |
| 邮箱 | 独立 Mail 是发放与持久权益权威；Bridge/UI/Core 通过公开 API 调用 |
| 保存屏障 | Sync 确认玩家完成加载、独占会话及库存事务提交，回执未知时隔离并只读恢复 |
| 市场 / 委托 | 由后端状态机处理担保、退款、验收、到期和平台介入，不由客户端自行结算 |
| 付款动画 | 1–3 秒模拟识别期间启动已确认请求；识别结束且后端成功才显示勾，不采集人脸 |
| 图片生命周期 | 活跃业务引用阻止回收；无当前用途后迁入专用 gc 前缀，30 天对象生命周期，订单记录保留 |

## 当前节点

Login 禁领且不参加库存同步；Amiya/Odyssey 使用 Sync 共享库存域 `survival`；MEK 同域，但保持停止及禁领，模组兼容未完成验收。四服安装同批 Core/Mail/Sync/XConomy，经济权威节点只能配置一个。

## 产品结构

四入口为商城、市场、信息、我的。钱包/账单/转账在我的；公共聊天、私聊、AI、公告和委托在信息。主题只有系统/浅色/深色；保留柔光玻璃、灵动视效和动态追光。完整已确认规则见 [产品基线](docs/prd/deuterium-current.md)。

旧工作区路径、旧 Ktor/Mohist 1.20.1 和 01–09 演示迭代保留为历史追溯资料，不是当前部署指令。整理前入口快照见 [历史入口](docs/archive/pre-consolidation/CONTEXT.md)。
