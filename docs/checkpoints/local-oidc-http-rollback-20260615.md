# 本地 Git checkpoint：OIDC HTTP 交付回退点

生成时间：2026-06-15。

## 目标

本 checkpoint 用于把当前本地仓库状态固定为可回滚锚点，不推送 GitHub。

当前本地进度已核对为回到 `DeuteriumAPP-backend-oidc-http-20260606023851` 交付边界：保留 OIDC HTTP 交付成果，移除 2026-06-11/12 之后的 CS 平台后续开发残留。

## Git 管理范围

纳入本地 Git 提交：

- 源码。
- 项目文档。
- Gradle wrapper 等工程必要文件。
- 本 checkpoint 说明。

不纳入本地 Git 提交：

- `delivery/` 交付目录和 zip。
- `dist/` 生成产物。
- APK、JAR、ZIP 等二进制交付物。
- `application.conf`、`local.properties`、`.env`、keystore 等本地配置或密钥文件。

这些路径由 `.gitignore` 排除，避免把可能包含本地配置、密钥或大体积产物的文件提交到源码历史。

## 交付包校验记录

目标交付包：

```text
C:\DeuteriumAPP\delivery\DeuteriumAPP-backend-oidc-http-20260606023851.zip
```

校验结果：

- zip 大小：`27709504`
- zip SHA256：`F97A948737535B0F536BEC684FF011578A941A4ED16F525521BDD1F17B755D49`
- zip 文件数：`77`
- 本地交付目录文件数：`77`
- zip 与本地交付目录逐文件 SHA256 差异数：`0`

## 少删与误删检查记录

少删检查结果：未发现后续 CS 残留。以下关键字和路径扫描无命中：

```text
CsRoutes
CsRepository
CsModels
V6__cs_platform
cs-web
cs.webRoot
cs.initialAdmin
WikiJwt
deuterium_cs_session
/api/v1/cs
/cs-app
/dcs
ProtocolLib
team_deathmatch
DeuteriumAPP-cs
```

误删检查结果：OIDC 相关源码、迁移、测试和契约文档仍存在：

- `backend-api/src/main/kotlin/com/deuterium/backend/oidc/OidcModels.kt`
- `backend-api/src/main/kotlin/com/deuterium/backend/oidc/OidcRoutes.kt`
- `backend-api/src/main/kotlin/com/deuterium/backend/oidc/OidcService.kt`
- `backend-api/src/main/kotlin/com/deuterium/backend/oidc/OidcSigning.kt`
- `backend-api/src/main/kotlin/com/deuterium/backend/repository/OidcRepository.kt`
- `backend-api/src/main/resources/db/migration/V5__oidc_provider.sql`
- `backend-api/src/test/kotlin/com/deuterium/backend/OidcRoutesTest.kt`
- `docs/contracts/oidc-provider-v1.md`
- `docs/contracts/runtime-endpoints.md`
- `docs/contracts/app-backend-api-v1.md`

## 回滚方式

后续需要回到这个本地状态时，优先切换到本地 checkpoint 分支：

```powershell
git switch local/checkpoint-oidc-http-20260615
```

如果只想从该分支恢复部分文件，先查看文件差异，再按文件恢复，避免覆盖无关后续工作。
