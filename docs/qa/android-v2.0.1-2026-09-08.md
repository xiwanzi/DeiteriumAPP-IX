# Android 2.0.1 (20100) 缓存与存储空间验收

2026-09-08。包名 com.deuterium.app.uilab，Android 8.0+，沿用当前签名，版本入口以 BuildConfig 为准。发布状态另见后端工作区 docs/deployment/deuterium-2.0.1-live.md。

## 变更

统一 Coil 3.3.0 图片加载、内存/磁盘缓存、稳定 assetId 缓存身份、下载合并与完整性校验。签名变化复用图片；当前头像启动预热，上传完成直接缓存已有字节，列表/头像按控件大小解码。新增“我的 → 存储空间”，使用 iOS 风格分类占用条和分组操作，读取真实占用并支持分类/全部清理。更新在下载/验证/安装时受到保护，当前资源、账号与业务记录保留。选图和裁切暂存文件及时释放，旧暂存文件按已知名称、年龄和现有引用安全清理。

历史订单继续复用原图；后端判断无当前用途后才进入 30 天生命周期。客户端识别 retainUntil / EXPIRED 并保留订单其余内容。详细契约见后端工作区 docs/contracts/images-cache-lifecycle-v201.md。

## 本次验证

- 使用工程 gradlew.bat 运行 :ui-lab:assembleDebug :ui-lab:assembleDebugAndroidTest :ui-lab:testDebugUnitTest :ui-lab:lintDebug；最终构建通过。
- 单元测试 108 项通过，0 failures / 0 errors / 0 skipped；Lint 0 errors / 29 warnings。
- Android 15 模拟器：实际 Coil 管线磁盘 → 内存命中、换签仍命中内存、96px 目标解码、到期不从内存复活、私有凭证不缓存、清理后磁盘及内存缓存为零。
- 真实文件统计与“清理所有缓存”确认流程通过，清理后业务记录哨兵文件完整。
- 浅色/深色页面已截图检查；默认42dp、小头像21dp、资料98dp及当前账号42dp的布局回归通过。
- 在原已安装版本上使用 adb install -r 验证覆盖安装。未进行用户真机手感验收，未创建真实交易。

截图：C:/DeuteriumAPP/output/cache201/storage-light.png、storage-dark.png、storage-confirmation.png、storage-cleared.png。截图中用于计量/清理的图片和下载文件由独立 androidTest APK 创建；这些测试文件和入口不会打入用户 APK。
