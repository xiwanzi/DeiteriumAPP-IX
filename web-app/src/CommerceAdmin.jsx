import React, { useState, useMemo, useEffect } from "react";
import {
  Search,
  Download,
  ArrowUpRight,
  ChevronLeft,
  ChevronRight,
  ReceiptText,
  SlidersHorizontal,
  Info,
} from "lucide-react";
import { players } from "./data.js";
import { money } from "./domain.js";
import {
  allPlayerSales,
  transactionRows,
  filterTransactions,
  salesSummary,
  downloadCsv,
} from "./commerce.js";
import {
  Avatar,
  Badge,
  Button,
  Empty,
  Modal,
  Price,
  Tabs,
} from "./components.jsx";

const person = (id) =>
  id === "official"
    ? "Deuterium 官方"
    : players.find((p) => p.id === id)?.name || "待接取";
export function SalesOverview({ state, open }) {
  const [query, setQuery] = useState(""),
    [from, setFrom] = useState(""),
    [to, setTo] = useState("");
  const rows = allPlayerSales(state, { query, from, to }),
    sum = (k) => rows.reduce((n, p) => n + p[k], 0);
  return (
    <section>
      <div className="workspace-section-head">
        <div>
          <h2>玩家销售概览</h2>
          <p>查看每位玩家的市场成交情况，点击账号进入完整交易明细。</p>
        </div>
        <Badge tone="neutral">平台审计权限 · 演示账目</Badge>
      </div>
      <div className="finance-metrics">
        {[
          ["净出售金额", sum("sold")],
          ["已结算收入", sum("settled")],
          ["担保中", sum("held")],
          ["实际退款", sum("refund")],
        ].map(([label, v]) => (
          <div key={label}>
            <span>{label}</span>
            <strong>
              {money(v)}
              <small>信用点</small>
            </strong>
          </div>
        ))}
      </div>
      <div className="admin-filterbar">
        <label className="search-input">
          <Search size={17} />
          <input
            aria-label="搜索玩家销售"
            placeholder="搜索游戏 ID / QQ"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </label>
        <div className="date-range">
          <input
            aria-label="销售起始日期"
            type="date"
            value={from}
            max={to || undefined}
            onChange={(e) => setFrom(e.target.value)}
          />
          <span>至</span>
          <input
            aria-label="销售截止日期"
            type="date"
            value={to}
            min={from || undefined}
            onChange={(e) => setTo(e.target.value)}
          />
        </div>
      </div>
      <div className="panel table-wrap finance-table">
        <table>
          <thead>
            <tr>
              <th>玩家</th>
              <th>净出售金额</th>
              <th>成交原额</th>
              <th>已退款</th>
              <th>已结算</th>
              <th>担保中</th>
              <th>成交订单</th>
              <th>明细</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((p) => (
              <tr key={p.id}>
                <td>
                  <button
                    className="table-player"
                    onClick={() =>
                      open({
                        type: "player-commerce",
                        item: p,
                        filters: { from, to },
                      })
                    }
                  >
                    <Avatar user={p.id} />
                    <span>
                      <strong>{p.name}</strong>
                      <small>{p.qq}</small>
                    </span>
                  </button>
                </td>
                {["sold", "gross", "refund", "settled", "held"].map((k) => (
                  <td key={k} className={k === "sold" ? "finance-total" : ""}>
                    {money(p[k])}
                  </td>
                ))}
                <td>{p.orderCount} 笔</td>
                <td>
                  <button
                    className="text-button"
                    onClick={() =>
                      open({
                        type: "player-commerce",
                        item: p,
                        filters: { from, to },
                      })
                    }
                  >
                    查看 <ArrowUpRight size={15} />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {!rows.length && (
          <Empty title="没有匹配的玩家" text="请调整账号关键词或日期范围。" />
        )}
      </div>
      <div className="metric-explanation">
        <Info size={15} />
        <span>
          净出售金额 = 已付款的玩家市场成交原额 −
          实际退款。未付款取消不计；委托报酬和转账在逐笔交易中独立展示，不并入商品销售额。
        </span>
      </div>
    </section>
  );
}
export function TransactionBrowser({
  state,
  open,
  player = "",
  initialFilters = {},
}) {
  const [filters, setFilters] = useState({
      query: "",
      channel: "全部渠道",
      status: "全部状态",
      role: "全部身份",
      from: "",
      to: "",
      ...initialFilters,
    }),
    [page, setPage] = useState(1);
  const rows = filterTransactions(transactionRows(state), {
      ...filters,
      player,
    }),
    size = 10,
    pages = Math.max(1, Math.ceil(rows.length / size)),
    current = Math.min(page, pages);
  const update = (k, v) => {
    setFilters((f) => ({ ...f, [k]: v }));
    setPage(1);
  };
  const filteredTotal = rows.reduce((n, t) => n + t.paidAmount - t.refunded, 0);
  return (
    <section className="transaction-browser">
      <div className="workspace-section-head">
        <div>
          <h2>{player ? "逐笔交易" : "全站交易明细"}</h2>
          <p>
            {player
              ? "买入、卖出、委托与转账，按同一账号范围查询。"
              : "每笔交易保留渠道、参与方、款项状态与成交信息。"}
          </p>
        </div>
        <Button
          secondary
          disabled={!rows.length}
          onClick={() => downloadCsv(rows)}
        >
          <Download size={16} />
          导出筛选结果
        </Button>
      </div>
      <div className="admin-filterbar">
        <label className="search-input">
          <Search size={16} />
          <input
            aria-label="搜索交易"
            value={filters.query}
            placeholder="交易号、商品、玩家…"
            onChange={(e) => update("query", e.target.value)}
          />
        </label>
        <select
          aria-label="交易渠道"
          value={filters.channel}
          onChange={(e) => update("channel", e.target.value)}
        >
          {["全部渠道", "玩家市场", "官方商城", "委托", "转账"].map((x) => (
            <option key={x}>{x}</option>
          ))}
        </select>
        <select
          aria-label="交易状态"
          value={filters.status}
          onChange={(e) => update("status", e.target.value)}
        >
          {[
            "全部状态",
            "待付款",
            "待发货",
            "已发货",
            "待开工",
            "施工中",
            "待验收",
            "已完成",
            "已领取",
            "已退款",
            "已取消",
            "待接取",
            "进行中",
          ].map((x) => (
            <option key={x}>{x}</option>
          ))}
        </select>
        {player && (
          <select
            aria-label="交易身份"
            value={filters.role}
            onChange={(e) => update("role", e.target.value)}
          >
            {["全部身份", "卖出 / 收入", "买入 / 支出"].map((x) => (
              <option key={x}>{x}</option>
            ))}
          </select>
        )}
        <div className="date-range">
          <input
            aria-label="交易起始日期"
            type="date"
            value={filters.from}
            max={filters.to || undefined}
            onChange={(e) => update("from", e.target.value)}
          />
          <span>至</span>
          <input
            aria-label="交易截止日期"
            type="date"
            value={filters.to}
            min={filters.from || undefined}
            onChange={(e) => update("to", e.target.value)}
          />
        </div>
      </div>
      <div className="panel table-wrap finance-table">
        <table>
          <thead>
            <tr>
              <th>交易内容 / 编号</th>
              <th>渠道</th>
              <th>付款方 → 收款方</th>
              <th>成交 / 实退</th>
              <th>状态</th>
              <th>创建时间</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {rows.slice((current - 1) * size, current * size).map((t) => (
              <tr key={`${t.kind}:${t.id}`}>
                <td>
                  <button
                    className="transaction-title"
                    onClick={() => open({ type: "transaction", item: t })}
                  >
                    <strong>{t.title}</strong>
                    <small>{t.number}</small>
                  </button>
                </td>
                <td>{t.channel}</td>
                <td>
                  {person(t.buyer)} <span className="muted">→</span>{" "}
                  {person(t.seller)}
                </td>
                <td>
                  <strong>{money(t.paidAmount)}</strong>
                  <small className={t.refunded ? "refund-text" : "muted"}>
                    {t.refunded ? `退款 ${money(t.refunded)}` : "无退款"}
                  </small>
                </td>
                <td>
                  <Badge
                    tone={
                      ["已完成", "已领取"].includes(t.status)
                        ? "sage"
                        : t.status === "已退款"
                          ? "neutral"
                          : "blue"
                    }
                  >
                    {t.status}
                  </Badge>
                </td>
                <td>
                  {new Date(t.time).toLocaleDateString("zh-CN")}
                  <small className="muted">
                    {new Date(t.time).toLocaleTimeString("zh-CN", {
                      hour: "2-digit",
                      minute: "2-digit",
                    })}
                  </small>
                </td>
                <td>
                  <button
                    className="round-link"
                    aria-label={`查看交易 ${t.number}`}
                    onClick={() => open({ type: "transaction", item: t })}
                  >
                    <ArrowUpRight size={15} />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {!rows.length && (
          <Empty
            title="没有符合条件的交易"
            text="试试调整渠道、状态或日期范围。"
          />
        )}
      </div>
      <footer className="table-pagination">
        <span>
          共 {rows.length} 笔 · 筛选净额 {money(filteredTotal)} 信用点
        </span>
        <div>
          <button
            disabled={current <= 1}
            aria-label="上一页交易"
            onClick={() => setPage(current - 1)}
          >
            <ChevronLeft size={16} />
          </button>
          <span>
            {current} / {pages}
          </span>
          <button
            disabled={current >= pages}
            aria-label="下一页交易"
            onClick={() => setPage(current + 1)}
          >
            <ChevronRight size={16} />
          </button>
        </div>
      </footer>
    </section>
  );
}
export function PlayerCommerce({ item: p, state, open, close, filters }) {
  const summary = salesSummary(
    filterTransactions(transactionRows(state), filters),
    p.id,
  );
  return (
    <Modal title={`${p.name} · 账号交易档案`} close={close} wide>
      <div className="commerce-player-heading">
        <Avatar user={p.id} size="large" />
        <div>
          <h2>{p.name}</h2>
          <p>
            游戏 ID · {p.name} <span>QQ · {p.qq}</span>
          </p>
        </div>
        <Badge tone="neutral">平台查询</Badge>
      </div>
      <div className="finance-metrics compact">
        {[
          ["净出售金额", summary.sold],
          ["实际退款", summary.refund],
          ["已结算", summary.settled],
          ["担保中", summary.held],
        ].map(([l, v]) => (
          <div key={l}>
            <span>{l}</span>
            <strong>{money(v)}</strong>
          </div>
        ))}
      </div>
      <TransactionBrowser
        state={state}
        open={open}
        player={p.id}
        initialFilters={filters}
      />
    </Modal>
  );
}
export function TransactionDetail({ item: t, close }) {
  return (
    <Modal title="交易档案" close={close} wide>
      <div className="transaction-detail-heading">
        <span className="stat-icon blue">
          <ReceiptText size={25} />
        </span>
        <div>
          <Badge tone="neutral">{t.channel}</Badge>
          <h2>{t.title}</h2>
          <p>{t.number}</p>
        </div>
        <Badge>{t.status}</Badge>
      </div>
      <div className="finance-metrics compact">
        {[
          ["成交原额", t.paidAmount],
          ["已退款", t.refunded],
          ["已结算", t.settled],
          ["担保中", t.held],
        ].map(([l, v]) => (
          <div key={l}>
            <span>{l}</span>
            <strong>{money(v)}</strong>
          </div>
        ))}
      </div>
      <dl className="detail-list">
        <div>
          <dt>付款方</dt>
          <dd>{person(t.buyer)}</dd>
        </div>
        <div>
          <dt>收款方</dt>
          <dd>{person(t.seller)}</dd>
        </div>
        <div>
          <dt>创建时间</dt>
          <dd>{new Date(t.time).toLocaleString("zh-CN")}</dd>
        </div>
        <div>
          <dt>原始记录 ID</dt>
          <dd className="mono">{t.id}</dd>
        </div>
        {t.snapshot?.place && (
          <div>
            <dt>成交时交付地点</dt>
            <dd>{t.snapshot.place}</dd>
          </div>
        )}
      </dl>
      <h3>成交内容</h3>
      <p className="description">
        {t.snapshot?.description ||
          t.lines?.map((l) => `${l.title} × ${l.qty}`).join("；") ||
          t.title}
      </p>
      <div className="notice-box">
        本机演示账目。正式交易明细和金额必须由有权限的后台接口返回，不能根据网页传入的数据执行退款。
      </div>
    </Modal>
  );
}
