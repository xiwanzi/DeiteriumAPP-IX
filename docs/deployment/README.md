# 部署与维护手册

当前 App 为 [2.0.11（21100）](deuterium-2.0.11-live.md)，Web/Go 沿用 [2.0.10](deuterium-2.0.10-live.md)，游戏组件沿用 [2.0.9](deuterium-2.0.9-live.md) 配套（Core 1.0.2、Mail 0.6.2/API3、Bridge 0.6.1、XConomy .3；Mail UI/Sync 保留实服文件）。三常驻服运行、MEK 停止。SMTP 和经营参数以后台现有配置为准；下方流程用于后续有授权的部署。

## 拓扑和路径

| 对象 | 现行位置 / 要求 |
| --- | --- |
| 公网入口 | `https://47.103.99.34`，443，同源 API；域名在当时仍等待备案 |
| Go | systemd `deuterium.service`，`/opt/deuterium/current/deuterium`，监听 `127.0.0.1:8080` |
| Go 发布目录 | `/opt/deuterium/releases/release-2.0.10-65c08a290896`；`current` 指向已验证版本 |
| 配置 | `/etc/deuterium/config.json`、`backend.env`、`releases.json`；AI 配置与提示词由环境中的文件路径指定 |
| 后端库 | MariaDB `deuterium_backend`，独立账号；禁止复用游戏经济数据库身份 |
| Nginx | 现有宝塔路径 `/www/server/panel/vhost/nginx/deuterium-test.conf`；80 ACME webroot `/var/www/deuterium-acme` |
| 网站 | 发布目录 `/var/www/deuterium/releases/web-2.0.10-65c08a290896/dist`；实际生效 root 从 Nginx 配置核对 |
| APK | Nginx `/downloads/` 公开渠道，发布时必须核对实际 alias/root，不上传私有目录 |
| 云备份 | `/var/backups/deuterium/`，含数据库一致性备份、前版配置和更新清单，受限访问 |
| 游戏备份 | 游戏主机 `E:/Deuterium_IX/deployment-backups/app-v209-20260910-20901`，不是云端公开目录 |

保留 IP 证书续期定时器和成功续期后的 Nginx 重载。更新域名时同步证书、Nginx、Go `publicOrigin`、App API 与更新地址，不能只改网站链接。SSH 连接、服务账号凭据和签名材料走现有受限运维渠道，本仓库不记录密码。

## 配置清单

模板：[Go 配置](../../backend-next/config.example.json)、[环境变量](../../ops/backend.env.example)、[AI 配置](../../ops/ai-config.example.json)、[systemd](../../ops/deuterium.service)、[Nginx](../../ops/nginx.conf.example)、[Core](../../deuterium-core/src/main/resources/config.yml)。模板用于人工配置，不是线上配置原件。

| 配置 | 内容与边界 |
| --- | --- |
| `DEUTERIUM_DSN` | 后端独立数据库；迁移使用可执行 DDL 的受限账号，运行身份仅本库所需权限 |
| `publicOrigin` / `trustedProxyCidrs` | 精确 HTTPS Origin；只信任真实反代地址，不使用全网 CIDR |
| `nodes` | login/amiya/odyssey/mek 独立 tokenSha256、聊天、领取、库存域、Mail 集群和兼容档案；经济权威只能一个 |
| `DEUTERIUM_S3_*` | endpoint/region/bucket/prefix/access/secret；短时签名直传，密钥仅服务端持有 |
| `DEUTERIUM_ASSET_GC_ENABLED` | 回收开关；生产专用前缀 `deuterium-test/gc/` 的 30 天生命周期，不能对整个 uploads 设置过期 |
| `DEUTERIUM_AI_CONFIG_FILE` / `DEUTERIUM_AI_PROMPT_FILE` | JSON 文件；模型、额度、联网策略及客服提示词；不得把真实供应商 key 放入模板 |
| `DEUTERIUM_RELEASE_MANIFEST` | APK/资源发布 JSON 路径，启动加载；校验 URL/大小/SHA-256/包名/版本范围 |

Core 用 `deuterium core-key` 为每节点生成随机密钥，原 token 仅给对应插件，Go 节点只保存摘要。插件数据库各自归属 Core、Mail、XConomy、Sync；其中 Sync 在 Amiya/Odyssey/MEK 共用已有 `minecraft` 库和 `survival` 库存域。不要为当前同步系统换成空库。

## 后端与网站升级

1. 对照组件锁定和发布摘要，完成隔离环境构建/测试。记录旧 `current`、Nginx root、现有配置及服务健康状态。
2. 将新二进制和网站静态内容上传到新的版本目录，校验 SHA-256；保留旧目录。受限备份后端库、配置、环境和更新清单。需要跨库一致性的游戏升级必须按下一节停服备份。
3. 对迁移 SQL 摘要做检查。新增迁移执行 `deuterium migrate`，使用单独迁移身份；不改已执行迁移，不恢复旧账号导入来覆盖现有业务。
4. systemd 使用 [版本目录模板](../../ops/deuterium.service)。在同一文件系统用临时符号链接加原子 rename 切换 `current`；重启 `deuterium.service`，回环请求 `/health/live`、`/health/ready`，检查 journal 与实际二进制摘要。
5. 网站仅切换静态 root/链接，API 和 bridge 继续直接代理 Go。`nginx -t` 成功才 reload；验证深层路由刷新、未登录状态、Cookie/CSRF、WebSocket 与 SSE 增量输出。禁止将 `/api/` 错误回退为 HTML。
6. APK 先完整公开下载核对摘要、包名、签名和版本，再原子替换 `releases.json`，重启服务加载。旧客户端应发现新版本、新版本不能提示自身更新。账号更新通知需本轮明确授权的接收范围。

新库首次部署可使用后端 README 中的 `export-legacy` / `import-legacy` 只读预检与事务导入。当前真实库已经完成账号迁移，该流程不是日常升级步骤。导出包含密码哈希，应受限保存，不能提交 Git。

## 游戏插件升级

Core 1.0.0、Mail 0.6.1 / API2、Sync .3、XConomy 扩展必须配套。Core 放 plugins；Mail 插件与 Bridge 放 plugins；Mail UI 放客户端和服务端 mods，外置客户端依赖见 Mail 仓库。只部署正式产物，`integration-probe` 不进入生产。

1. 从服务器管理器核对四节点状态、在线玩家、已安装摘要、数据库和库存域。MEK 既有停止/禁领状态保留；不能因安装新包擅自开放。
2. 正常停止受影响节点并确认 Java/包装进程退出、旧异步保存完成。共享库存/经济改动不能新旧混跑或热加载。
3. 同一停服窗口对 Core、Mail、Sync/XConomy 涉及库做一致性备份，并备份插件配置、玩家/模组文件。校验压缩包 CRC 和逐文件摘要，记录对应时间点。
4. 替换配套 JAR，保留原包和配置备份。四服 XConomy 统一；Login 禁领且不启用 Sync 集成；Amiya/Odyssey 必须真实上报 `playerDataReady` 和 Mail 能力；MEK 保留停止禁领。
5. 正常恢复原常驻节点，确认 `Done`、Core WSS ACK、Mail API2、资金 provider、Sync 保存屏障、数据库连接及无类加载错误。真实游戏客户端领取与切服另作实测；自动探针不能代替。

## 备份、回退和故障处理

- 发布失败先关闭入口写操作或保留当前健康版本，按已记录的旧 `current`/网站 root 回退可兼容的程序。不要自动恢复旧数据库覆盖升级后新增订单、聊天和资金。
- 图片生命周期迁移 018 为增量列/索引。回退旧后端前先关闭 GC 并停用专用 OSS 生命周期规则，避免旧代码重新绑定即将过期对象；通常优先保持当前程序、关闭回收诊断。
- 付款未知：保持原 requestId、operationId 和确认参数，调用原结果查询；不换键再付款。邮件未知：只核对原持久保存证明，不重新发物；`NOT_FOUND` 不能作为已撤回/可退款证明。
- Sync 失联/隔离：先核对玩家会话代次、库事务、原保存回执和邮件状态；控制台解隔离不等于邮件已回滚。没有真实适配能力时保持禁领。
- 备份恢复是单独维护操作：先停相关写入、评估新业务和跨库一致性，再按确切快照恢复。保留备份摘要、审批/操作时间和恢复后账本核对记录。

## 日常只读检查

```sh
systemctl status deuterium.service --no-pager
readlink -f /opt/deuterium/current
sha256sum /opt/deuterium/current/deuterium
curl -fsS http://127.0.0.1:8080/health/ready
journalctl -u deuterium.service --since '15 minutes ago' --no-pager
nginx -t
```

检查日志时不输出环境变量或私有配置全文。公网健康、游戏节点 ACK、资产 GC 失败、未知资金/邮件操作和证书续期失败都应按实际症状处理；不得用创建真实订单或付款作为默认健康探测。
