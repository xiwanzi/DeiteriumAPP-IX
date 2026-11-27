import React, { useMemo, useState } from "react";
import { ArrowLeft, Plus, Trash2, Save, UploadCloud, ChevronDown, ChevronRight, AlertCircle, CheckCircle2 } from "lucide-react";
import { Badge, Button, Field, Modal } from "./components.jsx";
import { useUnsavedChanges } from "./unsaved-changes.js";
import { poolKinds, ruleText, scheduleInput, scheduleISO, rewardFromItem, poolIssues, supplyGrades } from "./supply-model.js";
import { mutateSupply, pendingSupplyRequest, retrySupply } from "./supply-requests.js";
import { SupplyArtField, SupplyArtPreview } from "./SupplyArtwork.jsx";
import SupplyInventoryPicker from "./SupplyInventoryPicker.jsx";

const sections = [
  ["basic", "基本信息与排期"],
  ["rewards", "奖品与概率配置"],
  ["art", "美术素材"],
  ["preview", "预览与发布"],
];

export default function SupplyPoolEditor({ client, initial, assets, models, scope, onSaved, onAsset, close }) {
  const [d, setD] = useState(() => structuredClone(initial.definition));
  const [section, setSection] = useState("basic");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [picker, setPicker] = useState(false);
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
  const addReward = (item, stars = 6) => {
    const created = { ...rewardFromItem(item), stars, up: stars === 6 };
    setD((old) =>
      old.rewards.some((reward) => reward.itemRef === item.itemRef && reward.itemRevision === item.revision)
        ? old
        : { ...old, rewards: [...old.rewards, created] },
    );
    setExpandedKeys((old) => ({ ...old, [created.key]: true }));
  };

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

  const issues = poolIssues(d);

  // Group rewards by star tier (6 -> 5 -> 4) with live in-tier probability % calculation
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

  const toggleExpand = (rewardKey) =>
    setExpandedKeys((old) => ({ ...old, [rewardKey]: !old[rewardKey] }));

  return (
    <div className="supply-editor">
      <div className="supply-editor-head">
        <Button secondary disabled={busy} onClick={leave}>
          <ArrowLeft size={16} />返回卡池
        </Button>
        <div>
          <h2>{initial.id ? d.name : initial.sourcePoolId ? "安排活动复刻" : "新建卡池"}</h2>
          <small>{initial.id || "保存后生成卡池编号"}</small>
        </div>
        <Badge tone={changes.dirty ? "amber" : "neutral"}>
          {changes.dirty ? "有未保存修改" : "草稿已同步"}
        </Badge>
      </div>

      <nav className="adm-subtabs" aria-label="卡池编辑步骤">
        {sections.map(([sid, label]) => (
          <button
            key={sid}
            type="button"
            className={section === sid ? "is-active" : ""}
            onClick={() => setSection(sid)}
          >
            <span>{label}</span>
            {sid === "rewards" && (
              <span className="adm-subtab-count">{(d.rewards || []).length}</span>
            )}
            {sid === "preview" && (
              <span
                className="adm-subtab-count"
                style={
                  issues.length
                    ? { background: "var(--adm-amber-bg)", color: "var(--adm-amber-fg)", borderColor: "transparent" }
                    : { background: "var(--adm-sage-bg)", color: "var(--adm-sage-fg)", borderColor: "transparent" }
                }
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
          <div className="supply-panel">
            <div className="supply-form-grid">
              <Field label="卡池类型">
                <select
                  value={d.kind}
                  disabled={!!initial.id || !!initial.sourcePoolId}
                  onChange={(e) =>
                    patch({
                      kind: e.target.value,
                      seriesId: "",
                      eventGoldPpm: e.target.value === "EVENT" ? 5000 : 0,
                      eventPity: e.target.value === "EVENT" ? 80 : 0,
                      endsAt: e.target.value === "STANDARD" ? null : d.endsAt,
                    })
                  }
                >
                  {Object.entries(poolKinds).map(([v, label]) => (
                    <option key={v} value={v}>
                      {label}
                    </option>
                  ))}
                </select>
              </Field>
              <Field label="运营名称">
                <input
                  required
                  maxLength={64}
                  value={d.name}
                  onChange={(e) => patch({ name: e.target.value })}
                  placeholder="用于卡池列表、订单与侧栏"
                />
              </Field>
              <Field label="页面标题（最多 12 字）">
                <input
                  required
                  maxLength={12}
                  value={d.title}
                  onChange={(e) => patch({ title: e.target.value })}
                  placeholder="游戏中显示的标题"
                />
              </Field>
              <Field label="副标题">
                <input
                  maxLength={120}
                  value={d.subtitle}
                  onChange={(e) => patch({ subtitle: e.target.value })}
                />
              </Field>
              <Field label="排列优先级">
                <input
                  type="number"
                  min={-100000}
                  max={100000}
                  value={d.sort}
                  onChange={(e) => patch({ sort: Number(e.target.value) })}
                />
              </Field>
            </div>
            <Field label="补给介绍">
              <textarea
                rows={4}
                maxLength={4000}
                value={d.description}
                onChange={(e) => patch({ description: e.target.value })}
                placeholder="玩法介绍或活动说明。概率、费用与保底由实际规则自动展示。"
              />
            </Field>
            <div className="supply-rule-box">
              {d.kind === "LIMITED" ? (
                <>
                  <strong>同一组 UP，两个抽取页面</strong>
                  <p>{ruleText("LIMITED", "80")}</p>
                  <p>{ruleText("LIMITED", "120")}</p>
                  <small>
                    所有限定 80 共享进度；所有限定 120 共享另一份进度。两组各自跨期继承，均无概率递增。
                  </small>
                </>
              ) : d.kind === "STANDARD" ? (
                <>
                  <strong>{ruleText("STANDARD")}</strong>
                  <p>长期开放，使用常驻券，获得6星后保底归零。</p>
                </>
              ) : (
                <>
                  <strong>活动独立保底 · 使用限定券 · 100% UP</strong>
                  <div className="supply-form-grid">
                    <Field label="每抽6 星概率（%）">
                      <input
                        type="number"
                        min=".0001"
                        max="100"
                        step=".0001"
                        disabled={!!initial.sourcePoolId || !!initial.publishedRevision}
                        value={d.eventGoldPpm / 10000}
                        onChange={(e) =>
                          patch({ eventGoldPpm: Math.round(Number(e.target.value) * 10000) })
                        }
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
                      : "首次发布后规则固定；后续请从原活动使用“复刻”入口。"}
                  </small>
                </>
              )}
            </div>
            <h3>排期 · UTC+8</h3>
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
            {d.startsAt && d.endsAt && (
              <p>
                持续约{" "}
                {Math.max(0, (new Date(d.endsAt) - new Date(d.startsAt)) / 86400000).toFixed(2)}{" "}
                天。到期停止新抽取，已创建订单继续交付。
              </p>
            )}
          </div>
        )}

        {section === "rewards" && (
          <div className="supply-panel">
            <div className="adm-toolbar" style={{ marginBottom: 12 }}>
              <div className="adm-toolbar-left">
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
              </div>
              <div className="adm-toolbar-right">
                <Button disabled={(d.rewards || []).length >= 100} onClick={() => setPicker("inventory")}>
                  <Plus size={15} />从玩家背包选取
                </Button>
                <Button
                  secondary
                  disabled={(d.rewards || []).length >= 100}
                  onClick={() => setPicker("library")}
                >
                  从物品库添加
                </Button>
              </div>
            </div>

            {!(d.rewards || []).length ? (
              <p className="muted">尚未配置奖品。点击右上角从背包或物品库添加。</p>
            ) : (
              groupedTiers.map((tier) => (
                <section className="adm-tier-section" key={tier.stars}>
                  <header className="adm-tier-head">
                    <strong>
                      <Badge tone={tier.badgeTone}>{tier.stars} 星</Badge>
                      {tier.label}
                    </strong>
                    <small>
                      共 {tier.entries.length} 件 · 组内总权重 {tier.totalWeight}
                    </small>
                  </header>
                  {!tier.entries.length ? (
                    <p className="muted" style={{ padding: "12px 16px", margin: 0, fontSize: 12 }}>
                      该星级暂未配置奖品
                    </p>
                  ) : (
                    tier.entries.map(({ reward, index }) => {
                      const pct =
                        tier.totalWeight > 0
                          ? ((Math.max(0, Number(reward.weight) || 0) / tier.totalWeight) * 100)
                          : 0;
                      const isOpen = Boolean(expandedKeys[reward.key]);
                      return (
                        <div className="adm-reward-item" key={reward.key}>
                          <div
                            className="adm-reward-summary"
                            role="button"
                            tabIndex={0}
                            aria-expanded={isOpen}
                            onClick={() => toggleExpand(reward.key)}
                            onKeyDown={(e) => {
                              if (e.key === "Enter" || e.key === " ") {
                                e.preventDefault();
                                toggleExpand(reward.key);
                              }
                            }}
                          >
                            {isOpen ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
                            <div className="adm-reward-main">
                              <strong>{reward.name}</strong>
                              <small>
                                {reward.itemRef} @ v{reward.itemRevision} · 权重 {reward.weight}
                                {reward.modelKey ? ` · 模型 ${reward.modelKey}` : ""}
                              </small>
                            </div>
                            <div className="adm-prob-meter">
                              <span>组内占比 {pct.toFixed(1)}%</span>
                              <div className="adm-prob-track">
                                <i style={{ width: `${Math.min(100, pct)}%` }} />
                              </div>
                            </div>
                            <Button
                              secondary
                              aria-label={`移除奖品 ${reward.name}`}
                              onClick={(e) => {
                                e.stopPropagation();
                                patch({ rewards: d.rewards.filter((_, i) => i !== index) });
                              }}
                            >
                              <Trash2 size={15} />
                            </Button>
                          </div>
                          {isOpen && (
                            <div className="adm-reward-body">
                              <div className="supply-form-grid">
                                <Field label="显示名称">
                                  <input
                                    maxLength={128}
                                    value={reward.name}
                                    onChange={(e) => updateReward(index, { name: e.target.value })}
                                  />
                                </Field>
                                <Field label="品质">
                                  <select
                                    value={reward.stars}
                                    onChange={(e) =>
                                      updateReward(index, {
                                        stars: Number(e.target.value),
                                        up: Number(e.target.value) === 6,
                                      })
                                    }
                                  >
                                    {supplyGrades.map((grade) => (
                                      <option key={grade.stars} value={grade.stars}>
                                        {grade.label}
                                      </option>
                                    ))}
                                  </select>
                                </Field>
                                <Field label={`组内权重（当前占比 ${pct.toFixed(1)}%）`}>
                                  <input
                                    type="number"
                                    min={1}
                                    max={1000000}
                                    value={reward.weight}
                                    onChange={(e) =>
                                      updateReward(index, { weight: Number(e.target.value) })
                                    }
                                  />
                                </Field>
                                <Field label="枪械 ID">
                                  <input
                                    value={reward.gunId}
                                    onChange={(e) => updateReward(index, { gunId: e.target.value })}
                                    placeholder="命名空间:枪械"
                                  />
                                </Field>
                                <Field label="展示模型">
                                  <select
                                    value={reward.modelKey}
                                    onChange={(e) => {
                                      const modelKey = e.target.value;
                                      updateReward(index, {
                                        modelKey,
                                        art: {
                                          ...reward.art,
                                          detail: modelKey ? `builtin-detail-${modelKey}` : "",
                                          fallback: modelKey ? `builtin-fallback-${modelKey}` : "",
                                        },
                                      });
                                    }}
                                  >
                                    <option value="">使用备用图片</option>
                                    {models.map((model) => (
                                      <option key={model} value={model}>
                                        {model}
                                      </option>
                                    ))}
                                  </select>
                                </Field>
                              </div>
                              <details style={{ marginTop: 10 }}>
                                <summary>武器展示图片与交付范围</summary>
                                <div className="supply-form-grid">
                                  <SupplyArtField
                                    client={client}
                                    role="detail"
                                    value={reward.art.detail}
                                    assets={assets}
                                    onAsset={onAsset}
                                    onChange={(detail) =>
                                      updateReward(index, { art: { ...reward.art, detail } })
                                    }
                                  />
                                  <SupplyArtField
                                    client={client}
                                    role="fallback"
                                    value={reward.art.fallback}
                                    assets={assets}
                                    onAsset={onAsset}
                                    onChange={(fallback) =>
                                      updateReward(index, { art: { ...reward.art, fallback } })
                                    }
                                  />
                                </div>
                                <p>
                                  兼容服务器：{(reward.servers || []).join("、")}
                                  。物品版本和内容摘要在发布时校验并冻结。
                                </p>
                              </details>
                            </div>
                          )}
                        </div>
                      );
                    })
                  )}
                </section>
              ))
            )}
          </div>
        )}

        {section === "art" && (
          <div className="supply-panel">
            <p>
              同一 UP 的 80 / 120 页面共用本组宣传素材，方案名称和概率提示自动区分。武器特写模型随所选奖品关联。
            </p>
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
            <Field label="主题配色">
              <select
                value={d.art.theme}
                onChange={(e) => patch({ art: { ...d.art, theme: e.target.value } })}
              >
                <option value="violet">银紫</option>
                <option value="red">绯红</option>
                <option value="blue">冰蓝</option>
              </select>
            </Field>
            <SupplyArtPreview definition={d} assets={assets} scheme={scheme} />
          </div>
        )}

        {section === "preview" && (
          <div className="supply-panel">
            <div className="supply-section-head">
              <h3>构图与规则预览</h3>
              {d.kind === "LIMITED" && (
                <div className="button-row">
                  <Button secondary={scheme !== "80"} onClick={() => setScheme("80")}>
                    80 方案
                  </Button>
                  <Button secondary={scheme !== "120"} onClick={() => setScheme("120")}>
                    120 方案
                  </Button>
                </div>
              )}
            </div>
            <SupplyArtPreview definition={d} assets={assets} scheme={scheme} />
            <p className="muted">此处检查图片构图与文案关系；游戏字体及三维演出由客户端呈现。</p>
            {issues.length ? (
              <div className="supply-checklist">
                <strong>
                  <AlertCircle size={15} style={{ verticalAlign: "-2px", marginRight: 5 }} />
                  发布前需要完成
                </strong>
                <ul>
                  {issues.map((issue) => (
                    <li key={issue}>{issue}</li>
                  ))}
                </ul>
              </div>
            ) : (
              <p className="supply-success">
                <CheckCircle2 size={15} style={{ verticalAlign: "-2px", marginRight: 5 }} />
                配置检查已完成。发布时服务端还会核对物品版本、素材状态和活动继承规则。
              </p>
            )}
            <div className="supply-rule-box">
              保存草稿不会改变玩家当前看到的版本。发布后按排期生效；单抽 1 券、十连 10
              券，背包优先，溢出奖励进入邮箱。
            </div>
            <div className="button-row">
              <Button
                disabled={!initial.id || changes.dirty || issues.length > 0}
                onClick={() => save("publish")}
              >
                <UploadCloud size={16} />发布当前草稿
              </Button>
              {changes.dirty && <small>请先保存当前修改</small>}
              {initial.publishedRevision > 0 && (
                <Button secondary onClick={showHistory}>
                  历史发布版本
                </Button>
              )}
            </div>
          </div>
        )}
      </fieldset>

      <footer className="supply-save-bar">
        <span>
          {busy
            ? "正在处理…"
            : changes.dirty
              ? "修改尚未保存"
              : issues.length
                ? `还有 ${issues.length} 项待补完才可发布`
                : "草稿已同步，可前往发布"}
        </span>
        <div className="button-row">
          {pendingSupplyRequest(key) && (
            <Button secondary disabled={busy} onClick={() => save("save", true)}>
              恢复原请求
            </Button>
          )}
          <Button secondary disabled={busy} onClick={leave}>
            关闭
          </Button>
          <Button secondary disabled={busy} onClick={() => save("save")}>
            <Save size={15} />保存草稿
          </Button>
          {section !== "preview" && (
            <Button disabled={busy} onClick={() => save("save", false, true)}>
              <UploadCloud size={15} />保存并去预览发布
            </Button>
          )}
        </div>
      </footer>

      {picker === "library" && (
        <SupplyItemPicker
          client={client}
          close={() => setPicker(false)}
          select={(item) => {
            addReward(item);
            setPicker(false);
          }}
        />
      )}
      {picker === "inventory" && (
        <SupplyInventoryPicker
          client={client}
          scope={scope}
          remaining={100 - (d.rewards || []).length}
          onAdded={addReward}
          close={() => setPicker(false)}
        />
      )}
      {history && (
        <Modal title="历史发布版本" close={() => setHistory(null)}>
          <p>选中的历史内容会载入草稿。再次保存并发布后生效，历史抽取与玩家资产保持原记录。</p>
          {history.map((item) => (
            <div className="supply-section-head" key={item.revision}>
              <span>
                版本 {item.revision} ·{" "}
                {new Date(item.createdAt).toLocaleString("zh-CN", { timeZone: "Asia/Singapore" })}
              </span>
              <Button secondary disabled={busy} onClick={() => restore(item.revision)}>
                载入草稿
              </Button>
            </div>
          ))}
        </Modal>
      )}
    </div>
  );
}

function SupplyItemPicker({ client, close, select }) {
  const [query, setQuery] = useState("");
  const [items, setItems] = useState([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const search = async () => {
    setBusy(true);
    setError("");
    try {
      const r = await client.request(
        `/api/v1/admin/core/items?limit=50&q=${encodeURIComponent(query)}`,
      );
      setItems((r.data.items || []).filter((item) => !item.archived));
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };
  return (
    <Modal title="选择物品库版本" wide close={close}>
      <p>使用现有物品库。先在游戏中保存样品，再在这里选择它的不可变版本。</p>
      <form
        className="supply-filter-row"
        onSubmit={(e) => {
          e.preventDefault();
          search();
        }}
      >
        <Field label="物品名称或编号">
          <input autoFocus value={query} onChange={(e) => setQuery(e.target.value)} />
        </Field>
        <Button type="submit" disabled={busy}>
          搜索物品
        </Button>
      </form>
      {error && <p role="alert">{error}</p>}
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>物品</th>
              <th>版本</th>
              <th>交付服务器</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={`${item.itemRef}-${item.revision}`}>
                <td>
                  {item.displayName}
                  <small>{item.itemRef}</small>
                </td>
                <td>{item.revision}</td>
                <td>{item.compatibleServerIds?.join("、")}</td>
                <td>
                  <Button secondary onClick={() => select(item)}>
                    选择
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {!items.length && <p className="muted">输入关键词搜索，也可以留空查询前 50 个版本。</p>}
    </Modal>
  );
}
