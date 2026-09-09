# Deuterium 2.0.6 部署记录

2026-09-09 21:46:10（UTC+8），**App 2.0.6（20600）、Web 2.0.6 与配套 Go 已发布上线**。

- [下载 App 2.0.6](https://47.103.99.34/downloads/Deuterium-2.0.6-20600-test.apk)，包名 `com.deuterium.app.uilab`，Android 8.0+。
- App/Web 经 [PR #9](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/9) 合并，源码 `4d9a001ebd0e`；构建提交 `1f8ada46265a` 与该合并源码树一致。
- Go 经 [PR #10](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/10) 补充未设商店头像的订单兼容修复，最终源码 `515c87fd4936`；Linux 二进制为该提交，`vcs.modified=false`。此补丁没有改变 App/Web 源码树。
- 维护工作区 `C:/DeiteriumAPP-IX/.worktrees/saki-admin-experience`，发布记录分支 `release/v206-live`。
- [小祥设置](https://47.103.99.34/admin?section=ai)、[账号与权限](https://47.103.99.34/admin?section=accounts)、[商店管理](https://47.103.99.34/merchant)。需管理员权限。

## 已上线内容

大额账单完整显示，根据文字宽度切换收支布局，并适配放大字体。网页按钮组、页头操作与查找表单统一间距。

后台支持小祥套餐名称、价格、额度、时段、有效期、提示词、参数与知识条目；账号可搜索分页、授予/撤销管理权限、封禁与解封。封禁后旧会话失效，自身和最后管理员受到保护。

Saki 购买使用已有信用点托管与结算，成功自动开通并留在套餐页，进入商城订单；显示 Saki AI 和小祥头像，联系卖家返回 AI 聊天，无领取或主动退款。**本轮未开启套餐售卖；当前购买开关为关闭，待管理员配置后主动开启。**

创建/编辑商店可上传头像、拖动和缩放裁切，预览实际头像尺寸。旧订单读取当前商店头像，未设置或图片已解绑时使用默认头像，不阻断订单。

内部执行步骤保留在审计，普通官方订单到货/退款为站内提醒，领取不重复提醒；客户文案清除执行证明与受益人绑定等术语。App 成功转账产生收款人通知，游戏 /pay 从已提交流水补取并去重。

## 发布验证

候选功能的 138 项 Go 顶层竞态/集成测试、60 项 Web 单测、126 项 Android 单测和原生/浏览器流程见 [候选验收](../qa/saki-admin-2026-09-09.md)，不重复计为本次新增测试。

本次发布重新通过：Android 20600 构建、126 项单测和 Lint（0 错误、27 警告）；Web 60 项单测与生产构建；系统安装器 20500 → 20600 升级、打开及再次打开；20600 原生大额账单浅深色/字体缩放、购买取消/确认/动画/订单/联系流程。兼容修复另通过 4 项相关竞态集成测试，含无头像、有效头像和解绑头像的订单读取。

实服检查发现旧订单的空头像原先被当成资产权限错误，已在 App/Web 分发前恢复旧 Go，合并并部署兼容补丁后重新验证通过；没有恢复旧数据库。最终：

- 公网 HTML、JS/CSS 与构建逐字节一致，网页配置 2.0.6，HTML `no-cache`。
- 公网完整 APK 的大小、SHA-256 与签名均已核对；20004、20101、20200、20300、20401、20500、20501 可更新到 20600；20600/20601 不重复更新或降级。
- 新后台可读取 70 个账号并分页；小祥设置、头像裁切预览、本人订单列表及详情实际浏览器可用，0 写入请求、0 运行错误、0 API 错误。
- 实际账单随后读取余额正常；通知返回系统提示策略。AI 上游在原参数和新增 temperature 参数下均返回完整成功结果；未修改用户聊天历史或私有配置。
- Go service active/ready，实际进程指向新程序；Login/Amiya/Odyssey 在线，MEK 保持离线。9 笔订单和 4 个委托保留，Core/Mail/Sync/XConomy 未更换或重启。
- 临时验收会话已撤销并验证 401；本机临时凭据已移除。

没有为了验收创建真实付款、开通付费套餐、修改玩家权限或商店头像。模拟器与隔离经济验证不替代真机手感或完整游戏客户端领取。

## 文件与迁移

| 文件 | 字节 | SHA-256 |
| --- | ---: | --- |
| Deuterium-2.0.6-20600-test.apk | 23368491 | `3b45ddd407aab468b346367604d5c62bbda996543bbf3b4a6a8db7cae2075645` |
| deuterium-linux-amd64 | 12816544 | `7d8a8c0336a2222c78dc3b9672110617eac8249498ff2916bfd73e586744d10c` |
| web-2.0.6.zip | 33677694 | `aff795fd2097d3197c5e14c7507357cf67254c8e8f252f8d182c7e01cd0e135d` |

APK 签名 SHA-256：`04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`，与现行测试版相同。本机交付目录 `C:/DeuteriumAPP/delivery/Deuterium-2.0.6`。

Go：`/opt/deuterium/releases/release-2.0.6-515c87fd4936/deuterium`。Web：`/var/www/deuterium/releases/web-2.0.6-4d9a001ebd0e/dist`。

新增 `021_saki_admin.sql` 已执行，共 17 个迁移记录。备份在 `/var/backups/deuterium/release-2.0.6-20260909-4d9a001ebd0e`，包含一致性 SQL gzip、配置、环境、原更新清单、systemd/Nginx 配置及旧程序/网页指向。SQL gzip 已验证完整性，232868 字节，SHA-256 `4ecaf2c0645e03c959c61efad56050edf206784a0b9676e2fe222fe7f1050f5c`；私有备份仅留在服务器。

配置和环境文件摘要前后一致。新的更新清单已原子写入并由 Go 重启载入；旧清单和旧 APK 保留。撤回分发时可恢复原清单，已安装 20600 不会自动降级。已有新 Saki 订单后必须保留数字订单识别、退款拒绝和原操作恢复能力，不直接退回旧后端，也不恢复旧 SQL 覆盖新增业务。

证据：[构建](artifacts/v206/build-manifest.json)、[备份](artifacts/v206/prepared.json)、[安装](artifacts/v206/install-proof.json)、[发布](artifacts/v206/published.json)、[公网验证](artifacts/v206/public-verification.json)、[只读后台](artifacts/v206/backend-readonly.json)、[真实浏览器](artifacts/v206/browser-live.json)、[AI 上游](artifacts/v206/ai-upstream.json)、[最终状态](artifacts/v206/final-state.json)。
