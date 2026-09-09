# 2.0.4 管理与统一流水验收

2026-09-09。本机代码 `018d1e25b0802a5b5130aa3eff4a04b0a612fd73`。生产环境未部署、未删除公告、未改经济余额、未发送真实邮件。

## 已完成检查

| 范围 | 本次结果 |
| --- | --- |
| Android | 121 项单测通过；Lint 0 errors / 29 warnings；2.0.4（20400）APK 构建、包名/版本/原签名和覆盖安装核对通过 |
| 模拟器 | 游戏内标签、最近流水排序、独立日汇总、旧查询在途时追加刷新、连续浅深色切换不重建 Activity/Composition、删除按钮浅深色渲染通过 |
| Web | Vite 生产构建通过；54 项既有 Node 测试通过；Playwright 18 项新增管理流程检查通过 |
| Go | 全量 race/integration：123 项顶层测试通过，0 失败，1 项既有人工浏览器夹具按默认跳过。`go vet ./...`、Linux amd64 CGO=0 编译通过 |
| OpenAPI | 2 种钱包与 5 种管理接口的实际隔离 HTTP 响应通过契约校验；补齐账单详情 `serverTime` 和可空字段 |
| Core | 29 项 Maven 测试通过；1.0.1 JAR 构建通过 |
| XConomy | 本机 MariaDB 下 10 项测试通过，0 跳过；构建/指定原始 JAR 摘要核验和 `.2` 补丁组装通过 |

## 关键行为证据

- 游戏原生 Vault 支出、含税 `/pay`、App 转账都从原始经济账本读取。重复 App 请求不多出流水，失败转账没有成功记录；双方金额、前后余额和当日汇总一致。覆盖相同时间戳的序号倒序、查询快照、新写入不混入后续分页、按玩家/来源筛选、他人明细隔离。
- Go 的个人流水参数强制会话 UUID，拒绝跨用户游标和自报玩家身份。管理员按 `playerRef` 查账仍重新鉴权，普通账号直接请求所有审计路由均被拒绝；管理员看到个人隐藏的退款订单和商品，审计详情没有交易操作按钮。
- 公告永久删除拒绝旧版本，重复同键删除只产生一次审计；正文、关联通知及历史幂等响应中的正文均不再存在，旧创建请求只能得到删除回执。
- 案件新申请和状态更新与队列同事务提交；重复请求只入队一次，失败的状态变更不发通知；领取有互斥租约，发送失败按退避重试，成功后不再次领取。
- SMTP 密码加密、错误密钥/篡改拒绝、配置版本/重试、权限隔离和密码不回显通过。使用临时证书和本机 SMTP 服务实际验证 TLS 与 STARTTLS、认证和 UTF-8 MIME；DATA 已确认而 QUIT 连接丢失时仍记成功，未提供 STARTTLS 的服务不会收到凭据。
- 浏览器检查包括全服与玩家审计、订单详情、下架商品、原操作审计、公告删除取消/确认、SMTP 配置及测试状态、保存并发布、防误触退出和 390px 商品编辑布局。接口使用隔离契约夹具；退出确认在测试中固定选择保留，未操作生产业务。

## 复核命令与证据

Android 使用项目 Wrapper：`:ui-lab:testDebugUnitTest :ui-lab:assembleDebug :ui-lab:assembleDebugAndroidTest :ui-lab:lintDebug`。独立 QA APK 使用 `releaseV204=true`，仅访问回环 MockWebServer。主题测试等待实际页面可见；首次固定延时检查有一次可访问性树尚未更新，改为状态等待后通过，未据此修改产品逻辑。

Go 使用 `go test -race -tags integration ./...`，测试 DSN 只指向回环独立 MariaDB，每项创建并清理自身随机 schema。Maven 使用 Java 21 与已有缓存依赖，经济测试显式设置回环 `DEUTERIUM_FUNDS_TEST_URL`。SMTP 单测只连接自己创建的本机监听端口。

[汇总](artifacts/v204/validation.json)、[模拟器结果](artifacts/v204/android-instrumentation.txt)、[浏览器 18 项](artifacts/v204/browser-checks.json)、[交付摘要](artifacts/v204/manifest.json)。截图：[钱包](artifacts/v204/v204-wallet-light.png)、[浅色删除](artifacts/v204/v204-delete-light.png)、[深色删除](artifacts/v204/v204-delete-dark.png)、[管理审计](artifacts/v204/v204-audit-light.png)、[手机商品编辑](artifacts/v204/v204-product-mobile.png)。

## 未执行范围

真实手机手感、真实游戏服务器换包/重启、真实 SMTP 收件、生产公告删除、线上更新清单和账户发布提醒均未执行。仓库可见性核对为公开，与旧私有维护说明不一致，因此代码暂存本地分支；没有自动更改仓库可见性、推送或部署。
