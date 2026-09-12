import React, { useEffect, useRef, useState } from "react";
import { ArrowUpRight, CheckCircle2, Copy, ImagePlus, LoaderCircle, Plus, RefreshCw, Save, Upload, Trash2 } from "lucide-react";
import { Button, Field, Modal, PageHead, Tabs } from "./components.jsx";
import { id } from "./format.js";
import { useUnsavedChanges } from "./unsaved-changes.js";
import "./desktop-launcher.css";

const copy = (value) => JSON.parse(JSON.stringify(value));
const formatDate = (value) => value ? new Date(value).toLocaleString("zh-CN") : "尚未发布";

async function sendMedia(client, file, progress) {
  if (!file) return null;
  if (file.size > 128 * 1024 * 1024) throw new Error("文件不能超过 128 MB。");
  const sha256 = [...new Uint8Array(await crypto.subtle.digest("SHA-256", await file.arrayBuffer()))].map((n) => n.toString(16).padStart(2, "0")).join("");
  const prepared = (await client.request("/api/v1/admin/desktop-launcher/media", { method: "POST", body: { contentType: file.type, size: file.size, sha256 } })).data;
  await new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("PUT", prepared.uploadUrl);
    for (const [key, values] of Object.entries(prepared.headers || {})) {
      if (!["host", "content-length"].includes(key.toLowerCase())) xhr.setRequestHeader(key, Array.isArray(values) ? values.join(",") : values);
    }
    if (!Object.keys(prepared.headers || {}).some((key) => key.toLowerCase() === "content-type")) xhr.setRequestHeader("Content-Type", file.type);
    xhr.upload.onprogress = (event) => progress(event.lengthComputable ? Math.round(event.loaded / event.total * 100) : 0);
    xhr.onload = () => xhr.status >= 200 && xhr.status < 300 ? resolve() : reject(new Error(`素材上传失败（HTTP ${xhr.status}）。`));
    xhr.onerror = () => reject(new Error("上传连接中断，请重试。"));
    xhr.ontimeout = () => reject(new Error("上传超时，请重试。"));
    xhr.timeout = 15 * 60 * 1000;
    xhr.send(file);
  });
  progress(100);
  return (await client.request("/api/v1/admin/desktop-launcher/media/complete", { method: "POST", body: { key: prepared.key } })).data;
}

function MediaButton({ label, accept = "image/png,image/jpeg,image/webp", disabled, onFile }) {
  const input = useRef(null);
  return <><Button secondary disabled={disabled} onClick={() => input.current?.click()}><Upload size={15} />{label}</Button><input ref={input} hidden type="file" accept={accept} onChange={(event) => { onFile(event.target.files?.[0]); event.target.value = ""; }} /></>;
}

export default function DesktopLauncherSettings({ client }) {
  const [data, setData] = useState(null), [draft, setDraft] = useState(null), [entryIndex, setEntryIndex] = useState(0),
    [tab, setTab] = useState("内容与素材"), [busy, setBusy] = useState(false), [error, setError] = useState(""), [message, setMessage] = useState(""),
    [upload, setUpload] = useState(null), [access, setAccess] = useState(null);
  const mounted = useRef(false), saveRequest = useRef(null), syncRequest = useRef(null);
  const changes = useUnsavedChanges(draft, busy);
  async function load(keepDraft = false) {
    try {
      const result = (await client.request("/api/v1/admin/desktop-launcher")).data;
      if (!mounted.current) return;
      setData(result);
      if (!keepDraft) { const next = copy(result.settings.draft); setDraft(next); changes.markSaved(next); }
      if (result.sync?.state !== "RUNNING") syncRequest.current = null;
      return result;
    } catch (e) { if (mounted.current) setError(e.message); }
  }
  useEffect(() => { mounted.current = true; load(); return () => { mounted.current = false; }; }, [client]);
  useEffect(() => {
    if (data?.sync?.state !== "RUNNING") return;
    const timer = setInterval(() => load(true), 3000);
    return () => clearInterval(timer);
  }, [data?.sync?.state, client]);
  const mutate = (operation) => setDraft((old) => { const next = copy(old); operation(next); return next; });
  const entry = draft?.content?.entries?.[entryIndex];
  const setEntry = (patch) => mutate((next) => Object.assign(next.content.entries[entryIndex], patch));
  const previewURL = (path) => !path ? "" : /^(https:|data:)/.test(path) ? path : `${data?.mcpatch?.assetBaseUrl || ""}${path}`;
  async function uploadFile(file, apply) {
    if (!file || busy) return;
    setBusy(true); setError(""); setUpload({ name: file.name, value: 0 });
    try {
      const media = await sendMedia(client, file, (value) => mounted.current && setUpload({ name: file.name, value }));
      if (!mounted.current) return;
      mutate((next) => { next.content.mediaAssets ||= {}; next.content.mediaAssets[media.url] = media; apply(next, media.url); });
      setMessage("素材已上传并校验。发布内容后，启动器会自动获取。");
    } catch (e) { if (mounted.current) setError(e.message); }
    finally { if (mounted.current) { setBusy(false); setUpload(null); } }
  }
  async function save(publish) {
    if (busy || !draft || !data) return;
    setBusy(true); setError(""); setMessage("");
    const body = { expectedVersion: data.settings.version, document: draft, publish }, fingerprint = JSON.stringify(body);
    if (saveRequest.current?.fingerprint !== fingerprint) saveRequest.current = { fingerprint, body: { ...body, clientRequestId: id() } };
    try {
      await client.request("/api/v1/admin/desktop-launcher", { method: "PUT", body: saveRequest.current.body });
      if (!mounted.current) return;
      saveRequest.current = null;
      await load(); setMessage(publish ? "启动器内容已发布。玩家会在启动或刷新时获取新内容。" : "草稿已保存，玩家看到的内容保持为上次发布版本。");
    } catch (e) { if (mounted.current) setError(e.message); }
    finally { if (mounted.current) setBusy(false); }
  }
  async function synchronize() {
    if (data?.sync?.state === "RUNNING") return;
    setError("");
    syncRequest.current ||= { clientRequestId: id() };
    let nativeToken = "";
    try { nativeToken = localStorage.getItem("dlauncher.mcpatch.token") || ""; } catch {}
    try {
      const result = (await client.request("/api/v1/admin/desktop-launcher/sync", { method: "POST", body: { ...syncRequest.current, ...(nativeToken ? { nativeToken } : {}) } })).data;
      if (mounted.current) setData((old) => ({ ...old, sync: result }));
      await load(true);
    } catch (e) { if (mounted.current) { setError(e.message); await load(true); } }
  }
  async function loginInformation() {
    try { const result = await client.request("/api/v1/admin/desktop-launcher/access", { method: "POST", body: {} }); if (mounted.current) setAccess(result.data); }
    catch (e) { if (mounted.current) setError(e.message); }
  }
  const sync = data?.sync, syncing = sync?.state === "RUNNING", versions = data?.mcpatch?.versions || [];
  return <section className="desktop-launcher-admin">
    <PageHead eyebrow="DESKTOP LAUNCHER" title="启动器" subtitle="管理客户端更新、安装基线和启动器上的内容。"><Button secondary disabled={busy} onClick={() => load(true)}><RefreshCw size={16} />刷新状态</Button></PageHead>
    {error && <div className="dl-message is-error" role="alert">{error}</div>}
    {message && <div className="dl-message is-success" role="status"><CheckCircle2 size={18} />{message}</div>}
    <div className="dl-publish-card panel">
      <div><span className="dl-eyebrow">客户端资源</span><h2>McPatch 与 OSS</h2><p>在 McPatch 上传文件、打包版本，完成后在这里同步给玩家。</p><div className="dl-version-pair"><span>最新打包<strong>{versions.at(-1)?.label || "尚未打包"}</strong></span><span>最近同步<strong>{sync ? formatDate(sync.updatedAt) : "尚未同步"}</strong></span></div></div>
      <div className="dl-publish-actions"><a className="dl-external" href={data?.mcpatch?.url || "/mcpatch/"} target="_blank" rel="noreferrer">打开 McPatch 管理页<ArrowUpRight size={17} /></a><Button secondary disabled={!data?.mcpatch?.connected} onClick={loginInformation}>管理页登录信息</Button><Button disabled={syncing || !data?.mcpatch?.connected} onClick={synchronize}>{syncing ? <><LoaderCircle className="dl-spin" size={17} />正在同步到 OSS…</> : <><RefreshCw size={17} />同步到 OSS</>}</Button></div>
      {sync && <div className={`dl-sync-result ${sync.state === "FAILED" ? "is-error" : ""}`} role="status">{syncing && <span className="dl-progress-indeterminate" />}<strong>{{ RUNNING: "同步进行中", SUCCEEDED: "同步完成", FAILED: "同步未完成" }[sync.state]}</strong><span>{sync.message}</span></div>}
      {data?.mcpatch?.error && <p className="auth-error">{data.mcpatch.error}</p>}
    </div>
    {!draft?.content?.entries ? <p className="dl-loading">{data ? "安装基线尚未初始化。" : "正在读取启动器配置…"}</p> : <>
      <div className="dl-editor-heading"><Tabs values={["内容与素材", "安装配置"]} value={tab} onChange={setTab} /><span>已发布版本 {data.settings.publishedVersion || "—"}</span></div>
      <fieldset disabled={busy} className="dl-editor-fieldset">
        {tab === "内容与素材" ? <>
          <div className="dl-entry-switch">{draft.content.entries.map((value, index) => <button type="button" key={value.key} className={index === entryIndex ? "is-selected" : ""} onClick={() => setEntryIndex(index)}><img src={previewURL(value.icon)} alt="" /><span>{value.name}</span></button>)}</div>
          {entry && <>
            <div className="dl-editor-grid"><div className="panel dl-form-card"><h3>页面与背景</h3><Field label="入口名称" value={entry.name} maxLength={60} onChange={(e) => setEntry({ name: e.target.value })} /><div className="dl-background-preview">{entry.background ? <img src={previewURL(entry.background)} alt="当前背景" /> : <ImagePlus size={30} />}</div><div className="button-row"><MediaButton label="更换静态背景" onFile={(file) => uploadFile(file, (next, image) => { next.content.entries[entryIndex].background = image; })} /><MediaButton label="更换入口图标" onFile={(file) => uploadFile(file, (next, image) => { next.content.entries[entryIndex].icon = image; })} /></div></div>
              <div className="panel dl-form-card"><h3>动态背景</h3>{entry.video ? <video className="dl-video-preview" src={previewURL(entry.video)} controls muted preload="metadata" /> : <div className="dl-media-empty">当前使用静态背景</div>}<div className="button-row"><MediaButton label={entry.video ? "更换视频" : "上传视频"} accept="video/mp4" onFile={(file) => uploadFile(file, (next, video) => { next.content.entries[entryIndex].video = video; })} />{entry.video && <Button secondary onClick={() => setEntry({ video: "" })}>使用静态背景</Button>}</div><p className="dl-note">支持 MP4，建议保持原画面比例。视频下载完成后再切换播放。</p></div></div>
            <div className="panel dl-form-card"><div className="dl-card-heading"><h3>轮播宣传图</h3><Button secondary disabled={(entry.banners || []).length >= 8} onClick={() => setEntry({ banners: [...(entry.banners || []), { title: "新宣传图", image: "" }] })}><Plus size={16} />添加</Button></div>{(entry.banners || []).map((banner, index) => <div className="dl-banner-row" key={index}><div className="dl-banner-preview">{banner.image && <img src={previewURL(banner.image)} alt={banner.title} />}</div><Field label={`宣传图 ${index + 1}`} value={banner.title} maxLength={100} onChange={(e) => mutate((next) => { next.content.entries[entryIndex].banners[index].title = e.target.value; })} /><MediaButton label="上传图片" onFile={(file) => uploadFile(file, (next, image) => { next.content.entries[entryIndex].banners[index].image = image; })} /><Button secondary aria-label={`删除宣传图 ${index + 1}`} onClick={() => setEntry({ banners: entry.banners.filter((_, n) => n !== index) })}><Trash2 size={16} /></Button></div>)}</div>
            <div className="panel dl-form-card"><div className="dl-card-heading"><h3>公告、新闻与资讯</h3><Button secondary disabled={(entry.news || []).length >= 60} onClick={() => setEntry({ news: [{ title: "", tab: entry.tabs[0], date: new Date().toLocaleDateString("zh-CN", { month: "2-digit", day: "2-digit" }), body: "", images: [] }, ...(entry.news || [])] })}><Plus size={16} />新建内容</Button></div>
              {(entry.news || []).map((news, index) => <details className="dl-news-editor" key={`${entry.key}-${index}`}><summary><span className="dl-news-category">{news.tab}</span><strong>{news.title || "未命名内容"}</strong><time>{news.date}</time></summary><div className="dl-news-fields"><Field label="标题" value={news.title} maxLength={150} onChange={(e) => mutate((next) => { next.content.entries[entryIndex].news[index].title = e.target.value; })} /><div className="dl-inline-fields"><Field label="分类"><select value={news.tab} onChange={(e) => mutate((next) => { next.content.entries[entryIndex].news[index].tab = e.target.value; })}>{entry.tabs.map((value) => <option key={value}>{value}</option>)}</select></Field><Field label="日期" value={news.date} onChange={(e) => mutate((next) => { next.content.entries[entryIndex].news[index].date = e.target.value; })} /></div><Field label="正文"><textarea rows={5} maxLength={20000} value={news.body || ""} onChange={(e) => mutate((next) => { next.content.entries[entryIndex].news[index].body = e.target.value; })} /></Field><div className="dl-news-images">{(news.images || []).map((image, n) => <figure key={image}><img src={previewURL(image)} alt={`正文图片 ${n + 1}`} /><button type="button" onClick={() => mutate((next) => { next.content.entries[entryIndex].news[index].images.splice(n, 1); })}>移除</button></figure>)}</div><div className="button-row"><MediaButton label="添加正文图片" onFile={(file) => uploadFile(file, (next, image) => { (next.content.entries[entryIndex].news[index].images ||= []).push(image); })} /><Button secondary onClick={() => setEntry({ news: entry.news.filter((_, n) => n !== index) })}>删除这条内容</Button></div></div></details>)}
            </div>
          </>}
        </> : <div className="panel dl-form-card dl-install-settings"><h3>首次安装基线</h3><p className="dl-note">先安装官方或镜像来源的基础游戏，再补齐选定基线，最后自动更新到 OSS 已同步的最新版本。</p><div className="dl-inline-fields"><Field label="Minecraft" readOnly value={draft.bootstrap.minecraftVersion} /><Field label="NeoForge" readOnly value={draft.bootstrap.loaderVersion} /></div><Field label="客户端实例" readOnly value={draft.bootstrap.instanceVersion} /><Field label="初始资源基线"><select value={draft.bootstrap.baselineVersion} onChange={(e) => mutate((next) => { next.bootstrap.baselineVersion = e.target.value; })}>{[...new Set([draft.bootstrap.baselineVersion, ...versions.map((v) => v.label)])].map((label) => <option key={label}>{label}</option>)}</select></Field><label className="checkbox-row"><input type="checkbox" checked={draft.bootstrap.enabled !== false} onChange={(e) => mutate((next) => { next.bootstrap.enabled = e.target.checked; })} />允许新玩家下载安装客户端</label><div className="dl-install-receipt"><strong>推荐设置</strong><span>分代 ZGC · Java 21</span><span>官方 / 镜像获取基础游戏，OSS 提供服务器补充内容</span></div></div>}
      </fieldset>
      {upload && <div className="dl-upload-progress" role="status"><span>{upload.name} · {upload.value === 100 ? "正在校验" : `${upload.value}%`}</span><progress value={upload.value} max={100} /></div>}
      <div className="dl-save-bar"><span>上次发布：{formatDate(data.settings.publishedAt)}</span><div><Button secondary disabled={busy} onClick={() => save(false)}><Save size={16} />保存草稿</Button><Button disabled={busy} onClick={() => save(true)}>{busy ? "正在处理…" : "发布内容"}</Button></div></div>
    </>}
    {access && <Modal title="McPatch 管理页登录" close={() => setAccess(null)}><p>此账号用于上传客户端文件与打包更新。</p><Field label="用户名" value={access.username} readOnly /><Field label="密码" type="password" value={access.password} readOnly /><div className="button-row"><Button secondary onClick={async () => { try { await navigator.clipboard.writeText(access.password); setMessage("McPatch 管理密码已复制。"); } catch { setError("复制失败，请从密码框选择复制。"); } }}><Copy size={15} />复制密码</Button><a className="dl-external" href={access.url} target="_blank" rel="noreferrer">打开管理页<ArrowUpRight size={16} /></a></div></Modal>}
  </section>;
}
