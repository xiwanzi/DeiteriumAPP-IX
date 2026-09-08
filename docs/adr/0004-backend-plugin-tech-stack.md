# ADR 0004: 后端与插件技术栈

## Status

Accepted

## Date

2026-04-29

## Context

ADR 0002 已决定 DeuteriumAPP 第一阶段采用以下系统边界：

```text
Android App -> 后端 API -> Minecraft 插件桥 -> Deuterium VIII 服务器
```

插件桥由 Minecraft 插件主动连接后端 WebSocket 长连接。ADR 0003 已决定玩家身份权威来自插件桥从 Deuterium VIII 当前服务器解析出的 UUID。

当前仍需决定后端与 Minecraft 插件的技术栈、数据库选择和第一版交付方式。该决策会影响后续工程脚手架、构建方式、运行环境、依赖选择和部署文档。

Deuterium VIII 当前服务端为 Mohist 1.20.1。Mohist 的插件兼容目标更接近 Spigot/Bukkit 生态，不能默认使用 Paper-only API。

用户期望后端交付形态是一个可复制到生产 Windows 机器的运行目录。该目录应能通过一键启动脚本启动后端，并连接生产环境已有的 MySQL/MariaDB 服务。

## Decision

DeuteriumAPP 第一阶段后端采用 Kotlin/JVM + Ktor + MySQL/MariaDB + Exposed + Flyway。

Minecraft 插件采用 Mohist 1.20.1 兼容路线：Spigot/Bukkit API + Java + Java-WebSocket。

后端第一版交付目标是可复制到生产 Windows 机器的后端运行目录，不采用 Docker Compose 作为默认交付方式。

## Backend Stack

后端技术栈规则如下：

- 使用 Kotlin/JVM 编写后端服务。
- 使用 Ktor 作为后端 HTTP API 与插件桥 WebSocket 服务框架。
- 使用 MySQL/MariaDB 作为第一版后端数据库。
- 使用 Exposed 作为 Kotlin 数据访问层。
- 使用 Flyway 管理数据库迁移。
- 后端运行时目标为 Java 17。

后端负责 App 鉴权、业务校验、限流、审计、请求编排和插件桥 WebSocket 服务端能力。后端不得让 App 直连数据库，也不得把数据库凭据暴露给 App。

## Plugin Stack

Minecraft 插件技术栈规则如下：

- 目标服务端为 Mohist 1.20.1。
- 插件优先使用 Spigot/Bukkit API，避免使用 Paper-only API。
- 插件使用 Java 实现，优先保证 Mohist/Bukkit 兼容性。
- 插件 WebSocket 客户端使用 Java-WebSocket。
- Java-WebSocket 等插件侧依赖在后续插件工程中按需 shade 打包，避免要求服务器额外安装库。
- 插件运行时目标为 Java 17。

插件负责连接后端插件桥、接收受控服务器动作、调用服务器侧能力，并把执行结果或服务器事件回传后端。插件不得提供通用远程命令执行入口。

## Production Delivery Baseline

后端第一版目标交付物是一个可复制到生产 Windows 机器的运行目录。

运行目录应包含：

- 后端可执行包。
- 配置模板或配置说明。
- 数据库迁移入口。
- Windows 一键启动脚本。
- 必要的运行说明。

后端连接生产环境已有 MySQL/MariaDB，默认端口为 `3306`。数据库服务不随第一版后端运行目录内置，也不通过 Docker Compose 编排。

生产数据库连接信息必须通过运行目录内的本地配置文件或环境变量提供。生产数据库用户名、密码和连接串不得硬编码进代码、ADR 或 Git 仓库。

如果生产环境当前使用特定数据库账号和密码，该信息只能进入部署时生成或手工填写的本地生产配置文件，并且该文件不得提交到仓库。

## Alternatives Considered

### Java + Spring Boot 后端

Spring Boot 生态成熟，适合大型后端服务和企业应用。但 DeuteriumAPP 第一阶段服务规模较小，核心需求是账号、钱包转账、聊天互通和插件桥 WebSocket。Spring Boot 对当前目标偏重，且会让后端与 Android 的 Kotlin 语言栈割裂。

因此不选择 Java + Spring Boot 作为第一版后端默认技术栈。

### Node.js + NestJS 后端

NestJS 适合快速构建 API 和实时通信服务，但会引入与 Android、JVM 插件不同的主语言栈。当前项目更希望降低跨语言维护成本。

因此不选择 Node.js + NestJS 作为第一版后端默认技术栈。

### Paper + Kotlin/JVM 插件

Paper 插件开发体验较好，Kotlin 也能与后端保持语言统一。但 Deuterium VIII 当前服务端是 Mohist 1.20.1，Paper-only API 不能保证兼容性。Kotlin 插件还会引入 Kotlin runtime 打包和类加载兼容问题。

因此插件端不选择 Paper-only API，也不选择 Kotlin/JVM 作为第一版默认实现语言。

### Spigot/Bukkit + Kotlin/JVM 插件

Spigot/Bukkit API 可以满足 Mohist 兼容目标，Kotlin/JVM 能统一后端和插件语言。但 Kotlin runtime 对插件 shade、类加载和服务器兼容性提出额外要求。第一版插件桥更看重稳定和可排障。

因此插件端选择 Java，而不是 Kotlin/JVM。

### Docker Compose 交付

Docker Compose 可以同时编排后端和数据库，适合标准化部署。但用户当前期望是可复制到生产 Windows 机器的一键启动后端目录，并连接生产环境已有 MySQL/MariaDB。

因此第一版不采用 Docker Compose 作为默认交付方式。

### SQLite 数据库

SQLite 部署轻量，但后续账号、会话、审计、转账记录和并发请求会更依赖事务与服务端数据库运维能力。当前生产环境已有 MySQL/MariaDB，更适合第一版后端。

因此不选择 SQLite 作为第一版后端数据库。

## Consequences

后续后端工程脚手架应以 Kotlin/JVM、Ktor、Exposed、Flyway 和 MySQL/MariaDB 为默认方向。

后续插件工程脚手架应以 Mohist 1.20.1、Spigot/Bukkit API、Java、Java-WebSocket 和 Java 17 为默认方向。

后续实现计划需要包含 Windows 一键启动目录的构建与交付方式，但不得把生产数据库凭据提交到仓库。

后续如果需要支持 Linux 部署、Docker Compose、开机自启、日志轮转、监控、备份或多实例部署，应通过新的 ADR 或实现计划推进。

## Non-Goals

本 ADR 不决定以下内容：

- 具体 Gradle 工程结构。
- API 路径、请求字段或响应字段。
- 数据库表结构和迁移脚本内容。
- 插件命令、权限节点或事件监听细节。
- WebSocket 具体消息 schema、重试细节或序列化格式。
- 生产机器备份、监控、日志轮转和开机自启方案。
- Linux、Docker Compose、Kubernetes 或多实例部署方案。
