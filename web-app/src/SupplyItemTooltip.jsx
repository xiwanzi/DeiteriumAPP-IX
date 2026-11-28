import React, { useEffect, useLayoutEffect, useRef, useState } from "react";

const HEX_COLOR = /^#[a-f\d]{6}$/i;
const png = (base64) => `data:image/png;base64,${base64}`;

function runStyle(run) {
  return {
    color: typeof run.color === "string" && HEX_COLOR.test(run.color) ? run.color : undefined,
    fontWeight: run.bold === true ? "bold" : undefined,
    fontStyle: run.italic === true ? "italic" : undefined,
    textDecoration:
      [run.underline === true ? "underline" : "", run.strikethrough === true ? "line-through" : ""]
        .filter(Boolean)
        .join(" ") || undefined,
  };
}

export default function SupplyItemTooltip({ client, name, itemId, anchor, preview, native }) {
  const tooltipRef = useRef(null);
  const [position, setPosition] = useState(null);
  const [shift, setShift] = useState(Boolean(anchor?.shift));
  const [compatibility, setCompatibility] = useState(null);
  const [compatibilityFailed, setCompatibilityFailed] = useState(false);

  const attachment = itemId === "tacz:attachment" ? preview?.variant : undefined;
  const showNative = Boolean(native?.length) && (native.length > 1 || !preview?.tooltip?.sections?.length);

  useLayoutEffect(() => {
    const el = tooltipRef.current;
    if (!el || !anchor) return;
    const update = () => {
      const { width, height } = el.getBoundingClientRect();
      const x = anchor.x + 14 + width <= window.innerWidth - 8 ? anchor.x + 14 : anchor.x - width - 14;
      const y = anchor.y + 10 + height <= window.innerHeight - 8 ? anchor.y + 10 : anchor.y - height - 10;
      setPosition({
        left: Math.max(8, Math.min(x, window.innerWidth - width - 8)),
        top: Math.max(8, Math.min(y, window.innerHeight - height - 8)),
      });
    };
    update();
    const ro = new ResizeObserver(update);
    ro.observe(el);
    window.addEventListener("resize", update);
    return () => {
      ro.disconnect();
      window.removeEventListener("resize", update);
    };
  }, [anchor]);

  useEffect(() => {
    if (!attachment) return;
    const onKey = (e) => setShift(e.shiftKey);
    const onBlur = () => setShift(false);
    window.addEventListener("keydown", onKey, true);
    window.addEventListener("keyup", onKey, true);
    window.addEventListener("blur", onBlur);
    return () => {
      window.removeEventListener("keydown", onKey, true);
      window.removeEventListener("keyup", onKey, true);
      window.removeEventListener("blur", onBlur);
    };
  }, [attachment]);

  useEffect(() => {
    if (!attachment || !shift || compatibility || !client) return;
    let alive = true;
    setCompatibilityFailed(false);
    client
      .request(`/api/v1/admin/supply/item-compatible-guns?attachment=${encodeURIComponent(attachment)}`)
      .then((res) => {
        if (alive) setCompatibility(res.data);
      })
      .catch(() => {
        if (alive) setCompatibilityFailed(true);
      });
    return () => {
      alive = false;
    };
  }, [attachment, shift, compatibility, client]);

  if (preview?.tooltip?.hidden) return null;
  const titleColor = preview?.tooltip?.titleColor;

  return (
    <div
      ref={tooltipRef}
      className="supply-mc-tooltip"
      role="tooltip"
      style={position || { left: 8, top: 8, visibility: "hidden" }}
    >
      {showNative ? (
        native.map((line, idx) => (
          <div className={idx === 0 ? "mc-tt-title" : undefined} key={idx}>
            {line.length
              ? line.map((run, ri) => (
                  <span key={ri} style={runStyle(run)}>
                    {run.text}
                  </span>
                ))
              : "\u00a0"}
          </div>
        ))
      ) : (
        <>
          <div
            className="mc-tt-title"
            style={{ color: titleColor && HEX_COLOR.test(titleColor) ? titleColor : undefined }}
          >
            {preview?.name || name}
          </div>
          {preview?.tooltip?.sections?.map((section, si) => (
            <div
              key={si}
              className="mc-tt-section"
              data-icon={section.some((line) => line.icon) || undefined}
            >
              {section.map((line, li) => (
                <div key={li} className="mc-tt-line" data-tone={line.tone}>
                  {line.icon && <img src={png(line.icon)} alt="" />}
                  {line.label && <span className="mc-tt-label">{line.label}： </span>}
                  {line.value}
                </div>
              ))}
            </div>
          ))}
          {!preview?.tooltip?.sections?.length && preview?.lines?.length > 0 && (
            <div className="mc-tt-section">
              {preview.lines.map((line, li) => (
                <div key={li} className="mc-tt-line" data-tone="muted">
                  {line}
                </div>
              ))}
            </div>
          )}
        </>
      )}
      {attachment &&
        (shift ? (
          <div className="mc-tt-section">
            <div className="mc-tt-compat-head">
              支持枪械{compatibility ? ` · ${compatibility.total}` : ""}
            </div>
            {compatibility ? (
              <>
                <div className="mc-tt-compat-icons">
                  {(compatibility.guns || []).map((gun, gi) =>
                    gun.icon ? (
                      <img key={gi} src={png(gun.icon)} alt={gun.name} />
                    ) : (
                      <span key={gi} className="mc-tt-compat-missing" aria-label={gun.name}>
                        •
                      </span>
                    ),
                  )}
                </div>
                {compatibility.total > (compatibility.guns || []).length && (
                  <div className="mc-tt-line" data-tone="muted">
                    还有 {compatibility.total - compatibility.guns.length} 种
                  </div>
                )}
              </>
            ) : (
              <div className="mc-tt-line" data-tone="muted">
                {compatibilityFailed ? "无法读取枪包兼容列表" : "读取支持枪械…"}
              </div>
            )}
          </div>
        ) : (
          <div className="mc-tt-section mc-tt-keyhint">按下 [Shift] 键查看支持枪械</div>
        ))}
      <div className="mc-tt-id">{preview?.variant || itemId}</div>
    </div>
  );
}
