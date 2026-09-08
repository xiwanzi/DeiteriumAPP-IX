# Deuterium 2.0.1 部署记录

2026-09-08。当前最新 App 为 **2.0.1 (20101)**，缓存、存储空间和联系人调整均已发布到在线更新通道，并已向 xiwanzi 发送更新提醒。Go 图片生命周期服务已部署。最新下载与验收见下方“联系人追加构建 20101”；20100 原始部署证据保留用于追溯。Android 系统安装仍由用户在手机上确认。

## 已发布产物

- APK：[Deuterium-2.0.1-20100-test.apk](https://47.103.99.34/downloads/Deuterium-2.0.1-20100-test.apk)，24,572,331 bytes。
- APK SHA-256：8713fae5f7a1f5444508817ee7036daa48b06cf0968b9e5a291b2232ed773346。
- 包名 com.deuterium.app.uilab，版本 2.0.1 (20100)，Android 8.0+。原签名证书 SHA-256 04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3，已核对与 20005 相同并覆盖安装。
- Android 源码 800c60a，工作区 ui-motion-lab；Go 源码 1b5d9d1c63f8af670409060f54a43aa2f981c163，工作区 test-release-v2。
- Go 二进制：/opt/deuterium/releases/cache-2.0.1-1b5d9d1c63f8/deuterium；SHA-256 b1d3564ad0b99d346151816c090821aa35c501ecbaa21af5f4b352e5e3b8ce25。
- 本机交付：C:/DeuteriumAPP/delivery/Deuterium-2.0.1/，含构建信息、发布清单和公网验证结果。没有自动推送或合并 Git main。

## 本次行为

自己和他人的头像、商品与委托图片使用共享内存及磁盘缓存，按控件尺寸解码，签名链接变化不重复下载。“我的 → 存储空间”显示真实占用，支持分类和全部清理；账号、聊天、交易记录和当前资源保留。新选图暂存文件及时释放。

历史订单继续共享原图。图片没有在售商品、有效委托等当前用途后，才转到 deuterium-test/gc/。雨云生命周期规则 deuterium-gc-30-days-v201 已配置且读回验证：仅此前缀，存在满 30 天异步删除。原先无生命周期规则，未给 uploads/ 或整个项目根前缀设置过期。

首轮 11 个资产中 6 个符合回收条件。运行后 5 ACTIVE、3 RETIRED、3 PURGED；PURGED 已核对对象本就缺失或已删除，不删除数据库业务记录。发布期间读取到 3 笔订单、1 份委托（备份时为 2 笔订单、1 份委托），本轮脚本未发起付款、下单或委托操作，不清理期间新增业务。

## 验证

- Android：108 项单元测试通过，Lint 0 errors / 29 warnings；实际 Coil 磁盘/内存命中、换签复用、96px 解码、到期/私有素材策略、真实文件清理及四种头像尺寸通过 Android 15 模拟器验证；浅深色页面已检查。详见 [Android 验收](../qa/android-v2.0.1-2026-09-08.md)。
- Go：119 项全量 race / integration 通过，仅既有人工浏览器夹具跳过；共享引用、恢复、并发和失败恢复见 [生命周期验收](../qa/assets-lifecycle-v201-2026-09-08.md)。
- 公网 HTTPS 完整读取 APK，字节数与 SHA-256 相同；20005 查询得到 20100，20100 不提示自身更新。
- APK 与 Go 二进制按已有部署/供应商/节点/会话凭据做 UTF-8、UTF-16LE 精确扫描，未发现这些凭据。
- 真实手机手感没有冒充已验收；未更换或重启游戏服务端插件。

## 备份与回退

部署前一致性备份：/var/backups/deuterium/cache-2.0.1-20260908-20100/deuterium_backend.sql.gz，89,117 bytes，SHA-256 a25769d5164e5abf233cc1695879d5a3b2badca9adc94b6aa827eed6d33e4be6。同目录保留原配置、环境和更新清单，均在服务端受限目录，不公开下载。

迁移 018 只增加生命周期列/索引并标记曾绑定资产；13 个既有迁移的工作文件、Git 与生产摘要全部一致。旧二进制仍可读取新增列以外的原字段。若回退到不支持恢复的旧版本，应先停止 GC 并停用此专用生命周期规则，避免旧代码重新绑定仍在到期前缀中的对象；不自动恢复旧数据库备份以覆盖新业务记录。优先保留当前服务、关闭回收开关后诊断。

详细边界见 [增量契约](../contracts/images-cache-lifecycle-v201.md) 和 [实施说明](../plans/image-cache-and-asset-lifecycle-v2.md)。

## 联系人追加构建 20101

同日已发布 2.0.1 (20101)，覆盖本页前述 20100 的 App 下载入口；Go 服务版本保持 1b5d9d1c63f8。新增联系人按私聊记录过滤、特别关心置顶、最近消息排序与全目录搜索，设置搜索包含存储空间。付款时序仅做解释，没有修改。

- 最新 APK：[Deuterium-2.0.1-20101-test.apk](https://47.103.99.34/downloads/Deuterium-2.0.1-20101-test.apk)，23,160,566 bytes。
- SHA-256：958d51e19ac22ebdea52e0e49419b10eb4e43e2febd7fedd1e2125ec60de0968；包名和签名与 20100 相同。
- Android 源码 d23bd79；113 项单元测试通过，Lint 0 errors / 29 warnings。模拟器验证默认联系人过滤、特别关心优先、新消息后重排、未聊过玩家搜索和资料入口；共享图片缓存管线再次通过。
- 公网完整下载摘要相同；20005 和 20100 均查询到 20101，20101 不提示自身更新。已向 xiwanzi 发送 APP_UPDATES 提醒，未广播测试私聊。
- 构建和验证命令：gradlew.bat :ui-lab:assembleDebug :ui-lab:assembleDebugAndroidTest :ui-lab:testDebugUnitTest :ui-lab:lintDebug --offline --console=plain。交付元信息与公网证据保存在 C:/DeuteriumAPP/delivery/Deuterium-2.0.1/ 的 20101 JSON 文件中。
- 联系人细节及支付问题说明：[联系人验收](../qa/contacts-v20101-2026-09-08.md)、[支付时序](../qa/payment-timing-2026-09-08.md)。

当前最新构建以本节 20101 为准，20100 证据保留用于追溯。
