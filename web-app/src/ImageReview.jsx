import React, { useCallback, useEffect, useRef, useState } from "react";
import { AlertTriangle, ChevronLeft, ChevronRight, RefreshCw, ScanEye, Trash2, Settings2 } from "lucide-react";
import { Button, Empty, Modal, PageHead } from "./components.jsx";
import "./image-review.css";
import ImageReviewSettings from "./ImageReviewSettings.jsx";

const endpoint = "/api/v1/admin/image-review";
const time = value => value ? new Date(value).toLocaleString("zh-CN", { hour12: false }) : "暂无";
const person = value => value.displayName || value.gameId || "账号已不可用";
const channels = { world: "世界频道", nearby: "附近频道", team: "团队频道", private: "私聊" };
const purposes = { CHAT_MEDIA: "聊天图片 / 贴纸上传", AVATAR: "头像", STORE_MEDIA: "商店图片", PRODUCT_MEDIA: "商品图片", MARKET_PHOTO: "市场图片", COMMISSION_COVER: "委托封面", DISPUTE_EVIDENCE: "交易凭证", ANNOUNCEMENT_MEDIA: "公告图片" };

function Evidence({ item, large = false }) {
  const [failed, setFailed] = useState(false);
  return failed ? <div className="review-image-missing"><AlertTriangle size={24} /><span>图片读取失败，请刷新重试</span></div> : <img className={large ? "review-image-large" : "review-image"} src={`${endpoint}/${item.id}/image`} alt="待复核图片" loading={large ? "eager" : "lazy"} referrerPolicy="no-referrer" onError={() => setFailed(true)} />;
}
function Scores({ item }) {
  return <div className="review-scores">{[["r18", "R18"], ["politics", "政治内容"], ["promotion", "宣传导流"]].map(([key, label]) => <span key={key}><span>{label}</span><strong className={item.result?.scores[key] > (item.thresholdUsed ?? 50) ? "review-score-high" : ""}>{item.result?.scores[key] ?? "—"}</strong></span>)}</div>;
}

export default function ImageReview({ client, navigate, search = "" }) {
  const settings = new URLSearchParams(search).get("view") === "settings";
  return settings ? <ImageReviewSettings client={client} onBack={() => navigate("/admin?section=image-review")} /> : <ImageReviewList client={client} onSettings={() => navigate("/admin?section=image-review&view=settings")} />;
}
function ImageReviewList({ client, onSettings }) {
  const [items, setItems] = useState([]), [summary, setSummary] = useState(null), [page, setPage] = useState(0), [more, setMore] = useState(false);
  const [loading, setLoading] = useState(true), [refreshing, setRefreshing] = useState(false), [error, setError] = useState(""), [notice, setNotice] = useState("");
  const [detail, setDetail] = useState(null), [detailLoading, setDetailLoading] = useState(false), [detailError, setDetailError] = useState("");
  const [remove, setRemove] = useState(null), [deleting, setDeleting] = useState(false), [deleteError, setDeleteError] = useState("");
  const alive = useRef(true), listing = useRef(null), details = useRef(null), busy = useRef(false);
  const load = useCallback(async () => {
    listing.current?.abort(); const controller = new AbortController(); listing.current = controller;
    setRefreshing(true);
    try {
      const { data } = await client.request(`${endpoint}?page=${page}`, { signal: controller.signal });
      if (!alive.current || controller.signal.aborted) return;
      setItems(data.items); setSummary(data.summary); setMore(data.hasMore); setError("");
    } catch (e) { if (alive.current && !controller.signal.aborted) setError(e.message); }
    finally { if (alive.current && !controller.signal.aborted) { setLoading(false); setRefreshing(false); } }
  }, [client, page]);
  useEffect(() => { alive.current = true; return () => { alive.current = false; listing.current?.abort(); details.current?.abort(); }; }, []);
  useEffect(() => { setLoading(true); load(); const timer = setInterval(() => { if (!document.hidden && !busy.current) load(); }, 30000); return () => { clearInterval(timer); listing.current?.abort(); }; }, [load]);
  const open = async item => {
    details.current?.abort(); const controller = new AbortController(); details.current = controller;
    setDetail(item); setDetailLoading(true); setDetailError("");
    try {
      const { data } = await client.request(`${endpoint}/${item.id}`, { signal: controller.signal });
      if (alive.current && !controller.signal.aborted) setDetail(data);
    } catch (e) { if (alive.current && !controller.signal.aborted) setDetailError(e.message); }
    finally { if (alive.current && !controller.signal.aborted) setDetailLoading(false); }
  };
  const closeDetail = () => { details.current?.abort(); setDetail(null); };
  const deleteRecord = async () => {
    if (busy.current || !remove) return;
    busy.current = true; setDeleting(true); setDeleteError(""); const id = remove.id;
    try {
      const { data } = await client.request(`${endpoint}/${id}`, { method: "DELETE" });
      if (!alive.current) return;
      setRemove(null); if (detail?.id === id) closeDetail();
      setNotice(data.state === "DELETING" ? "记录已移出列表，证据正在后台清理。" : "审核记录及证据图片已删除。");
      if (items.length === 1 && page > 0) setPage(page - 1); else await load();
    } catch (e) { if (alive.current) setDeleteError(e.message); }
    finally { busy.current = false; if (alive.current) setDeleting(false); }
  };
  const askDelete = item => { setRemove(item); setDeleteError(""); };
  const stale = summary && (!summary.lastScanAt || Date.now() - new Date(summary.lastScanAt).getTime() > 300000 || !summary.lastWorkAt || Date.now() - new Date(summary.lastWorkAt).getTime() > 300000);
  return <section className="image-review-page">
    <PageHead title="图片审核" subtitle={`仅审核启用后新上传的静态聊天图片，当前阈值 ${summary?.threshold ?? 50} 分。`}>
      <Button secondary onClick={onSettings}><Settings2 size={16} />设置</Button>
      <Button secondary disabled={refreshing || deleting} onClick={() => { setNotice(""); load(); }}><RefreshCw size={16} />{refreshing ? "刷新中…" : "刷新"}</Button>
    </PageHead>
    {error && <p role="alert" className="form-error">{error}</p>}
    {notice && <p role="status" className="review-notice">{notice}</p>}
    {summary && <>
      <div className="review-overview">
        <div><span>待复核</span><strong>{summary.flagged}</strong><small>按风险分数排序</small></div>
        <div><span>低于或等于阈值</span><strong>{summary.clear}</strong><small>不额外保留图片</small></div>
        <div><span>等待审核</span><strong>{summary.pending}</strong><small>{summary.retrying ? `${summary.retrying} 项等待重试或保存证据` : "新上传图片自动加入"}</small></div>
        <div><span>已跳过动图</span><strong>{summary.skipped}</strong><small>{summary.unavailable ? `${summary.unavailable} 项原图已失效或不支持` : "本轮只检查静态图片"}</small></div>
      </div>
      <div className="review-service"><span><ScanEye size={16} />{summary.modelLabel}</span><span>最近扫描：{time(summary.lastScanAt)}</span>{summary.deleting > 0 && <span>{summary.deleting} 份证据清理中</span>}</div>
      {(stale || summary.scanError || summary.retrying > 0) && <p className="review-warning" role="status"><AlertTriangle size={17} />{summary.scanError || stale ? "审核服务暂未正常更新，已有记录可继续查看。" : "部分图片正在重试或保存证据，尚未计入低风险结果。"}</p>}
    </>}
    {loading ? <p className="review-loading" role="status">正在读取审核记录…</p> : items.length === 0 ? <Empty title={error ? "暂未读取到审核记录" : "暂无需要复核的图片"} text={error ? "请刷新重试。" : `超过 ${summary?.threshold ?? 50} 分且证据保存成功的图片会出现在这里。`} /> : <div className="review-grid">
      {items.map(item => <article className="review-card" key={item.id}>
        <button className="review-preview" onClick={() => open(item)} aria-label={`查看 ${person(item)} 的图片审核详情`}><Evidence item={item} /><span className="review-risk">{item.score}<small> / 100</small></span></button>
        <div className="review-card-body"><Scores item={item} /><p className="review-summary">{item.result?.summary}</p><div className="review-uploader"><strong>{person(item)}</strong>{item.gameId && item.gameId !== person(item) && <span>{item.gameId}</span>}</div><p className="review-meta">上传于 {time(item.uploadedAt)}</p><p className="review-meta">{purposes[item.purpose] || "业务图片"}</p><div className="review-card-actions"><Button secondary onClick={() => open(item)}>查看详情</Button><Button danger onClick={() => askDelete(item)}><Trash2 size={15} />删除记录</Button></div></div>
      </article>)}
    </div>}
    {(page > 0 || more) && <div className="review-pagination"><Button secondary disabled={page === 0 || refreshing} onClick={() => setPage(page - 1)}><ChevronLeft size={16} />上一页</Button><span>第 {page + 1} 页</span><Button secondary disabled={!more || refreshing} onClick={() => setPage(page + 1)}>下一页<ChevronRight size={16} /></Button></div>}
    {detail && <Modal title={`图片审核 · ${detail.score} 分`} close={closeDetail}>
      <div className="review-detail"><Evidence key={detail.id} item={detail} large /><Scores item={detail} /><p>{detail.result?.summary}</p><ul className="review-reasons">{detail.result?.reasons.map((reason, n) => <li key={n}>{reason}</li>)}</ul>
        <dl className="review-facts"><dt>上传账号</dt><dd>{person(detail)}{detail.gameId && ` · ${detail.gameId}`}</dd><dt>账号标识</dt><dd>{detail.playerRef || detail.uploaderId || "已不可用"}</dd><dt>上传时间</dt><dd>{time(detail.uploadedAt)}</dd><dt>图片来源</dt><dd>{purposes[detail.purpose] || detail.purpose}</dd><dt>当时阈值</dt><dd>{detail.thresholdUsed ?? 50} 分</dd><dt>审核时间</dt><dd>{time(detail.reviewedAt)}</dd></dl>
        <h3>发送记录</h3>{detailLoading ? <p role="status">正在读取发送记录…</p> : detailError ? <p role="alert" className="form-error">{detailError}<Button secondary onClick={() => open(detail)}>重试</Button></p> : detail.events?.length ? <div className="review-events">{detail.events.map(event => <div key={event.id}><strong>{person(event)}</strong><span>{channels[event.channel] || event.channel}</span><time>{time(event.sentAt)}</time></div>)}{detail.eventsTruncated && <p className="review-meta">此处展示最近 100 条发送记录。</p>}</div> : <p className="review-meta">暂未发现关联聊天发送记录；上传不代表已发送。</p>}
        <p className="review-footnote">分数用于人工复核，不代表系统已对玩家作出处罚。</p><div className="button-row"><Button secondary onClick={closeDetail}>关闭</Button><Button danger onClick={() => askDelete(detail)}>删除记录</Button></div>
      </div>
    </Modal>}
    {remove && <Modal title="删除审核记录" close={() => { if (!deleting) setRemove(null); }}><p>删除这条审核记录及单独保存的证据图片，释放审核存储空间。</p><p className="review-meta">玩家原有聊天图片、贴纸及其他业务图片不受影响。</p>{deleteError && <p role="alert" className="form-error">{deleteError}</p>}<div className="button-row"><Button secondary disabled={deleting} onClick={() => setRemove(null)}>取消</Button><Button danger disabled={deleting} onClick={deleteRecord}>{deleting ? "删除中…" : "确认删除"}</Button></div></Modal>}
  </section>;
}
