import React, {useRef,useState} from "react";
import {ArrowUp,Trash2,Upload} from "lucide-react";
import {Badge,Button} from "./components.jsx";
import {uploadAsset} from "./assets.js";
export function MediaPicker({ client, images, onChange, purpose, businessType, businessRef = "", max, onBusy }) {
  const input = useRef(null), [progress, setProgress] = useState(null), [error, setError] = useState("");
  const upload = async (files) => {
    if (!files?.length || progress) return; setError(""); onBusy?.(true);
    const added = [];
    try {
      if (files.length + images.length > max) throw new Error(`最多上传 ${max} 张图片。`);
      for (const file of files) { const asset = await uploadAsset(client, file, purpose, businessType, businessRef, (value, label) => setProgress({ value, label })); added.push(asset); onChange([...images, ...added]); }
    } catch (e) { setError(e.message); } finally { setProgress(null); onBusy?.(false); if (input.current) input.current.value = ""; }
  };
  return <div className="field"><label>图片 · 第一张为封面</label><div className="foundation-grid" style={{ gridTemplateColumns: "repeat(auto-fill,minmax(130px,1fr))" }}>{images.map((asset, index) => <div className="panel" key={asset.assetId} style={{ padding: 8 }}>
    {asset.url ? <img src={asset.url} alt={asset.altText || `第 ${index + 1} 张图片`} style={{ width: "100%", aspectRatio: "1", objectFit: "cover", borderRadius: 8 }} /> : <p className="muted">图片预览暂不可用</p>}
    <div className="button-row">{index === 0 ? <Badge>封面</Badge> : <Button secondary aria-label={`将第 ${index + 1} 张设为封面`} onClick={() => onChange([asset, ...images.filter((a) => a.assetId !== asset.assetId)])}><ArrowUp size={14} /></Button>}<Button secondary aria-label={`移除第 ${index + 1} 张图片`} onClick={() => onChange(images.filter((a) => a.assetId !== asset.assetId))}><Trash2 size={14} /></Button></div>
  </div>)}</div><Button secondary disabled={Boolean(progress) || images.length >= max} onClick={() => input.current?.click()}><Upload size={16} />添加图片</Button><input ref={input} type="file" hidden multiple accept="image/png,image/jpeg,image/webp" onChange={(e) => upload(Array.from(e.target.files || []))} />
    {progress && <div role="status"><p>{progress.label} · {progress.value}%</p><progress max="100" value={progress.value} style={{ width: "100%" }} aria-label="图片上传进度" /></div>}{error && <p className="auth-error" role="alert">{error}</p>}
  </div>;
}
