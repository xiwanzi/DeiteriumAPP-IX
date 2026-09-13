import React, { useEffect, useRef, useState } from "react";
import { CheckCircle2, FileArchive, LoaderCircle, RefreshCw, ShieldCheck, Upload } from "lucide-react";
import launcherIcon from "./assets/launcher-icons/default.svg";
import { Button } from "./components.jsx";
import { id } from "./format.js";

function unpack(envelope) {
  if (!envelope || typeof envelope.payload !== "string" || typeof envelope.signature !== "string") throw new Error("请选择构建生成的版本清单。");
  return JSON.parse(new TextDecoder().decode(Uint8Array.from(atob(envelope.payload), (c) => c.charCodeAt(0))));
}
const megabytes = (value) => `${(value / 1048576).toFixed(1)} MB`;
const endpoint = "/api/v1/admin/desktop-launcher/application";

export default function LauncherApplicationRelease({ client, onBusyChange }) {
  const [data, setData] = useState(null), [selection, setSelection] = useState(null), [error, setError] = useState(""),
    [message, setMessage] = useState(""), [phase, setPhase] = useState("idle"), [progress, setProgress] = useState(0);
  const input = useRef(null), mounted = useRef(false), request = useRef(null), upload = useRef(null), operation = useRef(null);
  const busy = phase !== "idle";
  useEffect(() => { onBusyChange?.(busy); return () => onBusyChange?.(false); }, [busy, onBusyChange]);
  async function load() {
    const result = (await client.request(endpoint)).data;
    if (!result?.application) throw new Error("启动器版本信息格式不正确。");
    if (mounted.current) setData(result);
    return result;
  }
  useEffect(() => {
    mounted.current = true;
    load().catch((e) => mounted.current && setError(e.message));
    return () => { mounted.current = false; upload.current?.abort(); operation.current?.abort(); };
  }, [client]);
  async function choose(files) {
    if (busy) return;
    setError(""); setMessage(""); setSelection(null); request.current = null;
    try {
      const list = [...files], zip = list.find((f) => /\.zip$/i.test(f.name)), metadata = list.find((f) => /\.json$/i.test(f.name));
      if (list.length !== 2 || !zip || !metadata || metadata.size > 64000) throw new Error("请同时选择构建生成的更新 ZIP 和版本 JSON 两个文件。");
      const envelope = JSON.parse(await metadata.text()), release = unpack(envelope);
      if (!release || release.product !== "DLauncher" || release.platform !== "windows-x64"
          || typeof release.version !== "string" || !/^\d{1,3}\.\d{1,3}\.\d{1,3}$/.test(release.version)
          || !Number.isSafeInteger(release.versionCode) || release.versionCode < 1
          || typeof release.notes !== "string" || release.notes.length > 16000
          || zip.size !== release.package?.size || zip.size > 1024 ** 3) throw new Error("更新包与版本清单不匹配，请重新选择。");
      if (mounted.current) setSelection({ zip, envelope, release });
    } catch (e) { if (mounted.current) setError(e.message); }
  }
  async function publish() {
    if (busy || !selection || !data) return;
    setError(""); setMessage(""); setProgress(0); setPhase("preparing");
    const controller = new AbortController(); operation.current = controller;
    request.current ||= { clientRequestId: id(), expectedRevision: data.application.revision, release: selection.envelope };
    try {
      const prepared = (await client.request(`${endpoint}/upload`, { method: "POST", body: { release: selection.envelope }, signal: controller.signal })).data;
      if (!mounted.current) return;
      setPhase("uploading");
      await new Promise((resolve, reject) => {
        const xhr = new XMLHttpRequest(); upload.current = xhr;
        xhr.open("PUT", prepared.uploadUrl);
        for (const [name, value] of Object.entries(prepared.headers || {})) {
          if (!["host", "content-length"].includes(name.toLowerCase())) xhr.setRequestHeader(name, Array.isArray(value) ? value.join(",") : value);
        }
        if (!Object.keys(prepared.headers || {}).some((k) => k.toLowerCase() === "content-type")) xhr.setRequestHeader("Content-Type", "application/zip");
        xhr.timeout = 30 * 60 * 1000;
        xhr.upload.onprogress = (event) => mounted.current && setProgress(event.lengthComputable ? Math.round(event.loaded / event.total * 100) : 0);
        xhr.onload = () => xhr.status >= 200 && xhr.status < 300 ? resolve() : reject(new Error(`上传失败（HTTP ${xhr.status}），可以重试。`));
        xhr.onerror = () => reject(new Error("上传连接中断，请重试。"));
        xhr.ontimeout = () => reject(new Error("上传超时，请重试。"));
        xhr.onabort = () => reject(new Error("上传已取消。"));
        xhr.send(selection.zip);
      });
      if (!mounted.current) return;
      setPhase("verifying");
      await client.request(`${endpoint}/publish`, { method: "POST", body: request.current, signal: AbortSignal.any([controller.signal, AbortSignal.timeout(180000)]) });
      if (!mounted.current) return;
      request.current = null;
      await load();
      setMessage(`启动器 ${selection.release.version} 已发布。已接入自更新的旧版玩家，下次打开时会进入强制更新。`);
      setSelection(null);
    } catch (e) {
      if (!mounted.current) return;
      // A lost response can follow a successful commit; inspect the published release.
      try {
        const current = await load();
        if (current.application.release?.payload === selection.envelope.payload && current.application.release?.signature === selection.envelope.signature) {
          request.current = null; setMessage("已确认此版本发布成功。"); setSelection(null); return;
        }
      } catch {}
      setError(e.message);
    } finally {
      upload.current = null; operation.current = null;
      if (mounted.current) setPhase("idle");
    }
  }
  let published = null;
  try { published = data?.application?.release ? unpack(data.application.release) : null; } catch {}
  return <div className="dl-application-release">
    {error && <div className="dl-message is-error" role="alert">{error}</div>}
    {message && <div className="dl-message is-success" role="status"><CheckCircle2 size={18} />{message}</div>}
    <div className="dl-application-hero">
      <div className="dl-application-identity"><img src={launcherIcon} alt="启动器图标" /><div><span>DEUTERIUM IX</span><h2>启动器版本</h2><p>程序更新 · 玩家社区 · 随包更新器</p></div></div>
      <div className="dl-application-current"><small>当前发布</small><strong>{published ? `v${published.version}` : "尚未发布"}</strong><span>{data?.application?.publishedAt ? new Date(data.application.publishedAt).toLocaleString("zh-CN") : "上传版本文件后即可发布"}</span></div>
    </div>
    <div className="dl-editor-grid">
      <section className="panel dl-form-card">
        <div className="dl-card-heading"><h3>发布新版</h3><Button secondary disabled={busy} aria-label="刷新启动器版本" onClick={() => load().catch((e) => setError(e.message))}><RefreshCw size={16} /></Button></div>
        <p className="dl-note">选择构建生成的 ZIP 与 JSON。后台会验证版本签名和上传文件，校验通过后再发布。</p>
        <button className="dl-release-picker" type="button" disabled={busy} onClick={() => input.current?.click()}><FileArchive size={28} /><strong>{selection ? selection.zip.name : "选择版本文件"}</strong><span>{selection ? megabytes(selection.zip.size) : "同时选择更新包和版本清单"}</span></button>
        <input ref={input} type="file" accept=".zip,.json" multiple hidden onChange={(e) => { choose(e.target.files); e.target.value = ""; }} />
        {selection && <div className="dl-release-selection"><span>待发布版本<strong>v{selection.release.version}</strong></span><span>更新方式<strong>强制更新</strong></span></div>}
        {busy && <div className="dl-upload-progress" role="status"><span><LoaderCircle size={14} className="dl-spin" /> {phase === "preparing" ? "验证版本签名" : phase === "uploading" ? `上传更新包 · ${progress}%` : "校验更新包并发布，请稍候…"}</span><progress max={100} value={phase === "uploading" ? progress : undefined} /></div>}
        <div className="dl-release-action"><Button disabled={busy || !selection || !data?.uploadReady || selection.release.versionCode <= (data?.application?.versionCode || 0)} onClick={publish}><Upload size={16} />{busy ? "正在发布…" : "上传并发布"}</Button></div>
        {selection && selection.release.versionCode <= (data?.application?.versionCode || 0) && <p className="dl-note">请使用高于当前发布版本的新版本号。</p>}
      </section>
      <section className="panel dl-form-card"><h3>{selection ? "本次更新说明" : "已发布的更新说明"}</h3><div className="dl-release-notes">{selection?.release.notes || published?.notes || "选择版本文件后，将在这里显示更新说明。"}</div><div className="dl-release-assurance"><ShieldCheck size={18} /><span>玩家确认更新后会进入独立更新器；完成安装即可重新打开启动器。</span></div></section>
    </div>
  </div>;
}
