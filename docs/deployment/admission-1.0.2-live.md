# 申请站 1.0.2 配色优化上线

2026-09-13 **20:48:51（UTC+8）**，已确认的主页面和弹窗配色优化上线。[PR #41](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/41) 已合并，源码 `3541a73604f92a6c16226d2b332427c668176966`。

入口：[通行许可申请](https://47.103.99.34/admission/)。主页面保留“下一步”的原始亮黄色，收敛欢迎标签、箭头、定位符、标题阴影、卡片边缘和步骤下划线，说明区改为中性灰。弹窗查询按钮使用深色底与小面积黄标，返回按钮使用轻量描边。

本次只替换申请站 `community.css`，其余 HTML、JS、字体、图片逐文件哈希保持一致。未重启服务、操作业务数据库或改变申请/审核/邮件流程；后台和其他组件沿用各自当前版本。

发布前完成桌面、手机同尺寸前后截图与弹窗返回验证；发布后从公网读取 CSS 核对 SHA-256，浏览器核对主按钮颜色、侧边装饰、弹窗主次按钮及返回操作。此次没有提交生产申请。入口继续返回 `Cache-Control: no-cache`，刷新后获取新版样式。

- 当前目录：`/var/www/deuterium-admission/releases/admission-1.0.2-3541a73604f9/site`
- 前版保留：`/var/www/deuterium-admission/releases/admission-1.0.1-57878e526653/site`
- 备份及发布记录：`/var/backups/deuterium/admission-accent-3541a73604f9`
- CSS SHA-256：`2c89d2c3d9b55274752577b6c5b7f34ae8bea2889ba3531ef4d91d60efd8fa2e`
- 完整静态包 SHA-256：`f6928816f94b916d00eeb6d792f3034257f9d4f21d478ff600a5b53a89bb7d3c`

采用静态目录原子切换，异常时仅切回前版站点目录，不需要恢复任何数据库。[发布证据](artifacts/admission-v102/published.json)
