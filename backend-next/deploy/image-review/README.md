# 独立图片审核部署

只部署独立二进制、独立数据库及 Web。禁止使用主后端的 `migrate`、服务重启或整库恢复命令。现有 Go、游戏节点、模组、配置保持。模型使用官方 `deepseek-flash`（DeepSeek V4.1 Flash），规则与 50 分阈值见[契约](../../../docs/contracts/image-review.md)。

1. 从同一提交构建 `go build ./cmd/deuterium-image-review` 与 `web-app`。Linux 使用 `CGO_ENABLED=0 GOOS=linux GOARCH=amd64`。
2. 只读确认业务库包含媒体及身份表、对象路径、当前服务 PID/哈希、Web 链接。备份两个社区 HTTPS vhost、Web 链接及业务 ID/数量快照。确认对象桶未开启版本保留，且审核证据目录不匹配现有自动过期规则。
3. 创建单独数据库 `deuterium_image_review`，执行 `internal/imagereview/schema.sql`。也可对该库执行新二进制 `--migrate-only`；它拒绝非 `deuterium_image_review*` 名称。无需修改业务 schema 或迁移清单。
4. 创建源库只读账号，仅读取身份会话、权限、图片上传/对象及消息归属所需列；审核库账号仅 SELECT/INSERT/UPDATE/DELETE。运行账号不持有 root 数据库权限，也不持有主服务 EnvironmentFile。
5. 以独立系统用户 `deuterium-review` 运行所附 systemd unit。服务专用环境文件仅 root/运行用户可读，密钥不写入命令参数、代码或构建物。监听 127.0.0.1:8086。
6. 启动独立服务，确认本机 `/health/ready`、扫描心跳、队列。首次会补查既有仍存在图片；新上传优先，已删除原图标记不可用，动态图片跳过。
7. 在现有社区 HTTPS vhost 添加所附两个 location；原样保留现有代理头与其他路由。先 `nginx -t`，再原子切换 Web 链接和 **reload** Nginx，不能 restart 游戏或现有 Go 服务。
8. 验证管理员访问、匿名/普通账号/客服拒绝、CSRF、图片跳转与删除；核对主 Go PID/启动时间/二进制及业务 ID 快照。高分未出现不等于审核没运行，应检查统计与最近扫描时间。

## 凭据和存储

环境模板不可原样上线。图片用短期只读签名 URL 送给模型，服务只读至多 64 KiB 文件头，完整证据使用 OSS CopyObject。证据目标固定为 `<prefix>image-review/evidence/<record-id>.image`，删除只允许该目录和确切 ID。证据删除后 HEAD 确认不可读，再清除身份、理由及消息快照，留下最小上传标识防止重建。

## 回退

停止新审核服务、恢复旧 Web 链接及两个 vhost，`nginx -t` 通过后 reload。不恢复业务数据库，不停止现有后端，不删除审查数据库或已存证据。恢复审核时继续原队列；后台故障不会中断玩家上传、聊天或使用贴纸。
