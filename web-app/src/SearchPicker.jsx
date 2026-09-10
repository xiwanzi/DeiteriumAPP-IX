import React, { useEffect, useRef, useState } from "react";
import { Check, ChevronDown, Search, X, Package } from "lucide-react";
import { Button, Modal } from "./components.jsx";

/** Paginated server search, used for item versions, templates and promotion scopes. */
export default function SearchPicker({ label, placeholder = "搜索名称或编号", values = [], onChange, load, multiple = false, max = 100, showChips = true, getKey = (v) => v.id, getTitle = (v) => v.name, getDescription = (v) => v.summary, disabled = false, unavailable = () => false }) {
  const [open, setOpen] = useState(false), [query, setQuery] = useState(""), [rows, setRows] = useState([]), [cursor, setCursor] = useState(null),
    [picked, setPicked] = useState([]), [busy, setBusy] = useState(false), [error, setError] = useState("");
  const generation = useRef(0), loader = useRef(load), searchInput = useRef(null);
  loader.current = load;
  const fetchRows = async (more = false) => {
    const current = ++generation.current; setBusy(true); setError("");
    try {
      const result = await loader.current(query, more ? cursor : null);
      if (current !== generation.current) return;
      setRows((old) => more ? [...new Map([...old, ...result.items].map((item) => [getKey(item), item])).values()] : result.items);
      setCursor(result.nextCursor || null);
    } catch (e) { if (current === generation.current) setError(e.message); }
    finally { if (current === generation.current) setBusy(false); }
  };
  useEffect(() => {
    if (!open) return;
    setRows([]); setCursor(null); setBusy(true);
    const timer = setTimeout(() => fetchRows(), 220);
    return () => { clearTimeout(timer); generation.current++; };
  }, [open, query]);
  useEffect(() => { if (open) { const frame = requestAnimationFrame(() => searchInput.current?.focus()); return () => cancelAnimationFrame(frame); } }, [open]);
  const close = () => { generation.current++; setOpen(false); };
  const choose = (item) => {
    if (unavailable(item)) return;
    if (!multiple) { onChange([item]); close(); return; }
    const key = getKey(item), selected = picked.some((v) => getKey(v) === key);
    if (!selected && picked.length >= max) { setError(`最多选择 ${max} 项。`); return; }
    setPicked(selected ? picked.filter((v) => getKey(v) !== key) : [...picked, item]);
  };
  return <div className="search-picker">
    <button type="button" className="picker-trigger" disabled={disabled} aria-label={label} aria-haspopup="dialog" onClick={() => { setPicked(values); setQuery(""); setOpen(true); }}>
      <span>{values.length ? multiple ? `已选择 ${values.length} 项` : getTitle(values[0]) : label}</span><ChevronDown size={17} />
    </button>
    {showChips && multiple && values.length > 0 && <div className="selection-chips">{values.map((v) => <span key={getKey(v)}>{getTitle(v)}<button type="button" disabled={disabled} aria-label={`移除 ${getTitle(v)}`} onClick={() => onChange(values.filter((item) => getKey(item) !== getKey(v)))}><X size={13} /></button></span>)}</div>}
    {open && <Modal title={label} close={close} className="resource-picker-modal" dismissOnBackdrop={false}>
      <label className="picker-search"><Search size={19} /><input ref={searchInput} autoFocus value={query} maxLength={100} placeholder={placeholder} aria-label={placeholder} onChange={(e) => setQuery(e.target.value)} />{query && <button type="button" aria-label="清空搜索" onClick={() => setQuery("")}><X size={17} /></button>}</label>
      {multiple && <div className="picker-selection-head"><span>已选 {picked.length} / {max}</span><button type="button" disabled={!picked.length} onClick={() => setPicked([])}>清空选择</button></div>}
      {error && <p className="auth-error" role="alert">{error}</p>}
      <div className="picker-results" aria-busy={busy}>
        {rows.map((item) => { const selected = picked.some((v) => getKey(v) === getKey(item)); return <button type="button" className={`picker-result ${selected ? "selected" : ""}`} key={getKey(item)} role={multiple ? "checkbox" : undefined} aria-checked={multiple ? selected : undefined} disabled={unavailable(item)} onClick={() => choose(item)}>
          <span className="picker-item-icon">{item.imageUrl ? <img src={item.imageUrl} alt="" /> : <Package size={20} />}</span><span className="picker-item-copy"><strong>{getTitle(item)}</strong><small>{getDescription(item)}</small></span><span className={`picker-check ${selected ? "checked" : ""}`}>{selected && <Check size={14} />}</span>
        </button>; })}
        {!busy && !rows.length && <p className="picker-empty">{query ? "没有找到匹配项，试试其他名称。" : "暂无可选内容。"}</p>}
        {busy && <p className="picker-empty" role="status">正在查找…</p>}
        {cursor && <Button secondary disabled={busy} onClick={() => fetchRows(true)}>加载更多</Button>}
      </div>
      <div className="picker-footer"><span>{multiple ? "选择后可继续搜索，已选内容会保留。" : "选择一项即可应用。"}</span><Button secondary onClick={close}>取消</Button>{multiple && <Button onClick={() => { onChange(picked); close(); }}>确定 {picked.length ? `(${picked.length})` : ""}</Button>}</div>
    </Modal>}
  </div>;
}
