# 官方交付模板管理增量

2026-09-08。补充 App v2 中原有只读 `/merchant/stores/{storeId}/delivery-templates`。模板来自管理员批准的 Core 不可变物品版本；不接受物品字节、NBT 或命令。

所有路径加 `/api/v1`，要求认证、浏览器 CSRF 和对应店铺权限；编辑要求 PRODUCT_EDIT，读取要求该店铺成员或管理权限。

| 方法与路径 | 请求 | 返回 data |
| --- | --- | --- |
| GET `/merchant/stores/{storeId}/delivery-templates` | cursor、limit | 原契约 DeliveryTemplate[]，仅 templateRef/name/summary/active，保留分页外壳 |
| POST 同上 | clientRequestId + 下列内容字段 | 201 完整模板 |
| GET 同路径 `/{templateRef}` | 无 | 200 完整模板 |
| PUT 同路径 `/{templateRef}` | clientRequestId、expectedVersion + 下列内容字段 | 200 完整模板 |
| POST 同路径 `/{templateRef}/disable` | clientRequestId、expectedVersion | 200 完整模板，active=false |

内容字段全部必填：name（1–80）、summary（1–1000）、inventoryDomain（受控背包域）、allowedServerIds（1–32 个唯一节点）、attachments（1–32 个唯一物品引用）、active（boolean）。每个 attachment 只含 itemRef、revision（正整数）、quantity（1–99999）、payloadSha256（64 位小写十六进制）；数量同时受精确版本的 maxQuantity 限制。

完整模板另含服务端生成的 templateRef、storeId、version、createdAt、updatedAt。停用后可通过完整 PUT 且 active=true 重新启用，但必须重新通过 Core 目录、摘要、兼容与范围校验。内容更新不会改写先前订单已经冻结的模板快照。

存储使用既有 `catalog_records_v2` 的 `delivery_template` 类型，未修改已部署 011 迁移。创建/编辑在事务内核对 `item_versions` 和 `core_catalog_heads`：精确版本存在、HEAD 未归档、摘要相等、数量合法、codec=`bukkit-bytes-v1`、背包域一致且全部节点在每个附件兼容名单内。请求的节点还必须由后端配置允许领取。HEAD 表或相应同步记录不存在时明确失败；不把“未知”当可销售。

商品发布和新报价都重新核实批准模板及当前 Core HEAD。字段错误返回 INVALID_REQUEST；权限不足 FORBIDDEN；版本冲突 STATE_VERSION_CONFLICT；旧 key 不同内容 IDEMPOTENCY_CONFLICT；缺目录 CORE_CATALOG_UNAVAILABLE；版本缺失 ITEM_VERSION_UNAVAILABLE；归档 ITEM_ARCHIVED；摘要/数量/编码/背包域不符 ITEM_SNAPSHOT_MISMATCH；范围/兼容性错误 INVALID_DELIVERY_SCOPE 或 ITEM_INCOMPATIBLE；停用模板 DELIVERY_TEMPLATE_UNAVAILABLE。

同一模板/HEAD 使用行锁并按物品引用排序，创建/编辑/停用采用原客户端 key 和乐观锁。真实数据库测试覆盖越权、伪造摘要、超量、禁领节点、归档后发布/报价拒绝、停用及重新启用。此模板能力不代表同步保存屏障或实际邮箱发放已经验收。
