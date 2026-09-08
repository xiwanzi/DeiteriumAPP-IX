# ADR 0001: Android UI 技术栈

## Status

Accepted

## Date

2026-04-29

## Context

DeuteriumAPP 是面向 Minecraft 服务器 Deuterium VIII 玩家的一款原生 Android App。当前仓库仍处于文档先行阶段，还没有 Android 工程、历史 UI 代码或需要兼容的旧页面。

项目第一阶段会围绕账号、钱包转账和聊天互通展开。Android 端需要符合原生 Android 用户体验，并为后续页面实现提供清晰、稳定的 UI 技术栈默认选择。

Android 官方文档将 Jetpack Compose 描述为 Android 推荐的现代原生 UI 工具，并提供 Material 3 in Compose 作为组件、主题和 Material You 风格的实现路线。对于当前从零开始的项目，Compose-first 可以减少旧 View 体系的模板代码，也更适合后续以声明式 UI 方式表达账号表单、钱包状态和聊天列表。

本 ADR 只决策 Android UI 技术栈，不决策 Android 工程骨架、导航、状态管理、网络、依赖注入、本地数据库、最低 Android 版本、target SDK 或发布渠道。

## Decision

DeuteriumAPP 的 Android UI 层采用 Kotlin + Jetpack Compose + Material 3。

具体规则如下：

- 新界面默认使用 Jetpack Compose 实现。
- Android UI 层默认使用 Kotlin。
- 默认使用 Material 3 组件与主题系统，作为 App 的基础视觉和交互规范。
- 新页面不默认使用 XML Views。
- 只有在特定平台能力、第三方控件限制或后续 ADR/plan 明确需要时，才允许局部使用 View interop。
- 如果使用 View interop，对应实现计划必须说明原因、边界和退出条件。

## Alternatives Considered

### XML Views 作为主 UI 栈

XML Views 是成熟的 Android UI 方案，生态稳定，也适合维护已有 View 体系项目。但 DeuteriumAPP 当前没有历史 Android 代码，不存在迁移成本。若从零开始选择 XML Views，会引入更多模板代码和命令式 UI 更新方式，不符合本项目希望保持轻量、清晰和便于后续迭代的方向。

因此不选择 XML Views 作为主 UI 栈。

### Compose 与 XML Views 并列作为一等主方案

混合方案可以提高兼容性，但也会增加 UI 规范、测试方式、组件复用和代码审查复杂度。当前项目规模较小，且没有旧页面负担，没有必要在第一阶段把两套 UI 体系并列为默认方案。

因此不选择混合作为默认主方案，只保留局部 View interop 作为例外机制。

## Consequences

后续 Android 端 PRD、plan 和 issue 可以默认假设新页面使用 Compose 和 Material 3。

后续创建 Android 工程时，应选择支持 Jetpack Compose 的原生 Android 项目配置，并以 Compose 页面、Compose 主题和 Compose UI 测试作为默认方向。

后续如果需要决定 Navigation、状态管理、架构分层、最低 Android 版本、target SDK、模块结构或依赖选择，应分别通过新的 ADR 或实现计划推进，不应把这些决定回填到本 ADR。

Deuterium VIII 的品牌视觉可以在 Material 3 的主题系统上逐步建立，但本 ADR 不定义具体颜色、字体、图标、页面布局或聊天/钱包/账号页面设计。

## References

- [Jetpack Compose](https://developer.android.com/compose)
- [Material Design 3 in Compose](https://developer.android.com/develop/ui/compose/designsystems/material3)
- [Interoperability APIs](https://developer.android.com/develop/ui/compose/migrate/interoperability-apis)
