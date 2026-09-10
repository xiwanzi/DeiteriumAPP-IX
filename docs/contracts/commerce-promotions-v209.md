# 商城 2.0.9 增量契约

2026-09-10，适用于本轮实现，尚未表示生产部署。App 2.0.9（20900）、Web 2.0.9、配套 Go；信用点交付需要 Core 1.0.2、Mail 0.6.2（API 3）、Mail Bridge 0.6.1 与 XConomy 2.26.3-deuterium.3。Mail UI 沿用 0.6.0。

## 商品与交付

商品仍引用一个 `deliveryTemplateRef`，发布时冻结其内容。模板的 `attachments` 可包含 0–32 种物品，每种绑定 `itemRef`、`revision`、`payloadSha256`、`quantity`；同一物品引用只选择一个版本。零附件模板只能用于附带信用点的商品，空交付不能发布。数量、摘要、背包域与领取节点在发布、报价、创建订单时复核。

| 商品字段 | 规则 |
| --- | --- |
| `price` | 原价，精确十进制信用点字符串，必须大于零 |
| `discountRate` | 应付比例，整数 1–10000；缺省 10000，8500 为 8.5 折 |
| `deliveryCredits` | 每件商品附带的整数信用点，0–1000000000，缺省 0 |
| `purchaseLimits` | 可选限购对象；各项可同时生效 |
| 返回的 `effectivePrice` | 服务端按商品折扣计算的单价；不是可提交的商品字段 |

按单件价格乘比例，四舍五入到分，再乘购买件数。订单的原价总额和执行金额受现有单笔上限保护。赠送信用点按购买数量累计，单封最多 1000000000000；以 `creditAmount` 加入已签摘要的交付快照。没有信用点时不增加该字段，旧快照保持可恢复。

物品版本查询 `GET /admin/core/items` 保留原 `afterItemRef`、`afterRevision`，新增 `q`（名称/编号搜索）与 `inventoryDomain`。每页 50 条，`data.next` 为后续游标。`GET /admin/core/items/{itemRef}/versions/{revision}` 读取精确版本与归档状态；均要求 `core.read` 或平台管理权限。

`GET /merchant/stores/{storeId}/delivery-templates` 支持 `q`、`limit`、`cursor`，搜索覆盖完整集合。列表新增 `attachmentCount`、`inventoryDomain`。现有创建、读取、编辑与停用路由不变，编辑仍要求版本和请求键。

## 限购

`purchaseLimits` 的 `lifetime`、`daily`、`weekly`、`monthly` 均为 0–999999，0/缺省表示不限。单笔 `limitPerOrder` 沿用 1–999。

- `dailyTime`：`HH:mm`，缺省 `00:00`。
- `weeklyDay`：1–7，星期一至星期日，缺省 1；`weeklyTime` 缺省 `00:00`。
- `monthlyDay`：1–31，缺省 1；短月份取当月最后一天；`monthlyTime` 缺省 `00:00`。

全部按北京时间计算，不依赖客户端时钟或定时清空计数。按游戏资产 UUID 查询已有订单和正在付款的数量，删除账单/订单展示不改变计数。支付明确失败或全额退款后恢复额度。计数归属以订单创建时间确定，跨周期的退款不会错误增加新周期配额。提交时锁定身份和商品，重新计数，防止并发绕过。

## 优惠券

| 路由 | 行为 |
| --- | --- |
| `GET /store/coupons` | 当前玩家有效且未占用/使用的券，支持 `q`、`limit`、`cursor` |
| `GET /store/coupons/attention` | 当前玩家有效、未使用且未查看的券，支持 `limit`、`cursor`；每条增加 `announced` |
| `POST /store/coupons/attention` | 确认实际展示的一批券，提交 `couponIds`（1–100 个唯一 ID）及 `viewed` |
| `GET /admin/coupons` | 平台管理员搜索与分页；可用 `status=ACTIVE/SCHEDULED/EXPIRED/INACTIVE` |
| `POST /admin/coupons` | 平台管理员创建活动，必须有 `clientRequestId` |
| `PUT /admin/coupons/{couponId}` | 编辑/停用活动，必须有 `clientRequestId`、`expectedVersion` |

创建/修改字段为 `name`、`type`（ORDER/ITEM）、`benefit`（FIXED/PERCENT）、`amountOff`、`discountRate`、`minimumSpend`、`maxDiscount`、`stackWithProductDiscount`、`audience`（ALL/PLAYERS）、`playerRefs`、`storeIds`、`productIds`、`startsAt`、`endsAt`、`active`。金额字段用十进制字符串；限额 0 表示无上限，门槛 0 表示无门槛。单品券只允许 PERCENT；整单券可立减或打折。

ALL 活动在有效期内对所有已注册及后来注册的玩家生效，无需主动领取；PLAYERS 必须选择已绑定玩家的稳定 `playerRef`。每个活动每个资产 UUID 一张、一次使用。玩家接口和订单优惠快照不包含其他获券玩家的名单。

起止时间接收 RFC3339，保存为 UTC 秒精度，区间为 `[startsAt, endsAt)`。全员券具有统一的活动截止时间。客户端按服务端时间展示，并在截止时从页面移除；服务端同样过滤。后台保留活动管理与订单核账所需记录，不向玩家提供“已过期”列表。

店铺/商品范围留空表示全部，两者同时指定时取交集。整单券的门槛与优惠只计算适用范围内的商品，范围外商品保留本身折扣。同一订单沿用现有同店铺及共同领取范围限制。

**每单只自动使用一张最优惠券。** 可叠加表示在商品折后价上再用券。不叠加时，比较适用范围内“原价用券”和“商品折扣”的实付，选择较低金额；范围外不受影响。单品券只作用于一件商品，选择带来最大额外优惠的一件，其余件数保持商品折扣。相同优惠优先使用较早到期的券，再按稳定 ID 确定顺序；没有额外优惠的券不消耗。

### 到账提醒与查看

`GET /store/coupons/attention` 不修改权益或提醒状态，返回的券同样满足有效期、当前受众、未使用规则。`announced=false` 表示新券尚未提醒；已提醒但未查看仍返回，供“我的优惠”红点使用。全员券对后来注册玩家自动具有未提醒状态。

确认状态按 `(couponId, owner_uuid)` 唯一保存：`viewed=false` 只表示已展示浮窗/提醒条，`viewed=true` 同时表示已提醒与已查看，响应为 `data.acknowledged=true`。状态只前进，不回退，重复请求和两个设备并发确认安全。确认不减少券额度、不占用券、不触发交易；确认仅接受自己的受众范围，不公开其他玩家记录。券在展示与确认之间过期、被停用或使用，仍可保留原提醒确认；未开始活动不提前确认。

App 在启动完成或回到前台后读取未查看券并合并新券；页面、键盘或其他浮层忙碌时延后。启动后的前台新增使用顶部轻提示。相同券 ID 在收到确认后不因重新启动或更换设备再次提醒；同时在线设备已打开的浮窗不强制互相关闭。客户端记录待同步的确认，断网重开不会反复弹同一批；恢复连接后提交同样的券 ID。关闭 App 后不新增系统推送，下一次进入时处理未提醒券。

单张整单满减券且规则短、适用范围完整时，弹窗展示金额/门槛/范围/有效期，仅保留“好的”，关闭后视作已查看。单品、百分比、限制范围、长标题或多券场景增加“去看看”；仅关闭摘要保留红点，跳转“我的优惠”后确认已加载的券。客户端以服务端时间加单调计时移除已到期的提示、浮窗内容和红点，不展示“已过期”。

## 报价、下单与恢复

报价请求新增 `source=CART/DIRECT`。CART 记录购物袋数量快照，DIRECT 不清理购物袋；缺省兼容旧 App 的购物袋结算。客户端仍只提交商品 ID/数量/版本和交付选择，不提交折扣、收款人、优惠券金额或最终价格。

报价返回原有逐行商品折后 `unitPrice`、`subtotal`，以及 `originalTotal`、`productDiscount`、`couponDiscount`、`discountTotal`、`totalAmount`、`coupon`（可为空）、`storeName`、`storeId`。`subtotal` 合计减 `couponDiscount` 等于实付；原价合计减全部优惠等于实付。单品券返回选中的 `productId`/`productTitle`。`cartItems` 为服务端关联快照，只读。

创建订单只引用原报价 ID/版本，重新检查商品发布版本、库存、配额、券版本/资格/有效期/占用和交付能力。券变更时要求重新确认，不能静默增加扣款或换券。对 `(couponId, owner_uuid)` 的唯一占用和订单建立处于同一事务。技术性失败释放未使用券，结果未知继续占用；成功后券不因退款再次发放。

确认支付与邮件投递后，后端按已购买数量扣除购物袋并推进购物袋版本，在订单中记录已处理标记。多次回执/恢复不会重复扣除，其他商品与额外新增数量保留。明确失败和未知结果不会清空。订单固定实际店名、商品原价/折后价和优惠快照，后续店铺或优惠编辑不修改历史成交金额。

全额抵扣允许实付 `0.00`：不发送伪造的经济扣款/退款指令，仍使用同一订单、交付、取消证明和库存流程；真实领取后结束订单。普通转账、市场和委托继续使用原有正数金额要求。

新增主要错误：`PLAYER_PURCHASE_LIMIT`、`COUPON_CHANGED`、`COUPON_EXPIRED`、`COUPON_ALREADY_USED`、`CREDIT_DELIVERY_UNAVAILABLE`、`EMPTY_DELIVERY`、`DELIVERY_CREDITS_LIMIT`。原版本、鉴权、库存和未知结果错误继续有效。

## 独立邮箱边界

Mail 通过公开 API 接收与快照一致的 `creditAmount`，保留旧 Create 构造签名。领取先取得独占状态并完成物品保存，再通过 Core 的 MailPlayerDataBarrier 调用 XConomy `rewardMail`。资金操作 ID 固定为 `mailcredit_<claim operation UUID>`，使用 XConomy 自有幂等事务；没有把该能力开放为任意网络命令。

恢复先验证原玩家、背包域、会话、领取操作及真实保存证明，再补齐同一笔信用点。没有保存证明不重发物品，资金提交不确定不换键；领取完成与撤回仍互斥。后端在扣款前检查可用领取节点的 API 3 信用点能力，Core 同时按能力选择节点。

预览从原 `mail_delivery_items` 精确字节取得，只读 API 携带编解码版本与防修改副本，桥接器在服务端主线程转换为客户端预览。实际奖励仍读取同一份原始快照。预览无法解码时只降级显示，不重建或改写 TACZ 附件。

## 迁移与回退

新增迁移 `023_commerce_promotions.sql`，只建立优惠占用表与订单查询索引。已存在的表/索引会跳过，DDL 中断后可幂等续跑，已验证重试不改变订单与券占用。商品/券配置扩展既有 JSON 内容，旧商品缺省无折扣、无新限购、无附带信用点。上线前备份，使用专用维护连接执行迁移，运行账号不增加 DDL 权限。

提醒增加 `024_coupon_attention.sql`，仅创建账号级提醒/查看记录表与索引；可重复执行建表，不回填用户券或批量创建通知。回退 App/Go 时保留此表，不删除已有提醒记录。

回退前停用优惠活动及新增信用点商品，妥善处理未完成的新订单；保留新表、业务和原始快照。需要保留新资金/邮箱适配直到相关订单结束。不能整库覆盖、复用新请求键或降级安装以抹掉业务；APK 回退需更高 versionCode 的兼容修正版。
