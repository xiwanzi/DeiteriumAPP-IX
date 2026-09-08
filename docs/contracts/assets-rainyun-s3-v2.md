# 测试对象存储：雨云 S3 直传

2026-09-08 用户明确提供雨云对象存储供 2.0.0 测试。API endpoint 为 `https://cn-nb2.rains3.com`，bucket `xiwanzi`，仅使用 `deuterium-test/` 前缀。密钥只在受限后端配置，不能进入 App/Web、日志或交付包。本供应商选择覆盖此前仅支持 `ALIYUN_OSS` POST 的授权对象；业务 assetId/上传会话/权限边界不变。

## 实际验证

已用用户提供凭据验证 HeadBucket。专用随机探测对象预签名 PUT 首次 200、相同授权再次写入 412，精确 size/MD5 与 HEAD 一致；匿名 GET 为 403。测试结束仅删除本次探测对象。Bucket 目前未启用版本控制，不能靠版本 ID 做不可变保证，因此每次授权必须签入 `If-None-Match: *` 并使用唯一对象 key。

## 客户端协议

沿用 `/api/v1/assets/uploads` 创建、`/{uploadId}` 查询、`/renew` 续签、`/complete` 完成校验和 `/assets/{assetId}` 读取。创建请求仍为 JSON 元信息：clientRequestId、purpose、businessType、businessRef（按业务要求）、fileName、contentType、sizeBytes、contentMd5、altText。

`authorization` 新增供应商分支：

```json
{
  "provider": "RAINYUN_S3",
  "method": "PUT",
  "uploadUrl": "https://xiwanzi.cn-nb2.rains3.com/<signed-object>",
  "signedHeaders": {
    "Content-Type": "image/png",
    "Content-MD5": "<final-file-base64-md5>",
    "If-None-Match": "*"
  },
  "expiresAt": "<UTC timestamp>",
  "sizeBytes": 1024
}
```

按返回 method 上传原始文件 bytes；不是 multipart。逐项保留 signedHeaders，Host/Content-Length 由平台网络栈设置（实际字节数必须一致），不得附带 App Bearer/Cookie。浏览器 CORS 仅允许当前部署站点 Origin；Android 原生 HTTP 不受浏览器 CORS 限制。不得记录完整预签名 URL。

收到 S3 成功后调用 complete；只有服务器核验对象长度、MD5、实际图片类型、尺寸和字节 SHA-256 后进入 READY，再将 assetId 绑定到资料/商品/委托。授权和资产 READY 是不同状态。未知传输结果先查询/complete，不换 key 重复提交；412 表示对象已存在，应按原 uploadId 核验。

会话上限 1 小时，单次签名不超过 5 分钟。续签不会延长会话寿命。每个账号每小时最多 100 个新的授权（同请求重试不占新名额）；解码验收额外限制图片总像素不超过 1600 万，避免压缩炸弹。头像/普通业务图片向有权限的已登录用户返回短期 GET URL，争议凭证仅对其有权参与者开放。客户端以 assetId 保存关系，URL 到期重新查询，不将预签名地址持久当成永久 CDN URL。

公告后台新增 `purpose=ANNOUNCEMENT_MEDIA`、`businessType=ANNOUNCEMENT`，仅显式公告管理权限可申请。它不借用商家店铺权限。争议证据须在对应订单/委托/介入已实现可信成员校验后开放；当前缺少这一业务校验时明确拒绝，不因知道引用 ID 就签发。

资料/商品/公告将绑定引用写入同一数据库事务；已有绑定禁止资产软删除。多管理员修改同一业务可保留其既有绑定的旧资产，新增资产必须属于当前操作账号；公开读取先核验外层业务可见性，再由数据库保存的所有者解析图片。下载签名不作为新的业务所有权证明。

供应商凭据配置键：`DEUTERIUM_S3_ENDPOINT`、`DEUTERIUM_S3_REGION`、`DEUTERIUM_S3_BUCKET`、`DEUTERIUM_S3_PREFIX`、`DEUTERIUM_S3_ACCESS_KEY`、`DEUTERIUM_S3_SECRET_KEY`。当前实际连通与 CORS 记录在部署文档；未配置时上传明确不可用，不产生模拟资产。

## 2.0.1 增量

图片可本地缓存，历史引用不再永久阻止对象回收，资产前缀迁移保留 assetId。具体响应字段、30 天回收规则及恢复边界见 [2.0.1 图片与生命周期契约](images-cache-lifecycle-v201.md)。
