# Deuterium 2.0.2 部署记录

2026-09-08。App **2.0.2 (20200)** 已发布到在线更新通道，已向 xiwanzi 账号发送更新提醒。

- [最新 APK](https://47.103.99.34/downloads/Deuterium-2.0.2-20200-test.apk)，23,160,566 bytes。
- SHA-256：88059466f6cd548a888832342099177ebc1bc6194db5e14f513b356ecdfdba51。
- 包名 com.deuterium.app.uilab，原签名证书 SHA-256 04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3；支持覆盖安装。
- Android 源码 4299ce37d0bf，工作区 ui-motion-lab，分支 codex/payment-overlap-v202。
- Go 服务仍为 1b5d9d1c63f8；本次只更新 APK/更新清单并重启服务加载清单，没有更换后端二进制、游戏插件或执行数据迁移。

付款确认后立即启动原业务请求，与 1–3 秒模拟识别并行，移除原 470ms/100ms 固定等待。只有识别结束且后端成功才显示确认；较慢请求继续等待。转账固定已确认的请求 ID、收款人、金额和备注，保持原幂等和未知结果恢复。

本次构建使用 gradlew.bat :ui-lab:assembleDebug :ui-lab:assembleDebugAndroidTest :ui-lab:testDebugUnitTest :ui-lab:lintDebug --offline --console=plain。113 项单元测试通过，Lint 0 errors / 29 warnings。Android 15 模拟器 fast/slow/failure/recovered 四种时序通过，包含识别中启动、早到结果不提前显示、界面恢复不重复提交以及恢复成功不被失败覆盖。回调为独立测试夹具，没有发起真实付款；用户手机实际网络及手感未冒充本次实测。

已验证公网 HTTPS 完整下载字节数/摘要，20005、20100、20101 均得到 20200，20200 不提示自身更新；APK 已完成包名/版本/签名及已有部署凭据的精确扫描。更新清单备份为 /var/backups/deuterium/cache-2.0.1-20260908-20100/releases-before-20200.json，通知事件为 app.release:20200，仅目标账号一条。

本机交付目录：C:/DeuteriumAPP/delivery/Deuterium-2.0.2/。详情见 [Android 验收](../qa/payment-v202-2026-09-08.md) 和 [实现边界](../plans/payment-overlap-v202.md)。
