# 白名单申请页管理员锁定上线（Admission 1.0.3）

2026-11-27 **16:37（UTC+8）**，白名单通行申请页管理员访问锁与严肃风格 403 拒绝访问页上线。[PR #154](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/154) 已合并至 `main`，合并提交 `d6782ab3bc51`。

## 生效入口

- 官网域名入口：`https://www.deuteriumix.com/admission/`
- 社区域名入口：`https://chat.deuteriumix.com/admission/`
- IP 入口：`https://47.103.99.34/admission/`

## 访问控制行为

1. **前端默认拒绝（Fail-Closed）与严肃 403 拒绝页**：
   - 页面初始状态为 `data-gate="checking"`，在管理员权限校验通过前隐藏申请表单与侧边导航，零表单闪烁；禁用脚本时直接呈现拒绝页。
   - 核验逻辑优先校验官网登录态（`https://www.deuteriumix.com/api/v1/web/session`）及 `platform.admin` 权限：
     - 未登录访客：直接展示深曜石色严肃 403 拒绝页，提示前往官网登录。
     - 普通官网账号登录态（无 `platform.admin`）：直接拒绝访问，清晰标注当前已识别账号及权限不足原因，并提供切换管理员账号入口。
     - 具备 `platform.admin` 权限的管理员：正常解锁申请页并在顶部显示 `ADMIN LOCK ACTIVE` 状态提示条。
2. **服务端同步硬锁（防绕过）**：
   - 在 `deuterium-official.conf`、`deuterium-chat.conf`、`deuterium-test.conf` 的 `/api/v1/admission/` 入口部署 Nginx Lua 内部子请求鉴权（`/_internal/admission-admin-check` 代理校验 `/api/v1/admin/whitelist/summary` 的 `platform.admin` 权限），未登录或非管理员直接返回 HTTP `401` / `403`（`ADMISSION_ADMIN_ONLY`）。
   - `backend-next/internal/httpapi/admission.go` 同步在源码层为 `/api/v1/admission/config`、`POST /api/v1/admission/applications`、`POST /api/v1/admission/status` 加上 `platform.admin` 权限校验，作为纵深防御。
   - Velocity Gateway 桥接接口（`/bridge/v1/admission/check`、`/bridge/v1/admission/poll`）与后台白名单管理接口（`/api/v1/admin/whitelist/*`）完全保持不受影响。
   - `deuterium-official.conf` 放行 `https://chat.deuteriumix.com` 的跨子域凭据 CORS，使 `chat.deuteriumix.com/admission/` 可直接读取官网登录态。

## 产物、备份与回退

- 当前静态目录：`/var/www/deuterium-admission/releases/admission-1.0.3-d6782ab3bc51/site`
- 前版保留目录：`/var/www/deuterium-admission/releases/admission-1.0.2-3541a73604f9/site`
- Nginx 备份目录：`/var/backups/deuterium/admission-admin-lock-d6782ab3bc51/`
- 静态包 SHA-256：`321553cf3447dd2f9fb1cbae6d96bf0b05b82b736e7cb77638d6ba92ba9d9b17`
- `index.html` SHA-256：`aeb6509a564b98ef72990447653e413c229901f8ff6cd74e8fba86d56bb3b7a0`
- `gate.css` SHA-256：`15727f7042ba4a1e64f91d6953863a628d6b852ae76998f5eeba8993f2d1b091`
- `app.js` SHA-256：`61ffcd59b25e773883079736681bae1ba24dd2f28c8aecf6ae7fa76de95b74fe`
- Go 后端 PID `3917899` 全程未重启，无数据库迁移与游戏服操作。

### 核验结果

- 本地浏览器完成四态实测（桌面未登录拒绝、桌面普通成员登录拒绝＋重新核验、390px 移动端拒绝无横向溢出、桌面管理员解锁），控制台零报错。
- 生产环境 `nginx -t` 通过并热重载；三公网入口（`www.deuteriumix.com`、`chat.deuteriumix.com`、`47.103.99.34`）静态资源 SHA-256 一致，三入口 `/api/v1/admission/{config,applications,status}` 匿名访问均返回 `401 ADMISSION_ADMIN_ONLY`；`www.deuteriumix.com`、`chat.deuteriumix.com`、`help.deuteriumix.com` 首页均为 `200`。
- 如需重新对外开放白名单申请，仅需恢复前版静态软链 `/var/www/deuterium-admission/releases/admission-1.0.2-3541a73604f9/site` 与备份的 Nginx 配置并执行 `nginx -s reload`。
