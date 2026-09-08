import React, { useState } from "react";
import {
  Plus,
  Heart,
  MapPin,
  Clock3,
  ArrowUpRight,
  ShieldCheck,
  Upload,
  Check,
} from "lucide-react";
import { categories, players } from "./data.js";
import { visibleListings, visibleCommissions } from "./domain.js";
import {
  PageHead,
  Tabs,
  SectionHead,
  Art,
  Price,
  Avatar,
  Badge,
  Button,
  Modal,
  Field,
  Empty,
  media,
} from "./components.jsx";

export function Market({ state, act, open, query }) {
  const [category, setCategory] = useState("全部"),
    [mine, setMine] = useState(false),
    [sort, setSort] = useState("最新发布");
  let list = (
    mine
      ? state.listings.filter((p) => p.owner === state.user)
      : visibleListings(state)
  ).filter(
    (p) =>
      (category === "全部" || p.category === category) &&
      `${p.title}${p.description}`.includes(query),
  );
  if (sort === "价格从低到高")
    list = [...list].sort((a, b) => a.price - b.price);
  return (
    <>
      <PageHead
        eyebrow="THE COMMUNITY MARKET"
        title={
          <>
            你的闲置，<span className="muted-heading">别人的心动。</span>
          </>
        }
        subtitle="交换好物，分享创造。在这里遇见同样热爱这个世界的人。"
      >
        <Button onClick={() => open({ type: "publish" })}>
          <Plus size={17} />
          发布商品
        </Button>
      </PageHead>
      <div className="market-intro">
        <div>
          <span className="tiny-label">来自我们的世界</span>
          <h2>
            每一份创造，
            <br />
            都值得被看见。
          </h2>
          <p>
            建筑、装备、日常补给。
            <br />
            你的下一个发现，或许就在这里。
          </p>
          <span>
            <ShieldCheck size={16} /> 平台担保，安心交易
          </span>
        </div>
        <div className="market-scene">
          <img
            src={media("street-daylight.png")}
            alt="Deuterium 服务器街道与玩家实景"
          />
        </div>
      </div>
      <div className="filter-header">
        <Tabs
          values={["发现好物", "我的发布"]}
          value={mine ? "我的发布" : "发现好物"}
          onChange={(v) => setMine(v === "我的发布")}
        />
        <label className="sort-select">
          <span className="sr-only">商品排序</span>
          <select value={sort} onChange={(e) => setSort(e.target.value)}>
            <option>最新发布</option>
            <option>价格从低到高</option>
          </select>
        </label>
      </div>
      <Tabs
        values={categories}
        value={category}
        onChange={(v) => setCategory(v === category ? "全部" : v)}
      />
      <div className="result-count">
        {list.length} 件{mine ? "我的商品" : "在售好物"}
      </div>
      {list.length ? (
        <div className="market-grid">
          {list.map((p) => (
            <article className="market-card" key={p.id}>
              <div className="market-card-photo">
                <button
                  aria-label={`查看 ${p.title}`}
                  onClick={() => open({ type: "listing", item: p })}
                >
                  <Art kind={p.art} image={p.image} />
                </button>
                <button
                  className={`favorite ${state.favorites.includes(`${state.user}:${p.id}`) ? "selected" : ""}`}
                  aria-label={`${state.favorites.includes(`${state.user}:${p.id}`) ? "取消收藏" : "收藏"} ${p.title}`}
                  onClick={() => act({ type: "FAVORITE", id: p.id })}
                >
                  <Heart
                    size={17}
                    fill={
                      state.favorites.includes(`${state.user}:${p.id}`)
                        ? "currentColor"
                        : "none"
                    }
                  />
                </button>
                <Badge tone="neutral">
                  {!p.active ? "已下架" : p.stock ? p.category : "已售罄"}
                </Badge>
              </div>
              <div className="market-card-copy">
                <button
                  className="plain-title"
                  onClick={() => open({ type: "listing", item: p })}
                >
                  <h3>{p.title}</h3>
                </button>
                <div className="seller-row">
                  <Avatar user={p.owner} />
                  <span>{players.find((x) => x.id === p.owner)?.name}</span>
                  <span className="muted">· 剩余 {p.stock} 份</span>
                </div>
                <div className="price-row">
                  <Price value={p.price} />
                  <button
                    className="round-link"
                    aria-label={`打开 ${p.title}`}
                    onClick={() => open({ type: "listing", item: p })}
                  >
                    <ArrowUpRight size={18} />
                  </button>
                </div>
              </div>
            </article>
          ))}
        </div>
      ) : (
        <Empty
          title={mine ? "还没有发布商品" : "暂时没有符合条件的商品"}
          text="换个分类看看，或分享你的第一件好物。"
        />
      )}
    </>
  );
}
export function ListingDetail({ item, state, act, open, close }) {
  const p = state.listings.find((x) => x.id === item.id),
    own = p.owner === state.user;
  return (
    <Modal title="玩家商品" close={close} wide>
      <div className="product-detail">
        <Art kind={p.art} image={p.image} />
        <div>
          <Badge>{p.category}</Badge>
          <h2>{p.title}</h2>
          <div className="seller-row">
            <Avatar user={p.owner} />
            <span>{players.find((x) => x.id === p.owner)?.name}</span>
          </div>
          <p className="description">{p.description}</p>
          <Price value={p.price} />
          <dl className="detail-list">
            <div>
              <dt>交付地点</dt>
              <dd>{p.place}</dd>
            </div>
            <div>
              <dt>剩余库存</dt>
              <dd>{p.stock} 份</dd>
            </div>
            {p.hours && (
              <div>
                <dt>总工期</dt>
                <dd>{p.hours} 小时，含验收预留</dd>
              </div>
            )}
            <div>
              <dt>付款方式</dt>
              <dd>平台担保，确认后结算</dd>
            </div>
          </dl>
          {own ? (
            <Button
              danger
              secondary
              disabled={!p.active}
              onClick={() =>
                open({
                  type: "confirm",
                  title: "确认下架商品",
                  description: "下架后将不再出现在玩家市场中，已有订单保留。",
                  action: { type: "UNLIST", id: p.id },
                  success: "商品已下架",
                  danger: true,
                })
              }
            >
              {p.active ? "下架商品" : "已下架"}
            </Button>
          ) : (
            <Button
              disabled={!p.stock || !p.active}
              onClick={() =>
                open({
                  type: "confirm",
                  title: "确认担保付款",
                  description: `购买 1 份「${p.title}」，请先确认交付地点和约定。`,
                  amount: p.price,
                  action: { type: "BUY", id: p.id },
                  success: "担保订单已创建",
                })
              }
            >
              担保购买
            </Button>
          )}
        </div>
      </div>
    </Modal>
  );
}
const statusText = {
  OPEN: "待接取",
  ACTIVE: "进行中",
  COMPLETED: "待验收",
  CONFIRMED: "已完成",
  CANCELLED: "已取消",
};
export function Commissions({ state, open, query }) {
  const [tab, setTab] = useState("发现委托"),
    [urgency, setUrgency] = useState("全部");
  const list = (
    tab === "发现委托"
      ? visibleCommissions(state)
      : state.commissions.filter((c) =>
          tab === "我发布的" ? c.owner === state.user : c.worker === state.user,
        )
  ).filter(
    (c) =>
      (urgency === "全部" || c.urgency === urgency) &&
      `${c.title}${c.description}`.includes(query),
  );
  return (
    <>
      <PageHead
        eyebrow="CREATE SOMETHING TOGETHER"
        title={
          <>
            一个想法，<span className="muted-heading">一起完成。</span>
          </>
        }
        subtitle="把计划交给擅长的人，让每一份热爱都有回响。"
      >
        <Button onClick={() => open({ type: "publish-commission" })}>
          <Plus size={17} />
          发布委托
        </Button>
      </PageHead>
      <div className="commission-banner">
        <img src={media("plaza-sunset.png")} alt="Deuterium 广场实景" />
        <div>
          <span className="tiny-label">小小的委托，大大的可能</span>
          <h2>
            找一位搭档，
            <br />
            创造下一处风景。
          </h2>
          <p>发布时预付，完成后结算。</p>
        </div>
      </div>
      <div className="filter-header">
        <Tabs
          values={["发现委托", "我发布的", "我接取的"]}
          value={tab}
          onChange={setTab}
        />
        <select
          aria-label="紧急程度筛选"
          value={urgency}
          onChange={(e) => setUrgency(e.target.value)}
        >
          <option>全部</option>
          <option>普通</option>
          <option>较急</option>
          <option>紧急</option>
        </select>
      </div>
      {list.length ? (
        <div className="commission-grid">
          {list.map((c) => (
            <article key={c.id} className="commission-card">
              <div className="commission-card-top">
                <Badge tone={c.urgency === "普通" ? "sage" : "amber"}>
                  {c.urgency === "普通" ? "从容进行" : c.urgency}
                </Badge>
                <Badge tone="neutral">{statusText[c.status]}</Badge>
              </div>
              <button
                className="plain-title"
                onClick={() => open({ type: "commission", item: c })}
              >
                <h3>{c.title}</h3>
              </button>
              <p className="commission-desc">{c.description}</p>
              <div className="commission-meta">
                <span>
                  <MapPin size={15} />
                  {c.location}
                </span>
                <span>
                  <Clock3 size={15} />
                  {c.hours} 小时履约
                </span>
              </div>
              <div className="commission-reward">
                <div>
                  <span className="tiny-label">预付报酬</span>
                  <Price value={c.price} />
                </div>
                <button
                  className="round-link"
                  aria-label={`查看 ${c.title}`}
                  onClick={() => open({ type: "commission", item: c })}
                >
                  <ArrowUpRight size={19} />
                </button>
              </div>
              <div className="commission-owner">
                <Avatar user={c.owner} />
                <span>{players.find((p) => p.id === c.owner)?.name} 发布</span>
                <span className="status-dot" />
                预付演示
              </div>
            </article>
          ))}
        </div>
      ) : (
        <Empty
          title="还没有符合条件的委托"
          text={
            tab === "发现委托"
              ? "发布你的计划，让朋友们来帮忙。"
              : "接取或发布后，可以在这里跟进。"
          }
        />
      )}
    </>
  );
}
export function CommissionDetail({ item, state, open, close }) {
  const c = state.commissions.find((x) => x.id === item.id),
    own = c.owner === state.user,
    worker = c.worker === state.user;
  const action =
    c.status === "OPEN"
      ? own
        ? { type: "COMMISSION_CANCEL", id: c.id }
        : { type: "ACCEPT", id: c.id }
      : c.status === "ACTIVE" && worker
        ? { type: "COMMISSION_COMPLETE", id: c.id }
        : c.status === "COMPLETED" && own
          ? { type: "COMMISSION_CONFIRM", id: c.id }
          : null;
  const label =
    c.status === "OPEN"
      ? own
        ? "取消并退回预付"
        : "接取这份委托"
      : c.status === "ACTIVE"
        ? "提交完成"
        : "确认验收";
  return (
    <Modal title="委托详情" close={close}>
      <Badge>{statusText[c.status]}</Badge>
      <h2 className="detail-heading">{c.title}</h2>
      <p className="description">{c.description}</p>
      <dl className="detail-list">
        <div>
          <dt>发布者</dt>
          <dd>{players.find((p) => p.id === c.owner)?.name}</dd>
        </div>
        <div>
          <dt>履约地点</dt>
          <dd>{c.location}</dd>
        </div>
        <div>
          <dt>履约期限</dt>
          <dd>接取后 {c.hours} 小时</dd>
        </div>
        <div>
          <dt>验收期限</dt>
          <dd>提交完成后 72 小时</dd>
        </div>
      </dl>
      <Price value={c.price} />
      <p className="notice-box">
        体验版可手动走通接取与验收。自动计时结算、逾期和退款争议待服务端接入。
      </p>
      {action && (
        <Button
          className="full"
          onClick={() =>
            open({
              type: "confirm",
              title: label,
              description:
                c.status === "OPEN" && !own
                  ? "接取后将从公开大厅移除，仅双方可在我的委托中查看。"
                  : "请确认当前履约情况。本次只更改本机演示记录。",
              action,
              success: "委托状态已更新",
            })
          }
        >
          {label}
        </Button>
      )}
    </Modal>
  );
}
export function PublishForm({ commission = false, act, close, open }) {
  const [image, setImage] = useState(""),
    [category, setCategory] = useState("建材"),
    [error, setError] = useState("");
  const upload = (e) => {
    const f = e.target.files[0];
    if (!f) return;
    if (
      !["image/jpeg", "image/png", "image/webp"].includes(f.type) ||
      f.size > 2 * 1024 * 1024
    ) {
      setError("体验版支持 2 MiB 以内的 JPEG、PNG 或 WebP 封面");
      return;
    }
    const r = new FileReader();
    r.onload = () => {
      setImage(r.result);
      setError("");
    };
    r.readAsDataURL(f);
  };
  const submit = (e) => {
    e.preventDefault();
    const f = Object.fromEntries(new FormData(e.currentTarget));
    const data = {
      ...f,
      stock: Number(f.stock),
      hours: Number(f.hours || 72),
      art: commission || category === "建筑服务" ? "village" : "flowers",
      image: image || media("flower-shop.png"),
    };
    if (commission)
      open({
        type: "confirm",
        title: "预付并发布委托",
        description: `发布「${f.title}」，报酬将先进入演示担保。`,
        amount: Number(f.price) * 100,
        action: { type: "COMMISSION_PUBLISH", data },
        success: "委托已发布",
      });
    else if (act({ type: "PUBLISH", data }, "商品已发布")) close();
  };
  return (
    <Modal title={commission ? "发布委托" : "发布商品"} close={close}>
      <form onSubmit={submit} className="form-stack">
        <label className="cover-upload">
          <img src={image || media("flower-shop.png")} alt="发布封面预览" />
          <span>
            <Upload size={19} />
            {image ? "更换封面" : "使用现成示例封面，或上传图片"}
            <small>JPEG / PNG / WebP · 2 MiB 以内</small>
          </span>
          <input
            aria-label="上传封面"
            type="file"
            accept="image/jpeg,image/png,image/webp"
            onChange={upload}
          />
        </label>
        {error && <p className="form-error">{error}</p>}
        <Field
          label="标题"
          name="title"
          placeholder={
            commission ? "想邀请大家一起完成什么？" : "给你的好物起个名字"
          }
          required
          maxLength={60}
        />
        <Field label="详细说明">
          <textarea
            name="description"
            required
            minLength={5}
            maxLength={2000}
            rows={3}
            placeholder="描述内容、交付要求，以及需要提前说明的事项。"
          />
        </Field>
        {!commission && (
          <Field label="分类">
            <select
              name="category"
              value={category}
              onChange={(e) => setCategory(e.target.value)}
            >
              {categories.slice(1).map((c) => (
                <option key={c}>{c}</option>
              ))}
            </select>
          </Field>
        )}
        <div className="form-grid">
          <Field
            label={commission ? "预付报酬 / 信用点" : "单价 / 信用点"}
            name="price"
            type="number"
            step="0.01"
            min="0.01"
            max="9999999.99"
            placeholder="0.00"
            required
          />
          {commission ? (
            <Field
              label="履约期限 / 小时"
              name="hours"
              type="number"
              min="1"
              max="8760"
              defaultValue="72"
              required
            />
          ) : (
            <Field
              label="可售库存"
              name="stock"
              type="number"
              min="1"
              max="999"
              defaultValue="1"
              required
            />
          )}
        </div>
        {!commission && category === "建筑服务" && (
          <Field
            label="总工期 / 小时（含验收预留）"
            name="hours"
            type="number"
            min="1"
            max="8760"
            defaultValue="168"
            required
          />
        )}
        <Field
          label="交付地点"
          name={commission ? "location" : "place"}
          required
          placeholder="例如：主城东门，或详细坐标"
          maxLength={120}
        />
        {commission && (
          <Field label="紧急程度">
            <select name="urgency">
              <option>普通</option>
              <option>较急</option>
              <option>紧急</option>
            </select>
          </Field>
        )}
        <p className="notice-box">
          发布内容保存在当前浏览器，仅用于体验。图片是展示素材，不代表实际出售或委托的建筑。
        </p>
        <Button className="full" type="submit">
          {commission ? "下一步 · 确认预付" : "发布商品"}
        </Button>
      </form>
    </Modal>
  );
}
