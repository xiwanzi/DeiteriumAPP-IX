import { md5Base64 } from "./assets.js";

const THREE_DAYS = 3 * 86400000;
const digest = async (buffer) => [...new Uint8Array(await crypto.subtle.digest("SHA-256", buffer))].map(b => b.toString(16).padStart(2, "0")).join("");
const dbPromise = () => new Promise((resolve, reject) => { const r = indexedDB.open("deuterium-chat-outbox", 1); r.onupgradeneeded = () => r.result.createObjectStore("jobs", { keyPath: "key" }); r.onsuccess = () => resolve(r.result); r.onerror = () => reject(r.error); });
async function dbAction(mode, action) { const db = await dbPromise(); return new Promise((resolve, reject) => { const tx = db.transaction("jobs", mode), request = action(tx.objectStore("jobs")); tx.oncomplete = () => { db.close(); resolve(request.result); }; tx.onerror = () => { db.close(); reject(tx.error); }; tx.onabort = tx.onerror; }); }

export class ChatMediaClient {
  constructor(client, user, update, accepted) { this.client = client; this.user = user; this.update = update; this.accepted = accepted; this.jobs = []; this.stickers = []; this.cache = new Map(); this.cacheSizes = new Map(); this.requests = new Map(); this.urls = new Set(); this.stopped = false; this.abort = new AbortController(); this.error = ""; }
  active() { if (this.stopped) throw new DOMException("Canceled", "AbortError"); }
  async request(path, options = {}) { this.active(); const r = await this.client.request(`/api/v1${path}`, { ...options, signal: this.abort.signal, expectedPlayerRef: this.user.playerRef }); this.active(); return r.data; }
  emit() { if (!this.stopped) this.update({ jobs: [...this.jobs], stickers: [...this.stickers], error: this.error }); }
  localURL(blob) { const url = URL.createObjectURL(blob); this.urls.add(url); return url; }
  releaseURL(url) { if (url && this.urls.delete(url)) URL.revokeObjectURL(url); }
  cacheBlob(hash, blob) {
    this.active();
    if (this.cache.has(hash)) return this.cache.get(hash);
    let used = [...this.cacheSizes.values()].reduce((a, b) => a + b, 0);
    while (used + blob.size > 64 * 1024 * 1024 && this.cache.size) { const oldest = this.cache.keys().next().value; used -= this.cacheSizes.get(oldest) || 0; this.releaseURL(this.cache.get(oldest)); this.cache.delete(oldest); this.cacheSizes.delete(oldest); }
    const url = this.localURL(blob); this.cache.set(hash, url); this.cacheSizes.set(hash, blob.size); return url;
  }
  async start() {
    if (this.started) { this.emit(); return; } this.started = true;
    try { const jobs = await dbAction("readonly", s => s.getAll()); this.active(); this.jobs = jobs.filter(j => j.owner === this.user.playerRef).map(j => ({ ...j, status: "queued", localURL: j.image ? this.localURL(j.image) : j.file ? this.localURL(j.file) : null })); this.emit(); this.drain(); } catch (e) { if (!this.stopped) { this.error = "暂时无法恢复待发送图片，请检查浏览器存储空间。"; this.emit(); } }
    await this.expireFiles(); if (this.stopped) return; this.expiryTimer = setInterval(() => this.expireFiles().catch(() => {}), 3600000);
    await this.refreshStickers();
  }
  async expireFiles() {
    if (this.stopped) return;
    for (const job of this.jobs) if (Date.now() - job.createdAt > THREE_DAYS && job.status !== "sending") {
      this.releaseURL(job.localURL); job.localURL = null; job.file = null; job.image = null; job.thumbnail = null;
      await this.save(job); if (!job.attempted) { job.status = "failed"; job.error = "待发送图片已超过 3 天，请重新选择。"; }
    }
    this.emit();
  }
  async refreshStickers() { try { const r = await this.request("/chat/stickers"); this.stickers = r.stickers; this.emit(); } catch (e) { if (!this.stopped) { this.error = e.message; this.emit(); } } }
  persisted(job) { const { localURL, status, error, ...stored } = job; return stored; }
  async save(job) { this.active(); await dbAction("readwrite", s => s.put(this.persisted(job))); this.active(); }
  async enqueue(file, reply, sticker = null, importing = false, dimensions = {}) {
    this.active(); if (this.jobs.length >= 10) throw new Error("待发送图片较多，请稍后再试。");
    if (importing && this.stickers.length >= 10) throw new Error("贴纸已满（10/10），删除已有贴纸后可继续添加。");
    if (file && (file.size < 1 || file.size > 20 * 1024 * 1024)) throw new Error("请选择 20MB 以内的图片。");
    const id = crypto.randomUUID(), job = { key: `${this.user.playerRef}:${id}`, owner: this.user.playerRef, id, file, reply, media: sticker, importing, width: dimensions.width, height: dimensions.height, createdAt: Date.now(), status: "queued", type: sticker || importing ? "sticker" : "image", localURL: file ? this.localURL(file) : null };
    await this.save(job); this.jobs.push(job); this.error = ""; this.emit(); this.drain(); return true;
  }
  async retry(id) { const job = this.jobs.find(j => j.id === id); if (job) { job.status = "queued"; job.error = ""; this.emit(); this.drain(); } }
  async dismiss(id) { const job = this.jobs.find(j => j.id === id); if (!job || job.status === "sending") return; await dbAction("readwrite", s => s.delete(job.key)); this.releaseURL(job.localURL); this.jobs = this.jobs.filter(j => j !== job); this.emit(); }
  async compress(job) {
    const worker = new Worker(new URL("./chat-media-worker.js", import.meta.url), { type: "module" }); this.worker = worker;
    try {
      const buffer = await job.file.arrayBuffer(); this.active();
      const result = await new Promise((resolve, reject) => { this.workerReject = reject; worker.onmessage = ({ data }) => data.error ? reject(new Error(data.error)) : resolve(data); worker.onerror = () => reject(new Error("图片未能处理，请重新选择。")); worker.postMessage({ id: job.id, bytes: buffer, type: job.file.type, sticker: job.importing }, [buffer]); });
      this.active(); const previousURL = job.localURL; job.image = new Blob([result.image], { type: "image/webp" }); job.thumbnail = new Blob([result.thumbnail], { type: "image/webp" }); job.width = result.width; job.height = result.height; job.animated = result.animated; job.localURL = this.localURL(job.image); job.file = null;
      await this.save(job);
      this.emit(); setTimeout(() => this.releaseURL(previousURL), 1000);
    } finally { worker.terminate(); this.worker = null; this.workerReject = null; }
  }
  async put(authorization, blob, md5) {
    const u = new URL(authorization?.uploadUrl), headers = authorization.signedHeaders || {}, lower = Object.fromEntries(Object.entries(headers).map(([k, v]) => [k.toLowerCase(), v]));
    if (u.protocol !== "https:" || !u.hostname.endsWith(".rains3.com") || u.username || u.password || authorization.method !== "PUT" || authorization.sizeBytes !== blob.size || lower["content-md5"] !== md5 || lower["content-type"] !== "image/webp" || lower["if-none-match"] !== "*" || Object.keys(lower).some(k => !["content-type", "content-md5", "if-none-match"].includes(k))) throw new Error("图片发送暂不可用，请稍后重试。");
    this.active(); const r = await fetch(u, { method: "PUT", body: blob, headers, credentials: "omit", redirect: "error", signal: this.abort.signal }); if (!r.ok && r.status !== 412) throw new Error("图片发送未完成，请重试。");
  }
  async upload(job) {
    const main = await job.image.arrayBuffer(), thumb = await job.thumbnail.arrayBuffer(); this.active();
    const input = { clientRequestId: job.id, sha256: await digest(main), thumbnailSha256: await digest(thumb), contentMd5: md5Base64(main), thumbnailMd5: md5Base64(thumb), sizeBytes: main.byteLength, thumbnailBytes: thumb.byteLength };
    let session = await this.request("/chat/media/uploads", { method: "POST", body: input });
    if (session.status !== "READY") {
      // Even a failed PUT can have reached S3. Complete checks the same bytes;
      // retries repeat this same request/key rather than making a second upload.
      await Promise.allSettled([this.put(session.authorization, job.image, input.contentMd5), this.put(session.thumbnailAuthorization, job.thumbnail, input.thumbnailMd5)]); this.active();
      for (let i = 0; i < 4; i++) { try { session = await this.request(`/chat/media/uploads/${session.uploadId}/complete`, { method: "POST", body: {} }); break; } catch (e) { if (e.code !== "MEDIA_BUSY" || i === 3) throw e; await new Promise(r => setTimeout(r, 500 * (i + 1))); } }
    }
    if (session.status !== "READY") throw new Error("图片发送未完成，请重试。");
    job.media = { ...session.media, type: job.type }; await this.save(job); return job.media;
  }
  async seed(media, main, thumb) { if (main && await digest(await main.arrayBuffer()) === media.sha256) this.cacheBlob(media.sha256, main); if (thumb && await digest(await thumb.arrayBuffer()) === media.thumbnailSha256) this.cacheBlob(media.thumbnailSha256, thumb); }
  async drain() {
    if (this.running || this.stopped) return; this.running = true;
    try {
      for (const job of this.jobs.filter(j => j.status === "queued")) {
        if (this.stopped) return; job.status = "sending"; this.emit();
        try {
          let result;
          if (job.attempted && !job.importing) { const known = await this.request(`/chat/sends/${job.id}`); if (known.status === "accepted") result = known; }
          if (!result) {
            if (Date.now() - job.createdAt > THREE_DAYS) throw new Error("待发送图片已超过 3 天，请重新选择。");
            if (!job.media) { if (!job.image) await this.compress(job); await this.upload(job); }
            if (job.importing) { await this.request("/chat/stickers", { method: "POST", body: { mediaId: job.media.mediaId } }); await this.refreshStickers(); }
            else {
              job.attempted = true; await this.save(job);
              result = await this.request("/chat/messages", { method: "POST", body: { clientMessageId: job.id, content: job.type === "sticker" ? "[贴纸]" : "[图片]", media: { mediaId: job.media.mediaId, type: job.type }, ...(job.reply?.id ? { replyToMessageId: job.reply.id } : {}) } });
              if (result.status !== "accepted") throw new Error(result.error?.code === "RESULT_UNKNOWN" ? "发送结果待确认，点击重试。" : result.error?.message || "发送结果暂未确认，请重试。");
            }
          }
          if (job.media && job.image) await this.seed(job.media, job.image, job.thumbnail);
          if (result?.message) this.accepted(result.message);
          await dbAction("readwrite", s => s.delete(job.key)); this.jobs = this.jobs.filter(j => j !== job); this.emit(); setTimeout(() => this.releaseURL(job.localURL), 1000);
        } catch (e) { if (this.stopped) return; job.status = "failed"; job.error = e.code === "STORAGE_UNAVAILABLE" ? "图片发送暂不可用，请稍后重试。" : e.message || "发送未完成，请重试。"; if (job.importing) this.error = job.error; this.emit(); }
      }
    } finally { this.running = false; if (!this.stopped && this.jobs.some(j => j.status === "queued")) this.drain(); }
  }
  async mediaURL(media, messageID, thumbnail = false) {
    const hash = thumbnail ? media.thumbnailSha256 : media.sha256; if (this.cache.has(hash)) return this.cache.get(hash);
    const key = hash; if (this.requests.has(key)) { try { return await this.requests.get(key); } catch { this.active(); } }
    const work = (async () => {
      const access = await this.request(`/chat/media/files/${media.mediaId}/access${messageID ? `?messageId=${encodeURIComponent(messageID)}` : ""}`);
      const url = new URL(thumbnail ? access.thumbnailUrl : access.url); if (url.protocol !== "https:" || !url.hostname.endsWith(".rains3.com") || url.username || url.password) throw new Error("图片暂不可用。");
      const r = await fetch(url, { credentials: "omit", signal: this.abort.signal }); if (!r.ok) throw new Error("图片暂不可用，点击重试。");
      const data = await r.arrayBuffer(); this.active(); if (data.byteLength !== (thumbnail ? media.thumbnailBytes : media.sizeBytes) || await digest(data) !== hash) throw new Error("图片不完整，请重试。");
      return this.cacheBlob(hash, new Blob([data], { type: "image/webp" }));
    })(); this.requests.set(key, work); try { return await work; } finally { if (this.requests.get(key) === work) this.requests.delete(key); }
  }
  async deleteSticker(media) { await this.request(`/chat/stickers/${media.mediaId}`, { method: "DELETE" }); await this.refreshStickers(); }
  async collect(media, messageID) {
    if (media.width <= 512 && media.height <= 512 && (media.animated || media.sizeBytes <= 512 * 1024)) {
      try { await this.request("/chat/stickers", { method: "POST", body: { mediaId: media.mediaId } }); await this.refreshStickers(); return; } catch (e) { if (!["MEDIA_EXPIRED", "MEDIA_NOT_FOUND"].includes(e.code)) throw e; }
    }
    const url = await this.mediaURL(media, messageID); const blob = await (await fetch(url)).blob(); return this.enqueue(new File([blob], "sticker.webp", { type: "image/webp" }), null, null, true);
  }
  stop() { this.stopped = true; clearInterval(this.expiryTimer); this.abort.abort(); this.workerReject?.(new DOMException("Canceled", "AbortError")); this.worker?.terminate(); this.urls.forEach(URL.revokeObjectURL); this.urls.clear(); this.cache.clear(); this.cacheSizes.clear(); }
}

// The authenticated client outlives the chat route. Navigating to the shop
// detaches rendering callbacks, while durable sends continue in the background.
export function acquireChatMedia(client,user,update,accepted) {
  let service=client.chatMediaSession;
  if(service && (service.stopped || service.user.playerRef!==user.playerRef)){service.stop();service=null;}
  if(!service){service=new ChatMediaClient(client,user,update,accepted);client.chatMediaSession=service;}
  service.update=update;service.accepted=accepted;service.start();return service;
}
