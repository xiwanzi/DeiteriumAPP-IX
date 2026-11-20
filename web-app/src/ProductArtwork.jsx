import React, { useState } from "react";
import { Package } from "lucide-react";

export default function ProductArtwork({ image, className = "", imageClassName, lazy = false }) {
  const [failedUrl, setFailedUrl] = useState(null);
  return <span className={`product-artwork ${className}`} aria-hidden="true">
    {image?.url && image.url !== failedUrl ?
      <img className={imageClassName} src={image.url} alt="" loading={lazy ? "lazy" : undefined} decoding="async" onError={() => setFailedUrl(image.url)} /> :
      <Package size={22} />}
  </span>;
}
