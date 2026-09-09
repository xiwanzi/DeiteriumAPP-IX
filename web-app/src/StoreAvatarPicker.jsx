import React, { useEffect, useRef, useState } from "react";
import { Upload, RotateCcw } from "lucide-react";
import { Avatar, Button, Field, Modal } from "./components.jsx";
import { uploadAsset } from "./assets.js";
import { avatarCrop } from "./avatar-crop.js";

export default function StoreAvatarPicker({ client, storeId, name, value, onChange, onBusy }) {
  const input = useRef(null), [file, setFile] = useState(null), [error, setError] = useState(""), [busy, setBusy] = useState(false), [progress, setProgress] = useState("");
  const save = async (cropped) => {
    if (busy) return; setBusy(true); onBusy(true); setError("");
    try { const asset = await uploadAsset(client, cropped, "STORE_MEDIA", "STORE", storeId || "", (n, message) => setProgress(`${message} · ${n}%`)); onChange(asset); setFile(null); }
    catch (e) { setError(e.message); } finally { setBusy(false); onBusy(false); setProgress(""); }
  };
  return <div className="store-avatar-picker"><Avatar user={{ name: name || "商店", avatar: value }} size="large" /><div><strong>商店头像</strong><p>用于商店与订单中的商家头像，可拖动和缩放裁切。</p><div className="button-row"><Button secondary disabled={busy} onClick={() => input.current?.click()}><Upload size={16} />{value ? "更换头像" : "上传头像"}</Button>{value && <Button secondary disabled={busy} onClick={() => onChange(null)}>移除</Button>}</div></div>
    <input ref={input} type="file" hidden accept="image/png,image/jpeg,image/webp" onChange={(e) => { const next = e.target.files?.[0]; e.target.value = ""; setError(""); if (!next) return; if (next.size > 20 * 1024 * 1024 || !["image/png", "image/jpeg", "image/webp"].includes(next.type)) { setError("请选择不超过 20 MB 的 PNG、JPEG 或 WebP 图片。"); return; } setFile(next); }} />
    {error && !file && <p className="auth-error" role="alert">{error}</p>}
    {file && <Modal title="调整商店头像" close={() => { if (!busy) setFile(null); }} dismissOnBackdrop={false}><AvatarCropEditor file={file} busy={busy} onSave={save} /><p role="status">{progress}</p>{error && <p role="alert" className="auth-error">{error}</p>}</Modal>}
  </div>;
}

export function AvatarCropEditor({ file, busy, onSave }) {
  const image = useRef(null), drag = useRef(null), [src, setSrc] = useState(""), [dimensions, setDimensions] = useState(null), [zoom, setZoom] = useState(1), [center, setCenter] = useState(null), [error, setError] = useState("");
  useEffect(() => { const url = URL.createObjectURL(file); setSrc(url); setDimensions(null); setCenter(null); setZoom(1); return () => URL.revokeObjectURL(url); }, [file]);
  const crop = dimensions && avatarCrop(...dimensions, zoom, ...(center || dimensions.map((n) => n / 2)));
  const style = crop ? { width: `${dimensions[0] / crop.size * 100}%`, height: `${dimensions[1] / crop.size * 100}%`, left: `${-crop.x / crop.size * 100}%`, top: `${-crop.y / crop.size * 100}%` } : { visibility: "hidden" };
  const move = (dx, dy, side) => setCenter((current) => { const base = avatarCrop(...dimensions, zoom, ...(current || dimensions.map((n) => n / 2))); return [base.centerX - dx * base.size / side, base.centerY - dy * base.size / side]; });
  const finish = async () => {
    if (!crop || busy) return;
    try { const canvas = document.createElement("canvas"); canvas.width = canvas.height = 512; canvas.getContext("2d").drawImage(image.current, crop.x, crop.y, crop.size, crop.size, 0, 0, 512, 512); const blob = await new Promise((resolve) => canvas.toBlob(resolve, "image/png")); if (!blob) throw new Error("图片裁切失败，请重试。"); await onSave(new File([blob], "store-avatar.png", { type: "image/png" })); }
    catch (e) { setError(e.message); }
  };
  return <div className="avatar-crop-editor"><p className="muted">拖动图片调整位置，圆形范围就是头像显示的内容。</p>
    <div className="avatar-crop-frame" role="img" aria-label="头像裁切预览，可用方向键移动图片" tabIndex={0}
      onPointerDown={(e) => { if (!crop || busy) return; e.currentTarget.setPointerCapture(e.pointerId); drag.current = [e.clientX, e.clientY]; }}
      onPointerMove={(e) => { if (!drag.current || !crop || busy) return; move(e.clientX - drag.current[0], e.clientY - drag.current[1], e.currentTarget.clientWidth); drag.current = [e.clientX, e.clientY]; }} onPointerUp={() => { drag.current = null; }} onPointerCancel={() => { drag.current = null; }}
      onKeyDown={(e) => { if (!crop || busy) return; const delta = { ArrowLeft: [-10, 0], ArrowRight: [10, 0], ArrowUp: [0, -10], ArrowDown: [0, 10] }[e.key]; if (delta) { e.preventDefault(); move(...delta, e.currentTarget.clientWidth); } }}>
      <img ref={image} src={src || undefined} alt="" draggable={false} style={style} onLoad={(e) => { const im = e.currentTarget; if (im.naturalWidth * im.naturalHeight > 64000000) { setError("图片尺寸过大，请选择较小的图片。"); return; } setDimensions([im.naturalWidth, im.naturalHeight]); }} onError={() => setError("无法读取图片，请选择其他图片。")} /><div className="avatar-crop-mask" />
    </div>
    <Field label={`缩放 · ${zoom.toFixed(1)} 倍`}><input type="range" min={1} max={4} step={0.01} value={zoom} disabled={busy} onChange={(e) => setZoom(Number(e.target.value))} /></Field>
    {crop && <div className="avatar-crop-previews" aria-label="实际尺寸预览">{[64, 40, 28].map((side) => <div key={side} style={{ width: side, height: side }}><img src={src} alt={`${side} 像素头像预览`} style={style} /></div>)}</div>}
    {error && <p role="alert" className="auth-error">{error}</p>}<div className="button-row"><Button secondary disabled={busy} onClick={() => { setZoom(1); setCenter(null); }}><RotateCcw size={15} />居中还原</Button><Button disabled={busy || !crop || Boolean(error)} onClick={finish}>{busy ? "正在保存…" : "使用此头像"}</Button></div>
  </div>;
}
