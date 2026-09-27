# YSM 管理整合候选与上线步骤

2026-09-28：本候选已按用户授权发布为 Web 2.0.20 和配套 Go，栏目名称为「模型管理」。最终结果以[上线记录](model-management-live.md)为准，下方为发布前候选快照。

2026-09-28，**本机完成，未推送、未合并、未部署**。维护工作区为 `chat-domain`，分支 `xiwanzi/ysm-admin-integration`，基础源码 `df8f8df`。现有 App 2.0.14 / Web 2.0.19 的发布记录仍有效；本候选没有将提高版本号当作已发布。

本轮用户确认将 YSM 栏目、界面和登录统一到现有后台。[实现边界](../plans/ysm-admin-integration.md)、[接口契约](../contracts/ysm-admin.md)、[本机验收](../qa/ysm-admin-integration.md)。用户已于本轮授权推送 PR、发布 Web/Go，并要求栏目名为「模型管理」。发布过程与最终结果另记上线记录。

本机 Linux amd64 / CGO=0 Go 构建位于 `output/ysm-admin/deuterium-linux-amd64`，SHA-256 为 `0d01154d599aa8029916b7cecb1f4cbab45cd89f1c76870801378056bbdcd6e5`。Web 构建位于 `web-app/dist`；这些是本机候选产物，没有上传服务器。

## 发布范围

只发布 Web 和 Go，配置 Go 到 Velocity 回环管理服务的受限连接。无需数据库迁移、重新安装 YSM、重启游戏节点、重新上传模型或改动原 S3 配置。上线后入口为 `https://chat.deuteriumix.com/admin?section=ysm`（此候选尚未在该地址生效）。

## 连接与代理

1. 先只读核实生产 Velocity 管理监听地址与端口，以及现有 Cloud → 游戏主机 SSH 连通方式。源码和既有上线记录为 `127.0.0.1:17655`，不能用 17656 控制端口代替。
2. Go 与 Velocity 同机时直接接回环；异机时使用受限 SSH 转发，把 Cloud 回环端口转发到游戏主机的 `127.0.0.1:17655`。隧道仅本机监听、开启退出失败检测和保活、由服务管理器重启；密钥放服务账号私有目录，不写入源码或网页。只允许目的管理端口的转发，不开放公网免密管理页。
3. 向现有 Go 配置增加 `ysmAdminUrl`（隧道入口）和 `ysmAdminOrigin: http://127.0.0.1:17655`。两者端口不同时保留原 Host 校验。现有 SMTP、节点、域名、并发等字段全部保留。
4. 新旧站点 `/api/v1/admin/ysm/` 仍代理至现有 Go upstream。针对这一前缀配置 `client_max_body_size 512m`、`proxy_request_buffering off`、HTTP/1.1、`proxy_read_timeout 310s` 和 `proxy_send_timeout 310s`；保留原 Cookie、Origin、CSRF 头及受信代理设置。普通 API 的限制不变，不能把 `/api/` 直接代理至 YSM。

## 上线与核对

- 按现有发布流程创建 PR，发布前保存 Go 配置、站点代理配置和 Web/Go 原版本路径；为 Web 确定正式版本并同步元数据。
- 先验证隧道只读 `config/cluster/catalog`，只记录模型数量、版本、节点与摘要，不输出凭据。安装候选 Go 和 Web，正常停启 Go、校验并重载代理。
- 使用管理员账号检查模型列表、节点与原服务一致；匿名 401、普通玩家 403，写请求无正确来源或 CSRF 时拒绝。核对真实登录的栏目访问。
- 此步骤不确认发布、重载、迁移、回收或真实模型增删。任何真实写验收需使用另行明确授权的测试模型范围。
- 保存部署哈希与时间，更新上线记录和维护入口。没有实际完成上述步骤前，不称为上线。

## 回退

恢复旧 Web/Go 路径和原 Go 配置（旧 Go 严格解析配置，回退时必须移除新字段），恢复代理配置后校验并重载；停止新增隧道服务。不要恢复旧数据库或模型库覆盖线上新增业务。本次整合没有迁移表或更改 YSM 模型数据。
