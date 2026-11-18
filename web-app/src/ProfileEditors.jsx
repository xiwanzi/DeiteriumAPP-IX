import React, { useEffect, useRef, useState } from "react";
import { ArrowRight, ImagePlus } from "lucide-react";
import { Avatar, Button, Field, Modal } from "./components.jsx";
import { id } from "./format.js";
import { uploadAsset } from "./assets.js";

export function ProfileEditor({ client, profile, accept, onSaved, close }) {
  const [base, setBase] = useState(profile), [name, setName] = useState(profile.displayName || profile.gameId), [bio, setBio] = useState(profile.bio || "");
  const [busy, setBusy] = useState(false), [error, setError] = useState("");
  const pending = useRef(null), lock = useRef(false);
  const trimmed = name.trim(), nameLength = Array.from(trimmed).length, bioLength = Array.from(bio).length;
  const nameChanged = trimmed !== (base.displayName || base.gameId), bioChanged = bio !== (base.bio || "");
  const valid = nameLength > 0 && nameLength <= 24 && bioLength <= 200 && (nameChanged || bioChanged);
  const save = async event => {
    event.preventDefault();
    if (lock.current || !valid) return;
    lock.current = true; setBusy(true); setError("");
    const changes = {expectedVersion:base.version, ...(nameChanged ? {displayName:trimmed} : {}), ...(bioChanged ? {bio} : {})};
    const fingerprint = JSON.stringify(changes);
    if (pending.current?.fingerprint !== fingerprint) pending.current = {fingerprint, body:{...changes,clientRequestId:id()}};
    try {
      const response = await client.patchProfile(pending.current.body);
      onSaved(response.data); close();
    } catch (e) {
      if (e.code === "STATE_VERSION_CONFLICT") {
        try {
          const latest = (await client.profile(profile.playerRef)).data;
          accept(latest); setBase(latest);
          if (!nameChanged) setName(latest.displayName || latest.gameId);
          if (!bioChanged) setBio(latest.bio || "");
          pending.current = null;
          setError("其他设备已更新资料。已保留你编辑的内容，请核对后再保存。");
        } catch (refreshError) { setError(refreshError.message); }
      } else setError(e.message);
    } finally { lock.current = false; setBusy(false); }
  };
  return <Modal title="编辑个人资料" className="me-dialog" dismissOnBackdrop={false} close={()=>{if(!lock.current) close();}}>
    <form className="me-form" onSubmit={save}>
      <Field label="昵称" hint={nameLength > 24 ? "昵称最多 24 个字符。" : "昵称用于社区展示，游戏 ID 保持独立。"} required maxLength={48} value={name} disabled={busy} onChange={event=>setName(event.target.value)}/>
      <Field label="个人简介" hint={`${bioLength} / 200`}><textarea rows={4} maxLength={400} value={bio} disabled={busy} onChange={event=>setBio(event.target.value)}/></Field>
      {error && <p className="auth-error" role="alert">{error}</p>}
      <footer><Button secondary disabled={busy} onClick={close}>取消</Button><Button type="submit" disabled={busy || !valid}>{busy?"正在保存…":"保存"}</Button></footer>
    </form>
  </Modal>;
}

export function AvatarEditor({ client, user, profile, firstFile, accept, onSaved, close, upload = uploadAsset }) {
  const [file, setFile] = useState(firstFile), [src, setSrc] = useState(""), [ready, setReady] = useState(false);
  const [error, setError] = useState(""), [progress, setProgress] = useState(null), [busy, setBusy] = useState(false);
  const input = useRef(null), lock = useRef(false), pending = useRef(null);
  useEffect(() => {
    setReady(false); pending.current = null; setProgress(null); setError("");
    if (!["image/png","image/jpeg","image/webp"].includes(file.type) || !file.size || file.size>20*1024*1024) {
      setSrc(""); setError("请选择 20 MB 以内的 PNG、JPEG 或 WebP 图片。"); return;
    }
    const url = URL.createObjectURL(file);
    setSrc(url);
    return () => URL.revokeObjectURL(url);
  }, [file]);
  const save = async () => {
    if (lock.current || !ready) return;
    lock.current = true; setBusy(true); setError("");
    try {
      if (!pending.current) {
        const asset = await upload(client,file,"AVATAR","PROFILE",user.userId,(value,label)=>setProgress({value,label}));
        pending.current = {clientRequestId:id(),expectedVersion:profile.version,avatarAssetId:asset.assetId};
      }
      setProgress({value:100,label:"正在保存头像"});
      const response = await client.patchProfile(pending.current);
      onSaved(response.data); close();
    } catch (e) {
      setProgress(null);
      if (pending.current && e.code==="STATE_VERSION_CONFLICT") {
        try {
          const latest = (await client.profile(user.playerRef)).data;
          accept(latest);
          if (latest.avatar?.assetId === pending.current.avatarAssetId) { onSaved(latest); close(); return; }
          pending.current = {...pending.current, expectedVersion:latest.version,clientRequestId:id()};
          setError("资料已在其他设备更新，请再次确认头像。");
        } catch (refreshError) { setError(refreshError.message); }
      } else setError(e.message);
    } finally { lock.current = false; setBusy(false); }
  };
  return <Modal title="更换头像" className="me-dialog" close={()=>{if(!lock.current) close();}}>
    <div className="me-avatar-editor">
      <div className="me-avatar-pair"><figure><Avatar user={profile} size="large"/><figcaption>当前头像</figcaption></figure><ArrowRight size={19}/>
        <figure>{src ? <img src={src} alt="新头像预览" onLoad={()=>setReady(true)} onError={()=>{setReady(false);setError("无法读取图片，请重新选择。");}}/> : <span className="me-avatar-empty"><ImagePlus size={24}/></span>}<figcaption>新头像</figcaption></figure></div>
      <input ref={input} type="file" accept="image/png,image/jpeg,image/webp" hidden onChange={event=>{const next=event.target.files?.[0]; if(next) setFile(next); event.target.value="";}}/>
      {progress && <div className="me-progress" role="status"><span>{progress.label} · {progress.value}%</span><progress aria-label="头像上传进度" max="100" value={progress.value}/></div>}
      {error && <p className="auth-error" role="alert">{error}</p>}
      <footer><Button secondary disabled={busy} onClick={()=>input.current?.click()}>重新选择</Button><Button disabled={busy || !ready} onClick={save}>{busy?"正在保存…":"使用新头像"}</Button></footer>
    </div>
  </Modal>;
}
