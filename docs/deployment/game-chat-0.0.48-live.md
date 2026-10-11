# 游戏客户端 0.0.48-dev 上线

2026-10-11 **23:56（UTC+8）**同步成功，用户中断操作后已于 **10-12 00:02**补完公网校验，确认线上为 **0.0.48-dev / 聊天模组 0.1.4**。源码 `9e65494`，[PR #104](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/104)，合并 `f34ce2104c571131022402bb873c55d51dd07968`。

团队图标保持原比例、文字和气泡位置清晰度优化，小窗口保持字号并减少信息密度；T 重新绑定、TAC 配置键解绑、Alt+T 轮盘与 Enter 完整页面保持。小地图通过 Xaero 自身接口迁至右上；按用户选择彻底拦截原版背景音乐，保留自定义 BGM、唱片和其他音效。见[本机验收](../qa/game-chat-readability-20261011.md)。

JAR 18,896,478 字节，SHA-256 `2d634fe496faf058ea431cb7d490fceae45ec223e25f788bcd688e2e096e844e`。TAR 18,901,504 字节，SHA-256 `5532c309b8240e020b05bde5cd226edfa11757776a68f8dad894a9b1d0c592e7`。

实际启动器 47→48 更新、重复检查无操作及个人文件保留通过；公网完整包哈希和 Range 元数据通过，62 个旧索引条目保留。备份 `/var/backups/deuterium/game-chat-client-0.0.48-9e65494`。后端、Core、App/Web、原业务数据未改，三常驻节点在线、MEK 停服。

[公网验证](artifacts/game-chat-0048/public-verification.json) · [更新测试](artifacts/game-chat-0048/client-updater-verification.json) · [同步结果](artifacts/game-chat-0048/sync-result.json)
