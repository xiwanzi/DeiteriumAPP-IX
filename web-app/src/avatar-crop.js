export function avatarCrop(width, height, zoom = 1, centerX = width / 2, centerY = height / 2) {
  if (![width, height, zoom, centerX, centerY].every(Number.isFinite) || width <= 0 || height <= 0) throw new Error("图片尺寸无效");
  const size = Math.min(width, height) / Math.max(1, Math.min(4, zoom)), half = size / 2;
  const x = Math.max(half, Math.min(width - half, centerX)), y = Math.max(half, Math.min(height - half, centerY));
  return { x: x - half, y: y - half, size, centerX: x, centerY: y };
}
