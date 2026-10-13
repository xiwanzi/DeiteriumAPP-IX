# 聊天时效、排序与 App 消息恢复验收

2026-10-13，本机候选，未提交、推送或部署。工作区 `.worktrees/game-chat-ldlib`，分支 `xiwanzi/chat-expiry-order`；基于游戏聊天 0.1.9 与 App 2.0.26 源码。对应[计划](../plans/chat-delivery-recovery-20261013.md)及[契约](../contracts/chat-realtime-recovery.md)。

## 原因与修复

1. 游戏附近频道没有过期清理，服务端附近媒体历史窗口为 30 天。现按发送时间满五分钟过滤、拒收和定时清除，服务端查询窗口同步缩短；不删除其他业务数据。
2. 游戏聊天按 LinkedHashMap 插入顺序展示。私聊摘要先插入最新消息，随后历史补拉将旧消息追加在后面。现按完整发送时间排序，重复 ID 去重；缓存满时淘汰最旧发送时间，避免历史挤出新消息。
3. App 公共消息仅在连接建立/发送后补拉，没有页面进入或前台恢复补拉；客户端没有主动 WebSocket ping。私聊服务端原本只唤醒 Hub，并未查询推送私聊，实际依赖四秒轮询。现补充私聊 opt-in 推送、客户端保活、快速退避重连、前台/网络恢复、公共页面五秒兜底，并按成功同步位置跨页补齐。

## 本轮验证

- 游戏 `game-chat-client/gradlew.bat build`：23 项测试全部通过。其中新增 5 项覆盖 299999/300000 毫秒边界、无新消息过期刷新、历史/延迟包不复活、摘要先到后补历史的排序、重复 ID、替换、等时稳定及亚毫秒顺序。构建成功。
- Go `go test ./...`：全部通过。
- 本机隔离 MariaDB 集成：6 项定向测试全部通过。覆盖附近五分钟过期、重试不续期、团队历史保持、接收人隔离、游戏会话权限、私聊双方实时推送、重复消息不重复推送、第三人与旧客户端不接收私聊，以及既有图标事件兼容。
- Android `:ui-lab:testDebugUnitTest`：173 项测试通过。新增连接重复调用/停止后迟到事件测试，既有发送确认、拒绝及未知结果语义保持。
- Android `:ui-lab:assembleChatMediaQa :ui-lab:assembleChatMediaQaAndroidTest -PdeuteriumTestBuildType=chatMediaQa`：构建成功。
- Android 35 模拟器执行 `am instrument -w -e chatRealtime true com.deuterium.app.uilab.chatmediaqa.test/com.deuterium.app.uilab.LiveBackendInstrumentation`：PASS。测试只访问应用内回环 MockWebServer，不连接生产账号。
  - 公共、私聊推送在两秒检查窗口内收到，私聊不混入公共列表。
  - 连接仍显示在线但漏推时，HTTP 补拉恢复消息。
  - 回前台重建连接，公共与私聊各 125 条跨页缺口均恢复，无重复；会话摘要已包含最新消息也不会跳过缺口。
  - 服务端关闭连接后，四秒检查窗口内自动重连，无需重启 App。
- 最终 QA APK 元数据核对：包名 `com.deuterium.app.uilab.chatmediaqa`，版本 `2.0.26-chat-media-candidate` / `22600`。独立 QA 包仅供本地验证，未升级正式应用版本号或替换正式安装。
- `git diff --check` 通过。

首次 Android 回归脚本在测试线程遍历 Compose 列表时发生并发修改异常；改为主线程读取后完整回归通过，非 App 收消息失败。首次尝试复用旧本机数据库遇到认证插件不兼容，最终所有集成测试使用本轮新建的回环数据库及随机测试 schema。

## 限制与发布范围

本轮没有发布 APK、游戏资源或 Go，没有重启游戏服、修改生产数据库或向玩家发送消息。正式落地需要配套发布游戏聊天客户端、App 与 Go；Core 协议及程序无需为本轮调整。模拟器与回环测试证明恢复逻辑，不代表真机运营商网络的固定延迟承诺。真实 Wi-Fi/蜂窝切换和线上多设备体验尚待部署后的实测。
