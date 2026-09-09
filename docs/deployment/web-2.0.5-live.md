# Web 2.0.5 部署记录

2026-09-09 14:59:54（UTC+8），**Web 2.0.5 与配套 Go 已上线**。用户在本任务明确确认创建/合并 PR 和部署；[PR #7](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/7) 已合入 main，提交 `0386f1bd56641f08f0b73faf819632f3cad9efda`。

- [商店管理](https://47.103.99.34/merchant)、[官方管理](https://47.103.99.34/admin)。需要有权限的 Deuterium ID 登录。
- 构建提交 `edfae91758b9144811b494bfb7a733d7ef069c7b` 与合并提交源码树完全一致；Linux 程序嵌入构建提交，`vcs.modified=false`。
- 本机工作区 `C:/DeiteriumAPP-IX/.worktrees/desktop-admin`，部署文档分支 `release/web205`；交付目录 `C:/DeuteriumAPP/delivery/Web-2.0.5-admin-edfae91/`。

| 对象 | 已启用目录 / 文件 | SHA-256 |
| --- | --- | --- |
| Go | `/opt/deuterium/releases/web-admin-2.0.5-edfae91758b9/deuterium` | `32041fecabf67c9f31d38fccb5d682d375eac967041a03d5418604f90f54ce5b` |
| Web | `/var/www/deuterium/releases/web-2.0.5-edfae91758b9/dist` | ZIP `8054fe7ae8209b8ca0991bbc60cd18276576436a8e2cfa2a8dab0768d811d5a7` |

## 已发布行为

管理端改为 PC 侧栏导航、带封面的商品表格、筛选与分页。商品编辑采用宽屏双栏，支持草稿保存后继续编辑、固定保存按钮及未保存保护。手机预览对应当前 Android 的详情、推荐海报、双列卡片和购物袋，显示原图及实际居中裁切的像素范围；图片说明和封面顺序可编辑。

商品新增游戏邮件标题、正文及预览。草稿须发布后才影响新订单；文案在成交时固定到持久投递计划，重试不读取后续修改。多商品仍合为一封分节邮件，正文超出 Core 字节上限时在报价阶段提示分开购买。

公共聊天按玩家身份补取头像，消息和最近参与者使用同一资料；图片加载失败回退姓名首字。商品详情、图库、购物袋、交付、报价和付款界面已优化，保留服务端金额、报价有效期和未知结果恢复；修复付款恢复的浏览器 origin 被交付地点变量遮蔽的问题。

## 验证

开发候选的 **58 项 Web 单测、128 项 Go 竞态/数据库集成测试、40 项隔离浏览器检查**和 Linux 构建已通过，详见 [开发验收](../qa/desktop-admin-2026-09-09.md)。部署直接使用相同源码树的已验收文件，没有重复计算测试次数。

本次上线检查：

- 公网 `/` 和 `/merchant` 返回的 HTML 与构建逐字节一致；实际 JS/CSS 的大小和 SHA-256 一致，`web-config.json` 返回 2.0.5，HTML 使用 `no-cache`。
- 当前 systemd 服务 active，ready 正常，实际运行程序路径指向新 Go。
- 真实账号只读检查商店/商品/品牌/分类/模板、公告、审计、平台介入与 SMTP 页面数据；账单读取后余额查询正常。
- 浏览器读取真实商品，打开编辑、App/裁切预览和游戏邮件预览；公共聊天 32 条消息加载头像，0 运行错误、0 API 错误、0 写入请求。
- 线上现有 1 件官方商品为 **UNLISTED**，公共商城为空，与接口状态一致。没有为测试擅自上架；购买流程证据来自隔离验收，没有生产付款或游戏邮件投递测试。
- Login / Amiya / Odyssey 三节点在线，MEK 离线且保持原状态。既有 9 笔订单、3 个委托保留。15 分钟临时 Web 验收会话已提前撤销并验证 401，本机临时凭据文件已移除。

手机模拟预览不能代替实际 Android 字体/显示缩放或游戏邮箱渲染。本次不声称完成新的真机或游戏领取验收。

## 备份与回退

备份目录 `/var/backups/deuterium/web-2.0.5-20260909-edfae91758b9` 保存 config/env、App 更新清单、systemd/Nginx 配置与旧程序/网页指向，以及一致性 SQL gzip。SQL 211,250 字节，SHA-256 `996e051d1c51d82fbc54c43eea0bc41eca3b57c418372692e6431686786f0f71`，已验证 gzip 完整性；备份留在服务器私有目录。

16 项已执行迁移摘要与源码完全一致，**无需新迁移**。先上传新目录并核对摘要，备份后切换 Go、检查 ready，再原子切换网页目录。Go 服务重启一次，游戏服务器未重启。原 Web/Go 目录保留。

App 仍为 2.0.5（20500），未更换 APK 或推送新账号提醒；App 更新清单 SHA-256 仍为 `4fa1b329ef6b0b9080bd584ea0be168f89c1d8a2ebbd7803c22fb2de3d29f750`。后台配置和环境文件摘要前后一致。Core / Mail / Sync / XConomy 沿用此前发布，见 [App 2.0.5](deuterium-2.0.5-live.md) 和 [2.0.4 配套基线](deuterium-2.0.4-live.md)。

回退网页时恢复旧静态目录指向即可，保留兼容新邮件字段的 Go。新字段一旦写入商品，不应直接切回会拒绝这些字段的旧 Go；需要后端回退时保留字段兼容能力。不要恢复旧 SQL 覆盖新增业务记录。

证据：[构建](artifacts/web205/build-manifest.json)、[备份](artifacts/web205/backup.json)、[切换](artifacts/web205/activation.json)、[公网摘要](artifacts/web205/public-verification.json)、[只读接口](artifacts/web205/live-readonly.json)、[真实浏览器](artifacts/web205/browser-live.json)、[最终状态与会话撤销](artifacts/web205/final-state.json)。
