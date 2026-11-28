import React, { useMemo, useState } from "react";
import {
  ArrowLeft,
  Plus,
  Trash2,
  Save,
  UploadCloud,
  ChevronDown,
  ChevronRight,
  AlertCircle,
  CheckCircle2,
  ArrowUp,
  ArrowDown,
  Link2,
  Package,
  UserRound,
  Database,
  History,
} from "lucide-react";
import { Badge, Button, Field, Modal } from "./components.jsx";
import { useUnsavedChanges } from "./unsaved-changes.js";
import {
  poolKinds,
  ruleText,
  scheduleInput,
  scheduleISO,
  rewardFromItem,
  poolIssues,
  supplyGrades,
  isRewardBound,
  unboundRewardCount,
} from "./supply-model.js";
import { mutateSupply, pendingSupplyRequest, retrySupply } from "./supply-requests.js";
import { SupplyArtField, SupplyArtPreview } from "./SupplyArtwork.jsx";
import SupplyItemPickerModal from "./SupplyItemPickerModal.jsx";

const EDITOR_STEPS = [
  ["basic", "1. 基础与排期"],
  ["rewards", "2. 奖品与概率"],
  ["art", "3. 视觉与海报"],
  ["preview", "4. 预览与发布"],
];

const THEME_SWATCHES = [
  { id: "violet", name: "银紫", accent: "#dbd8ff", bg: "#101019" },
  { id: "red", name: "绯红", accent: "#f4b9ab", bg: "#1d1319" },
  { id: "blue", name: "冰蓝", accent: "#accafb", bg: "#0b1423" },
];

export default function SupplyPoolEditor({
  client,
  initial,
  assets,
  models,
  tickets = {},
  scope,
  onSaved,
  onAsset,
  close,
}) {
  const [d, setD] = useState(() => {
    const cloned = structuredClone(initial.definition);
    if (!cloned.labels) cloned.labels = { rail: "" };
    if (!Array.isArray(cloned.sections)) cloned.sections = [];
    return cloned;
  });
  const [section, setSection] = useState("basic");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [pickerMode, setPickerMode] = useState(null); // null | "creative" | "inventory" | "library" | "manual"
  const [bindTarget, setBindTarget] = useState(null);
  const [history, setHistory] = useState(null);
  const [scheme, setScheme] = useState("80");
  const [tierFilter, setTierFilter] = useState("ALL");
  const [expandedKeys, setExpandedKeys] = useState({});

  const changes = useUnsavedChanges(d, busy);
  const key = `${scope}:pool:${initial.id || "new"}`;
  const patch = (value) => setD((old) => ({ ...old, ...value }));

  const updateReward = (index, value) =>
    setD((old) => ({
      ...old,
      rewards: old.rewards.map((r, i) => (i === index ? { ...r, ...value } : r)),
    }));

  const bindItemToReward = (index, item, extra = {}) => {
    const updates = {
      itemRef: item.itemRef,
      itemRevision: item.revision,
      payloadSha256: item.payloadSha256,
      inventoryDomain: item.inventoryDomain,
      servers: [...(item.compatibleServerIds || [])],
    };
    if (extra.gunId) updates.gunId = extra.gunId;
    if (extra.displayId) updates.displayId = extra.displayId;
    updateReward(index, updates);
  };

  const handlePickerAdded = (item, extra = {}) => {
    if (bindTarget !== null) {
      bindItemToReward(bindTarget.index, item, extra);
      setBindTarget(null);
      return;
    }
    const created = rewardFromItem(item, extra);
    setD((old) =>
      old.rewards.some((reward) => reward.itemRef === item.itemRef && reward.itemRevision === item.revision)
        ? old
        : { ...old, rewards: [...old.rewards, created] },
    );
  };

  const addSection = () => {
    if ((d.sections || []).length >= 32) return;
    patch({ sections: [...(d.sections || []), { title: "", body: "" }] });
  };
  const fillDefaultSections = () => {
    const ruleBody =
      d.kind === "STANDARD"
        ? "常驻补给每抽 6 星概率 {goldProb}，最多 {maxPity} 抽必得 6 星，获得 6 星后重新计数。\n当前使用「{ticketName}」，单抽消耗 1 张、十连消耗 10 张；兑换单价：{price}。"
        : d.kind === "EVENT"
          ? "活动池每抽 6 星概率 {goldProb}，最多 {maxPity} 抽必得当期 6 星 UP；同活动复刻沿用进度。\n当前使用「{ticketName}」，单抽消耗 1 张、十连消耗 10 张；兑换单价：{price}。"
          : "每抽 6 星概率 {goldProb}，最多 {maxPity} 抽必得当期 6 星 UP（限定 120 方案在 {minPity}～{maxPity} 抽间随机确定保底）；同方案跨期共享保底进度。\n当前使用「{ticketName}」，单抽消耗 1 张、十连消耗 10 张；兑换单价：{price}。";
    const deliveryBody =
      d.deliveryMode === "MAIL"
        ? "单抽或十连的全部奖励将按抽取订单合并为一封邮件寄送至游戏邮箱；可在上方「补给记录」查看跨服历史抽取记录。"
        : "每件奖励优先进入当前背包，空间不足或玩家离线时自动按抽取订单合并寄送至游戏邮箱；可在上方「补给记录」查看跨服历史抽取记录。";
    const defaults = [
      { title: "抽取规则与保底", body: ruleBody },
      {
        title: "奖励交付与记录",
        body: deliveryBody,
      },
    ];
    if ((d.sections || []).length > 0 && !window.confirm("用默认规则与交付说明覆盖当前自定义分段？")) return;
    patch({ sections: defaults });
  };
  const updateSection = (idx, value) =>
    patch({ sections: (d.sections || []).map((item, i) => (i === idx ? { ...item, ...value } : item)) });
  const moveSection = (idx, delta) => {
    const list = [...(d.sections || [])];
    const next = idx + delta;
    if (next < 0 || next >= list.length) return;
    [list[idx], list[next]] = [list[next], list[idx]];
    patch({ sections: list });
  };
  const removeSection = (idx) => patch({ sections: (d.sections || []).filter((_, i) => i !== idx) });

  const save = async (action, retry = false, goPreview = false) => {
    setBusy(true);
    setError("");
    try {
      const body = {
        action,
        expectedVersion: initial.version,
        ...(action === "save" ? { definition: d, sourcePoolId: initial.sourcePoolId || "" } : {}),
      };
      const result = retry
        ? await retrySupply(client, key)
        : await mutateSupply(client, key, `/api/v1/admin/supply/pools${initial.id ? `/${initial.id}` : ""}`, body);
      changes.markSaved(result.definition);
      await onSaved(result);
      if (goPreview) setSection("preview");
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };

  const leave = () => {
    if (!changes.dirty || window.confirm("放弃当前未保存的卡池修改？")) close();
  };

  const showHistory = async () => {
    setError("");
    try {
      const r = await client.request(`/api/v1/admin/supply/pools/${initial.id}/versions`);
      setHistory(r.data.items);
    } catch (e) {
      setError(e.message);
    }
  };

  const restore = async (revision) => {
    setBusy(true);
    try {
      const r = await client.request(`/api/v1/admin/supply/pools/${initial.id}/versions/${revision}`);
      setD(r.data);
      setHistory(null);
      setSection("preview");
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };

  const issues = poolIssues(d, initial.id);
  const unboundCount = unboundRewardCount(d);
  const ticketEntries = useMemo(() => {
    const merged = {
      STANDARD: { name: "常驻券", enabled: true },
      LIMITED: { name: "限定券", enabled: true },
      ...(tickets || {}),
    };
    return Object.entries(merged).filter(([, info]) => info && info.enabled !== false);
  }, [tickets]);

  const probStats = useMemo(() => {
    const goldPct =
      d.kind === "EVENT"
        ? Number(d.eventGoldPpm || 0) / 10000
        : d.kind === "LIMITED" && scheme === "120"
          ? 1.2
          : 0.5;
    const nonSixPct = Math.max(0, 100 - goldPct);
    const fiveW = (d.rewards || []).filter((r) => Number(r.stars) === 5).reduce((s, r) => s + Math.max(0, Number(r.weight) || 0), 0);
    const fourW = (d.rewards || []).filter((r) => Number(r.stars) === 4).reduce((s, r) => s + Math.max(0, Number(r.weight) || 0), 0);
    const otherW = fiveW + fourW;
    return {
      goldPct,
      fivePct: otherW > 0 ? (nonSixPct * fiveW) / otherW : 0,
      fourPct: otherW > 0 ? (nonSixPct * fourW) / otherW : 0,
      otherW,
      nonSixPct,
    };
  }, [d.kind, d.eventGoldPpm, d.rewards, scheme]);

  const groupedTiers = useMemo(() => {
    const tiers = [
      { stars: 6, label: "6 星最高档（UP / 保底）", badgeTone: "amber" },
      { stars: 5, label: "5 星高阶奖品", badgeTone: "blue" },
      { stars: 4, label: "4 星常规奖品", badgeTone: "neutral" },
    ];
    return tiers
      .map((tier) => {
        const entries = (d.rewards || [])
          .map((reward, index) => ({ reward, index }))
          .filter(({ reward }) => Number(reward.stars) === tier.stars);
        const totalWeight = entries.reduce((sum, e) => sum + Math.max(0, Number(e.reward.weight) || 0), 0);
        return { ...tier, entries, totalWeight };
      })
      .filter((t) => tierFilter === "ALL" || Number(tierFilter) === t.stars);
  }, [d.rewards, tierFilter]);

  const toggleExpand = (rewardKey) => setExpandedKeys((old) => ({ ...old, [rewardKey]: !old[rewardKey] }));

  return (
    <div className="supply-editor">
      <div className="supply-editor-head">
        <Button secondary disabled={busy} onClick={leave}>
          <ArrowLeft size={16} />
          返回列表
        </Button>
        <div className="supply-editor-title">
          <h2>{initial.id ? d.title || d.name : initial.sourcePoolId ? "安排活动复刻" : "新建卡池"}</h2>
          <small>
            {initial.id || "保存后自动生成编号"} · {poolKinds[d.kind]}
            {initial.publishedRevision > 0 ? ` · 已发布 v${initial.publishedRevision}` : " · 未发布"}
          </small>
        </div>
        <Badge tone={changes.dirty ? "amber" : "sage"}>
          {changes.dirty ? "有未保存修改" : "草稿已同步"}
        </Badge>
      </div>

      <nav className="adm-subtabs supply-step-nav" aria-label="卡池编辑步骤">
        {EDITOR_STEPS.map(([sid, label]) => (
          <button
            key={sid}
            type="button"
            className={section === sid ? "is-active" : ""}
            onClick={() => setSection(sid)}
          >
            <span>{label}</span>
            {sid === "rewards" && <span className="adm-subtab-count">{(d.rewards || []).length}</span>}
            {sid === "preview" && (
              <span
                className={`adm-subtab-count ${issues.length ? "is-warn" : "is-ok"}`}
              >
                {issues.length ? `${issues.length} 项待补` : "可发布"}
              </span>
            )}
          </button>
        ))}
      </nav>

      {error && (
        <p className="form-error" role="alert">
          {error}
        </p>
      )}

      <fieldset disabled={busy}>
        {section === "basic" && (
          <div className="supply-cards-stack">
            <section className="supply-card">
              <header className="supply-card-head">
                <div>
                  <h3>卡池身份与文案</h3>
                  <small>配置卡池类型、展示标题与侧栏标签。单抽 / 十连按钮文案固定由客户端统一控制。</small>
                </div>
              </header>
              <div className="supply-form-grid">
                <Field label="卡池类型">
                  <select
                    value={d.kind}
                    disabled={!!initial.sourcePoolId || initial.kindEditable === false}
                    title={initial.kindEditable === false ? "该卡池已有玩家抽卡记录，类型已锁定" : undefined}
                    onChange={(e) => {
                      const nextKind = e.target.value;
                      const defaultTicket =
                        !d.ticketId || d.ticketId === "STANDARD" || d.ticketId === "LIMITED"
                          ? nextKind === "STANDARD"
                            ? "STANDARD"
                            : "LIMITED"
                          : d.ticketId;
                      patch({
                        kind: nextKind,
                        seriesId: nextKind === "EVENT" ? initial.id || "" : "",
                        ticketId: defaultTicket,
                        eventGoldPpm: nextKind === "EVENT" ? (d.eventGoldPpm || 5000) : 0,
                        eventPity: nextKind === "EVENT" ? (d.eventPity || 80) : 0,
                        endsAt: nextKind === "STANDARD" ? null : d.endsAt,
                      });
                    }}
                  >
                    {Object.entries(poolKinds).map(([v, label]) => (
                      <option key={v} value={v}>
                        {label}
                      </option>
                    ))}
                  </select>
                </Field>
                <Field label="页面主标题（≤12 字）">
                  <input
                    required
                    maxLength={12}
                    value={d.title}
                    onChange={(e) => patch({ title: e.target.value })}
                    placeholder="如：墨影流光 / 雨夜鸣金"
                  />
                </Field>
                <Field label="运营名称（后台与订单显示）">
                  <input
                    required
                    maxLength={64}
                    value={d.name}
                    onChange={(e) => patch({ name: e.target.value })}
                    placeholder="后台列表与抽卡订单展示名称"
                  />
                </Field>
                <Field label="副标题（可选）">
                  <input
                    maxLength={120}
                    value={d.subtitle || ""}
                    onChange={(e) => patch({ subtitle: e.target.value })}
                    placeholder="显示在游戏标题下方或详情顶部"
                  />
                </Field>
                <Field label="侧栏短标签（留空显示：常驻 / 限定 / 活动）">
                  <input
                    maxLength={32}
                    value={d.labels?.rail || ""}
                    onChange={(e) => patch({ labels: { ...(d.labels || {}), rail: e.target.value } })}
                    placeholder="留空自动按卡池类型显示：常驻 / 限定 / 活动"
                  />
                </Field>
                <Field label="排列优先级（越大越靠前）">
                  <input
                    type="number"
                    min={-100000}
                    max={100000}
                    value={d.sort}
                    onChange={(e) => patch({ sort: Number(e.target.value) })}
                  />
                </Field>
              </div>
            </section>

            <section className="supply-card">
              <header className="supply-card-head">
                <div>
                  <h3>票券与保底规则</h3>
                  <small>指定抽取消耗的票券类型与图标；限定池自动提供 80 抽 / 120 抽双方案。</small>
                </div>
              </header>
              <div className="supply-form-grid">
                <Field label="消耗票券">
                  <select
                    value={d.ticketId || (d.kind === "STANDARD" ? "STANDARD" : "LIMITED")}
                    onChange={(e) => patch({ ticketId: e.target.value })}
                  >
                    {ticketEntries.map(([tid, info]) => (
                      <option key={tid} value={tid}>
                        {info.name || tid} ({tid})
                      </option>
                    ))}
                  </select>
                </Field>
                <Field label="票券图标（可选）">
                  <select value={d.ticketArt || ""} onChange={(e) => patch({ ticketArt: e.target.value })}>
                    <option value="">跟随默认票券图标</option>
                    {Object.values(assets).map((asset) => (
                      <option key={asset.assetId} value={asset.assetId}>
                        {asset.altText || asset.assetId}
                      </option>
                    ))}
                  </select>
                </Field>
                <Field label="发奖方式">
                  <select
                    value={d.deliveryMode === "MAIL" ? "MAIL" : "INVENTORY"}
                    onChange={(e) => patch({ deliveryMode: e.target.value })}
                  >
                    <option value="INVENTORY">优先背包（满或离线时十连合寄一封邮件）</option>
                    <option value="MAIL">直接发往邮箱（单抽/十连合并为一封邮件）</option>
                  </select>
                </Field>
              </div>
              <div className="supply-rule-box">
                {d.kind === "LIMITED" ? (
                  <>
                    <strong>轮换限定 · 双保底方案</strong>
                    <p>{ruleText("LIMITED", "80")}</p>
                    <p>{ruleText("LIMITED", "120")}</p>
                    <small>所有限定 80 共享一组进度，所有限定 120 共享另一组进度，跨期继承且无概率递增。</small>
                  </>
                ) : d.kind === "STANDARD" ? (
                  <>
                    <strong>{ruleText("STANDARD")}</strong>
                    <p>长期开放，获得 6 星后保底归零。</p>
                  </>
                ) : (
                  <>
                    <strong>独立活动池 · 100% UP</strong>
                    <div className="supply-form-grid" style={{ marginTop: 10 }}>
                      <Field label="每抽 6 星概率（%）">
                        <input
                          type="number"
                          min=".0001"
                          max="100"
                          step=".0001"
                          disabled={!!initial.sourcePoolId || !!initial.publishedRevision}
                          value={d.eventGoldPpm / 10000}
                          onChange={(e) => patch({ eventGoldPpm: Math.round(Number(e.target.value) * 10000) })}
                        />
                      </Field>
                      <Field label="固定硬保底抽数">
                        <input
                          type="number"
                          min="1"
                          max="10000"
                          disabled={!!initial.sourcePoolId || !!initial.publishedRevision}
                          value={d.eventPity}
                          onChange={(e) => patch({ eventPity: Number(e.target.value) })}
                        />
                      </Field>
                    </div>
                    <small>
                      {initial.sourcePoolId
                        ? `复刻沿用活动 ${d.seriesId} 的规则与玩家保底。`
                        : "首次发布后规则锁定；后续复刻请从卡池列表点击「复刻」。"}
                    </small>
                  </>
                )}
              </div>
            </section>

            <section className="supply-card">
              <header className="supply-card-head">
                <div>
                  <h3>开放排期 · UTC+8</h3>
                  <small>到期自动停止新抽取，已生成订单会继续完成交付。</small>
                </div>
                {d.startsAt && d.endsAt && (
                  <Badge tone="blue">
                    持续 {Math.max(0, (new Date(d.endsAt) - new Date(d.startsAt)) / 86400000).toFixed(1)} 天
                  </Badge>
                )}
              </header>
              <div className="supply-form-grid">
                <Field label="预告展示时间（可选）">
                  <input
                    type="datetime-local"
                    value={scheduleInput(d.visibleAt)}
                    onChange={(e) => patch({ visibleAt: scheduleISO(e.target.value) })}
                  />
                </Field>
                <Field label="开放时间">
                  <input
                    type="datetime-local"
                    value={scheduleInput(d.startsAt)}
                    onChange={(e) => patch({ startsAt: scheduleISO(e.target.value) })}
                  />
                </Field>
                {d.kind !== "STANDARD" && (
                  <Field label="结束时间">
                    <input
                      type="datetime-local"
                      value={scheduleInput(d.endsAt)}
                      onChange={(e) => patch({ endsAt: scheduleISO(e.target.value) })}
                    />
                  </Field>
                )}
              </div>
            </section>

            <section className="supply-card">
              <header className="supply-card-head">
                <div>
                  <h3>详情页补给说明与规则分段（{(d.sections || []).length} / 32）</h3>
                  <small>
                    游戏内「补给说明」下方所有规则与交付文案（含「抽取规则与保底」「奖励交付与记录」）均在此自由编辑，首段自动红标高亮，换行自动分段；留空则使用系统默认文案。支持动态占位符：
                    <code>{"{goldProb}"}</code> <code>{"{minPity}"}</code> <code>{"{maxPity}"}</code>{" "}
                    <code>{"{ticketName}"}</code> <code>{"{price}"}</code>。
                  </small>
                </div>
                <div className="supply-row-actions">
                  <Button type="button" secondary onClick={fillDefaultSections}>
                    填入默认规则分段
                  </Button>
                  <Button type="button" secondary disabled={(d.sections || []).length >= 32} onClick={addSection}>
                    <Plus size={14} />
                    添加说明分段
                  </Button>
                </div>
              </header>
              <div className="supply-form-grid" style={{ marginBottom: 14 }}>
                <Field label="主推区标题（留空默认：本期主推枪械 / 常驻主推枪械）">
                  <input
                    maxLength={40}
                    value={d.labels?.featuredTitle || ""}
                    onChange={(e) => patch({ labels: { ...(d.labels || {}), featuredTitle: e.target.value } })}
                    placeholder="如：本期主推枪械 / 限定 UP 武器"
                  />
                </Field>
                <Field label="主推区引导文案（支持 {goldProb}，留空用默认）">
                  <input
                    maxLength={240}
                    value={d.labels?.featuredLead || ""}
                    onChange={(e) => patch({ labels: { ...(d.labels || {}), featuredLead: e.target.value } })}
                    placeholder="如：本池 6 星基础获得概率为 {goldProb}，抽中 6 星时 100% 为当期主推。"
                  />
                </Field>
              </div>
              <Field label="卡池简介">
                <textarea
                  rows={3}
                  maxLength={4000}
                  value={d.description || ""}
                  onChange={(e) => patch({ description: e.target.value })}
                  placeholder="简要介绍本期补给主题或亮点。"
                />
              </Field>
              {(d.sections || []).map((item, idx) => (
                <div className="supply-section-card" key={idx}>
                  <div className="supply-section-card-head">
                    <Field label={`分段 ${idx + 1} 标题`}>
                      <input
                        maxLength={80}
                        value={item.title || ""}
                        onChange={(e) => updateSection(idx, { title: e.target.value })}
                        placeholder="如：补给说明 / 奖励交付"
                      />
                    </Field>
                    <div className="supply-row-actions">
                      <Button type="button" secondary disabled={idx === 0} aria-label="上移" onClick={() => moveSection(idx, -1)}>
                        <ArrowUp size={14} />
                      </Button>
                      <Button
                        type="button"
                        secondary
                        disabled={idx === (d.sections || []).length - 1}
                        aria-label="下移"
                        onClick={() => moveSection(idx, 1)}
                      >
                        <ArrowDown size={14} />
                      </Button>
                      <Button type="button" secondary aria-label="删除" onClick={() => removeSection(idx)}>
                        <Trash2 size={14} />
                      </Button>
                    </div>
                  </div>
                  <Field label="正文">
                    <textarea
                      rows={2}
                      maxLength={2000}
                      value={item.body || ""}
                      onChange={(e) => updateSection(idx, { body: e.target.value })}
                      placeholder="填写展示给玩家的详细说明正文"
                    />
                  </Field>
                </div>
              ))}
            </section>
          </div>
        )}

        {section === "rewards" && (
          <div className="supply-cards-stack">
            <div className="supply-prob-overview">
              <div>
                <span>6 星基础概率</span>
                <strong>{probStats.goldPct.toFixed(2).replace(/\.?0+$/, "")}%</strong>
              </div>
              <div>
                <span>5 星综合概率</span>
                <strong>{probStats.fivePct.toFixed(2)}%</strong>
              </div>
              <div>
                <span>4 星综合概率</span>
                <strong>{probStats.fourPct.toFixed(2)}%</strong>
              </div>
              <div>
                <span>已配置奖品</span>
                <strong>{(d.rewards || []).length} / 100</strong>
              </div>
            </div>

            <section className="supply-card">
              <div className="supply-rewards-bar">
                <div className="adm-pill-group" role="group" aria-label="按星级筛选奖品">
                  {[
                    ["ALL", `全部 (${(d.rewards || []).length})`],
                    ["6", `6星 (${(d.rewards || []).filter((r) => Number(r.stars) === 6).length})`],
                    ["5", `5星 (${(d.rewards || []).filter((r) => Number(r.stars) === 5).length})`],
                    ["4", `4星 (${(d.rewards || []).filter((r) => Number(r.stars) === 4).length})`],
                  ].map(([val, label]) => (
                    <button
                      key={val}
                      type="button"
                      className={tierFilter === val ? "is-active" : ""}
                      onClick={() => setTierFilter(val)}
                    >
                      {label}
                    </button>
                  ))}
                </div>
                <div className="supply-row-actions">
                  <Button
                    type="button"
                    disabled={(d.rewards || []).length >= 100}
                    onClick={() => {
                      setBindTarget(null);
                      setPickerMode("creative");
                    }}
                  >
                    <Package size={15} />
                    打开物品选择器
                  </Button>
                  <Button
                    type="button"
                    secondary
                    disabled={(d.rewards || []).length >= 100}
                    onClick={() => {
                      setBindTarget(null);
                      setPickerMode("inventory");
                    }}
                  >
                    <UserRound size={15} />
                    在线背包
                  </Button>
                  <Button
                    type="button"
                    secondary
                    disabled={(d.rewards || []).length >= 100}
                    onClick={() => {
                      setBindTarget(null);
                      setPickerMode("library");
                    }}
                  >
                    <Database size={15} />
                    已有物品库
                  </Button>
                </div>
              </div>

              {unboundCount > 0 && (
                <div className="supply-warn-banner">
                  <AlertCircle size={16} />
                  <div>
                    <strong>当前有 {unboundCount} 件示例奖品尚未绑定真实游戏物品</strong>
                    <p>点击右侧「绑定物品」按钮即可从创造物品栏、在线背包或物品库一键入库并绑定，无需重填名称和权重。</p>
                  </div>
                </div>
              )}

              {!(d.rewards || []).length ? (
                <div className="supply-empty-box">
                  <p>当前奖池还没有奖品。</p>
                  <Button
                    type="button"
                    onClick={() => {
                      setBindTarget(null);
                      setPickerMode("creative");
                    }}
                  >
                    <Plus size={15} />
                    从物品选择器添加奖品
                  </Button>
                </div>
              ) : (
                groupedTiers.map((tier) => (
                  <div className="supply-tier-group" key={tier.stars}>
                    <header className="supply-tier-head">
                      <div>
                        <Badge tone={tier.badgeTone}>{tier.stars} 星</Badge>
                        <strong>{tier.label}</strong>
                      </div>
                      <small>
                        共 {tier.entries.length} 件 · 组内总权重 {tier.totalWeight}
                      </small>
                    </header>

                    {!tier.entries.length ? (
                      <p className="supply-tier-empty">该星级暂未配置奖品</p>
                    ) : (
                      tier.entries.map(({ reward, index }) => {
                        const pct =
                          tier.totalWeight > 0 ? (Math.max(0, Number(reward.weight) || 0) / tier.totalWeight) * 100 : 0;
                        const isOpen = Boolean(expandedKeys[reward.key]);
                        const bound = isRewardBound(reward);
                        return (
                          <div className={`supply-reward-row ${isOpen ? "is-open" : ""}`} key={reward.key}>
                            <div className="supply-reward-line">
                              <button
                                type="button"
                                className="supply-reward-toggle"
                                aria-expanded={isOpen}
                                onClick={() => toggleExpand(reward.key)}
                              >
                                {isOpen ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
                                <div className="supply-reward-info">
                                  <strong>
                                    {reward.name}
                                    {!bound && <Badge tone="amber">待绑定</Badge>}
                                  </strong>
                                  <small>
                                    {bound
                                      ? `${reward.itemRef} @ v${reward.itemRevision}`
                                      : "未绑定游戏物品（仅预览不可抽取）"}
                                    {reward.gunId ? ` · ${reward.gunId}` : ""}
                                  </small>
                                </div>
                              </button>

                              <div className="supply-reward-inline">
                                <label className="supply-inline-ctrl" title="快捷修改星级">
                                  <span>品质</span>
                                  <select
                                    value={reward.stars}
                                    onChange={(e) => {
                                      const s = Number(e.target.value);
                                      updateReward(index, { stars: s, up: s === 6 });
                                    }}
                                  >
                                    <option value={6}>6★</option>
                                    <option value={5}>5★</option>
                                    <option value={4}>4★</option>
                                  </select>
                                </label>
                                <label className="supply-inline-ctrl" title="快捷修改组内权重">
                                  <span>权重</span>
                                  <input
                                    type="number"
                                    min={1}
                                    max={1000000}
                                    value={reward.weight}
                                    onChange={(e) => updateReward(index, { weight: Math.max(1, Number(e.target.value) || 1) })}
                                  />
                                </label>
                                <div className="supply-inline-prob">
                                  <span>{pct.toFixed(1)}%</span>
                                  <div className="supply-prob-bar">
                                    <i style={{ width: `${Math.min(100, pct)}%` }} />
                                  </div>
                                </div>
                                <Button
                                  type="button"
                                  secondary
                                  className="supply-bind-btn"
                                  onClick={() => {
                                    setBindTarget({ index, name: reward.name });
                                    setPickerMode("creative");
                                  }}
                                >
                                  <Link2 size={14} />
                                  {bound ? "换绑" : "绑定物品"}
                                </Button>
                                <Button
                                  type="button"
                                  secondary
                                  aria-label={`移除 ${reward.name}`}
                                  onClick={() => patch({ rewards: d.rewards.filter((_, i) => i !== index) })}
                                >
                                  <Trash2 size={14} />
                                </Button>
                              </div>
                            </div>

                            {isOpen && (
                              <div className="supply-reward-detail">
                                <div className="supply-form-grid">
                                  <Field label="展示名称">
                                    <input
                                      maxLength={128}
                                      value={reward.name}
                                      onChange={(e) => updateReward(index, { name: e.target.value })}
                                    />
                                  </Field>
                                  <Field label="TaCZ 枪械 ID（3D 展示用，非枪械可留空）">
                                    <input
                                      value={reward.gunId || ""}
                                      onChange={(e) => updateReward(index, { gunId: e.target.value })}
                                      placeholder="例如 tacz:ai_awp"
                                    />
                                  </Field>
                                  <Field label="TaCZ 皮肤展示 ID（可选）">
                                    <input
                                      value={reward.displayId || ""}
                                      onChange={(e) => updateReward(index, { displayId: e.target.value })}
                                      placeholder="例如 deuterium_ix:qbz_191_printstream"
                                    />
                                  </Field>
                                  <Field label="内置展示模型（可选）">
                                    <select
                                      value={reward.modelKey}
                                      onChange={(e) => {
                                        const modelKey = e.target.value;
                                        updateReward(index, {
                                          modelKey,
                                          art: {
                                            ...reward.art,
                                            detail: modelKey ? `builtin-detail-${modelKey}` : reward.art?.detail || "",
                                            fallback: modelKey ? `builtin-fallback-${modelKey}` : reward.art?.fallback || "",
                                          },
                                        });
                                      }}
                                    >
                                      <option value="">无内置模型（使用 3D 枪械或备用图）</option>
                                      {models.map((model) => (
                                        <option key={model} value={model}>
                                          {model}
                                        </option>
                                      ))}
                                    </select>
                                  </Field>
                                </div>
                                <div className="supply-form-grid" style={{ marginTop: 12 }}>
                                  <SupplyArtField
                                    client={client}
                                    role="detail"
                                    value={reward.art.detail}
                                    assets={assets}
                                    onAsset={onAsset}
                                    onChange={(detail) => updateReward(index, { art: { ...reward.art, detail } })}
                                  />
                                  <SupplyArtField
                                    client={client}
                                    role="fallback"
                                    value={reward.art.fallback}
                                    assets={assets}
                                    onAsset={onAsset}
                                    onChange={(fallback) => updateReward(index, { art: { ...reward.art, fallback } })}
                                  />
                                </div>
                                <small>
                                  交付兼容服务器：{(reward.servers || []).join("、") || "绑定物品后自动同步"}
                                </small>
                              </div>
                            )}
                          </div>
                        );
                      })
                    )}
                  </div>
                ))
              )}
            </section>
          </div>
        )}

        {section === "art" && (
          <div className="supply-split-layout">
            <div className="supply-cards-stack">
              <section className="supply-card">
                <header className="supply-card-head">
                  <div>
                    <h3>主题色板与海报焦点</h3>
                    <small>右侧预览会实时反映主题色与焦点偏移；修改文案不会强制玩家重下未改动图片。</small>
                  </div>
                </header>
                <div className="supply-swatch-row" role="radiogroup" aria-label="选择主题配色">
                  {THEME_SWATCHES.map((sw) => (
                    <button
                      key={sw.id}
                      type="button"
                      role="radio"
                      aria-checked={(d.art.theme || "violet") === sw.id}
                      className={`supply-swatch ${(d.art.theme || "violet") === sw.id ? "is-active" : ""}`}
                      onClick={() => patch({ art: { ...d.art, theme: sw.id } })}
                    >
                      <span className="supply-swatch-dot" style={{ background: sw.bg, borderColor: sw.accent }}>
                        <i style={{ background: sw.accent }} />
                      </span>
                      <span>{sw.name}</span>
                    </button>
                  ))}
                </div>
                <div className="supply-form-grid" style={{ marginTop: 14 }}>
                  <Field label={`海报水平焦点：${Math.round(Number(d.art.heroX ?? 0.5) * 100)}%`}>
                    <input
                      type="range"
                      min="0"
                      max="1"
                      step="0.01"
                      value={d.art.heroX ?? 0.5}
                      onChange={(e) => patch({ art: { ...d.art, heroX: Number(e.target.value) } })}
                    />
                  </Field>
                  <Field label={`海报垂直焦点：${Math.round(Number(d.art.heroY ?? 0.5) * 100)}%`}>
                    <input
                      type="range"
                      min="0"
                      max="1"
                      step="0.01"
                      value={d.art.heroY ?? 0.5}
                      onChange={(e) => patch({ art: { ...d.art, heroY: Number(e.target.value) } })}
                    />
                  </Field>
                </div>
              </section>

              <section className="supply-card">
                <header className="supply-card-head">
                  <div>
                    <h3>宣传素材上传与选择</h3>
                    <small>支持从内置素材挑选，或直接上传本地图片并在线裁切。</small>
                  </div>
                </header>
                <div className="supply-form-grid">
                  <SupplyArtField
                    client={client}
                    role="hero"
                    value={d.art.hero}
                    assets={assets}
                    onAsset={onAsset}
                    onChange={(hero) => patch({ art: { ...d.art, hero } })}
                  />
                  <SupplyArtField
                    client={client}
                    role="rail"
                    value={d.art.rail}
                    assets={assets}
                    onAsset={onAsset}
                    onChange={(rail) => patch({ art: { ...d.art, rail } })}
                  />
                </div>
              </section>

              <section className="supply-card">
                <header className="supply-card-head">
                  <div>
                    <h3>主推奖品详情横幅（923 × 117）</h3>
                    <small>对应游戏内「补给详情」顶部主推条的背景横幅；支持挑选内置素材或上传并裁切。</small>
                  </div>
                </header>
                {(d.rewards || []).filter((r) => Number(r.stars) === 6).length === 0 ? (
                  <p className="muted">请先在「2. 奖品与概率」中配置至少 1 件 6 星主推奖品。</p>
                ) : (
                  <div className="supply-cards-stack">
                    {(d.rewards || []).map((reward, idx) =>
                      Number(reward.stars) === 6 ? (
                        <div key={reward.key} className="supply-section-card">
                          <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 8 }}>
                            <strong>{reward.name || reward.key}</strong>
                            <Badge tone="amber">6 星主推</Badge>
                          </div>
                          <SupplyArtField
                            client={client}
                            role="detail"
                            value={reward.art?.detail || ""}
                            assets={assets}
                            onAsset={onAsset}
                            onChange={(detail) => updateReward(idx, { art: { ...(reward.art || {}), detail } })}
                          />
                        </div>
                      ) : null,
                    )}
                  </div>
                )}
              </section>
            </div>

            <aside className="supply-sticky-preview">
              <section className="supply-card">
                <header className="supply-card-head">
                  <div>
                    <h3>实时构图预览</h3>
                    <small>拖动焦点滑块或切换主题色即可即时查看效果。</small>
                  </div>
                </header>
                <SupplyArtPreview definition={d} assets={assets} scheme={scheme} />
              </section>
            </aside>
          </div>
        )}

        {section === "preview" && (
          <div className="supply-split-layout">
            <section className="supply-card">
              <header className="supply-card-head">
                <div>
                  <h3>游戏内构图预览</h3>
                  <small>最终字体与三维枪械演出由客户端呈现；此处核对构图与规则。</small>
                </div>
                {d.kind === "LIMITED" && (
                  <div className="adm-pill-group">
                    <button type="button" className={scheme === "80" ? "is-active" : ""} onClick={() => setScheme("80")}>
                      80 方案
                    </button>
                    <button type="button" className={scheme === "120" ? "is-active" : ""} onClick={() => setScheme("120")}>
                      120 方案
                    </button>
                  </div>
                )}
              </header>
              <SupplyArtPreview definition={d} assets={assets} scheme={scheme} />
            </section>

            <section className="supply-card">
              <header className="supply-card-head">
                <div>
                  <h3>发布检查清单</h3>
                  <small>保存草稿不会影响线上玩家；点击「发布当前草稿」后按排期生效。</small>
                </div>
              </header>
              {issues.length ? (
                <div className="supply-warn-banner">
                  <AlertCircle size={16} />
                  <div>
                    <strong>发布前需先补全以下 {issues.length} 项：</strong>
                    <ul>
                      {issues.map((issue) => (
                        <li key={issue}>{issue}</li>
                      ))}
                    </ul>
                  </div>
                </div>
              ) : (
                <div className="supply-ok-banner">
                  <CheckCircle2 size={16} />
                  <span>基础配置检查通过，可直接发布给游戏客户端。</span>
                </div>
              )}

              {unboundCount > 0 && !issues.length && (
                <div className="supply-notice" style={{ marginTop: 12 }}>
                  当前仍有 {unboundCount} 件示例奖品未绑定游戏物品：发布后玩家可实时查看卡池展示，但抽取会被安全拦截且不扣券，直至完成绑定。
                </div>
              )}

              <div className="supply-publish-actions">
                <Button
                  type="button"
                  disabled={!initial.id || changes.dirty || issues.length > 0}
                  onClick={() => save("publish")}
                >
                  <UploadCloud size={16} />
                  发布当前草稿
                </Button>
                {initial.publishedRevision > 0 && (
                  <Button type="button" secondary onClick={showHistory}>
                    <History size={15} />
                    历史发布版本
                  </Button>
                )}
              </div>
              {changes.dirty && <small style={{ marginTop: 8 }}>请先点击底部「保存草稿」再发布。</small>}
            </section>
          </div>
        )}
      </fieldset>

      <footer className="supply-save-bar">
        <span className="supply-save-status">
          {busy
            ? "正在保存…"
            : changes.dirty
              ? "有未保存的修改"
              : issues.length
                ? `还有 ${issues.length} 项待补全`
                : "草稿已同步，可前往预览发布"}
        </span>
        <div className="supply-row-actions">
          {pendingSupplyRequest(key) && (
            <Button type="button" secondary disabled={busy} onClick={() => save("save", true)}>
              恢复原请求
            </Button>
          )}
          <Button type="button" secondary disabled={busy} onClick={leave}>
            返回列表
          </Button>
          <Button type="button" secondary disabled={busy} onClick={() => save("save")}>
            <Save size={15} />
            保存草稿
          </Button>
          {section !== "preview" && (
            <Button type="button" disabled={busy} onClick={() => save("save", false, true)}>
              <UploadCloud size={15} />
              保存并去发布
            </Button>
          )}
        </div>
      </footer>

      {pickerMode && (
        <SupplyItemPickerModal
          client={client}
          scope={scope}
          initialMode={pickerMode}
          targetName={bindTarget?.name || ""}
          remaining={100 - (d.rewards || []).length}
          onAdded={handlePickerAdded}
          close={() => {
            setPickerMode(null);
            setBindTarget(null);
          }}
        />
      )}

      {history && (
        <Modal title="历史发布版本" close={() => setHistory(null)}>
          <p className="muted">载入历史版本会覆盖当前草稿，需再次保存并发布后生效；历史抽卡记录不受影响。</p>
          <div className="supply-history-list">
            {history.map((item) => (
              <div className="supply-history-row" key={item.revision}>
                <div>
                  <strong>版本 v{item.revision}</strong>
                  <small>{new Date(item.createdAt).toLocaleString("zh-CN", { timeZone: "Asia/Singapore" })}</small>
                </div>
                <Button type="button" secondary disabled={busy} onClick={() => restore(item.revision)}>
                  载入草稿
                </Button>
              </div>
            ))}
          </div>
        </Modal>
      )}
    </div>
  );
}
