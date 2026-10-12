import React, { useEffect, useRef, useState } from "react";
import { ArrowLeft, RotateCcw, Save } from "lucide-react";
import { Button, Field, PageHead } from "./components.jsx";
import { useUnsavedChanges } from "./unsaved-changes.js";

export default function ImageReviewSettings({ client, onBack }) {
  const [saved, setSaved] = useState(null), [defaults, setDefaults] = useState(null), [threshold, setThreshold] = useState(50), [prompt, setPrompt] = useState("");
  const [loading, setLoading] = useState(true), [busy, setBusy] = useState(false), [error, setError] = useState(""), [notice, setNotice] = useState("");
  const alive = useRef(true), acting = useRef(false), controller = useRef(null);
  const dirty = !!saved && (Number(threshold) !== saved.threshold || prompt !== saved.prompt);
  useUnsavedChanges(dirty, busy);
  const load = async () => {
    controller.current?.abort(); const c = new AbortController(); controller.current = c; setLoading(true);
    try {
      const { data } = await client.request("/api/v1/admin/image-review/settings", { signal: c.signal });
      if (!alive.current || c.signal.aborted) return;
      setSaved(data.settings); setDefaults(data.defaults); setThreshold(data.settings.threshold); setPrompt(data.settings.prompt); setError("");
    } catch (e) { if (alive.current && !c.signal.aborted) setError(e.message); }
    finally { if (alive.current && !c.signal.aborted) setLoading(false); }
  };
  useEffect(() => { alive.current = true; load(); return () => { alive.current = false; controller.current?.abort(); }; }, [client]);
  const save = async e => {
    e.preventDefault(); if (acting.current || !saved) return;
    if (threshold === "" || !Number.isInteger(Number(threshold)) || Number(threshold) < 0 || Number(threshold) > 100 || !prompt.trim() || [...prompt].length > 12000) { setError("请填写 0–100 的整数阈值，以及不超过 12,000 字的评分提示词。"); return; }
    acting.current = true; setBusy(true); setError(""); setNotice("");
    try {
      const { data } = await client.request("/api/v1/admin/image-review/settings", { method: "PUT", body: { threshold: Number(threshold), prompt, expectedVersion: saved.version } });
      if (!alive.current) return;
      setSaved(data.settings); setThreshold(data.settings.threshold); setPrompt(data.settings.prompt); setNotice("设置已保存，后续开始的审核使用新规则。");
    } catch (e) { if (alive.current) setError(e.message); }
    finally { acting.current = false; if (alive.current) setBusy(false); }
  };
  return <section className="image-review-page">
    <PageHead title="图片审核设置" subtitle="修改标记阈值与评分标准，保存后用于后续开始的审核。"><Button secondary disabled={busy} onClick={onBack}><ArrowLeft size={16} />返回图片审核</Button></PageHead>
    {error && <p role="alert" className="form-error">{error} <Button secondary disabled={busy} onClick={() => { if (!dirty || window.confirm("重新载入会丢弃当前未保存的修改，继续？")) load(); }}>重新载入</Button></p>}
    {notice && <p role="status" className="review-notice">{notice}</p>}
    {loading ? <p role="status">正在读取设置…</p> : saved && <form className="review-settings-form" onSubmit={save}>
      <div className="review-settings-section"><h2>标记阈值</h2><Field label="风险分数阈值"><input aria-label="风险分数阈值" type="number" min="0" max="100" step="1" required value={threshold} disabled={busy} onChange={e => { setThreshold(e.target.value); setNotice(""); }} /></Field><p className="review-meta">总分严格大于阈值才标记并保存证据。例如设为 50，51–100 分会被保留；设为 100 则不自动标记。</p></div>
      <div className="review-settings-section"><h2>评分标准提示词</h2><p className="review-meta">可修改 R18、政治内容、宣传导流的判断标准和评分尺度。AI 仍按三个类别返回 0–100 分；总分取最高项，再加其余两项之和的四分之一，最高 100 分。</p><Field label="审核提示词"><textarea aria-label="审核提示词" rows="18" required value={prompt} disabled={busy} onChange={e => { setPrompt(e.target.value); setNotice(""); }} /></Field><div className="review-settings-caption"><span>{[...prompt].length.toLocaleString()} / 12,000 字</span><span>DeepSeek V4.1 Flash</span></div></div>
      <p className="review-meta">只审核启用后新上传的静态聊天图片。旧图、业务图片和动图仍跳过；已完成的审核记录不会因修改设置而重新评分或删除。</p>
      <div className="review-settings-actions"><Button secondary disabled={busy || !defaults} onClick={() => { setThreshold(defaults.threshold); setPrompt(defaults.prompt); setNotice("已填入默认设置，点击保存后生效。"); }}><RotateCcw size={16} />恢复默认</Button><Button type="submit" disabled={busy || !dirty}><Save size={16} />{busy ? "保存中…" : "保存设置"}</Button></div>
    </form>}
  </section>;
}
