# Public Release Checklist / 公开发布检查清单

Last updated / 更新时间：2026-05-02

## 中文

本文面向项目维护者。目标是把 DeuteriumAPP 作为一个开源产品发布，同时避免把生产配置、密钥、构建产物或私有交付资料带入公开仓库。

### 发布原则

- 公开仓库应包含源码、示例配置、接口合同、架构文档和开发文档。
- 公开仓库不应包含生产数据库密码、pepper、plugin bridge token、私有域名、APK、JAR、ZIP、JRE、本地 SDK 或真实 `application.conf`。
- 产品 README 应面向使用者和贡献者，而不是内部交付人员。
- 私有工作区可以保留生产交付文档，但公开导出必须排除或清洗这些内容。
- 当前公开许可证为 Apache License 2.0，公开仓库必须包含根目录 `LICENSE` 文件。

### 生成公开副本

```powershell
cd C:\DeuteriumAPP
.\scripts\export-public-clean.ps1
```

默认输出：

```text
C:\DeuteriumAPP-public
C:\Users\<you>\DeuteriumAPP-public.zip
```

脚本会：

- 复制源码、非敏感文档、Gradle Wrapper 和构建脚本。
- 排除 `.git`、`.tools`、`.gradle`、`build`、`delivery`、`dist`、APK、ZIP、非 wrapper JAR、本地 SDK 配置和插件本地资源。
- 把私有域名替换为 `example.com`。
- 把已填充的数据库密码、pepper 和 plugin token 替换为 `CHANGE_ME`。
- 扫描当前已知生产密钥和私有域名。

### 人工审核

发布前人工检查：

```powershell
Get-ChildItem C:\DeuteriumAPP-public -Recurse -File |
  Select-String -Pattern "password=|token=|token:|secret|pepper|keystore" -CaseSensitive:$false
```

预期结果：只出现 `CHANGE_ME` 这类占位示例。

还需要人工确认：

- README 是否像产品介绍，而不是内部备忘录。
- 快速开始是否能让外部开发者理解如何构建。
- 技术架构是否解释清楚三端关系。
- 没有真实服务器地址、真实账号、真实 token 或交付包。
- 已包含 Apache License 2.0 的根目录 `LICENSE` 文件。

### 推送公开仓库

仅在 `C:\DeuteriumAPP-public` 中执行 Git 发布命令：

```powershell
cd C:\DeuteriumAPP-public
git init
git add .
git commit -m "Initial public DeuteriumAPP source release"
git branch -M main
git remote add origin https://github.com/<owner>/<repo>.git
git push -u origin main
```

不要从私有工作区 `C:\DeuteriumAPP` 直接 push 到公开仓库。

## English

This checklist is for maintainers. The goal is to publish DeuteriumAPP as an open-source product while keeping production config, secrets, build artifacts, and private delivery material out of the public repository.

### Release Principles

- The public repository should contain source code, example config, interface contracts, architecture docs, and development docs.
- The public repository should not contain production database passwords, peppers, plugin bridge tokens, private domains, APKs, JARs, ZIPs, JRE files, local SDK tools, or filled `application.conf` files.
- The product README should speak to users and contributors, not internal delivery operators.
- The private workspace may keep production delivery docs, but public export must exclude or sanitize them.
- The current public license is Apache License 2.0. The public repository must include the root `LICENSE` file.

### Generate Public Copy

```powershell
cd C:\DeuteriumAPP
.\scripts\export-public-clean.ps1
```

Default output:

```text
C:\DeuteriumAPP-public
C:\Users\<you>\DeuteriumAPP-public.zip
```

The script:

- copies source code, non-sensitive docs, Gradle Wrapper, and build scripts;
- excludes `.git`, `.tools`, `.gradle`, `build`, `delivery`, `dist`, APKs, ZIPs, non-wrapper JARs, local SDK config, and private plugin resources;
- replaces private domains with `example.com`;
- replaces filled database passwords, peppers, and plugin tokens with `CHANGE_ME`;
- scans for currently known production secrets and private domains.

### Manual Review

Before publishing, run:

```powershell
Get-ChildItem C:\DeuteriumAPP-public -Recurse -File |
  Select-String -Pattern "password=|token=|token:|secret|pepper|keystore" -CaseSensitive:$false
```

Expected result: placeholder examples such as `CHANGE_ME` only.

Also confirm:

- README reads like a product introduction, not an internal memo.
- Getting started docs are understandable for external developers.
- Technical architecture explains the three-runtime-module relationship clearly.
- No real server address, account, token, or delivery artifact is present.
- The root `LICENSE` file for Apache License 2.0 is present.

### Push Public Repository

Run Git publication commands only inside `C:\DeuteriumAPP-public`:

```powershell
cd C:\DeuteriumAPP-public
git init
git add .
git commit -m "Initial public DeuteriumAPP source release"
git branch -M main
git remote add origin https://github.com/<owner>/<repo>.git
git push -u origin main
```

Do not push directly from the private workspace `C:\DeuteriumAPP`.
