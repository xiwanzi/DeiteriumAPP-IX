import { usePlayerNameState } from "./use-player-names.js";
import React, { useEffect, useRef, useState } from "react";
import { ArrowUpRight, RefreshCw, Search, X, RotateCcw } from "lucide-react";
import { Badge, Button, Empty, Field, Modal, PageHead, Tabs } from "./components.jsx";
import { credit } from "./format.js";
import { fundsLabels } from "./business.js";
import { ContentBlocks } from "./RemoteCollection.jsx";
import AuditEvents from "./AuditManagement.jsx";
import { CopyButton } from "./AdminPrimitives.jsx";

const tabs = ["资金流水", "玩家查询", "订单审计", "商品审计", "管理操作"];
const states = {
  PAYMENT_PROCESSING: "付款处理中",
  AWAITING_CLAIM: "待领取",
  CLAIMED: "已领取",
  PENDING: "待处理",
  PAID: "已付款",
  AWAITING_SHIPMENT: "待发货",
  SHIPPED: "已发货",
  WORK_STARTED: "施工中",
  WORK_COMPLETED: "待验收",
  CONFIRMED: "已完成",
  REFUNDED: "已退款",
  CANCELLED: "已取消",
  ACTIVE: "在售",
  DRAFT: "草稿",
  UNLISTED: "已下架",
  ARCHIVED: "已归档",
};
const date = (value) => (value ? new Date(value).toLocaleString("zh-CN", { hour12: false }) : "—");
const channel = (value) => (value === "OFFICIAL_STORE" ? "官方商城" : "玩家市场");
const dateDay = (d) =>
  new Date(d.getTime() + 8 * 3600000).toISOString().slice(0, 10);

export default function TradeAudit({ client }) {
  const [tab, setTab] = useState(tabs[0]);
  const [player, setPlayer] = usePlayerNameState(client, null);
  const [detail, setDetail] = useState(null);
  const select = (value, next = "资金流水") => {
    setPlayer(value);
    setTab(next);
  };
  return (
    <>
      <PageHead
        eyebrow="PLATFORM AUDIT"
        title="交易与玩家审计"
        subtitle="查询全服玩家收支，追溯订单、商品与后台管理操作记录。"
      />
      <Tabs values={tabs} value={tab} onChange={setTab} label="审计内容" />
      {player && tab !== "玩家查询" && tab !== "管理操作" && (
        <div className="audit-player-scope">
          <div>
            <strong>{player.gameId}</strong>
            <span>当前查看此玩家的记录</span>
          </div>
          <Button secondary onClick={() => setTab("玩家查询")}>
            更换玩家
          </Button>
          <Button secondary onClick={() => setPlayer(null)}>
            <X size={15} />查看全服
          </Button>
        </div>
      )}
      {tab === "管理操作" ? (
        <AuditEvents client={client} embedded />
      ) : tab === "玩家查询" ? (
        <PlayerBrowser client={client} onSelect={select} />
      ) : (
        <AuditCollection
          key={`${tab}:${player?.playerRef || "all"}`}
          client={client}
          tab={tab}
          player={player}
          onDetail={(id) =>
            setDetail({ collection: tab === "订单审计" ? "orders" : "products", id })
          }
        />
      )}
      {detail && (
        <Modal
          title={detail.collection === "orders" ? "订单详情" : "商品详情"}
          close={() => setDetail(null)}
          wide
        >
          <AuditDetail client={client} {...detail} />
        </Modal>
      )}
    </>
  );
}

function PlayerBrowser({ client, onSelect }) {
  const [query, setQuery] = useState("");
  const [items, setItems] = usePlayerNameState(client, []);
  const [cursor, setCursor] = useState(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const generation = useRef(0);
  const load = async (more = false, overrideQ = null) => {
    const qText = overrideQ !== null ? overrideQ : query;
    const current = ++generation.current;
    setBusy(true);
    setError("");
    try {
      const r = await client.request(
        `/api/v1/admin/players?${more && cursor ? `cursor=${encodeURIComponent(cursor)}` : `q=${encodeURIComponent(qText.trim())}`}`,
      );
      if (current !== generation.current) return;
      setItems((old) => (more ? [...old, ...r.data] : r.data));
      setCursor(r.page?.nextCursor);
    } catch (e) {
      if (current === generation.current) setError(e.message);
    } finally {
      if (current === generation.current) setBusy(false);
    }
  };
  useEffect(() => {
    load();
    return () => {
      generation.current++;
    };
  }, []);
  return (
    <>
      <form
        className="adm-toolbar"
        onSubmit={(e) => {
          e.preventDefault();
          load();
        }}
      >
        <div className="adm-toolbar-left">
          <label className="adm-search-box">
            <Search size={15} />
            <input
              aria-label="搜索玩家"
              value={query}
              maxLength={100}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="输入游戏名、QQ 或 UUID…"
            />
            {query && (
              <button
                type="button"
                className="adm-search-clear"
                aria-label="清空搜索"
                onClick={() => {
                  setQuery("");
                  load(false, "");
                }}
              >
                <X size={14} />
              </button>
            )}
          </label>
        </div>
        <div className="adm-toolbar-right">
          <Button type="submit" disabled={busy}>
            <Search size={15} />查找玩家
          </Button>
        </div>
      </form>
      {error && (
        <p className="notice-box" role="alert">
          {error}
        </p>
      )}
      <div className="panel table-wrap">
        <table>
          <thead>
            <tr>
              <th>玩家</th>
              <th>QQ</th>
              <th>账号状态</th>
              <th>查看记录</th>
            </tr>
          </thead>
          <tbody>
            {items.map((p) => (
              <tr key={p.uuid}>
                <td>
                  <strong>{p.gameId}</strong>
                  <div style={{ display: "flex", alignItems: "center", gap: 6, marginTop: 4 }}>
                    <small className="audit-id" style={{ margin: 0 }}>
                      {p.uuid}
                    </small>
                    <CopyButton value={p.uuid} label="UUID" compact />
                  </div>
                </td>
                <td>{p.qq || "—"}</td>
                <td>
                  <Badge tone={p.registered ? "sage" : "neutral"}>
                    {p.registered ? (p.status === "active" ? "已注册" : "账号受限") : "游戏玩家"}
                  </Badge>
                </td>
                <td>
                  <div className="adm-compact-row-actions">
                    <Button secondary onClick={() => onSelect(p)}>
                      流水
                    </Button>
                    <Button secondary onClick={() => onSelect(p, "订单审计")}>
                      订单
                    </Button>
                    <Button secondary onClick={() => onSelect(p, "商品审计")}>
                      商品
                    </Button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {!items.length && !busy && !error && (
          <Empty title="没有找到玩家" text="试试游戏名、QQ 或 UUID。" />
        )}
      </div>
      {busy && <p role="status">正在查找…</p>}
      {cursor && (
        <div style={{ marginTop: 14 }}>
          <Button secondary disabled={busy} onClick={() => load(true)}>
            加载更多玩家
          </Button>
        </div>
      )}
    </>
  );
}

function AuditCollection({ client, tab, player, onDetail }) {
  const ledger = tab === "资金流水";
  const orders = tab === "订单审计";
  const collection = ledger ? "transactions" : orders ? "orders" : "products";
  const emptyFilters = {
    q: "",
    direction: "",
    businessType: "",
    channel: "",
    kind: "",
    status: "",
    from: "",
    to: "",
  };
  const [filters, setFilters] = useState(emptyFilters);
  const [items, setItems] = usePlayerNameState(client, []);
  const [cursor, setCursor] = useState(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const generation = useRef(0);
  const set = (key) => (e) => setFilters((old) => ({ ...old, [key]: e.target.value }));
  const load = async (more = false, overrideFilters = null) => {
    const active = overrideFilters || filters;
    const current = ++generation.current;
    setBusy(true);
    setError("");
    try {
      const query = new URLSearchParams({ limit: "25" });
      if (more && cursor) query.set("cursor", cursor);
      else {
        if (player) query.set("playerRef", player.playerRef);
        for (const key of ledger
          ? ["direction", "businessType"]
          : ["q", orders ? "channel" : "kind", "status"])
          if (active[key]) query.set(key, active[key].trim());
        if (ledger) {
          if (active.from) query.set("from", new Date(`${active.from}T00:00:00+08:00`).toISOString());
          if (active.to)
            query.set(
              "to",
              new Date(
                Math.min(new Date(`${active.to}T00:00:00+08:00`).getTime() + 86400000, Date.now()),
              ).toISOString(),
            );
        }
      }
      const r = await client.request(`/api/v1/admin/${collection}?${query}`);
      if (current !== generation.current) return;
      setItems((old) => (more ? [...old, ...r.data] : r.data));
      setCursor(r.page?.nextCursor);
    } catch (e) {
      if (current === generation.current) setError(e.message);
    } finally {
      if (current === generation.current) setBusy(false);
    }
  };
  useEffect(() => {
    load();
    return () => {
      generation.current++;
    };
  }, []);

  const applyDays = (days) => {
    const now = new Date();
    const start = new Date(now.getTime() - days * 86400000);
    const next = { ...filters, from: dateDay(start), to: dateDay(now) };
    setFilters(next);
    load(false, next);
  };

  const resetFilters = () => {
    setFilters(emptyFilters);
    load(false, emptyFilters);
  };

  return (
    <>
      <form
        className="panel admin-form audit-filters"
        onSubmit={(e) => {
          e.preventDefault();
          load();
        }}
      >
        {ledger && (
          <div className="adm-quick-dates">
            <span>快捷日期：</span>
            {[
              [0, "今天"],
              [7, "近 7 天"],
              [30, "近 30 天"],
            ].map(([days, label]) => (
              <button
                key={days}
                type="button"
                className="adm-preset-chip"
                disabled={busy}
                onClick={() => applyDays(days)}
              >
                {label}
              </button>
            ))}
          </div>
        )}
        <div className="form-grid">
          {ledger ? (
            <>
              <Field label="收支方向">
                <select value={filters.direction} onChange={set("direction")}>
                  <option value="">全部收支</option>
                  <option value="income">收入</option>
                  <option value="expense">支出</option>
                </select>
              </Field>
              <Field label="交易来源">
                <select value={filters.businessType} onChange={set("businessType")}>
                  {[
                    ["", "全部来源"],
                    ["GAME", "游戏内收支"],
                    ["TRANSFER", "App / 网页转账"],
                    ["OFFICIAL_STORE", "官方商城"],
                    ["MARKET_ORDER", "玩家市场"],
                    ["COMMISSION", "委托"],
                  ].map(([key, label]) => (
                    <option key={key} value={key}>
                      {label}
                    </option>
                  ))}
                </select>
              </Field>
              <Field label="起始日期" type="date" value={filters.from} onChange={set("from")} />
              <Field label="结束日期" type="date" value={filters.to} onChange={set("to")} />
            </>
          ) : (
            <>
              <Field
                label={orders ? "订单号 / 商品名" : "商品名称 / 编号"}
                value={filters.q}
                maxLength={100}
                onChange={set("q")}
              />
              <Field label="类型">
                <select
                  value={orders ? filters.channel : filters.kind}
                  onChange={set(orders ? "channel" : "kind")}
                >
                  <option value="">全部类型</option>
                  <option value={orders ? "OFFICIAL_STORE" : "product"}>官方商城</option>
                  <option value={orders ? "PLAYER_MARKET" : "listing"}>玩家市场</option>
                </select>
              </Field>
              <Field label="状态">
                <select value={filters.status} onChange={set("status")}>
                  <option value="">全部状态</option>
                  {(orders
                    ? [
                        "PAYMENT_PROCESSING",
                        "AWAITING_SHIPMENT",
                        "SHIPPED",
                        "WORK_COMPLETED",
                        "AWAITING_CLAIM",
                        "CLAIMED",
                        "CONFIRMED",
                        "REFUNDED",
                        "CANCELLED",
                      ]
                    : ["ACTIVE", "DRAFT", "UNLISTED", "ARCHIVED"]
                  ).map((key) => (
                    <option key={key} value={key}>
                      {states[key]}
                    </option>
                  ))}
                </select>
              </Field>
            </>
          )}
        </div>
        <div className="button-row">
          <Button type="submit" disabled={busy}>
            <Search size={15} />查询
          </Button>
          <Button secondary disabled={busy} onClick={resetFilters}>
            <RotateCcw size={15} />重置
          </Button>
          <Button secondary disabled={busy} onClick={() => load()}>
            <RefreshCw size={15} />刷新
          </Button>
          <span className="muted">
            {ledger
              ? "默认最近一年 · 每行代表一个玩家账户的收支"
              : "包含已结束、已下架及玩家隐藏的记录"}
          </span>
        </div>
      </form>
      {error && (
        <p className="notice-box" role="alert">
          {error}
        </p>
      )}
      <div className="panel table-wrap audit-table">
        <table>
          <thead>
            <tr>
              {(ledger
                ? ["时间", "玩家", "交易", "金额", "交易后余额"]
                : orders
                  ? ["订单 / 商品", "买家 → 卖家", "金额", "状态", "创建时间", ""]
                  : ["商品", "类型", "价格 / 库存", "状态", "创建时间", ""]
              ).map((title, n) => (
                <th key={n}>{title || <span className="sr-only">详情</span>}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {items.map((item) =>
              ledger ? (
                <tr key={item.recordId}>
                  <td>{date(item.occurredAt)}</td>
                  <td>
                    <strong>{item.player?.gameId}</strong>
                  </td>
                  <td>
                    <strong>{item.title}</strong>
                    <small className="audit-id">
                      {item.otherPlayer?.gameId ? `对方：${item.otherPlayer.gameId}` : item.note}
                    </small>
                  </td>
                  <td className={item.direction === "income" ? "audit-income" : "audit-expense"}>
                    {item.direction === "income" ? "+" : "−"}
                    {credit(item.amount)}
                  </td>
                  <td>{credit(item.afterBalance)}</td>
                </tr>
              ) : orders ? (
                <tr key={item.orderId}>
                  <td>
                    <strong>
                      {item.items?.map((line) => line.title).join("、") || channel(item.channel)}
                    </strong>
                    <div style={{ display: "flex", alignItems: "center", gap: 6, marginTop: 4 }}>
                      <small className="audit-id" style={{ margin: 0 }}>
                        {item.orderId}
                      </small>
                      <CopyButton value={item.orderId} label="订单号" compact />
                    </div>
                  </td>
                  <td>
                    {item.buyer?.displayName || "—"}
                    <small className="audit-id">→ {item.seller?.displayName || "官方商店"}</small>
                  </td>
                  <td>{credit(item.amount)}</td>
                  <td>
                    <Badge>{states[item.status] || item.status}</Badge>
                    <small className="audit-id">
                      {fundsLabels[item.fundsStatus] || item.fundsStatus}
                    </small>
                  </td>
                  <td>{date(item.createdAt)}</td>
                  <td>
                    <Button secondary onClick={() => onDetail(item.orderId)}>
                      详情
                      <ArrowUpRight size={15} />
                    </Button>
                  </td>
                </tr>
              ) : (
                <tr key={item.productId}>
                  <td>
                    <strong>{item.title}</strong>
                    <small className="audit-id">{item.productId}</small>
                  </td>
                  <td>{item.kind === "product" ? "官方商品" : "市场商品"}</td>
                  <td>
                    {credit(item.price)}
                    <small className="audit-id">库存 {item.stock ?? "不限"}</small>
                  </td>
                  <td>
                    <Badge>{states[item.state] || item.state}</Badge>
                  </td>
                  <td>{date(item.createdAt)}</td>
                  <td>
                    <Button secondary onClick={() => onDetail(item.productId)}>
                      详情
                      <ArrowUpRight size={15} />
                    </Button>
                  </td>
                </tr>
              ),
            )}
          </tbody>
        </table>
        {!items.length && !busy && !error && (
          <Empty title="没有符合条件的记录" text="可以更换玩家、日期或状态后重新查询。" />
        )}
      </div>
      {busy && (
        <p role="status" className="muted">
          正在读取记录…
        </p>
      )}
      {cursor && (
        <div style={{ marginTop: 14 }}>
          <Button secondary disabled={busy} onClick={() => load(true)}>
            加载更多记录
          </Button>
        </div>
      )}
    </>
  );
}

function AuditDetail({ client, collection, id }) {
  const [value, setValue] = usePlayerNameState(client, null);
  const [error, setError] = useState("");
  useEffect(() => {
    let alive = true;
    client
      .request(`/api/v1/admin/${collection}/${encodeURIComponent(id)}`)
      .then((r) => {
        if (alive) setValue(r.data);
      })
      .catch((e) => {
        if (alive) setError(e.message);
      });
    return () => {
      alive = false;
    };
  }, [id, collection]);
  if (error)
    return (
      <p className="notice-box" role="alert">
        {error}
      </p>
    );
  if (!value) return <p role="status">正在读取详情…</p>;
  const product = value.draft || value;
  const images = value.draftImages || value.photos || value.images || [];
  return (
    <div className="audit-detail">
      <Badge tone="neutral">只读审计</Badge>
      {collection === "orders" ? (
        <>
          <h3>
            {channel(value.channel)} · {states[value.status] || value.status}
          </h3>
          <p>
            {value.buyer?.displayName} → {value.seller?.displayName || "官方商店"}
          </p>
          <p className="price">
            {credit(value.amount)}
            <small>信用点</small>
          </p>
          <p>{fundsLabels[value.fundsStatus] || value.fundsStatus}</p>
          <div className="panel admin-form">
            {value.items?.map((line, i) => (
              <p key={i}>
                <strong>{line.title}</strong> × {line.quantity} · {credit(line.unitPrice)} 信用点
              </p>
            ))}
          </div>
          <dl>
            <div>
              <dt>订单编号</dt>
              <dd>{value.orderId}</dd>
            </div>
            <div>
              <dt>创建时间</dt>
              <dd>{date(value.createdAt)}</dd>
            </div>
            <div>
              <dt>交付方式</dt>
              <dd>
                {{ MAILBOX: "游戏内邮箱", DOOR: "送货上门", PICKUP: "约定自取", WORKSITE: "工程现场" }[
                  value.delivery?.method
                ] || value.delivery?.method}
              </dd>
            </div>
            {value.delivery?.location && (
              <div>
                <dt>交付地点</dt>
                <dd>{value.delivery.location}</dd>
              </div>
            )}
          </dl>
          {value.refund && (
            <div className="notice-box">
              退款状态：{value.refund.status}
              <p>{value.refund.description || value.refund.reason}</p>
              {value.refund.rejectionReason && <p>拒绝原因：{value.refund.rejectionReason}</p>}
            </div>
          )}
        </>
      ) : (
        <>
          <h3>{product.title}</h3>
          <p>{product.subtitle}</p>
          <p className="price">
            {credit(product.price)}
            <small>信用点</small>
          </p>
          <p style={{ whiteSpace: "pre-wrap" }}>{product.description}</p>
          <ContentBlocks blocks={product.contentBlocks || []} media={images} />
          <p>{product.deliverySummary}</p>
          <p>{product.estimatedDelivery}</p>
          {product.includedItems?.length > 0 && (
            <ul>
              {product.includedItems.map((text, i) => (
                <li key={i}>{text}</li>
              ))}
            </ul>
          )}
        </>
      )}
      {images.length > 0 && (
        <div className="audit-images">
          {images.map(
            (image, i) =>
              image.url && (
                <img
                  key={image.assetId || i}
                  src={image.url}
                  alt={image.altText || "交易商品图片"}
                />
              ),
          )}
        </div>
      )}
    </div>
  );
}
