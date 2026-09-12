# Deuterium IX 通行申请站

独立静态站点，无 npm 依赖或构建步骤。使用 `index.html`、`styles.css`、`community.css`、`app.js` 和 `assets/` 发布，`/api/v1/admission/` 同源代理到 Go 服务。正式配置禁止模拟成功。

界面保持用户确认的终末地风格；背景是用户提供的 DSA 方块火箭发射图，版本标直接使用 Deuterium IX 既有徽章。字体沿用原型中的 HarmonyOS Sans / Novecento 资源，视觉来源见前端素材记录，不使用设计 skills 重新生成。

流程：身份及公约 → 可选游玩意向 → 核对 → 实际提交回执。查询凭证由 Web Crypto 生成，不放进 URL。Cookie 不随公开请求发送。网络结果未知时保存当前标签页请求并重试同一凭证，查询凭证单独保存在本浏览器。

前端固定公约版本应与 Go `AdmissionCovenantVersion` 一致；版本变化时要求刷新阅读。真实QQ群核验由后台审批人确认。入群邀请为已核对的 `https://qm.qq.com/q/HROB9FDSYE`，群号 490579956。

本轮只发布独立网页，不更改 Android App。接口和部署说明见 `../docs/contracts/admission-v1.md`。
