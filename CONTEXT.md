# Deuterium IX 项目上下文

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
          XConomy资金API  独立Mail API2  Sync保存屏障
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
