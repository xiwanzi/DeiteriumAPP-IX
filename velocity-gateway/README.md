# Deuterium Gateway

使用 Java 21、Velocity API 3.4 编译，已在与线上相同的 **Velocity 4.1.1 / Java 25** 隔离实例验证加载。统一白名单检查与持久化返回上次游玩服务器；默认入口 **amiya**，备用入口依配置优先使用 amiya、login。只向 Go 后端查询权限，不维护第二份白名单。

## 安装

使用 Java 21 执行 `mvn verify`，将 `target/Deuterium-Gateway-1.0.0.jar` 放入 Velocity `plugins/`。首轮部署必须先准备配置及导入原有玩家，再重启代理。后端子服保持回环绑定，Velocity 必须开启正版认证。

`plugins/deuterium-gateway/config.json`：

```json
{
  "apiBaseUrl": "https://47.103.99.34",
  "apiToken": "填写单独生成的网关凭据",
  "applicationUrl": "https://47.103.99.34/admission/",
  "defaultServer": "amiya",
  "fallbackServers": ["amiya", "login"],
  "requestTimeoutSeconds": 5
}
```

Go 配置保存凭据的 SHA-256，代理配置保存原值。凭据不能复用 Core 凭据、提交到仓库或放在网页中。接口仅支持准入查询、固定的在线断开指令及回执。配置无效、正版认证关闭、后端超时或返回无效时拒绝新的连接。

权限命令：`/dgate` 显示连接状态、默认服与已记录玩家数，需要 `deuterium.gateway.admin` 或控制台权限。

## 返回子服

记录成功进入的服务器，存于 `last-servers.json`，异步写入、原子替换并保留上一版本备份。自动回退不会覆盖原记录，后续手动切换成功后才更新。损坏文件不静默清空；有有效备份则恢复并保留损坏文件，无法恢复时停止准入等待维护。

登录检查完成后才选择子服，旧连接的迟到回调不会覆盖最新连接。后端撤销立刻影响下一次准入；选择同时移出在线玩家时，代理每 5 秒拉取固定指令，执行前重新核对权限版本，避免重新授予后执行旧指令。后台显示回执结果。
