import encode, { init } from "@jsquash/webp/encode.js";
import wasm from "@jsquash/webp/codec/enc/webp_enc.wasm?url";
import simdWasm from "@jsquash/webp/codec/enc/webp_enc_simd.wasm?url";
import { parseGIF, decompressFrame } from "gifuct-js";
import { animatedWebp, outputSize, webpChunks, frameWebp, imageOrientation } from "./chat-media-codec.js";

const initialized = init(undefined, { locateFile: (path) => path.includes("simd") ? simdWasm : wasm });
const encodeCanvas = async (canvas, quality, lossless = false) => { await initialized; return new Uint8Array(await encode(canvas.getContext("2d").getImageData(0, 0, canvas.width, canvas.height), { quality, lossless: lossless ? 1 : 0, method: 3, low_memory: 1, exact: 1 })); };
function scaled(source, width, height) { const c = new OffscreenCanvas(width, height); const ctx = c.getContext("2d", { colorSpace: "srgb" }); ctx.imageSmoothingQuality = "high"; ctx.drawImage(source, 0, 0, width, height); return c; }
function sourceSize(b) {
  const view = new DataView(b.buffer, b.byteOffset, b.byteLength);
  if (b.length >= 24 && b[0] === 137 && b[1] === 80) return [view.getUint32(16), view.getUint32(20)];
  if (b.length >= 10 && b[0] === 71 && b[1] === 73) return [view.getUint16(6, true), view.getUint16(8, true)];
  if (b[0] === 255 && b[1] === 216) { for (let p = 2; p + 9 < b.length;) { if (b[p] !== 255) break; const type = b[p + 1], n = view.getUint16(p + 2); if (n < 2) break; if ([192, 193, 194, 195, 197, 198, 199, 201, 202, 203, 205, 206, 207].includes(type)) return [view.getUint16(p + 7), view.getUint16(p + 5)]; p += n + 2; } }
  if (b.length >= 30 && b[0] === 82) { const c = webpChunks(b)[0], d = c.data; if (c.type === "VP8X") return [1 + (d[4] | d[5] << 8 | d[6] << 16), 1 + (d[7] | d[8] << 8 | d[9] << 16)]; if (c.type === "VP8 ") return [(d[6] | d[7] << 8) & 16383, (d[8] | d[9] << 8) & 16383]; if (c.type === "VP8L") { const bits = new DataView(d.buffer, d.byteOffset + 1, 4).getUint32(0, true); return [(bits & 16383) + 1, ((bits >>> 14) & 16383) + 1]; } }
  throw new Error("请选择 JPEG、PNG、WebP 或 GIF 图片。");
}

async function compressStatic(bytes, type, sticker) {
  const [w, h] = sourceSize(bytes), orientation = imageOrientation(bytes); const size = orientation >= 5 ? outputSize(h, w, sticker) : outputSize(w, h, sticker);
  const bitmap = await createImageBitmap(new Blob([bytes], { type }), { resizeWidth: size.width, resizeHeight: size.height, imageOrientation: "from-image" });
  let canvas = scaled(bitmap, bitmap.width, bitmap.height); bitmap.close();
  const textLike = size.long || type === "image/png";
  const ceiling = sticker ? 512 * 1024 : 2 * 1024 * 1024, target = sticker ? 200 * 1024 : textLike ? 1024 * 1024 : 800 * 1024;
  let image;
  const qualities = textLike ? [90, 88, 84] : sticker ? [85, 80, 76] : [82, 77, 72];
  for (let i = 0; i < 3; i++) { const attempt = await encodeCanvas(canvas, qualities[i], textLike && i === 0); if (!image || attempt.length < image.length) image = attempt; if (image.length <= target) break; }
  if (image.length > ceiling) throw new Error("图片压缩后仍然过大，请裁剪后再试。");
  const thumbScale = Math.min(1, 320 / Math.max(canvas.width, canvas.height));
  const thumb = await encodeCanvas(scaled(canvas, Math.max(1, Math.floor(canvas.width * thumbScale)), Math.max(1, Math.floor(canvas.height * thumbScale))), 70);
  return { image, thumbnail: thumb, width: canvas.width, height: canvas.height, animated: false };
}

async function compressAnimation(bytes, type) {
  const [w, h] = sourceSize(bytes); if (w > 2048 || h > 2048 || w * h > 4_000_000) throw new Error("动图尺寸过大，请选择较小的动图。");
  const size = outputSize(w, h, false, true), canvas = new OffscreenCanvas(w, h), ctx = canvas.getContext("2d");
  let inputs, loop = 0;
  if (type === "image/gif") {
    const gif = parseGIF(bytes.buffer); const frames = gif.frames.filter(f => f.image);
    if (frames.length > 600 || frames.length * w * h > 160_000_000) throw new Error("动图帧数或尺寸过大。");
    const app = gif.frames.find(f => f.application?.id?.startsWith("NETSCAPE"))?.application?.blocks;
    if (app?.length >= 3) loop = app[1] | app[2] << 8;
    inputs = frames.map(f => ({ duration: Math.max(20, (f.gce?.delay || 10) * 10), draw: () => { const d = f.image.descriptor; if (d.left + d.width > w || d.top + d.height > h) throw new Error("动图内容不完整。"); const frame = decompressFrame(f, gif.gct, true); const before = frame.disposalType === 3 ? ctx.getImageData(0, 0, w, h) : null; const patch = new OffscreenCanvas(d.width, d.height); patch.getContext("2d").putImageData(new ImageData(frame.patch, d.width, d.height), 0, 0); ctx.drawImage(patch, d.left, d.top); return () => { if (frame.disposalType === 2) ctx.clearRect(d.left, d.top, d.width, d.height); else if (before) ctx.putImageData(before, 0, 0); }; } }));
  } else {
    const chunks = webpChunks(bytes), anim = chunks.find(c => c.type === "ANIM"); if (!anim) throw new Error("动图内容不完整。"); loop = anim.data[4] | anim.data[5] << 8;
    const read24 = (d, p) => d[p] | d[p + 1] << 8 | d[p + 2] << 16;
    inputs = chunks.filter(c => c.type === "ANMF").map(({ data: d }) => ({ duration: Math.max(20, read24(d, 12)), draw: async () => { const x = 2 * read24(d, 0), y = 2 * read24(d, 3), fw = 1 + read24(d, 6), fh = 1 + read24(d, 9); if (x + fw > w || y + fh > h) throw new Error("动图尺寸不正确。"); const wrapper = new Uint8Array(12 + d.length - 16); wrapper.set([82, 73, 70, 70], 0); new DataView(wrapper.buffer).setUint32(4, wrapper.length - 8, true); wrapper.set([87, 69, 66, 80], 8); wrapper.set(d.slice(16), 12); const b = await createImageBitmap(new Blob([frameWebp(webpChunks(wrapper), fw, fh)], { type: "image/webp" })); if (d[15] & 2) ctx.clearRect(x, y, fw, fh); ctx.drawImage(b, x, y); b.close(); return () => { if (d[15] & 1) ctx.clearRect(x, y, fw, fh); }; } }));
  }
  if (!inputs.length || inputs.length > 600 || inputs.length * w * h > 160_000_000 || inputs.reduce((n, f) => n + f.duration, 0) > 30000) throw new Error("请选择 30 秒以内、较短的动图。");
  let best, first, outWidth = size.width, outHeight = size.height;
  const interval = Math.max(50, Math.ceil(inputs.reduce((n, f) => n + f.duration, 0) / 300));
  for (const [index, quality] of [75, 68, 62].entries()) {
    const factor = index === 2 ? .75 : 1, width = Math.max(1, Math.floor(size.width * factor)), height = Math.max(1, Math.floor(size.height * factor));
    ctx.clearRect(0, 0, w, h); const frames = []; let next = 0, elapsed = 0, encodedBytes = 0;
    for (const input of inputs) {
      const dispose = await input.draw();
      if (elapsed >= next || !frames.length) {
        if (frames.length) frames.at(-1).duration = elapsed - frames.at(-1).start;
        const current = scaled(canvas, width, height); if (!frames.length) first = current;
        const encoded = await encodeCanvas(current, quality); encodedBytes += encoded.length;
        frames.push({ bytes: encoded, start: elapsed, duration: input.duration }); next = elapsed + interval;
      }
      elapsed += input.duration; dispose();
      if (frames.length > 300 || encodedBytes > 6 * 1024 * 1024) break;
    }
    if (elapsed !== inputs.reduce((n, f) => n + f.duration, 0)) continue;
    frames.at(-1).duration = elapsed - frames.at(-1).start;
    const encoded = animatedWebp(frames, width, height, loop); if (!best || encoded.length < best.length) { best = encoded; outWidth = width; outHeight = height; }
    if (best.length <= 1024 * 1024) break;
  }
  if (!best || best.length > 2 * 1024 * 1024) throw new Error("动图压缩后仍然过大，请选择较短的动图。");
  const factor = Math.min(1, 320 / Math.max(first.width, first.height));
  return { image: best, thumbnail: await encodeCanvas(scaled(first, Math.max(1, Math.floor(first.width * factor)), Math.max(1, Math.floor(first.height * factor))), 70), width: outWidth, height: outHeight, animated: true };
}

self.onmessage = async ({ data }) => {
  try {
    if (data.bytes.byteLength < 1 || data.bytes.byteLength > 20 * 1024 * 1024) throw new Error("请选择 20MB 以内的图片。");
    const bytes = new Uint8Array(data.bytes), gif = bytes[0] === 71 && bytes[1] === 73;
    const webpAnimated = bytes[0] === 82 && webpChunks(bytes).some(c => c.type === "ANIM");
    const result = gif || webpAnimated ? await compressAnimation(bytes, gif ? "image/gif" : "image/webp") : await compressStatic(bytes, data.type, data.sticker);
    if (result.thumbnail.length > 80 * 1024) throw new Error("图片预览未能处理，请选择其他图片。");
    self.postMessage({ id: data.id, ...result }, [result.image.buffer, result.thumbnail.buffer]);
  } catch (error) { self.postMessage({ id: data.id, error: error.message || "图片未能处理，请重新选择。" }); }
};
