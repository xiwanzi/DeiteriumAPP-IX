import SparkMD5 from "spark-md5";
import { id } from "./format.js";

export function md5Base64(buffer) {
  const digest = SparkMD5.ArrayBuffer.hash(buffer);
  return btoa(digest.match(/../g).map((byte) => String.fromCharCode(parseInt(byte, 16))).join(""));
}
export async function uploadAsset(client, file, purpose, businessType, businessRef = "", onProgress = () => {}) {
  if (!["image/png", "image/jpeg", "image/webp"].includes(file.type) || file.size < 1 || file.size > 20 * 1024 * 1024) throw new Error("请选择 20 MB 以内的 PNG、JPEG 或 WebP 图片。");
  onProgress(0, "正在准备上传");
  const contentMd5 = md5Base64(await file.arrayBuffer());
  const created = await client.request("/api/v1/assets/uploads", { method: "POST", body: { clientRequestId: id(), purpose, businessType, businessRef, fileName: file.name, contentType: file.type, sizeBytes: file.size, contentMd5, altText: file.name } });
  const session = created.data;
  if (session.status === "READY" && session.asset) return session.asset;
  const authorization = session.authorization; let url;
  try { url = new URL(authorization?.uploadUrl); } catch { throw new Error("图片上传授权无效，请重试。"); }
  if (url.protocol !== "https:" || url.username || url.password || authorization?.method !== "PUT") throw new Error("图片上传授权无效，请重试。");
  let uploadError, uploadedPercent = 0;
  try {
    await new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest(); xhr.open("PUT", url.href); xhr.withCredentials = false; xhr.timeout = 120000;
      for (const [name, value] of Object.entries(authorization.signedHeaders || {})) xhr.setRequestHeader(name, value);
      xhr.upload.onprogress = (event) => { if (event.lengthComputable) { uploadedPercent = Math.round(event.loaded / event.total * 100); onProgress(uploadedPercent, "正在上传"); } };
      xhr.onload = () => xhr.status >= 200 && xhr.status < 300 ? resolve() : reject(new Error(`图片上传未完成（${xhr.status}）`));
      xhr.onerror = () => reject(new Error("图片上传连接中断，正在核对上传结果。"));
      xhr.ontimeout = () => reject(new Error("图片上传超时，正在核对上传结果。")); xhr.send(file);
    });
  } catch (error) { uploadError = error; }
  onProgress(uploadError ? uploadedPercent : 100, uploadError ? "正在核对上传结果" : "正在验证图片");
  const completeRequestId = id();
  for (let attempt = 0; attempt < 8; attempt++) {
    const result = await client.request(`/api/v1/assets/uploads/${encodeURIComponent(session.uploadId)}/complete`, { method: "POST", body: { clientRequestId: completeRequestId, ossRequestId: "" } });
    if (result.data.status === "READY" && result.data.asset?.assetId) return result.data.asset;
    if (["REJECTED", "EXPIRED"].includes(result.data.status)) throw new Error("图片验证未通过，请选择其他图片。");
    if (uploadError && result.data.status === "AUTHORIZED") throw uploadError;
    await new Promise((resolve) => setTimeout(resolve, Math.min(5000, Math.max(1000, (result.data.retryAfterSeconds || 2) * 1000))));
  }
  throw new Error("图片仍在验证，请稍后重试。");
}
