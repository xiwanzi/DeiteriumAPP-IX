# Deuterium 游戏聊天

2026-10-11 已随游戏内容 0.0.44-dev、Core 1.0.4 与配套 Go 发布，见[上线记录](../docs/deployment/game-chat-0.0.44-live.md)。Minecraft 1.21.1 / NeoForge，依赖 LDLib2 2.2.39；当前随包图片编解码组件为 Windows x64。

## 使用

- 客户端安装本模块与 LDLib2；替换原 ModernChat 聊天客户端，避免两套聊天界面同时接管按键。
- `Enter` 打开完整聊天；`T`、`/` 使用原版薄输入栏及指令补全；`Esc` 退出。
- 完整页面按物理像素布局，不随 Minecraft GUI 缩放重复缩小。世界 HUD 仍遵循 GUI × 0.6 的原消息尺度。
- 表情、图片、分享入口的点击区域为 36px，实际图形轮廓统一为 24px。内置/个人贴纸支持右键或按住 550ms 预览。
- 图片在后台压缩为 WebP；普通图片最长边 1920，长图和贴纸采用各自的尺寸与质量上限，沿用 App 契约。游戏中的动态图片、贴纸只显示首帧。
- 私聊与个人贴纸需要配套 Core 和 Go 服务签发的聊天会话；客户端不包含节点密钥或管理凭据。

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
