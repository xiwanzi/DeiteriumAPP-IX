// Android Storefront.kt and ProductMedia.kt: logical dp, centered ContentScale.Crop.
export function productImageFrame(mode, phoneWidth = 393) {
  return { poster: [308, 405], grid: [(phoneWidth - 52) / 2, 170], detail: [phoneWidth - 40, 260], bag: [80, 100] }[mode];
}

export function centeredCrop(imageWidth, imageHeight, frameWidth, frameHeight) {
  if (![imageWidth, imageHeight, frameWidth, frameHeight].every((n) => Number.isFinite(n) && n > 0)) return null;
  const scale = Math.max(frameWidth / imageWidth, frameHeight / imageHeight);
  const width = frameWidth / scale, height = frameHeight / scale;
  return { x: (imageWidth - width) / 2, y: (imageHeight - height) / 2, width, height, retained: width * height / (imageWidth * imageHeight) };
}

export const defaultMailBody = "购买的物品已按订单快照投递，请在支持领取的服务器打开邮箱。";
export function mailContentError(title, body) {
  for (const [text, label, maxChars, maxBytes] of [[title, "邮件标题", 80, 240], [body, "邮件正文", 1000, 3000]]) {
    if (Array.from(text).length > maxChars || new TextEncoder().encode(text).length > maxBytes) return `${label}过长，请减少文字或表情（最多 ${maxChars} 字、${maxBytes} 字节）。`;
    if (/[\u0000-\u0009\u000b-\u001f\u007f-\u009f]/u.test(text) || label === "邮件标题" && text.includes("\n")) return `${label}包含不支持的控制字符。`;
  }
  return "";
}
