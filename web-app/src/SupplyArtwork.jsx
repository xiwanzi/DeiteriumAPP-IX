import React, { useEffect, useRef, useState } from "react";
import { Button, Field, Modal } from "./components.jsx";
import { uploadAsset } from "./assets.js";
import { ruleText } from "./supply-model.js";

const sizes = { hero: [1536, 864], rail: [357, 234], detail: [923, 117], fallback: [960, 640] };
const labels = { hero: "首页宣传图", rail: "侧栏封面", detail: "详情横幅", fallback: "备用展示图" };

export function SupplyArtField({ client, role, value, assets, onChange, onAsset, disabled }) {
  const [file, setFile] = useState(null), [error, setError] = useState("");
  const input = useRef(null), selected = assets[value];
  return <div className="supply-art-field">
    <Field label={`${labels[role]} · ${sizes[role].join(" × ")}`}><select value={value || ""} disabled={disabled} onChange={e => onChange(e.target.value)}><option value="">选择素材</option>{Object.values(assets).map(asset => <option key={asset.assetId} value={asset.assetId}>{asset.altText || asset.assetId}</option>)}</select></Field>
    {selected?.url ? <img className={`supply-art-thumb ${role}`} src={selected.url} alt={labels[role]} /> : <div className={`supply-art-empty ${role}`}>尚未选择{labels[role]}</div>}
    <Button secondary disabled={disabled} onClick={() => input.current?.click()}>上传并裁切</Button>
    <input ref={input} hidden type="file" accept="image/png,image/jpeg,image/webp" onChange={e => { const next = e.target.files?.[0]; e.target.value = ""; if (!next) return; if (next.size > 20 * 1024 * 1024) { setError("原图最大 20 MiB"); return; } setFile(next); setError(""); }} />
    {error && <p role="alert" className="form-error">{error}</p>}
    {file && <SupplyCropEditor client={client} file={file} role={role} close={() => setFile(null)} onSaved={asset => { onAsset(asset); onChange(asset.assetId); setFile(null); }} />}
  </div>;
}

function drawCrop(canvas, image, role, x, y, zoom) {
  const [width, height] = sizes[role]; canvas.width = width; canvas.height = height;
  const ctx = canvas.getContext("2d"); ctx.clearRect(0, 0, width, height);
  if (role === "fallback") {
    const scale = Math.min(width / image.width, height / image.height);
    ctx.drawImage(image, (width - image.width * scale) / 2, (height - image.height * scale) / 2, image.width * scale, image.height * scale);
  } else {
    const scale = Math.max(width / image.width, height / image.height) * zoom;
    const sw = width / scale, sh = height / scale;
    ctx.drawImage(image, (image.width - sw) * x, (image.height - sh) * y, sw, sh, 0, 0, width, height);
  }
  if (role === "hero") {
    ctx.globalCompositeOperation = "destination-in";
    const horizontal = ctx.createLinearGradient(0, 0, width, 0);
    [[0, 0], [.03, 1], [.92, 1], [1, 0]].forEach(([stop, a]) => horizontal.addColorStop(stop, `rgba(0,0,0,${a})`));
    ctx.fillStyle = horizontal; ctx.fillRect(0, 0, width, height);
    const vertical = ctx.createLinearGradient(0, 0, 0, height);
    [[0, 0], [.06, 1], [.87, 1], [1, 0]].forEach(([stop, a]) => vertical.addColorStop(stop, `rgba(0,0,0,${a})`));
    ctx.fillStyle = vertical; ctx.fillRect(0, 0, width, height); ctx.globalCompositeOperation = "source-over";
  }
  if (role === "detail") {
    const fade = ctx.createLinearGradient(0, 0, width, 0);
    [[0, 1], [.36, .96], [.51, .67], [.77, 0], [1, 0]].forEach(([stop, a]) => fade.addColorStop(stop, `rgba(245,245,245,${a})`));
    ctx.fillStyle = fade; ctx.fillRect(0, 0, width, height);
  }
}

function SupplyCropEditor({ client, file, role, close, onSaved }) {
  const [image, setImage] = useState(null), [x, setX] = useState(.5), [y, setY] = useState(.5), [zoom, setZoom] = useState(1);
  const [busy, setBusy] = useState(false), [error, setError] = useState(""), [progress, setProgress] = useState(""); const canvas = useRef(null);
  useEffect(() => { let live = true, bitmap; createImageBitmap(file).then(value => { bitmap = value; if (value.width * value.height > 40_000_000) { value.close(); throw new Error("原图像素过大，请先缩小到 4000 万像素以内"); } if (live) setImage(value); else value.close(); }).catch(e => { if (live) setError(e.message); }); return () => { live = false; bitmap?.close(); }; }, [file]);
  useEffect(() => { if (image && canvas.current) drawCrop(canvas.current, image, role, x, y, zoom); }, [image, role, x, y, zoom]);
  const upload = async () => {
    setBusy(true); setError("");
    try {
      const blob = await new Promise((resolve, reject) => canvas.current.toBlob(value => value ? resolve(value) : reject(new Error("无法生成裁切图片")), "image/png"));
      if (blob.size > 8 * 1024 * 1024) throw new Error("裁切后的图片超过 8 MiB，请使用较简洁的原图");
      const prepared = new File([blob], `${file.name.replace(/\.[^.]+$/, "")}-${role}.png`, { type: "image/png" });
      const asset = await uploadAsset(client, prepared, "SUPPLY_ART", "SUPPLY", "", (percent, label) => setProgress(`${label} · ${percent}%`));
      onSaved(asset);
    } catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  return <Modal title={`裁切${labels[role]}`} wide close={() => { if (!busy) close(); }}>
    <div className="supply-crop"><canvas ref={canvas} aria-label="裁切效果预览" />{!image && <p>正在读取图片…</p>}</div>
    {role !== "fallback" && <div className="supply-form-grid"><Field label="水平主体位置"><input type="range" min="0" max="1" step=".01" value={x} disabled={busy} onChange={e => setX(Number(e.target.value))} /></Field><Field label="垂直主体位置"><input type="range" min="0" max="1" step=".01" value={y} disabled={busy} onChange={e => setY(Number(e.target.value))} /></Field><Field label="放大"><input type="range" min="1" max="3" step=".05" value={zoom} disabled={busy} onChange={e => setZoom(Number(e.target.value))} /></Field></div>}
    <p className="muted">{role === "hero" ? "自动保留海报边缘渐隐；请把武器主体放在标题和按钮以外的区域。" : role === "detail" ? "左侧自动渐隐，为武器名称和概率说明留白。" : "预览即导出的图片构图。"}</p>
    {progress && <p role="status">{progress}</p>}{error && <p role="alert" className="form-error">{error}</p>}
    <div className="button-row"><Button secondary disabled={busy} onClick={close}>取消</Button><Button disabled={!image || busy} onClick={upload}>{busy ? "上传中…" : "使用此图片"}</Button></div>
  </Modal>;
}

export function SupplyArtPreview({ definition: d, assets, scheme = "80" }) {
  const hero = assets[d.art?.hero], rail = assets[d.art?.rail];
  const heroX = Number.isFinite(Number(d.art?.heroX)) ? Number(d.art.heroX) * 100 : 50;
  const heroY = Number.isFinite(Number(d.art?.heroY)) ? Number(d.art.heroY) * 100 : 50;
  const defaultRail = d.kind === "STANDARD" ? "常驻" : d.kind === "EVENT" ? "活动" : "限定";
  const railLabel = d.labels?.rail?.trim() || defaultRail;
  const featured = (d.rewards || []).filter(r => r.stars === 6).slice(0, 2);
  return <div className={`supply-preview theme-${d.art?.theme || "violet"}`} aria-label={`${scheme} 方案构图预览`}>
    {hero?.url && <img className="supply-preview-hero" style={{ objectPosition: `${heroX}% ${heroY}%` }} src={hero.url} alt="" />}
    <aside><small>D9 · ARMORY</small><strong>枪械补给</strong>{rail?.url && <img src={rail.url} alt="卡池封面" />}<span>{railLabel}{d.kind === "LIMITED" ? ` · ${scheme}` : ""}</span><small>补给详情</small></aside>
    <div className="supply-preview-copy">
      <h3>「{d.title || "页面标题"}」</h3>
      <span>{d.subtitle || "本期武器补给"}</span>
      <p>{ruleText(d.kind, scheme, d)}</p>
      {featured.length > 0 && <div className="supply-preview-featured">{featured.map(r => <em key={r.key}>{r.name}</em>)}</div>}
    </div>
    <div className="supply-preview-actions"><span>抽取一次 · 1 券</span><span>抽取十次 · 10 券</span></div>
  </div>;
}
