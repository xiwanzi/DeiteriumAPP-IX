# 2.0.16 更新版本无效修复

2026-10-05，按用户报告已修正生产 App 发布清单。App 仍为 **2.0.16（21600）**，APK、签名、Go 二进制及网页版本没有变更。

## 原因与修正

本轮发布脚本将新条目的 `minAppVersionCode` 填成 `0`。Go 的清单读取允许该值，但现有 App `AppUpdates.decode` 要求 `minApp > 0`，因此 HTTP 200 和正确安装包摘要仍不能证明 App 能接受清单。这是本轮发布参数错误。

已仅将 `deuterium-2.0.16-21600-test` 的 `minAppVersionCode` 从 0 改为 **1**；最大适用版本保持 **21599**。另 16 个既有条目逐对象一致，新条目其他字段一致。发布脚本的候选生成值同步修正。保留既有文件权限并重启当前 Go 读取，无数据库写入。

## 本次验证

- 公网 IP 更新接口分别以 20004、21300、21500 请求，均提供 21600、最低版本为 1；21600 不提供重复 APK 更新。
- 根据当前 App 的真实解析规则检查最低版本正数、最大/最低范围、目标版本、ID/版本名/说明长度、HTTPS URL、包体大小及 SHA-256 格式。21500 与当前源码的 `AppUpdates.kt` 一致。
- 服务 active / ready。此前 APK 版本、同签名和完整性验收保持；本次没有重建 APK，也不将源代码规则检查称为手机更新安装实测。
- 回退备份 `/var/backups/deuterium/v216-update-min-version-20261005/releases-before.json`。该备份含导致报错的旧条目，仅作审计，不应重新激活。

用户在 App 中重新点击“检查更新”即可重新取得正确清单。后续发布必须核对客户端解析约束，不能只验证 HTTP 200 和下载摘要。

[修复和公网验证](artifacts/v216/update-manifest-fixed.json) · [原发布记录](registration-code-check-live-20261005.md)
