import React, { useState } from "react";
import {
  ArrowUpRight,
  ArrowDownLeft,
  WalletCards,
  History,
  LockKeyhole,
  ArrowRight,
  Package,
  Heart,
  Settings,
  LogOut,
  Shield,
  Monitor,
  Sun,
  Moon,
  PenLine,
  Bell,
  Check,
} from "lucide-react";
import { players } from "./data.js";
import { money, myOrders, dateKey } from "./domain.js";
import {
  PageHead,
  SectionHead,
  Button,
  Avatar,
  Badge,
  Tabs,
  Empty,
  Modal,
  Field,
  Price,
  Toggle,
  media,
} from "./components.jsx";
export function Wallet({ state, open }) {
  const [filter, setFilter] = useState("全部"),
    [from, setFrom] = useState(""),
    [to, setTo] = useState("");
  const records = state.records
    .filter((r) => r.user === state.user)
    .filter(
      (r) =>
        filter === "全部" ||
        (filter === "收入"
          ? r.amount > 0
          : filter === "支出"
            ? r.amount < 0
            : r.kind === "冻结"),
    )
    .filter(
      (r) =>
        (!from || dateKey(r.time) >= from) && (!to || dateKey(r.time) <= to),
    );
  const held =
    state.orders
      .filter((o) => o.buyer === state.user && o.channel === "玩家市场")
      .reduce((n, o) => n + o.amount, 0) +
    state.commissions
      .filter(
        (c) =>
          c.owner === state.user &&
          ["OPEN", "ACTIVE", "COMPLETED"].includes(c.status),
      )
      .reduce((n, c) => n + c.price, 0);
  const today = dateKey(new Date()),
    todays = state.records.filter(
      (r) => r.user === state.user && dateKey(r.time) === today,
    );
  return (
    <>
      <PageHead
        eyebrow="YOUR WALLET"
        title={
          <>
            每一份积累，<span className="muted-heading">都有迹可循。</span>
          </>
        }
        subtitle="信用点、担保和每一笔往来，在这里清楚掌握。"
      />
      <div className="wallet-grid">
        <div className="balance-card">
          <div className="balance-top">
            <span>
              <WalletCards size={20} /> 可用信用点
            </span>
            <span className="wallet-mark">D</span>
          </div>
          <div className="balance-number">
            {money(state.balances[state.user])}
          </div>
          <div className="balance-bottom">
            <span>
              DEUTERIUM CREDIT <small>本机体验余额</small>
            </span>
            <Button onClick={() => open({ type: "transfer" })}>
              转账 <ArrowUpRight size={17} />
            </Button>
          </div>
        </div>
        <div className="wallet-stat">
          <span className="stat-icon sage">
            <ArrowDownLeft size={22} />
          </span>
          <span>今日收入</span>
          <strong>
            +
            {money(
              todays
                .filter((r) => r.amount > 0)
                .reduce((n, r) => n + r.amount, 0),
            )}
          </strong>
          <small>信用点</small>
        </div>
        <div className="wallet-stat">
          <span className="stat-icon blue">
            <LockKeyhole size={22} />
          </span>
          <span>担保中的信用点</span>
          <strong>{money(held)}</strong>
          <small>市场与委托预付</small>
        </div>
      </div>
      <SectionHead title="账单明细" description="每一笔往来，都记在这里。" />
      <div className="filter-header">
        <Tabs
          values={["全部", "收入", "支出", "冻结"]}
          value={filter}
          onChange={setFilter}
        />
        <div className="date-range">
          <input
            aria-label="账单起始日期"
            type="date"
            value={from}
            onChange={(e) => setFrom(e.target.value)}
          />
          <span>至</span>
          <input
            aria-label="账单截止日期"
            type="date"
            value={to}
            onChange={(e) => setTo(e.target.value)}
          />
        </div>
      </div>
      <div className="panel">
        {records.length ? (
          records.map((r) => (
            <div className="record-row" key={r.id}>
              <span className={`stat-icon ${r.amount > 0 ? "sage" : "blue"}`}>
                {r.amount > 0 ? (
                  <ArrowDownLeft size={20} />
                ) : (
                  <ArrowUpRight size={20} />
                )}
              </span>
              <div>
                <strong>{r.title}</strong>
                <small>
                  {new Date(r.time).toLocaleString("zh-CN")} · {r.kind}
                </small>
              </div>
              <strong className={r.amount > 0 ? "positive" : ""}>
                {r.amount > 0 ? "+" : ""}
                {money(r.amount)}
              </strong>
            </div>
          ))
        ) : (
          <Empty
            title="还没有相关账单"
            text="体验一次购买、委托预付或转账，这里就会记录下来。"
          >
            <Button secondary onClick={() => open({ type: "transfer" })}>
              体验转账 <ArrowUpRight size={16} />
            </Button>
          </Empty>
        )}
      </div>
    </>
  );
}
export function Transfer({ state, open, close }) {
  return (
    <Modal title="信用点转账" close={close}>
      <form
        className="form-stack"
        onSubmit={(e) => {
          e.preventDefault();
          const f = Object.fromEntries(new FormData(e.currentTarget));
          open({
            type: "confirm",
            title: "确认转账",
            description: `转给 ${players.find((p) => p.id === f.to).name}${f.note ? ` · ${f.note}` : ""}`,
            amount: Number(f.amount) * 100,
            action: { type: "TRANSFER", ...f },
            success: "演示转账已完成",
          });
        }}
      >
        <div className="notice-box">
          可用信用点 {money(state.balances[state.user])}
        </div>
        <Field label="收款人">
          <select name="to" required>
            {players
              .filter((p) => p.id !== state.user)
              .map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name} · {p.qq}
                </option>
              ))}
          </select>
        </Field>
        <Field
          label="转账金额"
          name="amount"
          placeholder="0.00"
          type="number"
          min="0.01"
          step="0.01"
          max="9999999.99"
          required
        />
        <Field
          label="备注（选填）"
          name="note"
          maxLength={100}
          placeholder="给对方留句话"
        />
        <Button className="full" type="submit">
          下一步 · 核对转账
        </Button>
      </form>
    </Modal>
  );
}
export function Me({ state, act, open, navigate }) {
  const p = players.find((p) => p.id === state.user),
    profile = state.profile[state.user] || {},
    orders = myOrders(state);
  const [tab, setTab] = useState("全部订单");
  const list = orders.filter((o) => tab === "全部订单" || o.channel === tab);
  return (
    <>
      <PageHead
        eyebrow="YOUR SPACE"
        title={
          <>
            你好，<span className="muted-heading">{p.name}。</span>
          </>
        }
        subtitle="把喜欢的留下，把生活过成自己的样子。"
      />
      <div className="profile-card">
        <div className="profile-identity">
          {profile.avatar ? (
            <img
              className="profile-avatar"
              src={profile.avatar}
              alt="我的头像"
            />
          ) : (
            <Avatar user={p.id} size="large" />
          )}
          <div>
            <h2>
              {p.name}{" "}
              <Badge tone="neutral">
                {p.admin ? "体验管理员" : "演示玩家"}
              </Badge>
            </h2>
            <p>{profile.bio ?? p.bio}</p>
            <small>
              游戏 ID · {p.name}　/　QQ · {p.qq}
            </small>
          </div>
        </div>
        <Button secondary onClick={() => open({ type: "profile-edit" })}>
          <PenLine size={16} />
          编辑资料
        </Button>
      </div>
      <div className="quick-grid">
        <button onClick={() => navigate("/wallet")}>
          <WalletCards />
          <span>
            我的钱包<small>{money(state.balances[p.id])} 信用点</small>
          </span>
          <ArrowUpRight size={18} />
        </button>
        <button onClick={() => open({ type: "favorites" })}>
          <Heart />
          <span>
            我的收藏
            <small>
              {
                state.favorites.filter((k) => k.startsWith(`${state.user}:`))
                  .length
              }{" "}
              件好物
            </small>
          </span>
          <ArrowUpRight size={18} />
        </button>
        <button onClick={() => navigate("/commissions")}>
          <Package />
          <span>
            我的委托<small>查看发布与接取</small>
          </span>
          <ArrowUpRight size={18} />
        </button>
      </div>
      <SectionHead
        title="我的订单"
        description="从每一次心动，到每一份收到。"
      />
      <Tabs
        values={["全部订单", "官方商城", "玩家市场"]}
        value={tab}
        onChange={setTab}
      />
      <div className="panel">
        {list.length ? (
          list.map((o) => (
            <button
              className="order-row"
              key={o.id}
              onClick={() => open({ type: "order", item: o })}
            >
              <span className="stat-icon blue">
                <Package size={22} />
              </span>
              <div>
                <strong>{o.title}</strong>
                <small>
                  {o.channel} · {o.buyer === p.id ? "我买入的" : "我卖出的"}
                </small>
              </div>
              <div className="order-end">
                <Badge tone="neutral">{o.status}</Badge>
                <Price small value={o.amount} />
              </div>
              <ArrowRight size={17} />
            </button>
          ))
        ) : (
          <Empty
            title="还没有订单"
            text="你的商城与市场交易，将一起保存在这里。"
          />
        )}
      </div>
      <SectionHead title="偏好与设置" />
      <div className="settings-grid">
        <div className="panel settings-panel">
          <h3>外观</h3>
          <p>同一个世界，也有不同光线。</p>
          <div className="theme-options">
            {[
              ["system", "系统", Monitor],
              ["light", "浅色", Sun],
              ["dark", "深色", Moon],
            ].map(([v, l, Icon]) => (
              <button
                key={v}
                className={state.theme === v ? "selected" : ""}
                onClick={() => act({ type: "THEME", value: v })}
              >
                <Icon size={20} />
                {l}
              </button>
            ))}
          </div>
          <Toggle
            label="灵动视效"
            description="页面切换与轻量反馈"
            checked={state.motion}
            onChange={(value) => act({ type: "MOTION", value })}
          />
          <Toggle
            label="柔光玻璃"
            description="导航与浮层的模糊材质"
            checked={state.glass}
            onChange={(value) => act({ type: "GLASS", value })}
          />
        </div>
        <div className="panel settings-panel">
          <h3>通知偏好</h3>
          <p>重要的事，按你喜欢的方式提醒。</p>
          <Toggle
            label="交易与委托"
            description="订单、退款和履约进展"
            checked={state.preferences.trade}
            onChange={(value) =>
              act({ type: "PREFERENCES", name: "trade", value })
            }
          />
          <Toggle
            label="提及我的消息"
            checked={state.preferences.mentions}
            onChange={(value) =>
              act({ type: "PREFERENCES", name: "mentions", value })
            }
          />
          <Toggle
            label="特别关心"
            checked={state.preferences.following}
            onChange={(value) =>
              act({ type: "PREFERENCES", name: "following", value })
            }
          />
          <small className="muted">当前保存偏好，系统推送待服务端接入。</small>
        </div>
      </div>
      <div className="account-footer">
        <Button secondary onClick={() => open({ type: "about" })}>
          关于 Deuterium
        </Button>
        <Button secondary onClick={() => act({ type: "LOGOUT" })}>
          <LogOut size={16} />
          退出演示账号
        </Button>
      </div>
    </>
  );
}
export function ProfileEdit({ state, act, close }) {
  const p = players.find((p) => p.id === state.user),
    profile = state.profile[p.id] || {},
    [avatar, setAvatar] = useState(profile.avatar),
    [error, setError] = useState("");
  const upload = (e) => {
    const f = e.target.files[0];
    if (!f) return;
    if (
      !["image/jpeg", "image/png", "image/webp"].includes(f.type) ||
      f.size > 1024 * 1024
    ) {
      setError("请选择 1 MiB 以内的图片");
      return;
    }
    const r = new FileReader();
    r.onload = () => {
      setAvatar(r.result);
      setError("");
    };
    r.readAsDataURL(f);
  };
  return (
    <Modal title="编辑资料" close={close}>
      <form
        className="form-stack"
        onSubmit={(e) => {
          e.preventDefault();
          if (
            act(
              {
                type: "PROFILE",
                bio: new FormData(e.currentTarget).get("bio"),
                avatar,
              },
              "资料已保存",
            )
          )
            close();
        }}
      >
        <div className="avatar-edit">
          {avatar ? (
            <img src={avatar} alt="头像圆形预览" />
          ) : (
            <Avatar user={p.id} size="large" />
          )}
          <label className="button secondary">
            选择头像
            <input
              className="sr-only"
              aria-label="选择头像"
              type="file"
              accept="image/jpeg,image/png,image/webp"
              onChange={upload}
            />
          </label>
        </div>
        {error && <p className="form-error">{error}</p>}
        <Field label="个人简介">
          <textarea
            name="bio"
            rows={4}
            maxLength={200}
            defaultValue={profile.bio ?? p.bio}
          />
        </Field>
        <Button type="submit">保存资料</Button>
      </form>
    </Modal>
  );
}
export function OrderDetail({ item, close }) {
  return (
    <Modal title="订单详情" close={close}>
      <Badge>{item.status}</Badge>
      <h2 className="detail-heading">{item.title}</h2>
      <Price value={item.amount} />
      <dl className="detail-list">
        <div>
          <dt>订单编号</dt>
          <dd className="mono">{item.id}</dd>
        </div>
        <div>
          <dt>订单来源</dt>
          <dd>{item.channel}</dd>
        </div>
        <div>
          <dt>创建时间</dt>
          <dd>{new Date(item.time).toLocaleString("zh-CN")}</dd>
        </div>
        <div>
          <dt>交付方式</dt>
          <dd>
            {item.channel === "官方商城" ? "游戏内邮箱" : item.snapshot?.place}
          </dd>
        </div>
      </dl>
      <div className="notice-box">
        这是已保存的本机成交快照。真实发货、领取、退款和自动结算需接入后端，本版不模拟服务器回执。
      </div>
    </Modal>
  );
}
