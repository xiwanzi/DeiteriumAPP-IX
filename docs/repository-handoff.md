# 源码归档与仓库交接

2026-09-08。用户授权整理现有网站、App、后端、Core 与相关修改，提交到 `xiwanzi/DeiteriumAPP-IX` 私有仓库；已有独立仓库的组件通过 PR 合并回原仓库。此次整理不重新部署、不修改服务器数据、不清理旧工作区。

## 来源和收录范围

| 内容 | 原工作区 / 基准 | 本仓库归属 |
| --- | --- | --- |
| 当前 Android、资源、测试和 Gradle 工程 | ui-motion-lab，`4299ce37d0bf` | android-app；当前模块 ui-lab，旧 app 模块保留但标明历史 |
| 网站、资源、测试、锁文件 | web-player-v1，`2455abb`（发布源码 `4de1ee7`） | web-app |
| Go、SQL 迁移、Core、XConomy 补丁 | test-release-v2，`f07f69e`（Go 发布 `1b5d9d1c63f8`；Core 与 `5ae6706` 相同） | backend-next、deuterium-core、adapters/xconomy |
| XConomy 原版对应源码 | 既有交付的 XConomy 2.26.3 源码包 | third_party/XConomy-2.26.3，保留原 GPL 许可证 |
| 契约、ADR、实施和 QA | 根目录及上述工作区；同路径优先部署集成记录，独有 App/Web 文档保留 | docs 与模块 docs |
| Mail .6.0 基线 + .6.1 | 独立 Mail 原工作区 e1f74e3 及既有未提交源码 | 原 Mail 仓库 PR，不复制业务实现到主仓库 |
| Sync .3 | 既有发布适配源码，原独立仓库保留 .2 修补器 | 原 Sync 仓库 core-integration；主仓库只保留入口 |

每份导入文件的原工作区、提交、路径、原 SHA-256 与整理后规范化 SHA-256 见 [源码来源清单](source-import-manifest.json)。源码路径保持原结构；Go import 路径保持旧名称，以免归档产生无关代码改动。整理后的文档可以与原摘要不同，原来源仍可追溯。

## 独立组件合并

| 组件 | 已合并 PR | 合并提交 |
| --- | --- | --- |
| Mail | [Deuterium-Mail #1](https://github.com/Deuterium-IX/Deuterium-Mail/pull/1) | `164efb13eefa7d53f769bf80fbbf078fe189cb98` |
| YouerModSync | [YouerModSync-Patches #1](https://github.com/Deuterium-IX/YouerModSync-Patches/pull/1) | `1144679a96ec2e04bb3dfe703ee8c8b84363b598` |

两个 PR 均核对提交后按用户授权自动合并，未绕过失败检查。GitHub 当时未返回配置的 PR 检查，构建和测试在独立检出目录完成并记录。配套版本固定在 [components.lock.json](../components.lock.json)。

XConomy 是第三方上游项目，本轮未发现用户所有的独立扩展仓库，项目专用修改保留在主仓库 adapter，原版源码/许可证配套归档；未冒充向第三方上游取得合并权限。

## 文档整理

重新编写 README、CONTEXT、AGENTS、项目状态、当前 PRD、构建指南和部署手册，清除当前入口中同时声称“已部署”和“未实现”的矛盾。原始入口保留在 `docs/archive/pre-consolidation`，历史 01–09 计划和 QA 继续保留日期及适用范围。

将原工作区绝对文件链接转换为仓库相对链接。确实未纳入的旧 Ktor 审查代码、个别截图/本机交付物保留为有说明的历史路径文字，避免给出失效链接；详情见 [链接整理记录](link-migration-report.json)。当前正文的模块和部署入口均可从本仓库导航。

新增 `ops/` 受控示例：版本目录 systemd 单元、同源 API/WebSocket/SSE Nginx 模板、后端环境变量和 AI 占位配置。它们是维护模板，不是线上私有配置副本。服务器主机、路径、备份、权限、配置变量、升级顺序和回退限制见 [部署手册](deployment/README.md)。

## 本次验证

| 对象 | 本次结果 |
| --- | --- |
| Android | 新目录 assembleDebug、113 项单元测试、Lint 通过；0 errors / 27 warnings；APK 包名、版本、最低 SDK 核对为 com.deuterium.app.uilab / 2.0.2 (20200) / 26 |
| Web | npm ci、54 项 Node 测试和 Vite 生产构建通过；Node 24.14.1 |
| Go | go test ./...、go vet ./...、Go 1.27.1 Linux amd64 无 CGO 编译通过；本次未重跑数据库 integration/race |
| Mail | Maven clean verify 与 UI Gradle clean build 通过；合计 280 项测试，无失败/错误/跳过；29 个交付补丁文件匹配既有 .6.1 清单 |
| Core | 29 项测试通过；新目录构建并安装配套 API 成功 |
| Sync | 显式 CoreProject 构建通过，产物摘要与已发布 .3 完全一致；9 项数据库测试因未配置专用隔离库跳过 |
| XConomy | 编译与补丁生成通过，产物摘要与已发布扩展完全一致；8 项数据库测试因未配置专用隔离库跳过 |
| Git / 文档 | 文件来源和规范化摘要、主文档相对链接、git diff --check 校验；原工作区未提交修改保持原样 |
| 敏感信息 | 10 个既有部署/供应商/节点/会话凭据的 UTF-8、UTF-16LE 精确扫描无命中；常见密钥格式扫描候选均为测试夹具或契约占位，不提交凭据原值 |

本次 Android 验证包 SHA-256 为 `fcc81f62f03ceef8e85dee17140cd37f6ac19a43163083e80828b5bafe49617d`，仅用于验证新检出可构建；不替换线上 APK，线上摘要继续以 2.0.2 部署记录为准。构建路径/时间与工具环境可以影响归档字节，不能把源码相同当作 APK 字节相同。

历史已发布 App、Core 和后端测试/实服证据另列原发布文档。本次未新跑真机、游戏客户端领取/切服、MEK 模组兼容或生产资金测试。

## 明确未纳入

- `.tools`、SDK、JDK、Maven/Gradle/npm 缓存、运行日志和本机临时部署脚本；构建工具与依赖按指南重新获取。
- 私钥、签名材料、token、密码、生产 JSON/env、原账号密码哈希导出、玩家数据库和备份。
- APK/JAR/ZIP、旧演示截图与一键工具压缩包；当前发布的必要元信息按明确清单保留在 `docs/deployment/artifacts`。
- 根目录旧 Ktor 后端、旧 Bukkit Bridge、网站视觉原型与其他任务尚未交付的分支，不覆盖当前 Go/Core/Web 代码；这些仍保留原工作区，没有删除。
- 未修改的第三方 MOD/插件不复制为本项目源码。Mail/Sync 的既有私有仓库保持独立；旧 D8 Interface Framework 不混入 IX 邮箱。

## 后续维护

以新主仓库为代码和运维文档入口；新增版本同时更新状态、部署记录、组件锁定和摘要。外部仓库修改通过 PR 合并后，再更新主仓库的固定提交。旧本机工作区只作来源/在途工作保留，避免双向自动覆盖。
