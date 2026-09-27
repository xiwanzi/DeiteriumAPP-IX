# 模型管理 / Web 2.0.20 上线

2026-09-28（UTC+8）：配套 Go 于 **05:45:51** 上线，最终 Web **2.0.20** 于 **05:51:28** 完成发布。用户本轮授权推送 PR、发布 Web/Go，并将栏目名称确定为「模型管理」。

入口：[模型管理](https://chat.deuteriumix.com/admin?section=ysm)。使用原 Deuterium ID 管理员登录；普通玩家不能访问。

## 范围与源码

- 后台统一导航、界面、登录和 `platform.admin` 权限，接入模型库、节点、任务与审计、存储与配置。
- Go 经受限 SSH 回环连接原 Velocity 管理 API；模型、配置、签名发布、暂存及回收仍由原 YSM 服务负责。原免密管理端口没有暴露公网。
- 保留上传预览、版本检查、一次性确认，确认票据加密并绑定当前登录会话和操作；结果未知不自动重试。中央管理审计补充操作者与请求关联。
- 实际接口的 `directory` 是完整模型路径，因此分类按原管理页的 `modelId` 一级目录规则处理；生产为 **10 个分类**。
- App 保持 **2.0.14（21400）**，游戏插件不更换，无数据库迁移，游戏节点未重启。本轮没有对真实模型执行新增、替换、删除、发布、重载、存储迁移或回收。

| 内容 | 来源 |
| --- | --- |
| 功能及命名 | [PR #50](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/50)，合并 `918584f72edf5b615b28daa112661a17cc42a0eb` |
| 分类规则对齐 | [PR #51](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/51)，合并 `25931e2ce897459888cf8f05eb469172bd80fc47` |
| 已部署 Go | 从 `918584f72edf` 干净源码构建，`vcs.modified=false` |
| 最终 Web | `25931e2ce897`；构建源 `928772f` 与合并版本的 Web 内容一致 |
| 上游 YSM 协议依据 | `Deuterium-IX/DeuteriumYSM` / `b45a3e42cc197f890ed12ca6f13abd2899fb1428`，本轮不改其代码 |

维护：`.worktrees/chat-domain`，上线记录分支 `xiwanzi/model-management-record`。

## 产物与运行位置

| 产物 | 字节数 | SHA-256 |
| --- | ---: | --- |
| Go Linux amd64 / CGO=0 / Go 1.27.1 | 27,846,623 | `0238c92ad316e3e300e0967eb0965fd373a4e45045dd207a0f01614bc9d9681d` |
| 最终 Web ZIP | 33,720,117 | `618e7dea3790a10f4575a6aec2602d6b4dcabb4361b69ecba80bdb6e0a5651f7` |

- Go：`/opt/deuterium/releases/model-management-918584f72edf`。
- Web：`/var/www/deuterium/releases/web-2.0.20-25931e2ce897/dist`。
- 新旧入口 `chat.deuteriumix.com` / `47.103.99.34` 共用现有 Web 链接，最终 JS 为 `index-06PVVIIS.js`。
- 配置仅增加 `ysmAdminUrl` 和 `ysmAdminOrigin`，均为 `http://127.0.0.1:17655`。原域名、SMTP、节点、并发、App 清单、启动器发布配置与 Go 服务文件保留。
- 两个 Nginx 站点仅增加 `/api/v1/admin/ysm/` 专用代理，最多 512 MiB、310 秒、关闭请求缓冲；实际 YSM 配置仍将模型上传限制为 64 MiB。

## 受限连接

Cloud 新建 `deuterium-ysm-tunnel.service` 和独立服务账号/密钥，仅监听 `127.0.0.1:17655`，转发至游戏主机回环 17655。沿用已经核实的 SSH 主机指纹，禁止自动接受新指纹；开启连接失败退出、保活和自动重启。

Windows 实际使用 `C:\Users\ovo\.ssh\authorized_keys`。新增独立公钥条目保留其他密钥，以 `restrict`、允许转发、`permitopen="127.0.0.1:17655"` 和拒绝命令的强制命令约束。私钥只留在 Cloud 服务目录。曾向未生效的管理员公共密钥文件添加条目，发现后已精确撤回；实际用户密钥文件已单独备份，原条目保留。没有修改 sshd 配置或重启 sshd。

实测该专用密钥的命令执行被拒绝，转发到其他目的端口被拒绝；Cloud 监听地址仅为回环。源模型管理服务和四个游戏节点全程保留。

## 验证与发布期间的并行更新

- 本轮命名与升版后的 Web 构建、71 项前端测试通过。Go 的全量单元、4 项 YSM 集成/单元、12 项本机浏览器检查见[本机验收](../qa/ysm-admin-integration.md)，这些是此前候选阶段验证，不冒充此次生产写验收。最终分类小修完成 Web 构建，并在真实页面检查全部 10 类与接口一致。
- 正常 TLS 校验下，新旧公网入口均返回 Web 2.0.20；页面及静态 JS 包含「模型管理」和最终分类规则。匿名模型请求返回 401。
- 临时短期会话核对：普通玩家 403；管理员缺少 CSRF、外站 Origin 均 403；管理员读取 catalog / cluster / config / tasks / audit / diff / migration / gc 和诊断包成功。原 YSM Cookie、CSRF 及存储密钥不透传。
- 真实管理员浏览器验证 256 模型、10 分类、分类筛选、四个页面、四个节点以及 390px 窄屏；模型写请求为 0，无 pageerror。正常会话探测的 401 不计页面异常。临时验收会话全部撤销并再次确认 401，本机及服务器的临时会话凭据文件已删除。
- 最终 **256 模型 / 476 文件 / revision 11 / config revision 3**；清单摘要仍为 `af541f09d49d75ef743cf8f02106882f6c413a4009ce7562273c77c39a6fb717`。Amiya、Login、MEK、Odyssey 四个 YSM 节点均 READY，四个 Core 节点也在线。
- **72 条账号记录、29 笔订单、5 个委托、1 条已有注销记录保留**；停稳时的账号和业务资源 ID 集合全部保留，27 个迁移及原配置/服务文件核对通过。
- 初次切换的完整性校验发现，预检后 **05:41:26** 已完成另一轮启动器同步，`published_index` 更新但内容/发布版本仍为 9。脚本按旧预检摘要比较而触发自动回退；确认既有同步任务后，改用停稳时的最新摘要重新校验并完成切换。没有恢复旧索引、整库或覆盖用户并行更新。最终保留的新索引摘要为 `0a2a59859abc7fba9b125f5e29d22737a7453e9d7ce01e1e581d57114b262f18`。

没有通过生产真实写操作验证模型渲染、NPC 回退、客户端切服、存储迁移或大文件吞吐；这些不是本次发布验证结论。

## 备份与回退

备份目录 `/var/backups/deuterium/model-management-20260928-918584f72edf`，0700；敏感文件 0600。数据库一致性备份 **15,011,835 字节**，SHA-256 `19c5ba4deefa0d85355168357d7b23498ca17d4363313a405326f22f308ad66f`，解压校验通过；同时保存原 Go 配置、环境文件、App 清单、服务文件摘要、两份 Nginx 配置及业务 ID 集合。

回退时正常停止 Go，恢复原 `config.json`（旧 Go 严格解析配置，需移除新增字段），切回 `/opt/deuterium/releases/release-2.0.14-4e95ae0df69b` 和 `/var/www/deuterium/releases/web-2.0.19-25a2ed8/dist`；恢复两个站点配置，校验并重载 Nginx，启动 Go 并验证就绪。可停止新增隧道服务，并仅删除专用公钥行；不能用整份旧密钥文件覆盖后续更新。**不得恢复数据库或模型库覆盖新业务。**

[构建清单](artifacts/model-management/manifest.json) · [最终 Web](artifacts/model-management/manifest-web-final.json) · [切换](artifacts/model-management/activated.json) · [Web 最终发布](artifacts/model-management/web-final.json) · [权限验证](artifacts/model-management/account-verification.json) · [公网验证](artifacts/model-management/public-verification.json) · [浏览器验证](artifacts/model-management/browser-verification.json) · [最终状态](artifacts/model-management/final-state.json)
