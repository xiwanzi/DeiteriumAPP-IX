# 2.0.1 图片缓存与资产生命周期验证

2026-09-08。本机验证对象为本分支 backend-next；部署状态另见发布记录。

- Go 全量 race / integration：119 项通过、0 失败；仅既有 TestSocialBrowserFixtureV2 人工浏览器夹具跳过。
- 新增 7 类真实数据库生命周期用例：共享有效/历史引用、委托未验收与资金未知、复制超时及删除失败恢复、恢复至有效前缀、引用与移动竞争、图片过期不破坏订单、未提交上传宽限与旧头像解绑。
- 新增 S3 SDK HTTP 测试：保留 CopySource 路径分隔符、复制条件、目标摘要、复用已有副本、不重置年龄、源消失后从已校验目标恢复、损坏/跨前缀拒绝。
- 雨云隔离探测：未启用版本控制，生命周期查询可用且原规则为空；复制路径和字节摘要通过。源 LastModified 为 2026-09-08T09:00:26Z，目标为 09:00:28Z，确认复制到新 key 重新开始对象存在时间。仅创建并删除本次随机探测对象。
- 部署前核对 13 个既有迁移：生产记录、当前工作文件及 Git 内容摘要全部一致。新增 018 迁移将在生产备份后执行。
- Android 108 项单元测试通过，Lint 0 errors / 29 warnings；缓存、真实清理、浅深色页面、四种头像尺寸已在 Android 15 模拟器验证。详细说明位于 Android 工作区 docs/qa/android-v2.0.1-2026-09-08.md。

证据原件：C:/DeuteriumAPP/.tools/deployment/cache201-go-final-tests.jsonl、cache201-s3-preflight.json、cache201-android-summary.json。本机脚本、凭据及运行时不进入交付源代码。没有用模拟器结果代替用户手机手感，未发起真实交易。
