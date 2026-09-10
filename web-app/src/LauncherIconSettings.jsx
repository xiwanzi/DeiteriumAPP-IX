import React, { useEffect, useRef, useState } from "react";
import { Check, CheckCircle2, RefreshCw, Smartphone, ArrowRight, LoaderCircle } from "lucide-react";
import { Button, PageHead } from "./components.jsx";
import { id } from "./format.js";
import defaultIcon from "./assets/launcher-icons/default.svg";
import anniversaryIcon from "./assets/launcher-icons/anniversary-911.svg";
import "./launcher-icons.css";

const artwork = { default: defaultIcon, anniversary_911: anniversaryIcon };
const titles = { default: "默认图标", anniversary_911: "双子节图标" };

export default function LauncherIconSettings({ client }) {
  const [data, setData] = useState(null);
  const [selected, setSelected] = useState("default");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const request = useRef(null);
  const mounted = useRef(false);

  async function load(keepSelection = false) {
    setLoading(true);
    try {
      const response = await client.request("/api/v1/admin/launcher-icon");
      if (!mounted.current) return;
      setData(response.data);
      if (!keepSelection) setSelected(response.data.settings.iconId);
      setError("");
      return response.data;
    } catch (e) {
      if (mounted.current) setError(e.message);
    } finally {
      if (mounted.current) setLoading(false);
    }
  }

  useEffect(() => {
    mounted.current = true;
    load();
    return () => { mounted.current = false; };
  }, [client]);

  async function publish() {
    if (!data || saving || loading || selected === data.settings.iconId) return;
    setSaving(true); setError(""); setMessage("");
    const input = { expectedVersion: data.settings.version, iconId: selected };
    const fingerprint = JSON.stringify(input);
    if (request.current?.fingerprint !== fingerprint) {
      request.current = { fingerprint, body: { ...input, clientRequestId: id() } };
    }
    try {
      const response = await client.request("/api/v1/admin/launcher-icon", { method: "PUT", body: request.current.body });
      if (!mounted.current) return;
      // Replays can return the original success after another administrator has
      // since changed the target. Always re-read the current durable selection.
      request.current = null;
      setData((old) => ({ ...old, settings: response.data }));
      setSelected(response.data.iconId);
      setMessage("图标配置已发布。在线 App 将收到切换通知，离线设备会在下次打开时同步。");
      await load();
    } catch (e) {
      if (!mounted.current) return;
      if (e.code === "VERSION_CONFLICT" || e.code === "SOCIAL_VERSION_CONFLICT" || e.status === 409) {
        request.current = null;
        const latest = await load(true);
        if (!mounted.current) return;
        if (latest?.settings.iconId === input.iconId) {
          setError("");
          setMessage("当前已启用你选择的图标，无需重复发布。");
        } else {
          setError(latest ? "配置已被更新，已读取最新状态。请核对选择后再次发布。" : "配置已变化，暂时无法读取最新状态。请刷新后重试。");
        }
      } else {
        setError(e.message);
      }
    } finally {
      if (mounted.current) setSaving(false);
    }
  }

  const current = data?.settings.iconId;
  const changed = data && selected !== current;
  const busy = loading || saving;
  return <section className="launcher-settings" aria-busy={busy}>
    <PageHead eyebrow="APP APPEARANCE" title="应用图标" subtitle="为 Deuterium 切换日常或周年活动外观。">
      <Button secondary disabled={busy} onClick={() => load()}><RefreshCw size={16} />刷新状态</Button>
    </PageHead>

    {error && <div className="launcher-feedback launcher-error" role="alert">{error}{!data && <Button secondary disabled={busy} onClick={() => load()}>重试</Button>}</div>}
    {message && <div className="launcher-feedback launcher-success" role="status"><CheckCircle2 size={19} /><span>{message}</span></div>}

    {data ? <>
      <div className="launcher-current">
        <img src={artwork[current]} alt="" />
        <div><span className="launcher-label">当前启用</span><strong>{titles[current] || current}</strong></div>
        <span className="launcher-live"><i />已发布</span>
        <time dateTime={data.settings.updatedAt}>更新于 {new Date(data.settings.updatedAt).toLocaleString("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" })}</time>
      </div>

      <div className="launcher-layout">
        <fieldset className="launcher-options" disabled={busy}>
          <legend>选择图标</legend>
          <div className="launcher-card-grid">
            {data.icons.map((icon) => <label key={icon.id} className={`launcher-choice ${selected === icon.id ? "is-selected" : ""}`}>
              <input type="radio" name="launcher-icon" value={icon.id} checked={selected === icon.id} disabled={!artwork[icon.id]} onChange={() => { setSelected(icon.id); setMessage(""); }} />
              <span className="launcher-choice-check" aria-hidden="true">{selected === icon.id && <Check size={15} strokeWidth={3} />}</span>
              <span className={`launcher-art-stage ${icon.id === "anniversary_911" ? "is-anniversary" : ""}`}>
                <img src={artwork[icon.id]} alt={`${icon.name}预览`} />
              </span>
              <span className="launcher-choice-heading"><strong>{icon.name}</strong>{current === icon.id && <span>当前</span>}</span>
              <span className="launcher-choice-description">{icon.description}</span>
            </label>)}
          </div>
        </fieldset>

        <aside className="launcher-preview-panel">
          <h2><Smartphone size={18} />桌面预览</h2>
          <div className="launcher-mask-preview">
            <figure><img className="mask-rounded" src={artwork[selected]} alt="圆角方形预览" /><figcaption>圆角方形</figcaption></figure>
            <figure><img className="mask-circle" src={artwork[selected]} alt="圆形预览" /><figcaption>圆形</figcaption></figure>
          </div>
          <p>图标外框由手机桌面决定。</p>
          <div className="launcher-compatibility"><strong>适用于 App 2.0.8 及以上</strong><span>旧版本需先升级。系统主题图标可能显示简洁的单色 D。</span></div>
        </aside>
      </div>

      <div className="launcher-publish-bar">
        <div>{changed ? <><span>准备切换</span><strong>{titles[current]} <ArrowRight size={15} /> {titles[selected]}</strong></> : <><strong>正在使用{titles[current]}</strong><span>选择另一款图标即可切换。</span></>}</div>
        <Button disabled={!changed || busy} onClick={publish}>{saving ? <><LoaderCircle size={17} className="launcher-spinner" />正在发布…</> : changed ? `启用${titles[selected]}` : "当前图标已启用"}</Button>
      </div>
      <p className="launcher-footnote">可切换的图标已内置于 App。新增款式需随新版 App 发布；这里的发布状态不代表所有设备已完成桌面刷新。</p>
    </> : !error && <div className="launcher-loading" role="status"><LoaderCircle size={22} className="launcher-spinner" />正在读取图标配置…</div>}
  </section>;
}
