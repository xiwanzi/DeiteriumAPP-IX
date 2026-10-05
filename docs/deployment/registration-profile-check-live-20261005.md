# 注册正版 ID 校验上线

2026-10-05 11:26:36（UTC+8）配套 Go 已上线。源码 `6271f468b718`，PR [#68](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/68)，合并 `8ec0df614938bcb4ce3ff291dcedfd47f4cfd636`。用户已明确授权推送、合并与部署。App **2.0.15**、社区 Web **2.0.25** 以及官网、Help 静态版本保持。

## 已生效行为

官网、Help、社区和 App 在最终提交注册时，邮箱验证码检查通过后查询 Minecraft 官方玩家档案。不存在或名称格式无效时返回 `MINECRAFT_PLAYER_NOT_FOUND` 并拒绝创建账号；查询超时、限流或异常响应时返回 `MINECRAFT_LOOKUP_UNAVAILABLE`，提示稍后重试。

查询使用邮箱验证码记录绑定的 `gameId`，失败不创建账号/会话、不消耗验证码。玩家名存在只证明公开档案存在，不代表持有账号；注册成功仍未完成游戏身份验证，也不绑定 UUID 或授予通行资格。旧账号不重新核验、不清理。

## 验证与保留

[本机验收](../qa/registration-profile-check-20261005.md)已通过，发布包与 PR 合并后的 `backend-next` 源码逐文件一致。

公网实测官网来源、Help 来源、社区来源及 App 无 Origin 的最终注册接口：对本次确认不存在的随机测试玩家名均返回 400 `MINECRAFT_PLAYER_NOT_FOUND`；没有创建账号，测试验证码保持 ACTIVE、错误次数为 0。临时验证码记录与只读验收会话均已定向清理，不发送真实验证码或邮件，没有注册真实测试账号。

服务 active、就绪检查通过，Login/Amiya/Odyssey/MEK 四个 Core 节点全部在线。重启前后均为 **45 个非 deleted 账号、27 笔订单、5 个委托、12 个工单、1 条已有注销记录**。配置和环境文件摘要一致，全部迁移版本/校验值一致。没有改 App、Web、游戏插件、邮件配置或业务数据。

## 程序与回退

- 当前 Go：`/opt/deuterium/releases/registration-profile-6271f468b718/deuterium`。
- SHA-256：`7b5cfa2c6ce8e41ef3392a830ee9f6976e9d770ab92e645ef760c174cf1a28fa`。
- 前版：`/opt/deuterium/releases/support-thread-fix-54c389cc9a43`，保留。
- 备份：`/var/backups/deuterium/registration-profile-20261005-6271f468b718`，包含旧程序、配置、数据库及发布/验收记录；服务停止后单事务备份已检查 gzip 完整性。
- 无新增迁移。回退仅切回前版 Go 并重启服务，不能还原整库覆盖上线后用户数据。

当前已完成推送、合并、部署与线上验收，无待发布项。
