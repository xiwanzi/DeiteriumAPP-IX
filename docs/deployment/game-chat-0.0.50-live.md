# 游戏聊天 0.0.50-dev：黑色渐变与分享功能

2026-10-12 **00:31（UTC+8）**，最终客户端 **0.0.50-dev / 聊天模组 0.1.6** 完成发布和公网核验。中间版本 0.0.49 于 00:25 发布，最终版补齐历史轮询先返回时的物品富文本合并。

- 主要源码 `cf2227b`，[PR #105](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/105)，合并 `aa073cf6438d1cf089258ebd397de0c913f89a48`。
- 最终边界修正 `fbb2d7e`，[PR #106](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/106)，合并 `065c3962999513d3539b8248b1e85c497c9fc76f`。

## 最终行为

聊天背景由偏绿改为中性黑灰，使用平滑曲线过渡并减弱中段场景颜色干扰。Logo 保留原字体，取消非等比横向拉伸，提高字形采样并对齐像素。分享浮层也改为中性深灰。

分享面板只有“当前位置”和“当前物品”：当前位置沿用旧实时位置图标，只发送一次服务器确认的位置；持续实时共享入口撤下。当前物品按用户确认仅支持世界频道，直接使用主手物品，通过已启用的 TrChat `[item]` 流程发送，保留服务器读取、权限及 30 秒冷却。空手提示，其他频道不会转发到世界。

物品翻译键按客户端当前语言归一化，与原始 TrChat 富文本归并；原始消息先到、桥接先到、原始消息已显示及 HTTP 历史先到均覆盖。只显示一条物品气泡，保留物品悬停和点击，不再额外显示原始格式行。旧客户端尚未更新时仍使用它自身的显示行为。

本次保留此前离线头像、中文输入豁免、小屏可读性、T / Alt+T / Enter、右上小地图及原版背景音乐屏蔽。用户需要完成启动器更新并重新启动游戏。

## 验证

- 构建和单元测试通过；中文实际 NeoForge + Modern UI + IMBlocker + Xaero 联合场景 `41348_7222e168`，136 步、11 项检查通过。
- 最终专项 `34856_53425450`，30 步、6 项检查通过，增加“历史轮询不能覆盖物品悬停”验证；截图确认中文名称与单条气泡。
- 正式 TrChat 配置只读确认，未向生产聊天发送物品测试消息；完整边界见[验收记录](../qa/game-chat-share-20261012.md)。
- 实际 DLauncher 48→49、49→50 更新均通过，旧 JAR 被替换、个人文件和存档保留、二次检查无重复更新。
- 两版公网完整 TAR SHA-256 校验，最终 HTTPS 索引及 Range 元数据校验通过。最终保留 64 个旧索引条目，其他客户端模组和配置未改。

| 最终产物 | 字节 | SHA-256 |
| --- | ---: | --- |
| 聊天模组 0.1.6 | 18,900,560 | `f990a39c49742708ddbc1e2295f4de92c50f1427e64eb0f02f830b7a432a705c` |
| 0.0.50-dev.tar | 18,905,088 | `554d54ea84abbe0f19d1e4758a219512352cff331625f183f1b0ff8340e545c5` |

## 保留项和回退

Go 二进制及数据库迁移不变，Core **1.0.4** / App **2.0.25** / Web **2.0.35** 保持。**61 账号、47 订单、5 委托、3,085 在售商品、12 工单、2 注销记录**保留；三个常驻节点在线且存储健康，MEK 停服。本轮不重启游戏服，不修改 TrChat 配置或服务端聊天权限。

备份 `/var/backups/deuterium/game-chat-client-0.0.49-cf2227b` 与 `/var/backups/deuterium/game-chat-client-0.0.50-fbb2d7e`。已更新客户端如需回退，发布新的 McPatch 修正版，不只回退索引。发布临时会话已撤销。

维护 `.worktrees/game-chat-ldlib` / `xiwanzi/game-chat-ldlib`。

[最终公网验证](artifacts/game-chat-0050/public-verification.json) · [最终更新测试](artifacts/game-chat-0050/client-updater-verification.json) · [最终同步](artifacts/game-chat-0050/sync-result.json) · [专项 UI](artifacts/game-chat-0050/ui-summary.json) · [49 公网记录](artifacts/game-chat-0049/public-verification.json)
