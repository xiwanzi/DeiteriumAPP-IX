# 邮箱 0.0.52 与 TAB 头衔接入（2026-10-12）

17:45（UTC+8）邮箱客户端已发布，17:47 公网完整性验收通过。用户随后明确授权直接重启 Amiya；18:01 Amiya 正常重启完成，18:01:42 TAB 配置切换并成功 reload，18:02 完成四节点回读。

## 当前状态

- 新变量 `%deuterium_title%` / `%deuterium_title_prefix%` 已实现并通过 Core 39 项测试；第二个变量带灰色方括号，保留称号 RGB 渐变，末尾 reset 保持 ID 颜色。
- Core 1.0.6 已更新四节点。Login / Amiya / Odyssey 启动及 PAPI 注册通过；MEK 保持停止。Amiya 按用户最终授权正常保存并重启。
- Login 及重启后的 Amiya 上实际执行 `papi parse xiwanzi %deuterium_title_prefix%` 返回 `[云端筑梦师]`，读取当前佩戴投影。
- 代理 TAB 已切换：列表保留服务器标识，列表与头顶统一读取新变量，刷新 1000 ms。应用前核对四节点 Core JAR，备份旧配置；应用后配置哈希及 TAB 成功 reload 日志通过。最终无人在线，代理在线玩家 parse 无法执行，头顶/列表实际画面等待玩家重新进入确认，不将后台变量解析冒充实机画面验收。
- 游戏资源 0.0.52-dev 已发布；邮箱 UI 0.7.19-native，源码 `41c5e2c86242ee0427a08c646a5140c5ec0edc33`。与线上 0.7.18 比较，JAR 内仅字体文件和版本元数据变化，字体为 HarmonyOS Sans SC Medium。
- 四节点邮箱业务插件与桥接器已是上游最新 0.7.3，无需重复替换；邮件、附件与配置未修改。

主源码 `00cc6b2`，合并 main 后为 `56b504933b20`，通过 [PR #110](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/110) 合并为 `362e694f44669bf06319ab290b7a52a41cab94b9`。

## 验证及数据

本轮 Core clean verify 39 项、邮箱 Maven clean verify 213 项、邮箱 UI clean build 129 项，均无失败或跳过。邮箱架构门禁通过。启动器真实更新器验证 0.0.51 → 0.0.52，移除唯一旧邮箱 JAR，保留个人设置、存档及其他模组，第二次更新为无操作。

OSS 同步 `ls_17c0b2f48d4590ae766336cff491ac8b` 成功，临时发布会话已撤销。公网 HTTPS 完整 TAR 哈希、长度及 HTTP 206 元数据验证通过，保留旧 66 条版本记录。

云端 Go/Web/官网/客服站链接、程序哈希、配置、迁移及业务数量保持：61 账号、47 订单、5 委托、3,085 在售商品、12 工单、2 注销记录。App 2.0.25 / Web 2.0.36 保持。已更新节点的配置内容保持，仅启动写入的 properties 注释时间变化。

## 产物与回退

| 产物 | SHA-256 |
| --- | --- |
| Core 1.0.6 | `509589801e38d6963c6ebedd819e53529c525705ea0d4eb4944b264a0a2382ef` |
| 邮箱 UI 0.7.19-native | `18368f505d26a6e852ce2ac0a769fe58d765513c9e2c64eb11fa57e96cc28b0a` |
| 0.0.52-dev TAR（9,707,008 字节） | `6bb1002513c82e46c8e3514a9c03ecb8270c5e557d396e293a7e12c4f33d6680` |

云端备份 `/var/backups/deuterium/titles-mail-client-0.0.52`；游戏端各节点备份 `C:/Users/ovo/CodexStaging/title-tab-core-1.0.6-<node>/<node>`。TAB 新旧配置及哈希计划保存在 `C:/Users/ovo/CodexStaging/title-tab-config-1.0.6`。

操作证据和续跑脚本在本工作区忽略目录 `output/title-tab-release`。四节点插件清单确认仅 Core 更新，邮箱及其他插件、全部模组和 Core 配置哈希保留；三常驻节点在线，MEK 仍停止。回退只恢复对应 JAR / TAB 配置 / 客户端索引，禁止恢复整库覆盖新业务。
