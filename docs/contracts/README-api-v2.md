# 接口文档阅读入口

**2026-09-12 / 永久注销候选：** 管理工作台使用当前管理员密码确认注销，App/Web 同步清理旧账号引用，见[注销契约](account-permanent-deletion.md)与[OpenAPI](openapi-account-deletion.yaml)。尚未部署。

**2026-09-10 / 2.0.8 候选：** 新增内置桌面图标选择、管理端发布与实时补同步，见 [图标切换契约](launcher-icons-v208.md) 和 [OpenAPI](openapi-launcher-icons-v208.yaml)。已完成本机实现与隔离验证，是否上线以部署记录为准。

**2026-09-09 / 2.0.3：** 新增账号级记录隐藏，公共聊天 HTTP 发送已实现，详情见 [2.0.3 接入说明](app-v203.md)。OpenAPI 同步显示字段和真实回执；下方原始操作数是历史快照。

**2026-09-08 实现扩展说明：** 当前 `openapi-app-v2.yaml` 已在原上游快照上加入商品草稿图片及真实交易恢复字段，原初始 SHA-256 `ce8091745107bab16822c53475aece94bed7104a149a89ad0850aa3d03316a48` 仅标识导入时版本。订单/委托/介入已具备本机实现和隔离验收，见 [交易编排契约](commerce-http-seam-v2.md) 与 [本机金融验收](../qa/commerce-v2-2026-09-08.md)。以下体验版 07 的操作计数、`existing/contract_only` 标签及“尚未实现”描述保留为原交付快照；当前是否部署以 [项目状态](../project-status.md) 为准。

**2026-09-08 契约补充：**

- 第 **2.1 节**：图片改为客户端直传 OSS，后端只授权/验证/写业务记录。新增申请授权、查询、续签、完成验证接口；旧 multipart POST /assets 移除，文件不经过业务 API。
- 第 **4.1 节**：委托接取后退出公开大厅；市场订单仅双方可见，售罄/下架商品隐藏，多库存保留剩余可售部分。详情、快照、退款、证据、分页、缓存及通知统一鉴权。
- 第 **8.1 节**：远程外观配置，新增 `/app/appearance` 和 `/admin/appearance/global`、`/admin/appearance/players/{playerRef}` 的 GET/PATCH。底栏玻璃、浮层玻璃、顶部渐变分别配置；支持全局/玩家覆盖、独立重置、版本及生效规则。
- 本次仅补接口契约，新增后端尚未实现，07 APK 尚未接入 OSS 直传、远程配置及独立材质数值。


体验版 07 配套的完整业务契约。**153 个 HTTP 操作、235 个模型、230 份经结构校验的请求/响应示例。** 服务端新增能力仍待实现，本轮交付 App 体验和契约。

- [完整业务说明与所有字段](app-api-complete-v2.md)：先读第 1–9 节，再查具体端点和数据模型。
- [OpenAPI 3.0.3](openapi-app-v2.yaml)：可导入支持 OpenAPI 的接口工具。
- [完整 JSON 示例](api-v2-examples.json)：按 method/path 查请求和响应。
- [担保与游戏邮箱桥接增量](bridge-commerce-v2.md)：后端/插件正式接入时所需的新能力。
- [校验记录](api-v2-validation.json)：示例与模型校验结果。
- [原有 v1 契约](app-backend-api-v1.md)：既有账号、钱包、聊天、AI、WebSocket 与插件协议。

## 范围

| 功能组 | HTTP 操作数 |
| --- | --- |
| 应用更新 | 1 |
| 账号与资料 | 9 |
| 钱包与转账 | 7 |
| 聊天与在线状态 | 13 |
| AI 助手 | 7 |
| 媒体上传 | 6 |
| 资金操作查询 | 2 |
| 官方商城 | 11 |
| 订单与退款 | 17 |
| 玩家市场 | 9 |
| 委托 | 13 |
| 通知 | 8 |
| 网页商家端 | 23 |
| 平台管理端 | 26 |
| 远程外观配置 | 1 |

## 推荐对接顺序

1. 保留既有账号、身份和会话机制，完成资料/媒体/通知基础接口。
2. 先落实经济担保和游戏邮箱的可恢复执行，再接商城、市场和委托的扣款/退款/结算。
3. 商家端完成商品草稿、完整信息编辑、发布和库存管理；成交快照永远独立保留。
4. 平台端接收介入申请、证据和服务端交易快照，并通过受控资金操作完成裁决。
5. 配置正式更新域名、包名/频道和发布权限，验证 APK 与资源包流程。

## 端点索引

所有路径均以 `/api/v1` 开头。`existing_v1` 为仓库已有路由；`extension_pending` 为已有路由的待实现扩展；`contract_only` 为新增契约。

| 方法 | 路径 | 用途 | 状态 |
| --- | --- | --- | --- |
| GET | `/app/update-check` | 检查当前 App 版本 | extension_pending |
| POST | `/account/registration-code` | 请求注册验证码 | existing_v1 |
| POST | `/account/register` | 提交注册 | existing_v1 |
| POST | `/account/login` | 登录 | existing_v1 |
| POST | `/account/password-reset-code` | 请求密码重设验证码 | existing_v1 |
| POST | `/account/password-reset` | 提交新密码 | existing_v1 |
| POST | `/account/logout` | 退出当前设备登录 | existing_v1 |
| GET | `/account/me` | 获取当前用户资料 | existing_v1 |
| GET | `/wallet/balance` | 获取当前已知余额 | extension_pending |
| POST | `/wallet/balance/refresh` | 刷新信用点余额 | existing_v1 |
| GET | `/wallet/recipients/search` | 搜索收款玩家 | existing_v1 |
| POST | `/wallet/transfers` | 提交信用点转账 | existing_v1 |
| GET | `/wallet/transfers/{transferId}` | 查询转账结果 | existing_v1 |
| GET | `/wallet/records` | 获取基础信用点流水 | extension_pending |
| GET | `/chat/messages` | 获取最近公共聊天 | extension_pending |
| POST | `/chat/messages` | 发送公共聊天或回复 | contract_only |
| GET | `/chat/presence` | 获取服务器在线概览 | existing_v1 |
| GET | `/chat/online-players` | 获取服务器在线玩家列表 | existing_v1 |
| GET | `/chat/player-directory` | 获取玩家目录 | existing_v1 |
| GET | `/chat/follows` | 获取当前账号关心玩家 | existing_v1 |
| POST | `/chat/follows` | 关心玩家 | existing_v1 |
| DELETE | `/chat/follows/{playerRef}` | 取消关心玩家 | existing_v1 |
| GET | `/ai/me` | 获取当前 AI 状态 | existing_v1 |
| GET | `/ai/plans` | 获取可购买 AI 套餐 | existing_v1 |
| GET | `/ai/messages` | 获取当前 AI 会话消息 | existing_v1 |
| POST | `/ai/chat/stream` | 流式发送 AI 消息 | extension_pending |
| POST | `/ai/conversation/reset` | 重置当前 AI 会话 | existing_v1 |
| POST | `/ai/purchases` | 购买 AI 套餐 | existing_v1 |
| GET | `/ai/purchases/{purchaseId}` | 查询 AI 套餐购买结果 | existing_v1 |
| PATCH | `/account/me/profile` | 更新个人简介或头像 | contract_only |
| GET | `/players/{playerRef}` | 获取玩家资料 | contract_only |
| POST | `/assets/uploads` | 申请客户端直传 OSS 的上传授权 | contract_only |
| GET | `/assets/uploads/{uploadId}` | 读取本人上传会话和验证结果 | contract_only |
| POST | `/assets/uploads/{uploadId}/renew` | 续签未完成上传的 OSS 授权 | contract_only |
| POST | `/assets/uploads/{uploadId}/complete` | 通知后端验证 OSS 对象并确认资产 | contract_only |
| GET | `/assets/{assetId}` | 查询媒体处理结果 | contract_only |
| POST | `/assets/{assetId}/remove` | 删除尚未被业务引用的本人媒体 | contract_only |
| GET | `/operations/{operationId}` | 查询资金操作状态 | contract_only |
| GET | `/operations/by-client-request` | 按幂等键恢复未收到编号的资金操作 | contract_only |
| GET | `/wallet/records/{recordId}` | 获取账单详情 | contract_only |
| GET | `/store/stores` | 列出受控官方店铺 | contract_only |
| GET | `/store/stores/{storeId}` | 获取官方店铺信息 | contract_only |
| GET | `/store/stores/{storeId}/homepage` | 获取商城首页区域和排序 | contract_only |
| GET | `/store/brands` | 获取品牌筛选项 | contract_only |
| GET | `/store/categories` | 获取商城商品分类 | contract_only |
| GET | `/store/products` | 浏览和搜索已发布商品 | contract_only |
| GET | `/store/products/{productId}` | 获取商品所有展示信息 | contract_only |
| GET | `/store/cart` | 获取购物袋 | contract_only |
| PUT | `/store/cart/items/{productId}` | 添加商品或修改数量 | contract_only |
| POST | `/store/cart/items/{productId}/remove` | 移除购物袋商品 | contract_only |
| POST | `/checkout/quotes` | 生成服务端结算报价 | contract_only |
| POST | `/store/orders` | 按报价创建官方商城订单并付款 | contract_only |
| GET | `/market/categories` | 获取六个市场分类 | contract_only |
| GET | `/market/listings` | 浏览或搜索市场商品 | contract_only |
| POST | `/market/listings` | 发布带图片的商品或建筑服务 | contract_only |
| GET | `/market/listings/{listingId}` | 获取市场商品详情 | contract_only |
| PUT | `/market/listings/{listingId}` | 保存下架商品的编辑内容 | contract_only |
| GET | `/market/me/listings` | 查看我发布的商品 | contract_only |
| POST | `/market/listings/{listingId}/unlist` | 确认下架商品 | contract_only |
| POST | `/market/listings/{listingId}/republish` | 重新校验完整表单并上架 | contract_only |
| POST | `/market/orders` | 按报价购买玩家商品并冻结货款 | contract_only |
| GET | `/orders` | 获取我的订单 | contract_only |
| GET | `/orders/{orderId}` | 获取订单详情和可用动作 | contract_only |
| GET | `/orders/{orderId}/snapshot` | 获取不可变成交条款快照 | contract_only |
| POST | `/orders/{orderId}/ship` | 卖家确认普通商品已交付 | contract_only |
| POST | `/orders/{orderId}/start-work` | 卖家开始施工 | contract_only |
| POST | `/orders/{orderId}/complete-work` | 卖家提交工程已完成 | contract_only |
| POST | `/orders/{orderId}/confirm` | 买家确认收货或工程验收 | contract_only |
| GET | `/orders/{orderId}/mailbox` | 查询官方物品邮箱发放或领取状态 | contract_only |
| POST | `/orders/{orderId}/refunds` | 申请订单退款 | contract_only |
| GET | `/orders/{orderId}/refunds/{refundId}` | 获取退款详情 | contract_only |
| POST | `/orders/{orderId}/refunds/{refundId}/withdraw` | 买方撤回退款申请 | contract_only |
| POST | `/orders/{orderId}/refunds/{refundId}/resolve` | 卖方同意或拒绝退款 | contract_only |
| GET | `/commissions` | 浏览委托大厅 | contract_only |
| POST | `/commissions` | 预付报酬并发布委托 | contract_only |
| GET | `/commissions/me` | 查看我发布或接取的委托 | contract_only |
| GET | `/commissions/{commissionId}` | 获取委托详情和进度 | contract_only |
| GET | `/commissions/{commissionId}/snapshot` | 获取发布和接取时锁定的委托条款 | contract_only |
| POST | `/commissions/{commissionId}/accept` | 接取委托 | contract_only |
| POST | `/commissions/{commissionId}/cancel` | 取消尚未接取的委托并退款 | contract_only |
| POST | `/commissions/{commissionId}/complete` | 接取者提交已完成 | contract_only |
| POST | `/commissions/{commissionId}/confirm` | 发布者确认完成并结算报酬 | contract_only |
| POST | `/commissions/{commissionId}/refunds` | 发布者申请委托退款 | contract_only |
| POST | `/commissions/{commissionId}/refunds/{refundId}/withdraw` | 撤回委托退款申请 | contract_only |
| POST | `/commissions/{commissionId}/refunds/{refundId}/resolve` | 接取者处理委托退款 | contract_only |
| GET | `/chat/conversations` | 获取私聊列表 | contract_only |
| POST | `/chat/conversations` | 获取或创建与玩家的私聊 | contract_only |
| GET | `/chat/conversations/{conversationId}/messages` | 分页读取私聊消息 | contract_only |
| POST | `/chat/conversations/{conversationId}/messages` | 发送私聊消息或回复 | contract_only |
| POST | `/chat/conversations/{conversationId}/read` | 同步私聊已读位置 | contract_only |
| GET | `/announcements` | 读取官方公告列表 | contract_only |
| GET | `/announcements/{announcementId}` | 读取官方公告详情 | contract_only |
| GET | `/notifications/preferences` | 获取账号通知偏好 | contract_only |
| PATCH | `/notifications/preferences` | 修改通知分组开关 | contract_only |
| POST | `/notifications/devices` | 注册或轮换本机推送令牌 | contract_only |
| POST | `/notifications/devices/{deviceRef}/unregister` | 注销本机推送登记 | contract_only |
| GET | `/notifications` | 获取交易和消息收件箱 | contract_only |
| POST | `/notifications/read` | 标记本人通知已读 | contract_only |
| GET | `/merchant/me` | 获取商家店铺授权 | contract_only |
| GET | `/merchant/stores/{storeId}` | 读取可编辑店铺资料 | contract_only |
| PUT | `/merchant/stores/{storeId}` | 编辑店铺全部展示资料 | contract_only |
| GET | `/merchant/stores/{storeId}/homepage` | 读取首页编排 | contract_only |
| PUT | `/merchant/stores/{storeId}/homepage` | 编辑轮播、网格、品牌和分类顺序 | contract_only |
| GET | `/merchant/stores/{storeId}/brands` | 管理品牌列表 | contract_only |
| POST | `/merchant/stores/{storeId}/brands` | 创建品牌 | contract_only |
| PUT | `/merchant/stores/{storeId}/brands/{entryId}` | 修改名称、图片或排序品牌 | contract_only |
| GET | `/merchant/stores/{storeId}/categories` | 管理商城分类列表 | contract_only |
| POST | `/merchant/stores/{storeId}/categories` | 创建商城分类 | contract_only |
| PUT | `/merchant/stores/{storeId}/categories/{entryId}` | 修改名称、图片或排序商城分类 | contract_only |
| GET | `/merchant/stores/{storeId}/delivery-templates` | 列出可选游戏邮箱模板 | contract_only |
| GET | `/merchant/stores/{storeId}/products` | 管理商品、草稿和上下架状态 | contract_only |
| POST | `/merchant/stores/{storeId}/products` | 新建商品完整草稿 | contract_only |
| GET | `/merchant/products/{productId}` | 读取预填编辑表单及已发布版本 | contract_only |
| PUT | `/merchant/products/{productId}` | 更新所有商品内容字段 | contract_only |
| POST | `/merchant/products/{productId}/publish` | 校验并发布商品草稿 | contract_only |
| POST | `/merchant/products/{productId}/unlist` | 下架官方商品 | contract_only |
| POST | `/merchant/products/{productId}/archive` | 归档已下架商品 | contract_only |
| POST | `/merchant/products/{productId}/stock-adjustments` | 调整实时可售库存并记录原因 | contract_only |
| GET | `/merchant/orders` | 查询有权限店铺的订单 | contract_only |
| GET | `/merchant/orders/{orderId}` | 查看官方订单与退款记录 | contract_only |
| POST | `/merchant/orders/{orderId}/delivery-retry` | 重试失败的游戏邮箱发放 | contract_only |
| POST | `/orders/{orderId}/interventions` | 申请平台介入订单争议 | contract_only |
| POST | `/commissions/{commissionId}/interventions` | 申请平台介入委托争议 | contract_only |
| GET | `/interventions/{caseId}` | 查询本人参与的介入申请 | contract_only |
| POST | `/interventions/{caseId}/evidence` | 补充本人介入说明和证据 | contract_only |
| POST | `/interventions/{caseId}/withdraw` | 撤回尚未裁决的介入申请 | contract_only |
| GET | `/admin/me` | 读取平台管理授权 | contract_only |
| GET | `/admin/interventions` | 管理员查询待处理争议 | contract_only |
| GET | `/admin/interventions/{caseId}` | 读取争议详情与权威交易快照 | contract_only |
| POST | `/admin/interventions/{caseId}/assign` | 领取争议处理任务 | contract_only |
| POST | `/admin/interventions/{caseId}/request-evidence` | 要求参与方补充说明 | contract_only |
| POST | `/admin/interventions/{caseId}/resolve` | 作出裁决并编排资金处理 | contract_only |
| GET | `/admin/audit-events` | 查询后台重要操作审计 | contract_only |
| GET | `/admin/stores/{storeId}/members` | 管理店铺成员授权 | contract_only |
| POST | `/admin/stores/{storeId}/members` | 授予或更新店铺成员权限 | contract_only |
| POST | `/admin/stores/{storeId}/members/{playerRef}/revoke` | 撤销店铺成员权限 | contract_only |
| GET | `/admin/announcements` | 管理公告草稿和已发布公告 | contract_only |
| POST | `/admin/announcements` | 创建官方公告草稿 | contract_only |
| PUT | `/admin/announcements/{announcementId}` | 编辑公告标题、摘要、封面和正文 | contract_only |
| POST | `/admin/announcements/{announcementId}/publish` | 发布公告并发送业务通知 | contract_only |
| POST | `/admin/announcements/{announcementId}/unpublish` | 撤下公告 | contract_only |
| POST | `/admin/releases/artifacts` | 上传并检查 APK 或资源更新包 | contract_only |
| GET | `/admin/releases/artifacts/{artifactRef}` | 查询更新包验证结果 | contract_only |
| GET | `/admin/releases` | 查询应用和资源发布记录 | contract_only |
| POST | `/admin/releases` | 创建更新发布草稿 | contract_only |
| PUT | `/admin/releases/{releaseId}` | 修改更新说明与兼容范围 | contract_only |
| POST | `/admin/releases/{releaseId}/publish` | 发布可供 App 检查的更新 | contract_only |
| POST | `/admin/releases/{releaseId}/withdraw` | 撤回问题更新的推荐 | contract_only |
| GET | `/app/appearance` | 读取当前玩家生效的远程外观配置 | contract_only |
| GET | `/admin/appearance/global` | 读取全局外观配置管理记录 | contract_only |
| PATCH | `/admin/appearance/global` | 修改或重置全局外观参数 | contract_only |
| GET | `/admin/appearance/players/{playerRef}` | 读取指定玩家的外观覆盖记录 | contract_only |
| PATCH | `/admin/appearance/players/{playerRef}` | 修改或重置指定玩家的外观参数 | contract_only |
