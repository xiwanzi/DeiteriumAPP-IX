# Deuterium 2.0.3 部署记录

2026-09-09（UTC+8）。**App 2.0.3 (20300) 和配套 Go 后端已发布**；01:08 在线更新清单生效，并向 xiwanzi 账号发送唯一更新提醒。网站和游戏插件未更换。

## 安装与源码

- [APK 下载](https://47.103.99.34/downloads/Deuterium-2.0.3-20300-test.apk)，23,319,247 bytes。
- APK SHA-256：`9ccfe155cb49e74964f7ff7f44fb39cad9585cb3140fab14b703825063e0ce51`。
- 包名 `com.deuterium.app.uilab`，Android 8.0+；签名沿用现有测试证书，SHA-256 `04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`，模拟器已覆盖安装。
- Android/Go 代码提交 `15bb45dc662586474b2307097b811e9463be05bc`，工作区 `release-v203`，私有仓库 [PR #2](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/2)。
- Go 当前目录 `/opt/deuterium/releases/app-2.0.3-15bb45dc6625`，SHA-256 `0f95a84703638f119228312fece80e7d1c6f3f970e5cede7168054de0f039bb7`，12,492,960 bytes；systemd `deuterium.service` 健康就绪。

## 最终行为

- 商城/市场只弹出“是否支付 ×× 信用点？”和取消/确认付款，不显示商品明细或内部报价提示；付款时序和幂等保护沿用 2.0.2。
- 修复新商品加入购物袋后动画读取无效本地资源的闪退，搜索缩略图与关键词使用真实商品数据。
- 已结束委托退出大厅，订单/委托/下架或售罄市场记录可从本人列表删除；按账号隐藏，不删除账本、业务源记录或对方记录。重新活跃或资金未知的业务仍显示。
- 联系人和信息入口显示未读红点，实际打开私聊并完成已读回执后清除。
- 公告/AI 正文渲染 Markdown；公共消息可直接提交，不等待历史同步，未知结果继续用原键重试。

## 验证

Android 119 项单元测试通过，Lint 0 errors / 29 warnings；加购真实回调、搜索、历史删除、未读清除、实际公告/AI 浅深色渲染、公共消息同键重试及付款四时序模拟器验证通过。Go 122 项 race/integration 通过，0 失败，仅既有人工浏览器夹具跳过；vet 和两类新响应契约验证通过。详情见 [本次 QA](../qa/app-v203-2026-09-09.md)。

公网完整下载 APK 的字节数和摘要与本机构建一致；20005、20100、20101、20200 均得到 20300，20300 不提示自身更新。发布提醒 `app.release:20300` 仅 xiwanzi 一条，未广播聊天或创建生产付款。

部署后通过临时、随后撤销的 QA 会话只读验证生产接口：本人 6 条订单、3 条历史委托和 1 条市场记录均正确返回删除能力；公开委托大厅为 0 条，商城有 1 件真实商品；联系人摘要含未读数，公共消息读取正常。未删除这些生产记录。Login/Amiya/Odyssey 在线，MEK 仍停止/禁领。账号实际收到系统弹窗及用户手机手感不冒充已实测。

源码、APK、Go 二进制对既有部署/供应商/节点/会话凭据做精确扫描，无命中；私钥、SQL 备份和私有配置未进入源码或公开下载目录。

## 迁移、备份与回退

14 个既有迁移文件的生产摘要与源码逐一一致。新增 `019_record_visibility.sql` 仅创建 `personal_record_visibility_v203`，摘要 `82a9a915b14078e91d8f6a09b33c002d7a16a4fd5f002575c81a5d4123ae3c7a`；不改旧业务表、账本或资金记录。App 发布前新偏好表为 0 行。

备份目录 `/var/backups/deuterium/app-2.0.3-20260909-20300`，包含原配置、环境、更新清单、旧 current 目标及一致性后端 SQL 压缩备份。SQL gzip 148,931 bytes，SHA-256 `464cba0cfe1a4881ffd296e13b0668bc6104156018cd8914968586bbc5c53243`，完整性检查通过。部署前后均保留 6 条订单、3 条委托。

旧程序目录 `/opt/deuterium/releases/cache-2.0.1-1b5d9d1c63f8` 保留。新增表对旧程序兼容；如回退，先恢复旧程序链接/清单并检查健康，不能恢复旧 SQL 覆盖新业务。旧程序不提供新的隐藏/公共发送入口，回退时须同步处理 App 通道和用户体验；图片 GC 配置沿用 2.0.1，本次未改其前缀或生命周期。

[发布元信息与公网验证](artifacts/v203/public-verification-20300.json)、[后端部署证据](artifacts/v203/backend-deployment.json)、[生产只读验证](artifacts/v203/live-readonly-verification.json)。本机完整交付目录为 `C:/DeuteriumAPP/delivery/Deuterium-2.0.3/`。
