# 图片审核 Web 2.0.37 上线

2026-10-12 **20:30（UTC+8）**，官方后台新增[图片审核](https://chat.deuteriumix.com/admin?section=image-review)。独立审核服务已启动，Web 2.0.37 已发布；源码 `2cc6c7dab15189fb5741d29dfca6aa8d68a3884d`，实现 [PR #114](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/114)，按用户最新要求收窄范围 [PR #115](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/115)。

## 最终范围

- **旧图全部跳过，只审启用后新上传的静态聊天图片**，包括该上传系统的静态贴纸。动态图片、头像、商品图等业务图片不参与。
- 启用时间固定为 `2026-10-12 12:30:17.244190 UTC`，写入独立数据库，服务重启不会移动边界。
- DeepSeek V4.1 Flash（`deepseek-flash`）直接读取 OSS 临时签名 URL，检查 R18、政治内容和宣传导流，合分严格超过 50 才保存独立证据并展示。
- 管理员可查看图片、三类分数、理由、上传账号/时间和关联发送账号/时间；删除只清理审核记录及证据，保留最小墓碑，原聊天图片不受影响。

## 发布与验证

- 新进程 `/opt/deuterium-image-review/releases/review-2cc6c7dab151`，二进制 SHA-256 `55262c968bc82b6f7d92ed6588c1ff7f2e887ee2919b6e40b346af7cd9a51c0b`，监听回环 8086。独立库 `deuterium_image_review`；业务源账号只有所需列 SELECT 权限，审核账号只有自己库的读写权限。
- Web `/var/www/deuterium/releases/web-2.0.37-2cc6c7dab151/dist`。两个社区 vhost 增加独立 API 代理，Nginx 校验后 reload；主 Go、游戏服、模组及 App 没有重启或更换。
- 主 Go PID `2763256`、原链接和二进制哈希 `1ba7f6098009fd482e7248061a93a9c42da5f9fb62202ef93b0d83dd83b6745b` 保持。61 账号、52 条交易资源、9,329 条目录记录、12 工单的已有 ID 集合全部保留。
- 公网管理员列表、扫描/工作心跳、匿名 401、无效 CSRF 403、删除端点身份校验通过。两个社区域名返回 Web 2.0.37，临时验收会话已撤销。
- 20:31 回读：**49 条旧聊天上传、3,078 条业务图片上传全部排除，审核库 0 条待审/旧图记录**。首次新聊天上传发生后才会新增审核任务，当前空列表符合要求。
- OSS 未启用版本保留，审核证据目录不匹配现有到期规则。供应商实际联调及本机/UI 验证见[验收记录](../qa/image-review-20261012.md)；收窄范围后补跑历史上传边界数据库测试及 Linux/Web 构建。

## 回退

备份 `/var/backups/deuterium/image-review-2cc6c7dab151`。停止独立审核服务，恢复旧 Web 链接和两个 vhost，Nginx 校验后 reload 即可；不恢复业务数据库、不重启游戏或主 Go、不删除已存审核证据。
