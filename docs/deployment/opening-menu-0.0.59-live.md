# 进服设置页返回黑幕修复上线

**2026-10-13 17:03（UTC+8）资源0.0.59-dev发布，17:04公网核验通过。** 开屏源码 [DeLauncher PR #6](https://github.com/Deuterium-IX/DeLauncher/pull/6)，合并 `dadd8c309c4b4ddb1e68e6829932d430e66b3bf6`。

用户反馈的是“进服已显示设置/菜单，点击回到游戏后重新黑屏渐出”。旧0.0.42失焦修复仍在，0.0.58与用户本机主开屏JAR哈希一致，没有回滚。遗漏在于菜单出现在首次渐出开始前时只隐藏开屏，没有结束连接过渡状态；关闭菜单又重新展示开屏。

本次已有世界与玩家时显示正常游戏菜单即结束本次交接，并覆盖菜单先于玩家/世界建立的情况。隔离真实客户端与回环服务器验证：旧版本复现，修复版菜单返回无重播并可移动；不打开菜单的正常进服仍有一次完整过渡。5项计时与2项early测试通过。

仅替换主开屏JAR和资源显示版本。JAR仍名为 `deuterium-opening-neoforge-1.21.1-0.0.22-reveal-fix.jar`，38,071字节，SHA-256 `9be4375b9fa71e7cffcf8ae054432914d26383b0710fa9f8e03930aabf43a086`；与旧包逐项比较只改变 `OpeningScreen.class`。

真实启动器0.0.58→0.0.59及重复无操作检查通过，聊天0.1.10、关闭世界内路标的配置、字号和其他模组保留。公网TAR全量哈希、长度与206元数据通过；TAR42,496字节，SHA-256 `ceffcc8e6aaa990210ff20ed53b06d69443556d3f0c18dc82d03b312ebab647a`。OSS同步 `ls_c337ff16ff9e6bb981f615675b036a35` 成功，临时发布会话已撤销，保留原73条资源版本。

App2.0.27、Web2.0.38、Go及配置保持；61账号、47订单、5委托、3,085在售商品、12工单、2注销记录保持。游戏服文件、配置及PID保持，三常驻节点在线、MEK停止，没有服务器重启。

备份 `/var/backups/deuterium/opening-menu-client-0.0.59`；回退仅涉及这两个客户端文件与资源索引，不回退数据库。源码维护位于 `C:/Mod/DeuteriumIX/work/DLauncher/.worktrees/opening-menu-reveal`；[详细源码验收](https://github.com/Deuterium-IX/DeLauncher/blob/main/docs/QA-OPENING-MENU-20261013.md)。发布证据在本工作区 `output/opening-menu-release/`。

同轮[旅者之乡雾效核查](../qa/travelers-home-fog-20261013.md)仅为来源研究，没有加入第三方光影或模组。
