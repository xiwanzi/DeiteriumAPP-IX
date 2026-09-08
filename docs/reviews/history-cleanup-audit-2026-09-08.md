# 历史内容清理审计

日期：2026-09-08。范围：`C:/DeuteriumAPP`。本轮只检查并生成清单，没有删除、迁移、提交或执行 Git 回收。

## 结论

目录文件逻辑大小约 **27.17 GiB**。优先处理重复交付展开目录、已过期 Git 临时对象、旧构建输出与安装下载包；历史源码与分支先保留。不要把根目录未跟踪内容整体视为垃圾。

统计不跟随符号链接/目录联接；5 个深层缓存文件未能读取元数据，且模拟器/开发任务仍在运行，因此为近似快照。模拟器镜像可能为稀疏文件，逻辑大小不等于实际可释放磁盘空间。下列父目录与子目录数据不能重复相加。

| 顶层目录 | 逻辑大小 |
| --- | ---: |
| `.tools` | 14.76 GiB |
| `.git` | 5.38 GiB |
| `android-app` | 3.72 GiB |
| `delivery` | 1.84 GiB |
| `.worktrees` | 0.58 GiB |
| `deliverables` | 0.35 GiB |
| `dist` | 0.35 GiB |

## 第一批：明确清理候选

以下是清理建议，不是无条件立即执行的删除列表。

| 内容 | 规模 | 建议与前提 |
| --- | ---: | --- |
| 14 组有完整 ZIP 的重复展开交付目录 | 合计约 1.07 GiB | 文件清单、大小、CRC 一致，ZIP testzip 通过；保留 ZIP，确认目录没有被服务、脚本或审查任务引用后，清除展开副本。当前生产/回滚目录须先确认，不因重复就直接删 |
| `.git/objects/*/tmp_obj_*` 共 7 个 | 约 1.56 GiB | `git count-objects -v` 明确报告 garbage；时间为 08-29 和 09-07。等待写入任务结束、确认无 Git 操作后，针对临时对象做维护，不手删正常对象，不重写历史 |
| `android-app/app/build`、`android-app/build`、`backend-api/build` | 合计约 0.44 GiB | 根目录旧构建输出。需要的 APK/JAR/报告先保留交付副本，确认不被构建或服务使用后可再生成 |
| `.tools/downloads` | 约 0.39 GiB | 已展开 SDK/Gradle 的安装下载包，核实安装完整后可删；不删除 SDK 或 Gradle 安装目录 |
| `web-aave-glass-prototype/node_modules` 与该原型的 `dist` | 约 0.07 GiB | 旧 Web 视觉试验的依赖/构建物；保留源码、package.json、lockfile、素材，恢复时重新安装构建 |

上述逻辑大小合计约 **3.5 GiB**，包含需要停止任务或确认引用的项目，不承诺现在即可全部回收。

## 第二批：空间大，但需先核实用途

| 内容 | 规模 | 处理方式 |
| --- | ---: | --- |
| `android-app/.gradle-home-test` | 2.63 GiB | 旧测试 Gradle home，需核实是否还有任务使用；删除意味着重新下载/生成依赖与转换缓存 |
| `android-app/gradle-user-home` | 0.30 GiB | 同上；检查离线构建需求和进程引用后再清 |
| `.tools/avd/DeuteriumUiLab35/snapshots/default_boot` 和 `DeuteriumUiVisual35/snapshots/default_boot` | 仅两个 ram.img 合计约 5.00 GiB | 可再生成的快速启动快照，但当前模拟器正在运行；停机后确认不需保留快照状态再处理，优先只清快照而非整台虚拟设备 |
| 两台 AVD 整体 | 共 7.55 GiB，含上述快照 | 保留当前测试设备。旧设备包含账号/订单体验状态、安装包和测试现场，不直接整个删 |
| SDK 两套 system-images | 共 5.15 GiB | `aosp_atd` 对应 UiLab35，`google_apis` 对应 UiVisual35；仅在对应设备退役后用 SDK 管理方式移除闲置镜像 |
| `android-app/local-maven` | 0.47 GiB | 离线依赖仓库，不是普通 build 输出。新版 settings 仍声明 local-maven 路径，但其工作区当前无该目录；需核实恢复/镜像用途后再决定，不按重复缓存直接删 |

目前有 Java 和模拟器进程；未确认其缓存持有关系，不在此审计中停止进程。

## 重复交付的核对结果

15 组 ZIP 与同名目录均通过 ZIP CRC 完整性检查。其中以下 14 组展开目录与 ZIP 的文件集合、大小和 CRC 一致：

- `delivery/DeuteriumAPP-backend-oidc-20260606021533`
- `delivery/DeuteriumAPP-backend-oidc-http-20260606023851`
- `delivery/DeuteriumAPP-backend-only-1.0.3-20260501`
- `delivery/DeuteriumAPP-backend-only-1.0.3-perf-20260503`
- `delivery/DeuteriumAPP-backend-only-1.0.3-perf2-20260503`
- `delivery/DeuteriumAPP-backend-only-1.0.3-wsfix-20260501`
- `delivery/DeuteriumAPP-backend-only-1.0.4-ai-sse-fix1-20260616`
- `delivery/DeuteriumAPP-pony-ultra-20260617-033059`
- `delivery/DeuteriumAPP-production-20260501`
- `delivery/DeuteriumAPP-production`
- `deliverables/DeuteriumAPP 后端 1.0.4 稳定版 20260715`
- `deliverables/DeuteriumAPP 后端 1.0.4 稳定版 20260715 配置完成版`
- `dist/deuterium-full-runtime-0.1.0-20260430`
- `dist/deuterium-full-runtime-0.1.0-20260501`

**例外：** `delivery/DeuteriumAPP-backend-only-1.0.4-ai-sse-20260616` 与其 ZIP 有 4 个不同文件：README、`backend/migrate-db.bat`、`backend/reset-database.bat`、`backend/start-backend.bat`。没有额外文件，但不能将其作为完全重复目录清除，必须先保存差异。未执行这些脚本。

建议以后采用一个交付入口，保留当前验收版、必要回滚版及对应说明/校验值，其他 ZIP 归档到项目之外。只把三个交付目录挪到项目内同一个文件夹不会释放空间。

`delivery/_prod_ref`（约 0.096 GiB）、`delivery/_plugin_compare` 是历史比对材料，可在审查结论和所需证据保存后归档/清理，本轮未将它们认定为已验证重复。

## 源码、工作区与文档

- **新版工作区保留。** `ui-motion-lab` 有未提交新版源码和刚整理的文档，当前还在迭代。07 交付目录已出现，但检查时只有截图，没有完成 APK/说明，不能据目录存在判定交付完成。06 及 07 当前材料保留。
- **旧体验版 01–05 建议归档。** 合计约 0.16 GiB，价值主要是设计回溯。可保留各版说明、关键图/录屏及必要 APK，不必作为默认开发入口；不是最高优先级空间项。
- **`minecraft-plugin-chatfix` 保留并归档来源。** 与新版工作区插件按统一换行/BOM 比较，仍有 6 个文件不同，包括 `BridgeClient.java`、`DeuteriumBridgePlugin.java`。不是纯缓存，也不是可以直接删的重复源码。
- **后端硬化工作区保留。** Git 工作树干净，但 `14ef1b8` 是相对新版分支独有提交，清理前需明确合并/归档策略。
- **旧优化和 README 工作区可退役候选。** `android-client-optimization`、`readme-consistency` 的普通 Git status 均干净，相对新版分支没有独有提交；但可能有忽略的产物和本机配置。检查这些内容与活跃任务后，用 `git worktree remove` 正常退役并保留分支，不手动删整个 `.worktrees`。
- **根目录三端路径不能整删。** 根 `android-app` 大量为构建缓存/本地依赖，根后端也有残留运行物；根插件仍有 `src/main/local-resources/config.yml`。应逐项分出可再生成输出与本机配置，不能把所有未跟踪文件都当旧源码重复。
- **文档归档，不按年份删除。** 01–05 体验计划可作历史索引；旧 PRD/ADR、契约、审查与回滚说明保留适用范围标记。它们包含兼容和资金规则，不因 UI 改版失效。当前入口已经在上一轮更新。
- **旧 Web 原型源码可归档。** 这是 liquid-glass-react 本地视觉试验，不等于后续商家/管理网页端；不建议继续作为当前产品入口。

## 必须保留或单独处理

- `.tools/android-sdk`、当前模拟器、Gradle 工具链和新版 build：本轮仍在使用，不能按历史工具整个清除。
- `.git` 正常对象、分支和 reflog：除临时 garbage 外约 3.8 GiB 仍需专门分析。两个大 blob 未在本次 all/reflog 路径枚举中匹配到，仍不能据此直接删除；本轮不执行 prune、gc 或历史重写。
- `deliverables/application.conf`、`oidc-signing-key.json` 及插件本机配置：可能是运行/身份凭据，保留受控私有备份，不提交或当旧文件删。本轮未读取密钥内容。
- 根目录缺少 `.gitignore`，这是后续整理应补的入口问题：明确排除工具、工作区、构建物和私有配置，但不能用忽略规则掩盖需要保存的新版源码。
- NapCat ZIP 只有约 1 MiB，先移出开发入口归档即可，空间收益很小。

## 建议执行顺序

1. 保存当前未提交工作与必要配置，等待正在进行的新版测试结束。
2. 先处理已验证重复的展开目录、旧安装包和明确的旧 build；生产/回滚引用逐个排除。
3. 单独处理 7 个 Git 临时 garbage，之后再复查对象体积，不混做历史重写。
4. 再决定是否清 Gradle home、快照、闲置模拟器和 SDK 镜像。
5. 最后退役无独有提交的旧工作区、整理交付入口与历史文档索引。

验证方式：目录体积扫描、Git 状态/提交关系、对象统计、源码规范化比较、ZIP 完整性和展开文件 CRC 比较。本轮没有运行业务构建测试，也没有执行删除。
