# Deuterium 2.0.0 测试部署计划与核对记录

创建：2026-09-08。本轮测试部署已完成，逐项实测范围以最终记录为准。

## 2.0.0 测试部署完成（2026-09-08）

最新依据为用户在[全栈部署任务](codex://threads/01a07e94-3ac4-7ab2-9d25-2a8698f58cf1)中的要求。服务对象已确认 **Deuterium IX**；App 保持现有设计，修正启动圆徽标、客服头像菜单及分类取消重叠，注册/关于统一版本标，清除演示数据并接真实服务。

当前发布：**App 2.0.0 (20004)**、网页 `4de1ee7`、Go `8744f5b`。入口 [https://47.103.99.34](https://47.103.99.34)，[下载 APK](https://47.103.99.34/downloads/Deuterium-2.0.0-20004-test.apk)。Core 1.0.0、Sync 0.7-deuterium.3、独立 Mail 0.6.1 已安装四服；Login/Amiya/Odyssey 运行验收通过，MEK 保持停止及禁领。DIMA/DaoYu 已初始化；真实转账、公共消息已核对。用户新增的订单/委托/图片是真实测试业务，不删除为演示数据。

14:25 已修复转账回执字段不兼容并只读恢复原交易；20004 修复授权图片显示及待确认交易自动查询。AI 为 DeepSeek V4 Flash、按需联网搜索、免费20次/24小时、管理员豁免，其他套餐9999999且禁购。部署、源码、备份及验证限制见[最终部署说明](deuterium-2.0.0-live.md)。真实手机手感与完整游戏客户端领取/切服不冒充已完成验证。

维护入口：Android `.worktrees/ui-motion-lab`，网页 `.worktrees/web-player-v1`，Go/部署 `.worktrees/test-release-v2`；Core 上游 `.worktrees/backend-rewrite`。下方旧07/09、Ktor、VIII及“新后端/网页未实现”仅为早期快照，与本节冲突时以本节为准。

详细执行与校验见[最终部署记录](deuterium-2.0.0-live.md)。以下保留早期计划和过程证据。

## 本轮授权与范围

依据用户当前任务「修复启动动画并完成全栈部署」：修复 APP 启动圆形版本徽标最初出现方形遮罩；其他 APP 视觉不改。清除玩家委托、公告、商品、聊天、余额等示例内容，移除模拟登录与体验版专属文案，保留正式功能入口；APP 更新为 2.0.0。随后核对并补齐 APP、网页、新后端的同步与部署缺口，以 APP 优先。最终部署后端与网页，并将验证后的 Core 和独立邮箱部署到实际 Minecraft 服务器。

`47.103.99.34` 是本轮提供的独立 Linux 部署机，**不是 Minecraft 主机**。`lnyozu.cn` 尚在备案，本轮不使用该域名作为访问入口。服务器密码、面板密码、数据库密码、节点密钥不写入本文或交付包。

## 源码与编辑所有权

| 对象 | 本机工作区 | 本轮核对时状态 |
| --- | --- | --- |
| Android | `.worktrees/ui-motion-lab/android-app/ui-lab` | UI 已多次验收；主要业务仍为本机模型，更新已有网络能力 |
| 网页 | `.worktrees/web-player-v1/web-app` | 演示与真实模式并存；真实登录、公共聊天和 Core 只读能力已接 |
| 新 Go 后端 / Core | `.worktrees/backend-rewrite` | 独立任务正在开发；当前源码比根索引更新，不能按旧 Ktor 状态推定能力 |
| 独立邮箱 | `C:/Mod/DeuteriumIX/work/mail/Deuterium-Mail` | 最新 0.6.0 本机产物待实服核查 |
| 旧后端/客户端工作线 | 其他 worktree | 保留原状，不自动合并或清理 |

后端任务继续拥有 backend-rewrite 的编辑权；APP/Web 各自在自己的工作区修改。集成发现须通过具体接口和测试证据交接。

## 已确认部署环境

2026-09-08 已以 SSH 只读核验：Alibaba Cloud Linux 3.2104 U12、x86_64，约 1.8 GiB RAM + 1 GiB swap，根盘余量约 24 GiB。已有宝塔及 Nginx，80/443/888/8888 和 SSH 22 在监听；没有运行中的数据库服务或 Docker 容器。

Minecraft 实际环境由既有 SSH 配置访问，Windows 路径 `E:/Deuterium_IX/servers`，四服为 Login、Amiya、Odyssey、MEK，Youer 1.21.1 / Java 21。主机、节点和现装版本将在独立 [Minecraft 准备检查](minecraft-readiness-2026-09-08.md) 中记录；根目录旧 Mohist 1.20.1 描述不能用于本次部署。

## 数据流和入口

已锁定访问入口为 `https://47.103.99.34`（标准 443，也可写 `https://47.103.99.34:443`）。8443 本机服务正常但公网测试被重置，因此不作为客户端默认地址。使用 IP 证书保留浏览器安全 Cookie、App 系统 TLS 验证和 Core WSS；现有宝塔域名站点继续按其 SNI 配置服务。

APP / 浏览器 → Nginx TLS → 网页网关 / 后端回环端口 → 后端专用数据库；Minecraft Core 从 Windows 主动连接后端桥。App/Web 不直连数据库，不持有游戏服务器凭据，不本地决定余额、发奖或资金结果。邮箱保持独立，Core 调用其公开接口。

用户本轮追加确认 XConomy 两个系统经济账户：`DIMA` 用于官方商城收款，`DaoYu` 用于冻结交易款。预付先转入 DaoYu，市场/委托结算到交易玩家；官方商城领取邮件后结算到 DIMA，退款从 DaoYu 原路返回。由 Core 初始化与管理身份/操作日志，先核对不存在不应占用的现有同名账户；实际创建及 UUID 以部署证据为准，不能在客户端虚构余额或创建普通登录账号冒充这两个专户。

## 实际能力核对基线

| 能力 | 初检结论 | 本轮验收要求 |
| --- | --- | --- |
| 登录/账号 | 新后端已支持旧账密导入、App Bearer、Web Cookie/CSRF；注册找回尚缺真实游戏验证 | 两端使用同一身份；失败明确；无模拟入口 |
| 钱包转账 | 客户端本机示例；新 Go 尚无完整真实接口 | XConomy 权威余额、幂等、结果未知恢复、收款隔离、真实受控操作 |
| 公共聊天转发 | 后端桥有持久投递和 ACK；Core 实服链路待验 | 四服来源、App/Web互通、去重重连、拒绝重复转发 |
| 私聊/引用/消息转发 | 多数未接后端 | 对齐真实支持范围；不存在本地假发送成功 |
| 商城/市场/委托/公告 | 客户端种子数据多，新后端并未全量实现 | 清空示例、真实数据同源；缺失接口明确列出并补齐 |
| 物品库/邮箱 | Core 与独立 Mail 正在对接 | 不可变物品版本、准确数量/限制服、领取撤回互斥、同步屏障 |
| 更新 | App 有真实下载能力；生产发布源未配置 | 2.0.0 元数据、签名/完整性、实际可下载 APK |

## 部署、失败和回滚顺序

1. 修复客户端并清除演示数据；旧体验缓存不导入真实账号资产。记录包名、版本号、签名、覆盖安装行为。
2. 准备独立数据库、运行用户、配置和 Nginx 站点，所有敏感文件仅存在受限运行目录。先做配置检查，再启动。
3. 按后端交付的路由核对客户端请求/响应；测未登录、错误响应、断线、重复提交与重启恢复。
4. 备份旧账号来源后只读导出；按后端工具预检冲突，确认导入数量与稳定 ID，不导入旧权限或会话。
5. 新版本 Core/Mail 必须先核对四服实际依赖与备份；以有序停服/替换/启动部署，保留原 JAR 和配置。无真实同步屏障时不能声称付费物品安全领取已完成。
6. 用明确测试身份完成 App/Web/游戏双向聊天、真实转账及邮箱交付验收；真实资金结果未知时查原 operation，不换 ID 重试。
7. 每次发布使用独立版本目录和校验清单；数据库迁移前备份，不通过回退二进制冒充撤销数据迁移。失败恢复上一版文件和配置，并核对持久操作状态。

## 完成证据与待确认

2026-09-08 09:24（Asia/Singapore）基础环境进度：

- 已安装专用 MariaDB 10.5.29，仅监听 127.0.0.1:3306；后端运行账号只有本库 SELECT/INSERT/UPDATE/DELETE 权限，迁移通过本机管理账号执行。
- 已部署此前交付的 Foundation 0.1.0 二进制作基础环境验收，SHA-256 `b58c2b8a32b0bfb2a8c77098f793a0ad314be239875824818a4e90a82f5258b5`；当前目录 `/opt/deuterium/releases/foundation-bootstrap-0.1.0`。它不代表本轮完整功能交付，后续按新产物替换。
- systemd `deuterium.service` 已启动；回环 `/health/ready` 返回 ready，公网 `/api/v1/account/me` 正确返回 401；当前尚未导入账号、注册游戏节点或执行资金操作。
- Let's Encrypt IP 证书首次有效期至 2026-09-14，系统 CA 验证通过；`deuterium-certbot-renew.timer` 每 6 小时检查，带 Nginx 重载 hook 的续期 dry-run 成功。
- 用户已明确授权：转出 `xiwanzi`、收款 `luoyinwuchen1`、网页管理员 `xiwanzi`；若 xiwanzi 余额不足允许游戏控制台补足。实际测试尚未开始。

2026-09-08 09:32 追加：网页 2.0.0 静态产物已发布至 `/var/www/deuterium/releases/web-2.0.0-20260908-01`，由 `/var/www/deuterium/current` 指向；压缩包 SHA-256 `0866871693ebb4b6f3917ca21f4391bb82b27796d542431dd98fa90ef94898a5`。Playwright 实际公网浏览器打开成功，未登录恢复请求返回 401，输入专门构造的不存在账号后真实显示“账号或密码错误”，未落入演示账号。截图为 `C:/DeuteriumAPP/output/playwright/deployed-login-v2.png`。

新建 `.worktrees/test-release-v2` / `codex/test-release-v2`（base `8db1109`）补社交/内容、更新等服务；另一后端任务继续持有 Core/Go RPC/注册/钱包编辑权。最终以合并产物替换当前 bootstrap，不能将当前只读/登录网页写成交易全部可用。

旧库导出发现一条 QQ 格式与新迁移校验不兼容的历史记录；当前官方工具安全拒绝整批，没有修改旧账号或密码，正修复兼容迁移策略。此时尚未替换 Minecraft 插件，业务验收与未完成能力继续记录。

2026-09-08 对象存储追加：用户指定雨云 S3 bucket `xiwanzi`、endpoint `https://cn-nb2.rains3.com` 供测试，使用 `deuterium-test/` 前缀。授权 PUT 首次 200、同 URL 再写 412；HEAD 长度与 MD5 一致、匿名 GET 403；探测对象已删除。原无 CORS 规则，现添加仅允许 `https://47.103.99.34` 的 GET/HEAD/PUT 规则，备份原配置且不改匿名访问策略。客户端直传协议补充位于 `.worktrees/test-release-v2/docs/contracts/assets-rainyun-s3-v2.md`，访问密钥未写本文。

2026-09-08 09:54 账号迁移完成：采用修复提交 `ff85b25` 的独立官方工具（Windows SHA-256 `a86f3a1ae3a04cd772b90097b3db4a31a46cb51fac923f120e00a29a55d7e471`，Linux `50efba8c9f5afa7f8e65aaf078e73e5ed9ae12614081e3e2d857ac14f97048bf`）。只读导出 70 行，Linux 预检 70 create，正式导入 70，重复预检 70 unchanged；70 个唯一 UUID 和 active 状态一致，建立 139 个合法登录别名。异常 QQ 保留显示值但不参与登录。源数据库、原密码与旧会话未修改，原库旧账单/聊天记录未迁移，不称为全量业务迁移。

迁移前新库备份在 `/var/backups/deuterium/migration-20260908/backend-before-import.sql.gz`；敏感 JSONL 只存受限备份目录。为用户明确授权的 xiwanzi 创建可撤销的 45 分钟 QA App/Web 会话并记服务端审计，未重置密码。Android 仪器已实测成功身份恢复、公共历史与认证 WebSocket 握手；游戏桥未安装，所以不能以握手成功认定四服消息显示已完成。

2026-09-08 10:27 部署集成阶段 `c050adfe1fdcbd7ce7af2c59183d4deb4ce21df9`，由干净的 Git 源码归档构建；完整 49 项测试及 race 检查通过，0 失败/0 跳过。Linux 二进制 SHA-256 `ea50bbceab204f0d61ce153a55a885b0e78ea7863419fa860e7a43a1631f11e9`，当前目录 `/opt/deuterium/releases/v2-stage-c050adfe1f`。备份在 `/var/backups/deuterium/stage-c050adfe1f/`。

已应用 001/010/011/012/014/015 迁移，70 个账号保留；通过 CLI 向用户指定 xiwanzi 授予 platform.admin。公网已登录的商城、市场、私聊列表、公告和通知均返回真实空数组。Web 浏览器真实 Cookie 会话显示 xiwanzi、正式管理入口和空商城。

Android 仪器针对实际公网服务通过社交、目录、购物袋读接口及 16×16 专用图片的签名 PUT → READY → 授权 GET 解码 → remove。测试没有修改 xiwanzi 的头像或发布公开示例商品/公告，专用探测对象随后精确清理；证据为 App 工作区 `docs/qa/android-v2-2026-09-08/live-social-catalog-assets.json`。当前资金/订单/委托仍待 Core 可靠经济与编排实现，不把这一阶段视为完整交付。

IP 证书方案依据 [Let's Encrypt 官方 Certbot IP 证书说明](https://letsencrypt.org/2026/03/11/shorter-certs-certbot)：Certbot 5.4+ 支持 webroot IP 验证，证书约 6 日，必须配置自动续期和 Nginx 重载。
