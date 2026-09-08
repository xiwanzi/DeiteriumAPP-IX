import React, { useState } from "react";
import {
  ShieldCheck,
  Package,
  MessageSquareWarning,
  Users,
  ArrowUpRight,
  ShieldBan,
  Search,
  Check,
  Clock3,
} from "lucide-react";
import { players } from "./data.js";
import { SalesOverview, TransactionBrowser } from "./CommerceAdmin.jsx";
import { AnnouncementAdmin } from "./Announcements.jsx";
import { isBanned, money } from "./domain.js";
import {
  PageHead,
  Tabs,
  Button,
  Avatar,
  Badge,
  Field,
  Modal,
  Empty,
  Price,
  SectionHead,
  media,
} from "./components.jsx";
export default function Admin({ state, act, open, query }) {
  const [tab, setTab] = useState(
    new URLSearchParams(location.search).get("section") === "announcements"
      ? "公告管理"
      : "概览",
  );
  if (!players.find((p) => p.id === state.user)?.admin)
    return (
      <>
        <PageHead eyebrow="OFFICIAL WORKSPACE" title="官方管理" />
        <Empty
          title="此账号没有管理权限"
          text="可在退出后选择 Mori 体验管理员页面。"
        />
      </>
    );
  const banned = players.filter((p) => isBanned(state, p.id));
  return (
    <>
      <PageHead
        eyebrow="OFFICIAL WORKSPACE"
        title={
          <>
            有序管理，<span className="muted-heading">安心相伴。</span>
          </>
        }
        subtitle="商品、申诉与账号管理，汇聚在同一个工作台。"
      >
        <Badge tone="sage">
          <ShieldCheck size={14} /> Mori · 体验管理员
        </Badge>
      </PageHead>
      <Tabs
        values={[
          "概览",
          "商品管理",
          "玩家销售",
          "交易明细",
          "公告管理",
          "申诉处理",
          "账号与封禁",
          "操作记录",
        ]}
        value={tab}
        onChange={setTab}
      />
      {tab === "概览" && (
        <>
          <div className="admin-stats">
            {[
              [Package, "官方商品", state.products.length, "商品管理"],
              [
                MessageSquareWarning,
                "待处理申诉",
                state.cases.filter((c) => c.status === "待处理").length,
                "申诉处理",
              ],
              [Users, "演示账号", players.length, "账号与封禁"],
              [ShieldBan, "当前封禁", banned.length, "账号与封禁"],
            ].map(([Icon, label, n, to]) => (
              <button
                key={label}
                className="admin-stat"
                onClick={() => setTab(to)}
              >
                <span className="stat-icon blue">
                  <Icon size={21} />
                </span>
                <span>{label}</span>
                <strong>{n}</strong>
                <ArrowUpRight size={17} />
              </button>
            ))}
          </div>
          <SectionHead
            title="需要关注"
            description="从待办开始，把每一件事处理清楚。"
          />
          <div className="panel">
            {state.cases.map((c) => (
              <button
                key={c.id}
                className="order-row"
                onClick={() => open({ type: "case", item: c })}
              >
                <span className="stat-icon amber">
                  <MessageSquareWarning size={22} />
                </span>
                <div>
                  <strong>{c.title}</strong>
                  <small>{c.id} · 建筑服务争议</small>
                </div>
                <Badge tone={c.status === "待处理" ? "amber" : "sage"}>
                  {c.status}
                </Badge>
                <ArrowUpRight size={18} />
              </button>
            ))}
          </div>
          <div className="notice-box admin-note">
            <ShieldCheck size={20} />
            <span>
              这是本机管理端演示。商品修改、账号封禁和案件意见只影响这份浏览器数据，正式管理权限与资金处理以服务端为准。
            </span>
          </div>
        </>
      )}
      {tab === "商品管理" && (
        <div className="admin-create-row">
          <Button onClick={() => open({ type: "product-create" })}>
            发布官方商品
          </Button>
        </div>
      )}
      {tab === "商品管理" && (
        <div className="panel table-wrap">
          <table>
            <thead>
              <tr>
                <th>商品</th>
                <th>价格 / 信用点</th>
                <th>库存</th>
                <th>状态</th>
                <th>
                  <span className="sr-only">操作</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {state.products
                .filter((p) =>
                  p.title.toLowerCase().includes(query.toLowerCase()),
                )
                .map((p) => (
                  <tr key={p.id}>
                    <td>
                      <div className="table-product">
                        <img src={media(p.image)} alt={p.title} />
                        <span>
                          <strong>{p.title}</strong>
                          <small>{p.brand}</small>
                        </span>
                      </div>
                    </td>
                    <td>{money(p.price)}</td>
                    <td>{p.stock}</td>
                    <td>
                      <Badge tone={p.stock ? "sage" : "neutral"}>
                        {p.stock ? "在售" : "售罄"}
                      </Badge>
                    </td>
                    <td>
                      <button
                        className="text-button"
                        onClick={() => open({ type: "product-edit", item: p })}
                      >
                        编辑 <ArrowUpRight size={16} />
                      </button>
                    </td>
                  </tr>
                ))}
            </tbody>
          </table>
        </div>
      )}
      {tab === "申诉处理" && (
        <div className="panel">
          {state.cases.map((c) => (
            <button
              key={c.id}
              className="order-row"
              onClick={() => open({ type: "case", item: c })}
            >
              <Avatar user={c.player} />
              <div>
                <strong>{c.title}</strong>
                <small>
                  {c.id} · 申请人 {c.player}
                </small>
              </div>
              <Badge tone={c.status === "待处理" ? "amber" : "sage"}>
                {c.status}
              </Badge>
              <ArrowUpRight size={18} />
            </button>
          ))}
        </div>
      )}
      {tab === "账号与封禁" && (
        <>
          <div className="notice-box">
            封禁限制 App 和网页登录，未完成交易需管理员接管；不联动 Minecraft
            游戏封禁。
          </div>
          <div className="panel table-wrap">
            <table>
              <thead>
                <tr>
                  <th>账号</th>
                  <th>QQ</th>
                  <th>账号状态</th>
                  <th>待跟进交易</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                {players
                  .filter((p) =>
                    `${p.name}${p.qq}`
                      .toLowerCase()
                      .includes(query.toLowerCase()),
                  )
                  .map((p) => {
                    const ban = state.bans.find(
                      (b) =>
                        b.player === p.id &&
                        !b.revoked &&
                        (!b.until || b.until > Date.now()),
                    );
                    const pending =
                      state.orders.filter(
                        (o) => o.buyer === p.id || o.seller === p.id,
                      ).length +
                      state.commissions.filter(
                        (c) =>
                          ["OPEN", "ACTIVE", "COMPLETED"].includes(c.status) &&
                          (c.owner === p.id || c.worker === p.id),
                      ).length;
                    return (
                      <tr key={p.id}>
                        <td>
                          <div className="table-product">
                            <Avatar user={p.id} />
                            <span>
                              <strong>{p.name}</strong>
                              <small>{p.admin ? "管理员" : "玩家"}</small>
                            </span>
                          </div>
                        </td>
                        <td>{p.qq}</td>
                        <td>
                          <Badge tone={ban ? "red" : "sage"}>
                            {ban ? "已封禁" : "正常"}
                          </Badge>
                        </td>
                        <td>
                          <button
                            className="text-button"
                            onClick={() =>
                              open({ type: "admin-player", item: p })
                            }
                          >
                            {pending} 笔 <ArrowUpRight size={14} />
                          </button>
                        </td>
                        <td>
                          {!p.admin ? (
                            <button
                              className={`text-button ${ban ? "" : "red-text"}`}
                              onClick={() =>
                                open({ type: "ban", item: p, ban })
                              }
                            >
                              {ban ? "解除封禁" : "封禁账号"}
                            </button>
                          ) : (
                            <span className="muted">受保护账号</span>
                          )}
                        </td>
                      </tr>
                    );
                  })}
              </tbody>
            </table>
          </div>
        </>
      )}
      {tab === "玩家销售" && <SalesOverview state={state} open={open} />}
      {tab === "交易明细" && <TransactionBrowser state={state} open={open} />}
      {tab === "公告管理" && <AnnouncementAdmin state={state} open={open} />}
      {tab === "操作记录" && (
        <div className="panel">
          {state.audit.length ? (
            state.audit.map((a) => (
              <div className="record-row" key={a.id}>
                <span className="stat-icon blue">
                  <HistoryIcon />
                </span>
                <div>
                  <strong>{a.title}</strong>
                  <small>{a.reason}</small>
                </div>
                <small>{new Date(a.time).toLocaleString("zh-CN")}</small>
              </div>
            ))
          ) : (
            <Empty
              title="还没有管理操作记录"
              text="编辑商品、处理案件或封禁账号后，将在这里留下记录。"
            />
          )}
        </div>
      )}
    </>
  );
}
const HistoryIcon = () => <Clock3 size={21} />;
export function BanDialog({ item: p, ban, act, close }) {
  return (
    <Modal
      title={ban ? `解除 ${p.name} 的封禁` : `封禁 ${p.name}`}
      close={close}
    >
      <form
        className="form-stack"
        onSubmit={(e) => {
          e.preventDefault();
          const f = Object.fromEntries(new FormData(e.currentTarget));
          if (
            act(
              ban
                ? { type: "UNBAN", id: ban.id, reason: f.reason }
                : {
                    type: "BAN",
                    player: p.id,
                    reason: f.reason,
                    days: Number(f.days),
                  },
              ban ? "演示账号已解封" : "演示账号已封禁",
            )
          )
            close();
        }}
      >
        <div className="notice-box">
          {ban
            ? `原封禁原因：${ban.reason}`
            : "App 与网页禁止登录，已有交易保留。不影响游戏登录；本次仅操作演示账号。"}
        </div>
        {!ban && (
          <Field label="封禁期限">
            <select name="days">
              <option value="1">1 天</option>
              <option value="7">7 天</option>
              <option value="30">30 天</option>
              <option value="0">永久</option>
            </select>
          </Field>
        )}
        <Field label={ban ? "解封原因" : "封禁原因"}>
          <textarea
            name="reason"
            rows={3}
            required
            minLength={2}
            maxLength={500}
            placeholder="请写明具体原因，操作将保留记录。"
          />
        </Field>
        <Button danger={!ban} type="submit">
          {ban ? "确认解封" : "确认封禁演示账号"}
        </Button>
      </form>
    </Modal>
  );
}
export function ProductEdit({ item: p, act, close }) {
  return (
    <Modal title="编辑官方商品" close={close}>
      <form
        className="form-stack"
        onSubmit={(e) => {
          e.preventDefault();
          const f = Object.fromEntries(new FormData(e.currentTarget));
          if (
            act(
              { type: "PRODUCT_SAVE", id: p.id, ...f, stock: Number(f.stock) },
              "商品信息已保存",
            )
          )
            close();
        }}
      >
        <Field label="商品标题" name="title" required defaultValue={p.title} />
        <div className="form-grid">
          <Field
            label="价格 / 信用点"
            name="price"
            type="number"
            min="0.01"
            max="9999999.99"
            step="0.01"
            defaultValue={p.price / 100}
            required
          />
          <Field
            label="可售库存"
            name="stock"
            type="number"
            min="0"
            max="999"
            required
            defaultValue={p.stock}
          />
        </div>
        <div className="notice-box">
          修改会同步到本机商城列表，已创建订单的成交信息保持原样。此首版提供编辑现有演示商品。
        </div>
        <Button type="submit">保存商品</Button>
      </form>
    </Modal>
  );
}
export function CaseDetail({ item, state, act, close }) {
  const c = state.cases.find((x) => x.id === item.id);
  return (
    <Modal title="申诉处理" close={close}>
      <Badge tone="amber">{c.status}</Badge>
      <h2 className="detail-heading">{c.title}</h2>
      <p className="description">{c.reason}</p>
      <dl className="detail-list">
        <div>
          <dt>申请人</dt>
          <dd>{c.player}</dd>
        </div>
        <div>
          <dt>交易对方</dt>
          <dd>{c.against}</dd>
        </div>
        <div>
          <dt>涉及金额</dt>
          <dd>{money(c.amount)} 信用点</dd>
        </div>
      </dl>
      <h3>处理时间线</h3>
      <ol className="timeline">
        {c.history.map((t, i) => (
          <li key={i}>{t}</li>
        ))}
      </ol>
      <form
        className="form-stack"
        onSubmit={(e) => {
          e.preventDefault();
          if (
            act(
              {
                type: "CASE",
                id: c.id,
                text: new FormData(e.currentTarget).get("text"),
              },
              "处理意见已记录",
            )
          )
            close();
        }}
      >
        <Field label="处理说明">
          <textarea
            name="text"
            required
            minLength={2}
            rows={3}
            placeholder="记录核对结果、需要补充的证据或处理建议。"
          />
        </Field>
        <p className="notice-box">
          本版仅记录意见，不执行资金裁决。真实退款与释放担保将由后端处理。
        </p>
        <Button disabled={c.status !== "待处理"} type="submit">
          保存处理意见
        </Button>
      </form>
    </Modal>
  );
}
