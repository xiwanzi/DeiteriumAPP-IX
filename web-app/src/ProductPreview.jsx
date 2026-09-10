import React, { useState } from "react";
import { ArrowLeft, Smartphone, ImageOff, Mail, ShoppingBag, Check, Moon, Sun } from "lucide-react";
import { Badge } from "./components.jsx";
import { credit } from "./format.js";
import { salePrice } from "./promotions.js";
import { centeredCrop, productImageFrame, defaultMailBody } from "./product-preview.js";

const modes = [["detail", "商品详情"], ["poster", "推荐海报"], ["grid", "双列卡片"], ["bag", "购物袋"]];
export default function ProductPreview({ value, images, included, brand, category, template }) {
  const [view, setView] = useState("phone"), [mode, setMode] = useState("detail"), [phoneWidth, setPhoneWidth] = useState(393), [dark, setDark] = useState(false), [imageIndex, setImageIndex] = useState(0), [dimensions, setDimensions] = useState({}), [failed, setFailed] = useState({});
  const selectedIndex = mode === "detail" ? Math.min(imageIndex, Math.max(0, images.length - 1)) : 0;
  const asset = images[selectedIndex], src = asset?.url, size = dimensions[src], frame = productImageFrame(mode, phoneWidth), crop = size && centeredCrop(size[0], size[1], ...frame);
  const title = value.title || "商品标题", price = credit(salePrice(value.price, value.discountRate || 10000)), scale = 320 / phoneWidth;
  const photo = (className = "", style = {}) => <div className={`preview-photo ${className}`} style={style}>{src && !failed[src] ? <img src={src} alt={asset.altText || title} onLoad={(e) => { const img = e.currentTarget; setDimensions((old) => old[src]?.[0] === img.naturalWidth && old[src]?.[1] === img.naturalHeight ? old : { ...old, [src]: [img.naturalWidth, img.naturalHeight] }); }} onError={() => setFailed((old) => ({ ...old, [src]: true }))} /> : <span><ImageOff size={26} />{src ? "图片加载失败" : "添加图片后预览"}</span>}</div>;
  return <aside className="product-preview" aria-label="App 实时预览">
    <div className="preview-heading"><div><Smartphone size={18} /><strong>App 实时预览</strong></div><Badge tone="sage">随编辑更新</Badge></div>
    <div className="preview-display-tabs" role="group" aria-label="预览内容">{[["phone","手机效果"],["crop","裁切尺寸"],["mail","游戏邮件"]].map(([key,label]) => <button type="button" key={key} aria-pressed={view === key} onClick={() => setView(key)}>{label}</button>)}</div>
    {src && <img hidden src={src} alt="" onLoad={(e) => { const img = e.currentTarget; setDimensions((old) => old[src]?.[0] === img.naturalWidth && old[src]?.[1] === img.naturalHeight ? old : { ...old, [src]: [img.naturalWidth, img.naturalHeight] }); }} onError={() => setFailed((old) => ({ ...old, [src]: true }))} />}
    <div hidden={view === "mail"}><div className="preview-tabs" role="group" aria-label="预览展示位置">{modes.map(([key, label]) => <button key={key} type="button" aria-pressed={mode === key} onClick={() => setMode(key)}>{label}</button>)}</div>
    <div className="preview-tools"><select aria-label="模拟手机宽度" value={phoneWidth} onChange={(e) => setPhoneWidth(Number(e.target.value))}>{[360, 393, 430].map((width) => <option value={width} key={width}>{width} dp 屏宽</option>)}</select><button type="button" className="icon-button" aria-label={dark ? "预览浅色" : "预览深色"} onClick={() => setDark(!dark)}>{dark ? <Sun size={17} /> : <Moon size={17} />}</button></div>
    </div><div hidden={view !== "phone"}><div className={`preview-phone ${dark ? "preview-dark" : ""}`}>
      <div className="phone-status"><span>9:41</span><span className="phone-island" /><span>▰ 100%</span></div>
      <div className="phone-viewport" style={{ height: 625 * scale }}><div className="phone-scroll" style={{ width: phoneWidth, height: 625, transform: `scale(${scale})` }}>
        <div className="phone-navbar"><ArrowLeft size={21} /><strong>{mode === "bag" ? "购物袋" : mode === "detail" ? "商品详情" : "商城"}</strong><ShoppingBag size={21} /></div>
        {mode === "detail" ? <>
          <div className="phone-copy"><small>{brand || "品牌"} · {category || "分类"}</small><h2>购买 {title}</h2><p>{value.subtitle || "一句话介绍这件商品"}</p></div>
          {photo("phone-detail-image", { width: frame[0], height: frame[1], margin: "0 20px", borderRadius: 24 })}
          <div className="phone-dots">{images.map((image, i) => <button type="button" aria-label={`预览第 ${i + 1} 张图片`} aria-pressed={i === selectedIndex} key={image.assetId} onClick={() => setImageIndex(i)} />)}</div>
          <div className="phone-copy"><h3>商品详情</h3><p>{value.description || "填写详细介绍，查看文字在手机上的排版。"}</p></div>
          <div className="phone-package"><h3>包装内容</h3>{included.split("\n").filter(Boolean).map((line, i) => <p key={i}>{line}</p>)}<p>单价 <strong>{price} 信用点</strong></p></div>
          <div className="phone-copy"><h3>{value.estimatedDelivery}</h3><p>{value.deliverySummary}</p></div>
        </> : mode === "poster" ? <><div className="phone-copy"><h2>新品推荐</h2></div><div className={`phone-poster ${value.posterTone === "DARK" ? "dark-art" : ""}`} style={{ width: 308, height: 405 }}>{photo()}<div className="poster-copy"><small>{category || "分类"}</small><h2>{title}</h2><p>{value.subtitle || "一句话简介"}</p><small>{price} 信用点</small></div></div><p className="phone-caption">推荐海报的文字叠在封面上，请留出清晰的文字区域。</p></> : mode === "grid" ? <><div className="phone-copy"><h2>探索更多</h2></div><div className="phone-grid"><article>{photo("", { height: 170 })}<div><h3>{title}</h3><p>{price} 信用点</p></div></article><div className="phone-next-card">相邻商品</div></div></> : <><div className="phone-bag-item">{photo("", { width: 80, height: 100, borderRadius: 15 })}<div><h3>{title}</h3><p>{price} 信用点</p><span>−　1　＋</span></div></div><div className="phone-copy"><p><Check size={16} /> 通过游戏内邮箱交付</p></div></>}
      </div></div>
      <div className="phone-bottom"><div><strong>{price}</strong><small>信用点</small></div><span>{mode === "bag" ? "去结算" : "加入购物袋"}</span></div>
    </div>
    </div><section hidden={view !== "crop"} className="crop-inspector" aria-label="图片裁切尺寸">
      <div className="preview-heading"><strong>实际裁切范围</strong><span>{frame[0]} × {frame[1]} dp</span></div>
      {mode === "detail" && images.length > 1 && <select aria-label="选择预览图片" value={selectedIndex} onChange={(e) => setImageIndex(Number(e.target.value))}>{images.map((image, i) => <option key={image.assetId} value={i}>第 {i + 1} 张{!i ? " · 封面" : ""}</option>)}</select>}
      {src && size && !failed[src] && crop ? <><div className="crop-original" style={{ aspectRatio: size[0] / size[1], width: Math.min(300, 300 * size[0] / size[1]) }}><img src={src} alt="原图与居中裁切区域" /><div className="crop-boundary" style={{ left: `${crop.x / size[0] * 100}%`, top: `${crop.y / size[1] * 100}%`, width: `${crop.width / size[0] * 100}%`, height: `${crop.height / size[1] * 100}%` }} /></div><dl className="crop-dimensions"><div><dt>原图</dt><dd>{size[0]} × {size[1]} px</dd></div><div><dt>保留区域（约）</dt><dd>{Math.round(crop.width)} × {Math.round(crop.height)} px</dd></div><div><dt>左 / 上裁去（约）</dt><dd>{Math.round(crop.x)} / {Math.round(crop.y)} px</dd></div><div><dt>画面保留</dt><dd>{Math.round(crop.retained * 100)}%</dd></div></dl></> : <p className="muted">{src ? "等待图片加载后计算像素尺寸。" : "上传封面后显示原图与裁切区域。"}</p>}
      <p className="preview-note">框内是 App 居中裁切后可见的内容，暗色部分会被裁掉。原图保持完整。dp 是布局尺寸，px 是原图像素；实际设备字体及显示缩放可能不同。</p>
    </section>
    <section hidden={view !== "mail"} className="mail-preview"><div className="preview-heading"><div><Mail size={17} /><strong>游戏内邮件预览</strong></div></div><small>Deuterium 官方商城 → 购买玩家</small><h3>{value.mailTitle?.trim() || "官方商城订单 D…"}</h3><p>{value.mailBody?.trim() || defaultMailBody}</p><div className="mail-attachments"><ShoppingBag size={17} /><span>{template?.summary || "选择交付模板后显示附件摘要"}</span></div><small>模拟单种商品购买。多商品订单使用订单标题，正文按商品分节合并。</small></section>
  </aside>;
}
