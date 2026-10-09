// Only RIFF framing lives here; pixel compression is performed by libwebp.
// Full-canvas frames use no-blend, preserving transparent pixels without trails.
const ascii = (s) => Uint8Array.from(s, (c) => c.charCodeAt(0));
const join = (arrays) => { const out = new Uint8Array(arrays.reduce((n, a) => n + a.length, 0)); let p = 0; for (const a of arrays) { out.set(a, p); p += a.length; } return out; };
const le = (n, size) => Uint8Array.from({ length: size }, (_, i) => n >>> (i * 8) & 255);
export function riffChunk(name, data) { return join([ascii(name), le(data.length, 4), data, new Uint8Array(data.length % 2)]); }
export function webpChunks(bytes) {
  const b = new Uint8Array(bytes), read = (p) => new DataView(b.buffer, b.byteOffset + p, 4).getUint32(0, true);
  if (b.length < 12 || String.fromCharCode(...b.slice(0, 4)) !== "RIFF" || String.fromCharCode(...b.slice(8, 12)) !== "WEBP" || read(4) !== b.length - 8) throw new Error("图片格式不正确。");
  const chunks = [];
  for (let p = 12; p < b.length;) { if (p + 8 > b.length) throw new Error("图片不完整。"); const n = read(p + 4), end = p + 8 + n; if (end + n % 2 > b.length) throw new Error("图片不完整。"); chunks.push({ type: String.fromCharCode(...b.slice(p, p + 4)), data: b.slice(p + 8, end) }); p = end + n % 2; }
  return chunks;
}
export function frameWebp(chunks, width, height) {
  const header = join([Uint8Array.of(chunks.some(c => c.type === "ALPH") ? 0x10 : 0, 0, 0, 0), le(width - 1, 3), le(height - 1, 3)]);
  const body = join([ascii("WEBP"), riffChunk("VP8X", header), ...chunks.map(c => riffChunk(c.type, c.data))]);
  return join([ascii("RIFF"), le(body.length, 4), body]);
}
export function animatedWebp(frames, width, height, loop = 0) {
  if (!frames.length || frames.length > 300 || width < 1 || height < 1 || width > 512 || height > 512 || frames.reduce((n, f) => n + f.duration, 0) > 30000) throw new Error("动图过长或尺寸过大。");
  const x = join([Uint8Array.of(0x12, 0, 0, 0), le(width - 1, 3), le(height - 1, 3)]);
  const parts = [ascii("WEBP"), riffChunk("VP8X", x), riffChunk("ANIM", join([new Uint8Array(4), le(loop, 2)]))];
  for (const frame of frames) {
    const chunks = webpChunks(frame.bytes).filter(c => ["ALPH", "VP8 ", "VP8L"].includes(c.type));
    if (!chunks.some(c => c.type === "VP8 " || c.type === "VP8L") || frame.duration < 10 || frame.duration > 30000) throw new Error("动图帧不完整。");
    const data = join([new Uint8Array(6), le(width - 1, 3), le(height - 1, 3), le(frame.duration, 3), Uint8Array.of(2), ...chunks.map(c => riffChunk(c.type, c.data))]);
    parts.push(riffChunk("ANMF", data));
  }
  const body = join(parts); return join([ascii("RIFF"), le(body.length, 4), body]);
}
export function outputSize(width, height, sticker = false, animation = false) {
  if (!(width > 0 && height > 0) || width * height > 40_000_000) throw new Error("图片尺寸过大，请裁剪后再试。");
  const long = height / width >= 2.5;
  const factor = Math.min(1, sticker || animation ? 512 / Math.max(width, height) : long ? Math.min(1440 / width, 8192 / height, Math.sqrt(8_000_000 / (width * height))) : 1920 / Math.max(width, height));
  return { width: Math.max(1, Math.floor(width * factor)), height: Math.max(1, Math.floor(height * factor)), long };
}

// Read just orientation before the browser allocates/resizes decoded pixels.
// Canvas still performs the actual orientation and sRGB conversion.
export function imageOrientation(bytes) {
  const b = new Uint8Array(bytes);
  function tiff(data) {
    try {
      let p = data[0] === 69 && data[1] === 120 ? 6 : 0;
      const little = data[p] === 73 && data[p + 1] === 73;
      if (!little && !(data[p] === 77 && data[p + 1] === 77)) return 1;
      const v = new DataView(data.buffer, data.byteOffset + p, data.length - p);
      if (v.getUint16(2, little) !== 42) return 1;
      const ifd = v.getUint32(4, little), count = v.getUint16(ifd, little);
      for (let i = 0; i < Math.min(count, 256); i++) { const at = ifd + 2 + i * 12; if (v.getUint16(at, little) === 0x112 && v.getUint16(at + 2, little) === 3 && v.getUint32(at + 4, little) === 1) { const orientation = v.getUint16(at + 8, little); return orientation >= 1 && orientation <= 8 ? orientation : 1; } }
    } catch { /* Malformed optional metadata does not control allocation. */ }
    return 1;
  }
  if (b[0] === 255 && b[1] === 216) {
    for (let p = 2; p + 4 <= b.length;) { if (b[p] !== 255) break; const type = b[p + 1], n = b[p + 2] * 256 + b[p + 3]; if (n < 2 || p + 2 + n > b.length) break; if (type === 225 && b[p + 4] === 69 && b[p + 5] === 120) return tiff(b.slice(p + 4, p + 2 + n)); p += n + 2; }
  } else if (b[0] === 82) { const exif = webpChunks(b).find(c => c.type === "EXIF"); if (exif) return tiff(exif.data); }
  else if (b[0] === 137) { const v = new DataView(b.buffer, b.byteOffset, b.byteLength); for (let p = 8; p + 12 <= b.length;) { const n = v.getUint32(p); if (n > b.length - p - 12) break; if (String.fromCharCode(...b.slice(p + 4, p + 8)) === "eXIf") return tiff(b.slice(p + 8, p + 8 + n)); p += n + 12; } }
  return 1;
}
