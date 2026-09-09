import React, { useEffect, useId, useRef } from "react";
import { confirmNavigation } from "./unsaved-changes.js";
import {
  X,
  ArrowUpRight,
  Check,
  Package,
  Pickaxe,
  Flower2,
  Carrot,
  Lamp,
  Map,
  Layers3,
  Heart,
  ChevronRight,
} from "lucide-react";
import { money } from "./format.js";
export const media = (name) => `/media/${name}`;
export function Avatar({ user = "玩家", size = "", onClick }) {
  const p = typeof user === "object" ? { name: user.gameId || user.name || "玩家", color: "blue", avatar: user.avatar } : { name: user || "玩家", color: "blue" };
  const content = p.avatar?.url ? <img className={`avatar ${p.color} ${size}`} src={p.avatar.url} alt="" aria-hidden="true" style={{ objectFit: "cover" }} /> : (
    <span className={`avatar ${p.color} ${size}`} aria-hidden="true">
      {p.name.slice(0, 1).toUpperCase()}
    </span>
  );
  return onClick ? (
    <button
      className="avatar-button"
      aria-label={`查看 ${p.name} 的资料`}
      onClick={onClick}
    >
      {content}
    </button>
  ) : (
    content
  );
}
export function PageHead({ eyebrow, title, subtitle, children }) {
  return (
    <div className="page-head">
      <div>
        <p className="eyebrow">{eyebrow}</p>
        <h1>{title}</h1>
        {subtitle && <p className="subtitle">{subtitle}</p>}
      </div>
      <div className="head-actions">{children}</div>
    </div>
  );
}
export function SectionHead({ title, description, action, onAction }) {
  return (
    <div className="section-head">
      <div>
        <h2>{title}</h2>
        {description && <p>{description}</p>}
      </div>
      {action && (
        <button className="text-button" onClick={onAction}>
          {action}
          <ChevronRight size={16} />
        </button>
      )}
    </div>
  );
}
export function Button({
  children,
  secondary,
  danger,
  className = "",
  ...props
}) {
  return (
    <button
      type="button"
      className={`button ${secondary ? "secondary" : ""} ${danger ? "danger" : ""} ${className}`}
      {...props}
    >
      {children}
    </button>
  );
}
export function Empty({
  title = "这里暂时没有内容",
  text = "换个筛选条件，再发现一些新东西。",
  children,
}) {
  return (
    <div className="empty">
      <Package size={34} />
      <h3>{title}</h3>
      <p>{text}</p>
      {children}
    </div>
  );
}
export function Tabs({ values, value, onChange, label = "筛选" }) {
  return (
    <div className="tabs" role="group" aria-label={label}>
      {values.map((v) => (
        <button
          key={v}
          type="button"
          className={value === v ? "active" : ""}
          aria-pressed={value === v}
          onClick={() => onChange(v)}
        >
          {v}
        </button>
      ))}
    </div>
  );
}
export function Modal({ title, children, close, wide = false, guardClose = false, dismissOnBackdrop = true }) {
  const requestClose = () => { if (!guardClose || confirmNavigation()) close(); };
  const ref = useRef(null),
    label = useId();
  useEffect(() => {
    const dialog = ref.current;
    const previous = document.activeElement;
    dialog.showModal();
    const overflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = overflow;
      previous?.focus?.();
    };
  }, []);
  return (
    <dialog
      ref={ref}
      className={`modal ${wide ? "wide" : ""}`}
      aria-labelledby={label}
      onCancel={(e) => {
        e.preventDefault();
        requestClose();
      }}
      onClick={(e) => {
        if (dismissOnBackdrop && e.target === ref.current) {
          const r = ref.current.getBoundingClientRect();
          if (
            e.clientX < r.left ||
            e.clientX > r.right ||
            e.clientY < r.top ||
            e.clientY > r.bottom
          )
            requestClose();
        }
      }}
    >
      <div className="modal-head">
        <h2 id={label}>{title}</h2>
        <button className="icon-button" aria-label="关闭弹窗" onClick={requestClose}>
          <X size={20} />
        </button>
      </div>
      <div className="modal-body">{children}</div>
    </dialog>
  );
}
export function Field({ label, children, hint, ...props }) {
  const fieldId = useId();
  const control = children || <input {...props} />;
  return (
    <div className="field">
      <label htmlFor={fieldId}>{label}</label>
      {React.cloneElement(control, {
        id: fieldId,
        "aria-describedby": hint ? `${fieldId}-hint` : undefined,
      })}
      {hint && <small id={`${fieldId}-hint`}>{hint}</small>}
    </div>
  );
}
export function Toggle({ label, description, checked, onChange }) {
  return (
    <label className="toggle-row">
      <span>
        <strong>{label}</strong>
        {description && <small>{description}</small>}
      </span>
      <input
        type="checkbox"
        role="switch"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
      />
      <span className="switch" aria-hidden="true" />
    </label>
  );
}
export function Price({ value, small = false }) {
  return (
    <span className={`price ${small ? "small" : ""}`}>
      {money(value)}
      <small>信用点</small>
    </span>
  );
}
export function Badge({ children, tone = "blue" }) {
  return <span className={`badge ${tone}`}>{children}</span>;
}
const icons = {
  wood: Layers3,
  pickaxe: Pickaxe,
  flowers: Flower2,
  carrot: Carrot,
  lantern: Lamp,
  stone: Layers3,
  map: Map,
};
export function Art({ kind = "wood", image, className = "" }) {
  if (image || ["village", "flowers", "map"].includes(kind))
    return (
      <div className={`item-art scene-art ${className}`}>
        <img
          src={
            image ||
            media(
              kind === "flowers"
                ? "flower-shop.png"
                : kind === "map"
                  ? "plaza-sunset.png"
                  : "street-daylight.png",
            )
          }
          alt="Deuterium 服务器实景展示"
          loading="lazy"
        />
      </div>
    );
  const Icon = icons[kind] || Package;
  return (
    <div className={`item-art art-${kind} ${className}`}>
      <div className="art-grid" />
      <span className="art-object">
        <Icon strokeWidth={1.25} />
      </span>
      <span className="art-caption">DEUTERIUM / {kind.toUpperCase()}</span>
    </div>
  );
}
export function ProductCard({ product: p, onOpen, onAdd }) {
  return (
    <article className="product-card">
      <button
        className={`product-image ${p.tone}`}
        onClick={() => onOpen(p)}
        aria-label={`查看 ${p.title}`}
      >
        <img src={media(p.image)} alt={p.title} loading="lazy" />
      </button>
      <div className="product-info">
        <span className="tiny-label">{p.brand} · 官方精选</span>
        <button className="plain-title" onClick={() => onOpen(p)}>
          <h3>{p.title}</h3>
        </button>
        <p>{p.subtitle}</p>
        <div className="price-row">
          <Price small value={p.price} />
          <button
            className="pill-button"
            disabled={!p.stock}
            onClick={() => onAdd(p)}
          >
            {p.stock ? "加入" : "售罄"}
          </button>
        </div>
      </div>
    </article>
  );
}
export function Confirm({
  title,
  description,
  amount,
  actionLabel = "确认",
  onConfirm,
  close,
  danger = false,
  pending = false,
}) {
  return (
    <Modal title={title} close={close}>
      <div className="confirm">
        <div className={`confirm-symbol ${danger ? "red" : ""}`}>
          <Check size={28} />
        </div>
        <p>{description}</p>
        {amount != null && <Price value={amount} />}
        <div className="notice-box">
          本机体验操作，不会影响真实账号或游戏信用点。
        </div>
        <div className="button-row">
          <Button secondary onClick={close}>
            取消
          </Button>
          <Button danger={danger} disabled={pending} onClick={onConfirm}>
            {pending ? "处理中…" : actionLabel}
          </Button>
        </div>
      </div>
    </Modal>
  );
}
