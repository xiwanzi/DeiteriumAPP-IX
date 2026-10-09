import React, { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { ImagePlus, Plus, X, Trash2, LoaderCircle, RefreshCw, BookmarkPlus } from "lucide-react";
import "./chat-media.css";

export const CHAT_EMOJI = ["😀", "😄", "😂", "🥹", "😊", "😍", "🥰", "😎", "🤔", "😭", "😮", "😴", "😅", "🙃", "🥲", "😡", "👍", "👎", "👏", "🙏", "🫶", "💪", "❤️", "💙", "✨", "🎉", "🔥", "💯", "👀", "🌿", "🌸", "🌙", "☀️", "🏡", "⛏️", "💎", "🎮", "🐱", "🐶", "🍉", "🍔", "☕", "✅", "❌", "💤", "🤝", "🎂", "🚀"];
const playing = new Set();
function useVisible(ref) {
  const [visible, setVisible] = useState(false);
  useEffect(() => { const o = new IntersectionObserver(([entry]) => setVisible(entry.isIntersecting), { threshold: .3 }); if (ref.current) o.observe(ref.current); return () => o.disconnect(); }, []);
  return visible;
}
function usePlayback(visible, enabled) {
  const key = useRef(Symbol()), [play, setPlay] = useState(false);
  useEffect(() => { if (!visible || !enabled) { playing.delete(key.current); setPlay(false); return; } const update = () => { const ok = !document.hidden && !matchMedia("(prefers-reduced-motion: reduce)").matches; if (!ok) playing.delete(key.current); else if (playing.size < 3) playing.add(key.current); setPlay(ok && playing.has(key.current)); }; update(); const timer = setInterval(update, 400); document.addEventListener("visibilitychange", update); return () => { clearInterval(timer); document.removeEventListener("visibilitychange", update); playing.delete(key.current); }; }, [visible, enabled]);
  return play;
}

export function ChatMediaBubble({ service, media, messageID, localURL, panel = false, previewOnly = false, onClick, onLongPress }) {
  const ref = useRef(), timer = useRef(), long = useRef(false), visible = useVisible(ref);
  const [thumb, setThumb] = useState(localURL), [full, setFull] = useState(null), [error, setError] = useState(""), [retry, setRetry] = useState(0);
  const play = usePlayback(visible, media?.animated && !panel && !previewOnly);
  useEffect(() => { let alive = true; if (localURL) { setThumb(localURL); return; } if (!visible || !media?.mediaId || !service) return; service.mediaURL(media, messageID, true).then(url => { if (alive) { setThumb(url); setError(""); } }).catch(e => { if (alive) setError(e.code === "MEDIA_EXPIRED" ? "图片已过期" : "点击重试"); }); return () => { alive = false; }; }, [service, media?.mediaId, messageID, localURL, visible, retry]);
  useEffect(() => { let alive = true; if (play && media?.mediaId && service) service.mediaURL(media, messageID).then(url => { if (alive) setFull(url); }).catch(() => {}); return () => { alive = false; }; }, [play, media?.mediaId, messageID, service]);
  const sticker = media?.type === "sticker" || panel, width = panel ? 76 : sticker ? 148 : Math.min(240, media?.width || 240), ratio = (media?.width || 4) / (media?.height || 3), height = panel ? 76 : Math.min(sticker ? 160 : 300, width / ratio);
  return <button ref={ref} type="button" className={`chat-media-tile ${sticker ? "sticker" : ""}`} style={{ width, height }} aria-label={sticker ? "查看贴纸" : "查看图片"}
    onPointerDown={() => { long.current = false; if (onLongPress) timer.current = setTimeout(() => { long.current = true; onLongPress(ref.current.getBoundingClientRect(), thumb, ref.current); }, 450); }} onPointerUp={() => clearTimeout(timer.current)} onPointerCancel={() => clearTimeout(timer.current)} onPointerLeave={() => clearTimeout(timer.current)}
    onClick={() => { if (long.current) return; if (!thumb && error) { setError(""); setRetry(n => n + 1); } else onClick?.(ref.current.getBoundingClientRect(), thumb, ref.current); }}>
    {thumb ? <img src={play && full ? full : thumb} alt={sticker ? "贴纸" : "聊天图片"} draggable="false" style={{ objectFit: sticker ? "contain" : "cover", objectPosition: "top" }} /> : <span className="chat-media-placeholder">{error || <LoaderCircle size={18} className="spin" />}</span>}
    {!panel && media?.height > (media?.width || 1) * 2.5 && <span className="chat-media-label">长图</span>}
    {!panel && media?.animated && !play && <span className="chat-media-label">GIF</span>}
  </button>;
}

export function ChatPictureButton({ service, reply, importing = false, disabled = false, onSent }) {
  const input = useRef(), [selected, setSelected] = useState(null), [error, setError] = useState(""), [busy, setBusy] = useState(false);
  useEffect(() => () => { if (selected?.url) URL.revokeObjectURL(selected.url); }, [selected?.url]);
  const choose = e => { const file = e.target.files?.[0]; e.target.value = ""; if (!file) return; if (file.size > 20 * 1024 * 1024) { setError("请选择 20MB 以内的图片。"); return; } setError(""); setSelected({ file, url: URL.createObjectURL(file) }); };
  return <><input ref={input} type="file" accept="image/jpeg,image/png,image/webp,image/gif" hidden onChange={choose} />
    <button type="button" className="chat-picture-button" disabled={disabled} aria-label={importing ? "添加贴纸" : "发送图片"} onClick={() => input.current.click()}>{importing ? <Plus size={20} /> : <ImagePlus size={20} />}{importing && "添加"}</button>
    {error && <span className="composer-error" role="alert">{error}</span>}
    {selected && createPortal(<div className="chat-image-selection" role="dialog" aria-modal="true" aria-label={importing ? "添加贴纸" : "发送图片"} onKeyDown={e => { if (e.key === "Escape") { e.stopPropagation(); if (!busy) setSelected(null); } }}>
      <div className="chat-selection-card"><header><strong>{importing ? "添加贴纸" : "发送图片"}</strong><button type="button" aria-label="取消" disabled={busy} onClick={() => setSelected(null)}><X /></button></header><img src={selected.url} alt="待发送图片" onLoad={e => { const { naturalWidth: width, naturalHeight: height } = e.currentTarget; setSelected(value => value ? { ...value, width, height } : value); }} />{error && <p className="composer-error" role="alert">{error}</p>}<footer><button type="button" disabled={busy} onClick={() => setSelected(null)}>取消</button><button type="button" className="primary" autoFocus disabled={busy || !selected.width} onClick={async () => { setBusy(true); try { await service.enqueue(selected.file, reply, null, importing, selected); setSelected(null); onSent?.(); } catch (e) { setError(e.message); } finally { setBusy(false); } }}>{importing ? "添加" : "发送"}</button></footer></div>
    </div>, document.body)}</>;
}

export function ChatExpressionPanel({ service, state, onEmoji, onSticker, onPreview }) {
  const [tab, setTab] = useState("emoji"), [manage, setManage] = useState(false), [error, setError] = useState("");
  return <div className="chat-expression-panel"><div className="chat-expression-tabs"><button type="button" className={tab === "emoji" ? "selected" : ""} onClick={() => setTab("emoji")}>Emoji</button><button type="button" className={tab === "sticker" ? "selected" : ""} onClick={() => setTab("sticker")}>贴纸 <small>{state.stickers.length}/10</small></button>{tab === "sticker" && <button type="button" className="manage" onClick={() => setManage(!manage)}>{manage ? "完成" : "管理"}</button>}</div>
    {tab === "emoji" ? <div className="chat-emoji-grid">{CHAT_EMOJI.map(e => <button type="button" key={e} onMouseDown={e => e.preventDefault()} onClick={() => onEmoji(e)}>{e}</button>)}</div> : <>
      <div className="chat-sticker-grid">{state.stickers.map(media => <div className="chat-sticker-cell" key={media.mediaId}><ChatMediaBubble service={service} media={media} panel onClick={(rect, thumb) => manage ? onPreview(media, rect, thumb) : onSticker(media)} onLongPress={(rect, thumb) => onPreview(media, rect, thumb)} />{manage && <button className="chat-sticker-delete" type="button" aria-label="删除贴纸" onClick={async () => { try { await service.deleteSticker(media); } catch (e) { setError(e.message); } }}><Trash2 size={14} /></button>}</div>)}<ChatPictureButton service={service} importing disabled={state.stickers.length >= 10 || state.jobs.some(j => j.importing && j.status !== "failed")} /></div>
      <p className="chat-expression-hint">{state.stickers.length >= 10 ? "贴纸已满，删除已有贴纸后可继续添加。" : "添加自己的图片或 GIF，长按预览。"}</p>
      {state.jobs.filter(j => j.importing).map(job => <div className="chat-import-status" key={job.id}>{job.status === "failed" ? <><span>{job.error}</span><button type="button" onClick={() => service.retry(job.id)}>重试</button><button type="button" onClick={() => service.dismiss(job.id)}>移除</button></> : <><LoaderCircle className="spin" size={15} />正在添加…</>}</div>)}
    </>}{(error || state.error) && <p className="composer-error" role="alert">{error || state.error}</p>}</div>;
}

export function ChatImageViewer({ service, media, messageID, preview, rect, source, close }) {
  const image = useRef(), shell = useRef(), pointers = useRef(new Map()), gesture = useRef(), transform = useRef({ scale: 1, x: 0, y: 0 });
  const [url, setURL] = useState(preview), [error, setError] = useState(""), [notice, setNotice] = useState("");
  const vw = innerWidth, vh = innerHeight, long = media.height > media.width * 2.5;
  const base = Math.min(1, (vw - 32) / media.width, long ? Infinity : (vh - 112) / media.height), width = media.width * base, height = media.height * base;
  const clamp = (v, low, high) => Math.max(low, Math.min(high, v));
  const paint = (next, dismiss = false) => { next.scale = clamp(next.scale, 1, 4); const bx = Math.max(0, (width * next.scale - vw + 24) / 2), by = Math.max(0, (height * next.scale - vh + 112) / 2); next.x = clamp(next.x, -bx, bx); if (!dismiss) next.y = clamp(next.y, -by, by); transform.current = next; if (image.current) image.current.style.transform = `translate(${next.x}px,${next.y}px) scale(${next.scale})`; };
  const dismiss = () => {
    if (!shell.current || matchMedia("(prefers-reduced-motion: reduce)").matches) { close(); return; }
    const bounds = source?.isConnected ? source.getBoundingClientRect() : null;
    if (image.current && bounds && bounds.bottom > 0 && bounds.top < innerHeight) image.current.animate([{ transform: image.current.style.transform || "translate(0,0) scale(1)" }, { transform: `translate(${bounds.left + bounds.width / 2 - vw / 2}px,${bounds.top + bounds.height / 2 - vh / 2}px) scale(${bounds.width / width})` }], { duration: 200, fill: "forwards", easing: "cubic-bezier(.2,.75,.25,1)" });
    shell.current.animate([{ opacity: 1 }, { opacity: 0 }], { duration: 200, fill: "forwards" }).finished.then(close);
  };
  useEffect(() => {
    const previous = document.activeElement, overflow = document.body.style.overflow; document.body.style.overflow = "hidden"; shell.current?.focus();
    const key = e => { if (e.key === "Escape") { e.stopPropagation(); dismiss(); } if (e.key === "Tab") { const buttons = [...shell.current.querySelectorAll("button:not(:disabled)")]; const pos = buttons.indexOf(document.activeElement); e.preventDefault(); buttons[(pos + (e.shiftKey ? buttons.length - 1 : 1)) % buttons.length]?.focus(); } }; document.addEventListener("keydown", key);
    if (long) paint({ scale: 1, x: 0, y: Math.max(0, (height - vh + 112) / 2) });
    else if (rect && image.current && !matchMedia("(prefers-reduced-motion: reduce)").matches) { const dx = rect.left + rect.width / 2 - vw / 2, dy = rect.top + rect.height / 2 - vh / 2, scale = rect.width / width; image.current.animate([{ transform: `translate(${dx}px,${dy}px) scale(${scale})`, borderRadius: "16px" }, { transform: "translate(0,0) scale(1)", borderRadius: "0" }], { duration: 250, easing: "cubic-bezier(.2,.75,.25,1)" }); }
    let alive = true; service.mediaURL(media, messageID).then(value => { if (alive) setURL(value); }).catch(e => { if (alive) setError(e.code === "MEDIA_EXPIRED" ? "清晰图片已过期，当前显示本机预览" : "清晰图片暂不可用"); });
    return () => { alive = false; document.body.style.overflow = overflow; document.removeEventListener("keydown", key); previous?.focus?.(); };
  }, []);
  const snapshot = () => { const p = [...pointers.current.values()]; return { ...transform.current, points: p, distance: p.length === 2 ? Math.hypot(p[1].x - p[0].x, p[1].y - p[0].y) : 0, center: p.length === 2 ? { x: (p[0].x + p[1].x) / 2 - vw / 2, y: (p[0].y + p[1].y) / 2 - vh / 2 } : null }; };
  return createPortal(<div className="chat-image-viewer" role="dialog" aria-modal="true" aria-label="查看图片" ref={shell} tabIndex={-1}>
    <header><button type="button" aria-label="关闭图片" onClick={dismiss}><X /></button><span>{media.type === "sticker" ? "贴纸" : "图片"}</span>{messageID && <button type="button" aria-label="添加到贴纸" onClick={async () => { try { await service.collect(media, messageID); setNotice("已加入添加队列"); } catch (e) { setNotice(e.message); } }}><BookmarkPlus /></button>}</header>
    <div className="chat-image-stage" onDoubleClick={e => { const t = transform.current, scale = t.scale > 1 ? 1 : 3; paint({ scale, x: scale === 1 ? 0 : -(e.clientX - vw / 2) * 2, y: scale === 1 ? 0 : -(e.clientY - vh / 2) * 2 }); }}
      onWheel={e => { const t = transform.current; if (long && !e.ctrlKey && t.scale === 1) paint({ ...t, y: t.y - e.deltaY }); else { const scale = clamp(t.scale * Math.exp(-e.deltaY * .002), 1, 4), ratio = scale / t.scale, x = e.clientX - vw / 2, y = e.clientY - vh / 2; paint({ scale, x: x - (x - t.x) * ratio, y: y - (y - t.y) * ratio }); } }}
      onPointerDown={e => { e.currentTarget.setPointerCapture(e.pointerId); pointers.current.set(e.pointerId, { x: e.clientX, y: e.clientY }); gesture.current = snapshot(); }}
      onPointerMove={e => { if (!pointers.current.has(e.pointerId)) return; pointers.current.set(e.pointerId, { x: e.clientX, y: e.clientY }); const start = gesture.current, current = snapshot(); if (pointers.current.size === 2 && start.distance) { const scale = clamp(start.scale * current.distance / start.distance, 1, 4), ratio = scale / start.scale; paint({ scale, x: current.center.x - (start.center.x - start.x) * ratio, y: current.center.y - (start.center.y - start.y) * ratio }); } else if (start.points.length === 1) paint({ scale: start.scale, x: start.x + e.clientX - start.points[0].x, y: start.y + e.clientY - start.points[0].y }, !long && start.scale === 1); }}
      onPointerUp={e => { if (pointers.current.size === 1 && !long && transform.current.scale === 1 && Math.abs(transform.current.y) > 120) dismiss(); else paint({ ...transform.current }); pointers.current.delete(e.pointerId); gesture.current = snapshot(); }} onPointerCancel={e => { pointers.current.delete(e.pointerId); paint({ ...transform.current }); gesture.current = snapshot(); }}>
      {url ? <img ref={image} src={url} alt="聊天图片" draggable="false" style={{ width, height }} /> : <LoaderCircle className="spin" />}
    </div><footer>{notice || error || "双指缩放 · 双击放大"}</footer>
  </div>, document.body);
}
