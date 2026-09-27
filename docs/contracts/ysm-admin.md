# YSM 统一管理接口

2026-09-28，本机候选。上游为 [Deuterium-IX/DeuteriumYSM](https://github.com/Deuterium-IX/DeuteriumYSM)，核对源码 `b45a3e42cc197f890ed12ca6f13abd2899fb1428` 的 `velocity-plugin` 管理 API。本轮没有修改上游协议或重新发布插件。

## 入口与权限

Web：`/admin?section=ysm`。API：`/api/v1/admin/ysm/`。

全部操作要求有效 Deuterium 会话和 `platform.admin`；浏览器写操作同时要求原有会话 CSRF、精确 Origin。沿用退出、封禁、永久注销与权限撤销的逐请求校验。

Go 配置：

```json
{
  "ysmAdminUrl": "http://127.0.0.1:17655",
  "ysmAdminOrigin": "http://127.0.0.1:17655"
}
```

两个字段为空时功能明确显示未配置。地址只接受带端口的字面 IPv4/IPv6 回环 HTTP origin，拒绝域名、外部 IP、路径、查询、凭据和重定向。`ysmAdminUrl` 是 Go 所在机器的隧道入口；`ysmAdminOrigin` 是 Velocity 实际监听地址，用于严格 Host/Origin/Referer 校验，缺省等于前者。上游当前示例管理端口是 **17655**，17656 是节点控制端口，不能混用。

## 路由

| 方法 | 后缀 | 用途 |
| --- | --- | --- |
| GET | config、cluster、dashboard、catalog | 公开配置、节点、概览、模型库 |
| GET | tasks、diff、audit、migration、gc | 任务、差异、审计、迁移与回收状态 |
| GET | diagnostics.zip | 原服务生成的无凭据诊断包 |
| POST | operations/preview、operations/confirm | 验证存储、目录扫描、发布、重载、回收、重试 |
| POST | models/stage | 二进制模型上传与预览 |
| POST | models/delete/preview、models/confirm | 删除预览与模型变更确认 |
| POST | config/preview、config/confirm | 配置差异与保存 |

JSON 返回使用现有 `{requestId, data}` 包装；错误使用 `{requestId, error:{code,message,details}}`。原上游 202 返回的 `taskId` 表示已提交后台任务，不表示节点已同步或发布已完成。

上传沿用上游 `Content-Type`（`application/zip` 或 `application/vnd.deuteriumysm.model`）、`X-Model-Id`（URI 编码）、`X-Model-Id-Encoding: percent`、`X-Model-Operation`（add/replace）、`X-Expected-Revision`。Go 流式转发，硬上限 512 MiB；上游配置可以更小。解包、Zip Slip、内容校验、暂存清理、版本 CAS 均仍由上游实现。普通 JSON 请求上限 1 MiB，响应上限 16 MiB，最多 4 个同时进行的管理请求。读取 15 秒，写操作 5 分钟；只延长本组鉴权接口的读写期限。

## 确认、隔离和结果未知

- 每次预览独立取得上游 CSRF，生成的确认票据由 Go AES-GCM 封装，附加认证数据绑定当前登录会话摘要和确切确认路径。浏览器看不到上游 CSRF、Cookie 或原始确认令牌。
- 预览与确认期间读配置、其他标签页或其他管理员预览不会改写已有确认的 CSRF。更换登录会话、跨操作类型、篡改、过期及 Go 重启均使票据失效。
- 上游继续负责票据的一次性消费和 revision 校验；Go 不重放 POST、不跟随重定向，也不把客户端 Cookie、Authorization 或转发头带到 YSM。
- 确认前写入原管理审计，记录操作者、动作和请求 ID；响应成功后追加接受记录与 taskId/configRevision。YSM 原审计仍显示 `web:loopback`；追溯真实后台操作者用中央管理审计。请求已记录、接受审计失败时不误报任务失败、不重试操作。
- 超时返回 `YSM_RESULT_UNKNOWN`，页面丢弃已提交的确认按钮，保留编辑草稿，并提示先核对任务、审计与版本。重新预览、重复确认不能作为自动恢复策略。

## 本机验证范围

见 [本机验收](../qa/ysm-admin-integration.md)。本轮用独立数据库和受控上游夹具验证，尚未配置生产隧道或用该页面修改真实模型；真实模型分发、NPC 回退与玩家切服不计本次验收。
