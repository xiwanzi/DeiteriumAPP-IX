# Go 并发隔离与查询合并上线

2026-09-15 **20:47:45（UTC+8）**，Go 后端已切换至并发优化版。用户明确授权推送、合并和上线；[PR #43](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/43) 已合并，运行源码 `a824088cb518e1cbccf9ffe0d6a52a05ad31cadf`。

## 本次变化

- 普通 HTTP、玩家 WebSocket、AI SSE、Core 长连接和准入/运维分别限额；默认分别为 256、1024、64、32、64。原生产配置未改，使用新代码默认值。
- 相同游标/Hub 修订的公共聊天读取合并，最多缓存 128 页、有效期 1 秒；保留逐用户会话校验、5 秒周期补查及失败恢复。
- 密码校验保持 2 路，额外最多 32 个请求短时等待 2 秒；AI 生成数沿用后台设置，最多 16 个额外请求等待 10 秒。
- 新增仅 `platform.admin` 可访问的 `/api/v1/admin/runtime/capacity`，提供容量、拒绝次数和数据库等待计数。

本次只替换 Go。App 保持 2.0.13、Web 2.0.19、Windows 启动器 0.6.1、申请站 1.0.2；未替换游戏插件、调整数据库结构或发送账号更新提醒。库存、扣款与交付状态机保留。

## 发布与证据

| 项目 | 结果 |
| --- | --- |
| Go 目录 | `/opt/deuterium/releases/backend-concurrency-a824088cb518` |
| Go SHA-256 | `ce12c2fb4bcc944d8abeae379ab9a888358eb64d427f3bf289fdffba7d2395e3` |
| 旧 Go 目录 | `/opt/deuterium/releases/dlauncher-0.6.1-25a2ed8` |
| 备份目录 | `/var/backups/deuterium/backend-concurrency-a824088cb518` |
| 数据库备份 SHA-256 | `d700b4bf1f1fa7825e69cbb27a6e21a3aaffcf280ac1a448b4206e2812cb3bf4` |
| 停启检查窗口 | 20:47:44–20:47:45；不代表所有客户端重连耗时 |
| 服务状态 | `active / running`，健康接口 `ready`，新进程文件摘要与发布包一致 |
| 业务记录 | 71 条账号记录、27 笔订单、5 个委托；数量和 ID 摘要前后一致 |
| 配置与 schema | 原配置文件摘要、迁移记录摘要均保持一致 |

发布前及停稳后均确认没有活动交易、没有未到期的 pending/streaming AI 请求。数据库以 single-transaction 方式备份，旧程序与配置副本保留；未执行数据库迁移或整库恢复。

上线后只读检查了运行文件、健康接口和业务摘要；公网 Web 配置仍为 2.0.19，申请站返回 200。新增容量接口匿名访问返回 401。既有本机管理员会话已不可用，复用请求同样返回 401；本轮未创建/刷新管理员会话，因此尚未完成生产管理员身份下的容量页面数据读取。

Linux 发布包使用 Go 1.27.1 从合并源码构建，`GOOS=linux / GOARCH=amd64 / CGO_ENABLED=0 / vcs.modified=false`。按用户要求未运行单元测试、数据库集成测试或压测。以上健康检查不是容量测试，不承诺 1024 活跃用户或百人抢购已经实测通过。

## 回退

若出现问题，正常停止 Go，将 `/opt/deuterium/current` 原子切回保留的旧目录，再启动并检查 ready；原配置未增加 concurrency 节，可直接供旧程序使用。此次无 schema 变化，不恢复数据库或业务数据。若以后添加 concurrency 配置，回退旧 Go 前先移除该节。

未知交易、未知 AI 结果仍沿用原请求和数据库记录恢复，不通过回退重复执行。本次部署脚本已包含启动/就绪失败时切回旧程序的处理，发布未触发回退。

[实现范围与限制](../plans/backend-concurrency.md) · [发布核对摘要](artifacts/backend-concurrency/verification.json)
