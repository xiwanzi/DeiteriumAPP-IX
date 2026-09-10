# 2.0.8 应用图标切换契约

实现：原生 Android `ui-lab`、Go `backend-next`、管理网页。协议新增于 2.0.8（20800），兼容原有 App / Web 请求。

## 可选图标与权限

| iconId | 名称 | 最低 App 版本 |
| --- | --- | --- |
| `default` | 默认图标，白底黑色弧线 D | 20800 |
| `anniversary_911` | 双子节图标，用户周年活动“911双子节” | 20800 |

安装包预置图标；后端选择图标标识，不接收 SVG、图片 URL、组件名或代码。初始配置为 `default`、版本 1。只有 `platform.admin` 可查看管理目录和发布配置；公共读取不需要登录，不包含管理员或玩家信息。

## HTTP

`GET /api/v1/app/launcher-icon` 返回统一响应外壳中的 `data`：

```json
{
  "iconId": "default",
  "version": 1,
  "minAppVersionCode": 20800,
  "updatedAt": "2026-09-10T02:00:00Z"
}
```

`GET /api/v1/admin/launcher-icon` 返回 `data.settings`（上述对象）和 `data.icons`（`id`、`name`、`description`、`minAppVersionCode` 的数组）。所有响应 `Cache-Control: no-store`。

`PUT /api/v1/admin/launcher-icon`：

```json
{
  "clientRequestId": "stable-unique-request-id",
  "expectedVersion": 1,
  "iconId": "anniversary_911"
}
```

成功返回更新后的配置。相同值不提高版本，也不新增切换审计。目标不同则版本加一。请求 ID 在账号 / 操作范围内幂等；响应未知时保留完整请求体重试。相同 ID 改变内容或 `expectedVersion` 过期返回 409；非法标识返回 400；未登录 401；权限、浏览器 Origin 或 CSRF 不符 403。事务内再次检查管理员权限。

管理页在提交成功后重新读取当前配置，避免将旧幂等回执当作最新全局状态。若并发操作已启用了同一目标，显示“无需重复发布”；其他冲突保留选择，让管理员核对后再操作。

## 实时提示及补同步

新 Android 聊天 WebSocket 握手携带 `X-Deuterium-Launcher-Icon: 1`，才会收到额外事件；旧 App / 浏览器不受新帧影响。

```json
{
  "type": "app.launcher-icon.changed",
  "sentAt": "2026-09-10T02:01:00Z",
  "payload": { "version": 2 }
}
```

连接建立时发送当前版本提示，随后只在配置版本增加时发送。配置提交后调用已有 Hub 唤醒；5 秒轮询覆盖提交后崩溃或丢失唤醒。事件不直接携带可执行指令，App 收到后补拉公开配置。未登录时仍可通过启动 / 前台读取同步。

App 在每次进入 STARTED 状态时读取，保持前台时每 30 秒补同步；请求串行化。旧版本响应、同版本却不同的标识、未知图标、不兼容最低版本、断网及旧后端 404 均不清除有效选择。成功核对 PackageManager 启用状态后保存本机版本和图标。配置属于设备上的安装实例，切换账号不会产生两套桌面图标。

## 资源与启动入口

默认 alias 延续 `.MainActivity` 组件标识；活动 alias 为 `.LauncherAnniversary`；两者都指向始终可用的 `.DeuteriumActivity`。内部通知直接指向真实 Activity。API 33+ 使用批量启用状态更新，26–32 先启用新入口，再停用旧入口，避免零入口。切换不主动结束正在运行的 Activity。

默认图标使用 VectorDrawable。活动图标由已确认 SVG 导出 1536px 透明前景，保留中央 72 单位构图并补齐视差区域；黑白 SVG 源稿随网页源码保存。API 33+ 系统主题图标使用简洁的单色 D，因此活动横幅可能不显示。App 中的图标预览读取当前选择；版本徽标不在本次切换范围。

## 发布、回退与状态语义

部署前备份配置与数据库；022 只新增图标配置表，不修改原有业务记录。顺序为兼容后端 / 迁移 → 管理网页 → APK 分发。APK 新装默认普通图标，后续按已发布配置同步。

“配置已发布”只表示服务器已保存并通知，不表示所有手机桌面已刷新。本轮没有加入设备追踪或送达统计。桌面实际刷新速度、固定位置与缓存行为依启动器而异，验收记录注明设备。

回退活动外观时在后台发布 `default`，形成更高配置版本。不要直接恢复旧数据库行或降低版本。若撤回到没有该接口的旧后端，已安装 App 会保留本机最后有效图标；需保留兼容接口直到需要的回退同步完成。不得回滚整库覆盖新增业务。
