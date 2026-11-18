# Deuterium 游戏聊天

2026-10-12 当前游戏内容为 0.0.50-dev，配套 Core 1.0.4 与 Go 已在线，见[上线记录](../docs/deployment/game-chat-0.0.50-live.md)。Minecraft 1.21.1 / NeoForge，依赖 LDLib2 2.2.39；当前随包图片编解码组件为 Windows x64。

## 使用

- 客户端安装本模块与 LDLib2；替换原 ModernChat 聊天客户端，避免两套聊天界面同时接管按键。
- 在配套游戏服首次进服会按正版 UUID 备份键位，之后只同步改动。玩家可在「选项 → 控制 → 按键绑定 → 云键位」开启远程协助、撤销最近一次远程改键。管理员在「工作台 → 玩家键位」查询和修改，详情见[键位同步契约](../docs/contracts/game-keybindings.md)。
- `Enter` 打开完整聊天；`T`、`/` 使用原版薄输入栏及指令补全；`Alt+T` 打开表情轮盘；`Esc` 退出。首次启动迁移这两个按键并解绑 TAC 配置入口，之后保留玩家自定义设置。
- 完整页面按物理像素布局，不随 Minecraft GUI 缩放重复缩小。世界 HUD 按物理像素保持可读字号，小窗口减少同屏信息。
- 表情、图片、分享入口的点击区域为 36px，实际图形轮廓统一为 24px。内置/个人贴纸支持右键或按住 550ms 预览。
- 图片在后台压缩为 WebP；普通图片最长边 1920，长图和贴纸采用各自的尺寸与质量上限，沿用 App 契约。游戏中的动态图片、贴纸只显示首帧。
- 私聊与个人贴纸需要配套 Core 和 Go 服务签发的聊天会话；客户端不包含节点密钥或管理凭据。

离线头像由服务端持久缓存，完整聊天页面已添加 IMBlocker 中文输入豁免。

分享面板支持一次发送的“当前位置”和世界频道“当前物品”（主手物品，沿用 TrChat 规则），不再提供持续位置共享。

## 配套代码

同一工作区的 `deuterium-core` 提供游戏身份、附近 50 米、已有团队模组的成员路由、结构化消息头和聊天会话；`backend-next` 复用现有媒体/贴纸/S3 协议并隔离附近、团队、私聊附件权限。App 转发正文不再重复拼接 ID/服务器/App 前缀。

生产配套服务已发布；本机测试与生产核验的范围分别记录，真实用户交互仍待实际使用反馈。

## 构建与验证

Java 21：

```powershell
./gradlew.bat build --console=plain
./gradlew.bat runClient -PldTest=deuterium_chat --console=plain
./gradlew.bat runClient -PldTest=deuterium_chat -PguiScale=2 --console=plain
./gradlew.bat runClient -PldTest=deuterium_chat -PguiScale=4 -Pwindow=1920x1080 --console=plain
```

UI 场景使用隔离世界和本机媒体夹具；它们及 `ui-test` 资源不会进入运行 JAR。构建产物位于 `build/libs`。具体实测范围、同背景 Web 对照及媒体/团队路由证据见 `../docs/qa/game-chat-ldlib-20261011.md`。
