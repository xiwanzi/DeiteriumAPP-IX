# 游戏聊天 0.0.45-dev 修复上线

2026-10-11 **22:55（UTC+8）**，客户端 **0.0.45-dev**（聊天模组 **0.1.1**）完成公网核验。源码 `1207c68`，[PR #99](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/99)，合并 `40f2baa82d3529d57183089255710bdb3693dd51`。

## 改动与原因

- 首次启动将原版快捷聊天改为 **T**，Sparkle Morpher 表情轮盘改为 **Alt+T**；**Enter** 保留完整聊天。迁移标记 `config/deuterium-chat/keybindings-v1` 防止以后覆盖玩家自行修改的按键，其余设置保留。
- 消息头像、私聊列表和联系人侧栏统一使用个人终端同款 `PlayerFaceRenderer`，绘制底层脸部与皮肤外层，保存并恢复混合状态。旧实现只采样底层脸部。
- “云端媒体暂不可用，请确认 App 已绑定游戏账号”不是本次用户账号问题。只读核对 xiwanzi 为 active 且已验证；Core 的新 HTTP 路径未被 Nginx 转发，公网 POST 返回 405，而 Core 对所有会话失败使用统一绑定提示。
- 已为 IP 与 chat 域名补充 `/bridge/v1/game-chat/` 普通 HTTP 反向代理，保留后端节点鉴权。修复后两域名的 session/media/team 无凭据请求均返回后端 401；实际 xiwanzi 在 22:49:28 自动取得游戏聊天会话，未手工改账号或签发测试用户会话。

## 验证与边界

- Gradle build / 单元测试通过；真实游戏 UI 场景 `45840_6418075c` 的 105 步、4 项断言通过。确认隔离客户端保存 `key_key.chat:key.keyboard.t`；现有 Sparkle Morpher 字节码确认轮盘通过带修饰键的映射匹配。轮盘实际模型动画效果未另行实测。
- DLauncher 实际 0.0.44 → 0.0.45 更新通过，旧 0.1.0 JAR 被删除，新 0.1.1 JAR 校验通过，个人设置/存档/无关模组保留，二次更新无操作。
- OSS 同步完成，保留 59 个旧索引条目；公网完整 TAR 与 Range 元数据检查通过。
- 新客户端 JAR：18,887,193 字节，SHA-256 `1668e6da2ecbb784dfc6defb299291d06d03b67b55ed1be06a8b6121b9bfafa7`。
- 新 TAR：18,892,288 字节，SHA-256 `292b0d51219a8e45ee102201783019e89898e6a97a3eb75bae5594e0a5c66890`。
- 三常驻 Core 节点在线、存储健康，MEK 保持停服。本次不重启游戏服，Core 1.0.4 / Go 二进制 / 数据库迁移 / App 2.0.25 / Web 2.0.35 保持。61 账号、47 订单、5 委托、3,085 在售商品、12 工单、2 注销记录保留。

## 代理配置与回退

两个 vhost `/www/server/panel/vhost/nginx/deuterium-test.conf` 与 `deuterium-chat.conf` 均增加下列规则；以后发布不能只检查 WebSocket `/bridge/v1/connect`：

```nginx
location ^~ /bridge/v1/game-chat/ {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-Proto $scheme;
    client_max_body_size 64k;
    proxy_read_timeout 30s;
}
```

配置备份 `/var/backups/deuterium/game-chat-proxy-20261011-r2`，`nginx -t` 后平滑重载。首次验证撞上重载交接窗口得到旧 405，脚本恢复原配置后加入等待，第二次完成六路径鉴权核验。

客户端备份 `/var/backups/deuterium/game-chat-client-0.0.45-hotfix`；已更新客户端需发布新的 McPatch 前向回退版本，不能只改旧索引。

[代理核验](artifacts/game-chat-0045/proxy-fix-result.json) · [公网校验](artifacts/game-chat-0045/public-verification.json) · [更新测试](artifacts/game-chat-0045/client-updater-verification.json) · [OSS 同步](artifacts/game-chat-0045/sync-result.json)
