# 注册正版 UUID 与未验证账号改名验收

日期：2026-10-14。工作树 `.worktrees/uuid-name-sync`，分支 `xiwanzi/registration-profile-uuid`。

## 问题及修正

注册的正版查询已经得到 UUID，但旧流程只保存查询成功标记，未保存 UUID。没有进服验证的账号改名后，无法通过新版同 UUID 登录逻辑找回原账号。

新增 `profile_uuid` 留存注册查询结果，与表示已验证游戏所有权的 `server_uuid` 分开。注册验证码与十分钟缓存一并保留 UUID，最终注册提交继续复用已有证明。新游戏名查询到同一 UUID 后，原密码正确才事务更新该平台账号的名字和登录别名。未验证账号保持原验证状态，不发布游戏资产身份映射。已验证账号继续走既有名称同步。

## 本次验证

- `go test ./...`、`go vet ./...` 通过。
- 隔离 MariaDB 回归通过：`go test -tags=integration ./internal/httpapi ./integration -run 'Test(RegisteredProfile|Registration|AuthAudit|PlayerName|AdminGameID|MailIdentity|EmailVerification|AccountDeletion|PasswordChange|ConcurrentFirstLogins|GameOnlyTransferHistory)' -count=1`；HTTP 测试 169.910 秒、凭据竞争测试 7.688 秒。
- 新增五组集成测试覆盖无游戏服的新名登录、错密码不改名、原账号/昵称/会话保留、游戏权限未开放、邮箱无外部查询、重复 UUID 注册、旧名字被其他 UUID 使用、迟到与并发名字证明、改密竞争、错误游戏 UUID 拒绝、管理员修正及注销释放 UUID。
- 注册旧验证码缺 UUID 时补查；新验证码留存 UUID，缓存过期后最终提交不重复查询官方服务。
- 只读核对 59 个未删除账号：8 个已有游戏绑定，50 个当前名字查询成功，1 个旧名失效。失效名字通过历史通行申请 UUID 与官方新名 UUID 一致确认；59 个目标 UUID 无重复。尚未操作真实玩家密码或替玩家登录。

本机回执在忽略目录 `output/profile-uuid/`；原账号资料和凭据不提交到仓库。具体上线与回填结果另记部署文档。
