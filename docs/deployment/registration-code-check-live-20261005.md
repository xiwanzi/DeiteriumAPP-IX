# 注册前置校验与即时倒计时上线

2026-10-05 后续修正：用户报告“更新版本无效”，已将新 APK 条目的最低适用版本从 0 修正为 1，保留包体及其他条目并重启读取，见[修复记录](v216-update-manifest-fix-20261005.md)。下方为本轮首次部署过程。

2026-10-05 11:47 开始切换，11:49 完成 App 更新读取修正，最终配套 Go、App **2.0.16（21600）**、社区 Web **2.0.26** 与官网/Help 的共享登录脚本已上线。源码 `4971fd7b61fd`，PR [#71](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/71)，合并 `d2f6453c5cc0133ffbf715b9bec554c90a5075ff`；沿用用户已明确授权的推送、合并、部署范围。

## 生效体验

点击获取注册验证码立即开始 60 秒倒计时，后台查询正版 ID，表单继续可填写。ID 不存在或查询服务故障时显示错误、终止倒计时并恢复按钮，不生成验证码或排队邮件。成功才发送邮箱验证码；成功档案按 ID 缓存 10 分钟，换邮箱复用；换 ID 或缓存过期重新查。最后注册使用验证码记录绑定的校验标志，不重复查外网。

修改字段、关闭或切换页面使旧请求的 UI 结果失效。旧 App 可正常使用新的发码接口；其即时倒计时交互需升级至 21600。迁移前已发出的旧验证码仍在最终提交时校验一次，不放行未校验的旧凭证。独立游戏身份验证、UUID 绑定、白名单、密码与验证码单次消费规则保持。

## 本次证据

- [本机验收](../qa/registration-code-check-20261005.md)：Go 单测/vet/集成、Web 71 项与共享会话 10 项测试、官网/Help/社区隔离浏览器交互、Android 152 项 JVM 测试及模拟器 `authRegistration` 全部通过。
- 四个真实注册发码来源实测：不存在的随机 ID 均返回 400 `MINECRAFT_PLAYER_NOT_FOUND`；验证码与邮件记录均无新增，没有创建账号。
- 为避免向无关邮箱发信，缓存验收使用唯一测试邮箱的临时冷却记录阻止真正发码；真实存在 ID 的查询成功缓存可被不同邮箱/大小写复用，缓存到期值未被第二次请求刷新。临时冷却记录和验收会话均已定向清理，没有发送邮件。
- 官网和 Help 的登录脚本公网 SHA-256 均为 `5308e6868adbb66d938d488dd1f1beb9f5782a4c15eb0da674b77b927b9fe0d5`；社区配置返回 2.0.26；公网 APK 大小和摘要一致。
- App 更新接口：21500 返回 21600，21600 不重复提供 APK 更新；原 16 个发布条目保留。上线检查发现发布清单权限被收紧导致更新接口 503，已恢复备份中的原 0644 权限并重启读取，最终接口正常；配置与环境文件未变。
- 服务 active / ready；Login、Amiya、Odyssey、MEK 全部在线。重启前后均为 **46 个非 deleted 账号、27 笔订单、5 个委托、12 个工单、1 条已有注销记录**，没有恢复旧已清账号。

## 程序与备份

当前 Go `/opt/deuterium/releases/registration-code-4971fd7b61fd`；社区 Web `/var/www/deuterium/releases/web-2.0.26-4971fd7b61fd/dist`；官网 `official-code-4971fd7b61fd`；Help `help-code-4971fd7b61fd`。前版 Go、Web 和两站目录保留。

备份 `/var/backups/deuterium/registration-code-20261005-4971fd7b61fd`，包含停止服务后的单事务数据库、旧程序、配置、发布清单、迁移摘要和验收记录。新增 034 只添加验证码标志及成功缓存表，原 29 个迁移摘要保持。前版 Go 忽略新字段，回退可切回前版程序/静态目录及发布清单并重启，不还原整库覆盖新增用户数据。

APK 8,891,918 字节，SHA-256 `9fc40a4c1bc9ab1ea8fe3535f2ce4b0f6578041f44396b192dafee7a0a319713`，与旧 App 同证书。Go SHA-256 `845228c4a1dff325931616d2483d45db0b28d1c3773279c3c8433c92c56a9160`。

官网维护源 `C:/Mod/DeuteriumIX/work/review/d9-endfield/account-auth.js`，Help 维护源 `.worktrees/support-center-preview/support-center/account-auth.js`（本机提交 `dba6a89`），本轮 [共享脚本快照](artifacts/v216/account-auth.js) 随主仓发布记录合并。游戏插件及邮件样式没有更换。

[产物清单](artifacts/v216/manifest.json) · [部署记录](artifacts/v216/deployment.json) · [线上验证](artifacts/v216/verification.json)
