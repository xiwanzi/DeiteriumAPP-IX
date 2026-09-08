# DeuteriumAPP 运行端口与 Base URL 约定 v1

## 端口

后端第一版使用两个不常见端口：

- `28657`：公开 App API 与 App Chat WebSocket 端口。
- `28658`：同机 Minecraft 插件桥端口，只监听 `127.0.0.1`。

内网穿透只需要转发 `28657`。不要把 `28658` 暴露到公网或内网穿透。

## Android Base URL

本机测试时：

```text
http://127.0.0.1:28657/api/v1
```

真实手机通过内网穿透访问时：

```text
https://<你的内网穿透域名>/api/v1
```

如果内网穿透只提供 HTTP，则临时测试地址为：

```text
http://<你的内网穿透域名>/api/v1
```

正式发布建议使用 HTTPS。

## Wiki.js OIDC Auth Origin

OIDC Provider 与公开 App API 共用 `28657` 端口，但不使用 `/api/v1` base path。

推荐生产入口：

```text
https://auth.deuterium.cafe
```

当前已确认的内部 HTTP-only 入口：

```text
http://authdeuterium.s.odn.cc
```

该内部入口映射到：

```text
127.0.0.1:28657
```

OIDC endpoints：

```text
/.well-known/openid-configuration
/.well-known/jwks.json
/oauth/authorize
/oauth/login
/oauth/token
/oauth/userinfo
/oauth/logout
```

内部 HTTP-only 环境必须显式设置 `oidc.allowInsecureHttp=true`。公网密码登录应使用 HTTPS。

OIDC access token 只用于 `/oauth/userinfo`，不得用于 `/api/v1/account/me` 或其他 Android App API。

## Chat WebSocket URL

通过 HTTPS 隧道时：

```text
wss://<你的内网穿透域名>/api/v1/chat/ws
```

通过 HTTP 隧道临时测试时：

```text
ws://<你的内网穿透域名>/api/v1/chat/ws
```

## 插件桥 URL

后端和 Minecraft 服务端在同一台机器时，插件配置保持：

```text
ws://127.0.0.1:28658/bridge/plugin/ws
```

插件桥使用 `Authorization: Bearer <plugin-bridge-token>` 鉴权。token 必须只写在生产机器本地配置里，不提交到仓库。
