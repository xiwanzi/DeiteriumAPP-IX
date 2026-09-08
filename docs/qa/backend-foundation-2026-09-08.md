# Backend Foundation 本机验收

日期：2026-09-08（Asia/Singapore）。对象：`codex/deuterium-backend-rewrite` 的 `backend-next`，Foundation 0.1。测试数据全部合成，本机数据库独立；没有操作生产账号或经济数据。

## 结论

首个后端基础里程碑已可运行，20 项测试通过，竞态检测无报告，Go vet 通过；实际 HTTP 响应通过复制的 App OpenAPI 校验。Linux amd64 已构建并检查构建信息，尚未验证 Linux 实际运行、systemd 安装、公网 WSS 或真实 Core 插件。

**邮箱保留独立插件。** 本次仅提供 [邮箱字段与时序](../contracts/mailbox-integration-v1.md)，没有修改或合并邮箱源码。商城付款/退款、Core 游戏适配和其他完整业务仍在后续阶段。

## 本次执行

环境：Windows amd64，AMD Ryzen 5 9600X，Go 1.27.1，MariaDB 11.8.8。Go 与 MariaDB 下载均核对官方 SHA-256，工具链留在本机 `.tools`，不进入源码或交付包。MariaDB 仅监听回环 33717；每项集成测试建立随机前缀库，完成后清理自身库。

| 验证 | 本次结果 |
| --- | --- |
| `go test -race -tags integration ./... -count=1` | 20 个测试全部通过，无竞态报告 |
| `go vet -tags integration ./...` | 通过 |
| 旧 Java 密码兼容 | 使用旧依赖 argon2-jvm 2.11、Argon2Factory.create()、3/65536/2 生成 Argon2i 合成哈希；Go 正确验证并拒绝错误密码 |
| 实际响应 → App OpenAPI | AuthResponse、UserProfileResponse、ChatMessagesResponse、ErrorResponse、LogoutResponse，5 类全部通过 |
| 增量契约 | 网页会话、Core 节点/物品查询的独立 OpenAPI 结构与外部模型引用校验通过 |
| Windows 二进制 CLI | 实际运行 migrate、只读 import 预检、apply、serve、登录及授权查询、logout；第二个服务实例被数据库实例锁正确拒绝，原实例仍 ready |
| Linux 构建 | GOOS=linux、GOARCH=amd64、CGO_ENABLED=0、trimpath；构建成功，不依赖 JVM |
| govulncheck 1.7.0 | 可达符号与已导入包均 0 项；x/crypto 模块含未使用 openpgp 的 GO-2026-5932 提示，本程序只使用 Argon2，不导入 openpgp |

App 契约快照与完成时源文件 SHA-256 一致：`ce8091745107bab16822c53475aece94bed7104a149a89ad0850aa3d03316a48`。原契约的 existing/contract_only 标签是 App 配套旧后端状态；本重写实现清单以 README 为准。

## 行为覆盖

- 只读预检不写身份；重复导入返回 unchanged；中途身份/别名冲突整批回滚；QQ 与游戏名交叉冲突拒绝；非法哈希内存参数预先拒绝；旧库只读导出与导入可往返。
- 旧密码正确登录后升级 Argon2id；App/网页同时首次登录可完成，真实改密后旧校验不能发会话；重复迁移不覆盖升级结果；禁用账号不能登录，校验后账号状态改变不能再发会话。
- 并发失败尝试不能绕过持久限次；创建新的身份服务仍保留锁定；游戏名与 QQ 共用账号主体的失败预算。
- 浏览器登录 Origin 限制、HttpOnly/主机 Cookie、CSRF、生产 Secure、App/网页会话类型隔离；普通迁移玩家没有后台权限。
- Core 密钥不能冒充别的节点；重复节点连接拒绝；越权命名空间、伪造来源字段和未知服务器范围拒绝。
- 16 个并发重复事件只生成一条消息；同事件 ID 修改内容冲突；物品同版本修改拒绝且回滚事件标记。
- 四个**模拟 Core 客户端**接入；游戏消息进入公共流并去重，不从后端绕回 TrChat；App 消息持久分发四节点、按节点 ACK；离线节点连接后补收；已接受消息离线重试仍返回原结果；新消息全服离线明确失败。
- 注销后已有实时连接不能再发送；WebSocket 拒绝错误 Origin 与 URL token；慢订阅唤醒合并、有界内存。

## 一次短测的观察值

这是同机回环 HTTP、单个合成账号、keep-alive、16 并发、1024 次 GET account/me 的短测；当时机器也在执行测试，没有做 CPU 隔离。总时长仅 0.167 秒，**不是稳定容量测试或新旧性能对比**。

| 指标 | 观察值 |
| --- | ---: |
| 请求错误 | 0 / 1024 |
| 平均 / P50 延迟 | 2.16 / 1.98 ms |
| P95 / P99 延迟 | 4.07 / 6.20 ms |
| 本次短窗口吞吐 | 6132.6 请求/秒，仅该样本 |
| 首次旧密码登录（含升级） | 137.67 ms |
| 服务启动后空闲工作集 | 11.73 MiB |
| 登录后工作集 | 142.69 MiB |
| 短测结束工作集 / 进程历史峰值 | 143.79 / 143.79 MiB |
| 稍后空闲复查（06:11） | 18.94 MiB，表明短测分配已被运行时回收；不作为固定常驻承诺 |
| Argon2 验证微基准（3 次） | 54.63 ms/次，约 64 MiB 分配/次 |

内存值是 Windows 进程工作集，不包含 MariaDB，也不是 Linux RSS。密码哈希需要内存，不能拿 11.73 MiB 当作包含并发登录时的常驻承诺。后续测量真实 TLS/WAN、实际数据量、长连接和完整业务，并与旧后端在相同条件下对比。

可复现脚本：`backend-next/scripts/smoke_load.py` 只允许回环地址；原始测试事件、响应和测量文件位于本工作区被忽略的 `backend-next/local/qa/`。交付包只包含精选非敏感证据，不含会话、数据库配置或账号导出。

## 尚未验证与下一步

1. 按独立邮箱契约，由用户完善能力查询、幂等投递、权威查询、领取/撤回互斥和持久事件；后端/Core 在这些能力齐备前不启用真实商品付款。
2. 开发 Core 的实际 TrChat/物品库/独立邮箱适配，验证 Youer/1.21.1 与四服行为。MEK 的 playerdata 开启不代表背包域、缺模组及保存屏障已验收，默认禁领。
3. 完成注册/改密验证码、官方商城交易纵切，逐步接入市场、委托、私聊/通知、OSS、更新、外观及 OIDC。
4. 确定 Linux 主机与域名后，在隔离环境做 WSS、TLS、真实数据库权限、备份恢复和部署演练；生产切换另按明确授权执行。
