import React, { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { X, Image, LoaderCircle } from "lucide-react";
import { stickerRadius } from "./chat-expressions.js";

export default function ChatAttachmentPreview({ selected, importing, busy, error, onReady, onChooseAgain, onSend, onClose }) {
  const sheet = useRef(), dialog = useRef(), drag = useRef(), moved = useRef(false), closeTimer = useRef();
  const [height, setHeight] = useState(null), [dragging, setDragging] = useState(false), [closing, setClosing] = useState(false);
  const [viewport, setViewport] = useState(() => window.visualViewport?.height || innerHeight);
  const full = Math.max(180, viewport - 16), half = Math.min(full, Math.max(320, viewport * .64));
  const current = Math.min(height ?? half, full), expanded = current > (half + full) / 2;
  useEffect(() => {
    const previous = document.activeElement, overflow = document.body.style.overflow;
    document.body.style.overflow = "hidden"; dialog.current?.focus();
    const resize = () => setViewport(window.visualViewport?.height || innerHeight);
    window.visualViewport?.addEventListener("resize", resize); window.addEventListener("resize", resize);
    return () => { clearTimeout(closeTimer.current); document.body.style.overflow = overflow; previous?.focus?.({ preventScroll: true }); window.visualViewport?.removeEventListener("resize", resize); window.removeEventListener("resize", resize); };
  }, []);
  const dismiss = () => {
    if (busy || closing) return;
    setClosing(true);
    closeTimer.current = setTimeout(onClose, matchMedia("(prefers-reduced-motion: reduce)").matches ? 0 : 220);
  };
  const finishDrag = cancelled => {
    if (!drag.current) return;
    const h = drag.current.height; drag.current = null; setDragging(false);
    if (!cancelled && h < half * .64) dismiss(); else setHeight(h > (half + full) / 2 ? full : half);
  };
  return createPortal(<div ref={dialog} tabIndex={-1} className={`chat-image-selection chat-attachment-preview ${closing ? "closing" : ""}`} role="dialog" aria-modal="true" aria-label={importing ? "添加贴纸" : "发送图片"}
    style={{ "--preview-dim": .16 + .15 * current / full }} onKeyDown={e => {
      if (e.key === "Escape") { e.stopPropagation(); dismiss(); }
      if (e.key === "Tab") {
        const nodes = [...dialog.current.querySelectorAll("button:not(:disabled)")]; const first = nodes[0], last = nodes.at(-1);
        if (e.shiftKey && (document.activeElement === first || document.activeElement === dialog.current)) { e.preventDefault(); last?.focus(); }
        else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first?.focus(); }
      }
    }}>
    <div className="chat-preview-scrim" onClick={dismiss} />
    <section ref={sheet} className={`chat-selection-card ${dragging ? "dragging" : ""}`} style={{ height: current }} aria-label="图片预览抽屉">
      <div className="chat-preview-drag-region" onPointerDown={e => {
        if (busy || closing || (e.target.closest("button") && !e.target.closest(".chat-preview-handle")) || e.button !== 0) return;
        moved.current = false; drag.current = { y: e.clientY, start: current, height: current };
      }} onPointerMove={e => { if (!drag.current) return; if (Math.abs(e.clientY - drag.current.y) > 4) { moved.current = true; setDragging(true); e.currentTarget.setPointerCapture(e.pointerId); } if (!moved.current) return; const h = Math.min(full, Math.max(100, drag.current.start - e.clientY + drag.current.y)); drag.current.height = h; setHeight(h); }} onPointerUp={() => finishDrag(false)} onPointerCancel={() => finishDrag(true)}>
        <div className="chat-preview-handle"><button type="button" disabled={busy || closing} aria-label={expanded ? "收起预览" : "展开预览"} onClick={e => { if (!e.detail || !moved.current) setHeight(expanded ? half : full); }}><span /></button></div>
        <header><button type="button" aria-label="取消" disabled={busy || closing} onClick={dismiss}><X size={21} /></button><strong>{importing ? "添加贴纸" : "发送图片"}</strong><button type="button" className="choose-again" disabled={busy || closing} onClick={onChooseAgain}>重新选择</button></header>
      </div>
      <div className={`chat-preview-image ${importing ? "sticker" : selected.height > selected.width * 2.5 ? "long" : ""}`}>
        <img src={selected.url} alt="待发送图片" style={importing ? {borderRadius: stickerRadius(selected.width, selected.height)} : undefined} onLoad={e => onReady(e.currentTarget.naturalWidth, e.currentTarget.naturalHeight)} />
      </div>
      {error && <p className="composer-error" role="alert">{error}</p>}
      <footer><span><Image size={18} />1 张图片</span><button type="button" className="primary" disabled={busy || closing || !selected.width} onClick={onSend}>{busy && <LoaderCircle size={16} className="spin" />}{importing ? "添加" : "发送"}</button></footer>
    </section>
  </div>, document.body);
}
