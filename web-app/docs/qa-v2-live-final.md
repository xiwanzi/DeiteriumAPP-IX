# 网页 2.0.0 最终部署组合只读验收

时间：2026-09-08 13:47（Asia/Singapore，UTC 05:47）。入口：[当前测试网站](https://47.103.99.34)。

本次直接加载公网部署的网页 `4de1ee7`，浏览器 document 为 HTTPS 200，实际 script 为 `/assets/index-tMukH1-X.js`。检查前已移除之前的本地 HTML/JS/CSS 替换；没有以本地候选页面代替线上文件。后端由主任务部署并核验为 `00fbdac`，包含 013 交易与 017 审计。

26 项浏览器/接口检查通过，0 JavaScript 运行异常。业务 HTTP 写入拦截记录为 0；没有发起付款、余额刷新 POST、消息发送、资料编辑、店铺创建或平台裁决。

| 对象 | 线上结果 |
| --- | --- |
| 订单 | `/orders` 刷新直达成功；`GET /api/v1/orders` 为 200 和空数组，界面显示“还没有订单”；390px 无横向溢出 |
| 委托 | 大厅、我发布的、我接取的均显示真实空态，公开列表 GET 200 和空数组 |
| 商店管理 | 当前账号尚未分配店铺；`GET /api/v1/merchant/me` 返回 403 `STORE_NOT_CONFIGURED`。界面显示服务端原说明，并保留平台管理员的创建入口；未创建商店 |
| 平台介入 | 管理页可打开；列表 GET 200 和空数组，界面显示“暂无介入案件” |
| 管理审计 | GET 200，读取到 13 条现有记录；页面显示实际最新动作。每条只有 eventId、actorId、action、resourceId、createdAt 五个字段 |
| 钱包 | 余额 GET 200，界面数值与服务端十进制字符串一致；原 `transfer_acad3a87845dcba74310068ae5ab734d791a7de1` 仍为 success、1.00，账单显示原收款方 luoyinwuchen1 |
| 公共聊天 | 实时连接正常；原 `msg_41ec45fa0ca4e1db827ab93ac9906f73` 在历史中恰好一条，正文与原验收消息一致，消息区恰好一个气泡。会话列表中的摘要不算第二条消息 |

复现脚本：[v2-live-final-readonly.cjs](../scripts/qa/v2-live-final-readonly.cjs)。需主任务提供短期受控浏览器会话；脚本和文档均不包含凭据。脚本允许真实 GET 通路，并在浏览器侧阻止其他业务 HTTP 方法。

截图位于被忽略的 `output/playwright/v2-final-live-audit.png` 与 `v2-final-live-chat.png`，已实际查看核对。该检查证明当前部署组合的读取与展示链路；没有新增资金或物品发放测试，Minecraft 最后换包由主任务单独记录。

本次网页静态归档保持为 `deuterium-web-2.0.0-4de1ee7-static.tar.gz`，SHA256 为 `C92BC90EDFAB96A76138BF416C1C82FC65C9C00111D97A9B8014A9B60DDDBB76`。本报告及复现脚本不会改变已部署静态包。
