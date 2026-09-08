# 网页 2.0.0 部署说明

日期：2026-09-08。对象为本工作区 `web-app/dist`。本文件提供可复现部署约束，远程实际部署证据由主任务统一记录。

## 本轮确认

- 当前使用 `https://47.103.99.34`；`lnyozu.cn` 尚在备案。无需配置域名，也不把域名写进前端请求地址。
- 网页与 App 共用后端账号和业务数据。浏览器一律请求同源 `/api/v1/*`。
- Nginx 提供可信 IP TLS，后端 `publicOrigin` 必须为 `https://47.103.99.34`。
- 后端绑定 `127.0.0.1:8080`；Nginx 直接代理 API，不经过 Node 网关。WebSocket 允许 Upgrade、Connection，保留真实 Origin，并将用户地址通过受信代理头交给后端。
- Core 的 `/bridge/v1/connect` 由同一入口直接代理后端；网页不持有 Core 凭据，不调用桥接口。

## 产物

在 `web-app` 执行 `npm ci`、`npm test`、`npm run build`。仅 `dist/` 是网站发布内容，包含静态 HTML、CSS、JS、现有品牌素材和 `web-config.json`。

`src/`、测试脚本、QA 截图、旧演示源码、`node_modules/`、用户凭据和本机数据均不需要上传。生产没有 Node 常驻依赖。

必须对前端业务路径配置 `try_files $uri $uri/ /index.html`，使 `/wallet`、`/me`、`/information` 刷新仍进入网站；`/api/` 和 `/bridge/` 不得回退 HTML。哈希静态资源可以缓存；`index.html` 和 `web-config.json` 应 `no-cache`/`no-store`，避免覆盖部署后仍用旧入口。

Node `server.mjs` 和 `gateway.mjs` 保留用于本机运行；其默认监听回环。未经额外受信代理配置，不在 Nginx 后再次用 Node 转发 API，以免覆写实际客户 IP 导致限流混用。

## 发布后验收

1. 未登录访问 `/wallet` 显示真实登录；`/preview/` 不能取得演示身份。不存在默认账号、合成余额和示例公告。
2. 登录、刷新恢复、退出后会话失效；Secure/HttpOnly Cookie 和 CSRF 检查生效。密码及 CSRF 不写 LocalStorage。
3. 网页、App 和 Core 双向公共聊天；WebSocket 重连恢复历史不重复。游戏服离线时不报告已发送。
4. 查询服务器余额、查找已确认收款人、受控小额转账。对同一 `clientRequestId` 重放不重复扣款。测试收付款及原游戏余额均可核对。
5. 响应丢失时保留请求并查询，余额失败显示不可用或上次已知值；不把未知结果显示成功，不根据本地数值记账。
6. 普通用户看不到管理入口；直接访问 `/admin` 的请求仍由后端拒绝。没有权限字段的兼容服务可直接访问该路径，由后端鉴权。
7. 尚未完成的服务显示不可用，空服务返回空列表；不能因本地测试通过而声称全部业务已接通。

网页转账恢复记录保存在当前标签页会话存储，并按用户引用隔离，不含密码、Cookie 或 CSRF。真实业务结果始终由服务端返回；终态成功会清理该记录。

本轮增加的订单购买/委托预付也保留原创建请求，以 `/operations/by-client-request` 或原 operationId 查询结果。订单/委托详情从服务端的 pendingOperationId 和 availableActions 恢复跨设备业务状态。部署资金页面必须同时挂载 013 接口；管理审计依赖独立 017 索引和 `/admin/audit-events` 路由。SSE 应关闭代理缓冲，保持 AI 增量响应。

原创建请求只有在无已知 operationId、原键查询明确 404、原账号与原 Origin 均吻合时，才显示“重试原请求”。点击后再次查原键，仍明确不存在才用不变正文与请求编号重放。业务资源 404、网络异常或 UNKNOWN 不构成重放依据。报价过期停止恢复并要求重新报价/确认，绝不自动换键再次付款。
