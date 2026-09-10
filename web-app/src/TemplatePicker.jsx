import React, { useEffect, useState } from "react";
import SearchPicker from "./SearchPicker.jsx";

export default function TemplatePicker({ client, storeId, reference, templates, onChange }) {
  const [cached, setCached] = useState(null);
  useEffect(() => {
    if (!reference || templates.some((t) => t.templateRef === reference)) return;
    let live = true;
    client.request(`/api/v1/merchant/stores/${encodeURIComponent(storeId)}/delivery-templates/${encodeURIComponent(reference)}`).then((r) => { if (live) setCached(r.data); }).catch(() => {});
    return () => { live = false; };
  }, [reference, storeId]);
  const selected = templates.find((t) => t.templateRef === reference) || (cached?.templateRef === reference ? cached : null);
  return <SearchPicker label="选择交付模板" values={reference ? [selected || { templateRef: reference, name: reference }] : []}
    getKey={(v) => v.templateRef} getTitle={(v) => v.name} getDescription={(v) => `${v.attachmentCount ?? "多"} 种物品 · ${v.summary || ""}${v.active === false ? " · 已停用" : ""}`} unavailable={(v) => !v.active}
    load={async (query, cursor) => { const r = await client.request(`/api/v1/merchant/stores/${encodeURIComponent(storeId)}/delivery-templates?limit=30&q=${encodeURIComponent(query)}${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`); return { items: r.data, nextCursor: r.page?.nextCursor }; }}
    onChange={([template]) => { setCached(template); onChange(template); }} />;
}
