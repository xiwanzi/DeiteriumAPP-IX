import {
  products,
  players,
  seedListings,
  seedCommissions,
  initialMessages,
} from "./data.js";
export const STORAGE_KEY = "deuterium-web-demo-v1";
export const id = () =>
  globalThis.crypto?.randomUUID?.() ??
  `demo-${Date.now()}-${Math.random().toString(36).slice(2)}`;
export const money = (cents) =>
  (cents / 100).toLocaleString("zh-CN", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
export function dateKey(value) {
  const d = new Date(value);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}
export function amount(value) {
  if (!/^\d{1,7}(\.\d{1,2})?$/.test(String(value)))
    throw new Error("请输入有效金额，最多两位小数");
  const [whole, fraction = ""] = String(value).split(".");
  const cents = Number(whole) * 100 + Number(fraction.padEnd(2, "0"));
  if (cents <= 0 || cents > 999999999)
    throw new Error("金额应在 0.01 至 9,999,999.99 之间");
  return cents;
}
export function initialState() {
  return {
    version: 1,
    user: "mori",
    profile: {},
    balances: Object.fromEntries(players.map((p) => [p.id, 2888600])),
    products: structuredClone(products),
    listings: structuredClone(seedListings),
    commissions: structuredClone(seedCommissions),
    cart: {},
    favorites: [],
    orders: [],
    records: [],
    messages: structuredClone(initialMessages),
    bans: [],
    notices: [],
    audit: [],
    cases: [
      {
        id: "case-001",
        title: "建筑交付与约定不一致",
        player: "aster",
        against: "luna",
        amount: 180000,
        status: "待处理",
        reason: "庭院与成交时确认的布局存在差异，希望平台协助核对。",
        history: ["买方提交申请与交付说明", "系统保存成交条款快照"],
      },
    ],
    processed: [],
    transfers: [],
    announcements: [],
    chatDrafts: {},
    readPositions: {},
    preferences: { trade: true, mentions: true, following: false },
    theme: "system",
    motion: true,
    glass: true,
  };
}
const fail = (message) => {
  throw new Error(message);
};
export const isBanned = (s, user) =>
  s.bans.some(
    (b) =>
      b.player === user && !b.revoked && (!b.until || b.until > Date.now()),
  );
export const visibleListings = (s) =>
  s.listings.filter((p) => p.active && p.stock > 0 && !isBanned(s, p.owner));
export const visibleCommissions = (s) =>
  s.commissions.filter((c) => c.status === "OPEN" && !isBanned(s, c.owner));
export const myOrders = (s) =>
  s.orders.filter((o) => o.buyer === s.user || o.seller === s.user);
export function transition(state, action) {
  const scope = action.key
    ? `${state.user}:${action.type}:${action.key}`
    : null;
  const payload = JSON.stringify(action);
  const previous = scope && state.processed.find((p) => p.scope === scope);
  if (previous) {
    if (previous.payload !== payload) fail("幂等键已用于不同的操作内容");
    return state;
  }
  const s = structuredClone(state),
    u = s.user;
  const now = new Date().toISOString();
  const notice = (text) =>
    s.notices.unshift({ id: id(), user: u, text, time: now, read: false });
  const requireLogin = () => {
    if (!u || isBanned(s, u)) fail("当前演示账号不可用，请重新选择账号");
  };
  const admin = () => {
    requireLogin();
    if (!players.find((p) => p.id === u)?.admin) fail("需要管理员权限");
  };
  const spend = (cents) => {
    if (!Number.isSafeInteger(cents) || cents <= 0) fail("金额无效");
    if (s.balances[u] < cents) fail("可用信用点不足");
    s.balances[u] -= cents;
  };
  const record = (title, cents, kind = "支出") =>
    s.records.unshift({
      id: id(),
      user: u,
      title,
      amount: cents,
      kind,
      time: now,
    });
  if (!["LOGIN", "LOGOUT", "THEME", "MOTION", "GLASS"].includes(action.type))
    requireLogin();
  switch (action.type) {
    case "LOGIN":
      if (!players.some((p) => p.id === action.user)) fail("请选择演示账号");
      if (isBanned(s, action.user))
        fail("该演示账号已封禁，暂时无法登录。请联系管理员处理。");
      s.user = action.user;
      break;
    case "LOGOUT":
      s.user = null;
      break;
    case "THEME":
      s.theme = action.value;
      break;
    case "MOTION":
      s.motion = action.value;
      break;
    case "GLASS":
      s.glass = action.value;
      break;
    case "PROFILE":
      s.profile[u] = {
        bio: action.bio.slice(0, 200),
        avatar: action.avatar ?? s.profile[u]?.avatar,
      };
      break;
    case "PREFERENCES":
      s.preferences[action.name] = action.value;
      break;
    case "FAVORITE": {
      const favorite = `${u}:${action.id}`;
      s.favorites = s.favorites.includes(favorite)
        ? s.favorites.filter((x) => x !== favorite)
        : [...s.favorites, favorite];
      break;
    }
    case "CART": {
      const p = s.products.find((x) => x.id === action.id);
      if (!p || !p.stock) fail("商品暂不可购买");
      s.cart[u] ??= {};
      const qty = (s.cart[u][p.id] || 0) + action.delta;
      if (qty > p.stock) fail("库存不足");
      if (qty <= 0) delete s.cart[u][p.id];
      else s.cart[u][p.id] = qty;
      break;
    }
    case "CHECKOUT": {
      const cart = s.cart[u] || {},
        entries = Object.entries(cart);
      if (!entries.length) fail("购物袋还是空的");
      const lines = entries.map(([pid, qty]) => {
        const p = s.products.find((x) => x.id === pid);
        if (!p || p.stock < qty) fail("商品库存已变化");
        return { ...p, qty };
      });
      const total = lines.reduce((n, p) => n + p.price * p.qty, 0);
      spend(total);
      for (const p of lines)
        s.products.find((x) => x.id === p.id).stock -= p.qty;
      s.orders.unshift({
        id: id(),
        title: lines.map((p) => p.title).join("、"),
        buyer: u,
        seller: "official",
        amount: total,
        channel: "官方商城",
        status: "待领取",
        time: now,
        lines,
      });
      record("官方商城 · 订单付款", -total);
      s.cart[u] = {};
      notice("商城演示订单已创建，可在我的订单查看。");
      break;
    }
    case "BUY": {
      const p = s.listings.find((x) => x.id === action.id);
      if (!p || !p.active || p.stock < 1 || isBanned(s, p.owner))
        fail("商品已售罄或下架");
      if (p.owner === u) fail("不能购买自己发布的商品");
      spend(p.price);
      p.stock--;
      s.orders.unshift({
        id: id(),
        title: p.title,
        buyer: u,
        seller: p.owner,
        amount: p.price,
        channel: "玩家市场",
        status: p.category === "建筑服务" ? "待开工" : "待发货",
        time: now,
        snapshot: { ...p },
      });
      record(`担保付款 · ${p.title}`, -p.price, "冻结");
      notice("市场演示订单已创建，款项进入本机担保记录。");
      break;
    }
    case "PUBLISH": {
      const p = action.data;
      if (
        !p.title?.trim() ||
        !p.description?.trim() ||
        !p.place?.trim() ||
        !p.category
      )
        fail("请补全商品信息");
      if (!Number.isInteger(p.stock) || p.stock < 1 || p.stock > 999)
        fail("库存应为 1–999");
      if (
        p.category === "建筑服务" &&
        (!Number.isInteger(p.hours) || p.hours < 1 || p.hours > 8760)
      )
        fail("请填写有效总工期");
      const price = amount(p.price);
      s.listings.unshift({
        ...p,
        title: p.title.trim(),
        price,
        id: id(),
        owner: u,
        active: true,
      });
      notice("商品已发布到本机演示市场。");
      break;
    }
    case "UNLIST": {
      const p = s.listings.find((p) => p.id === action.id);
      if (p?.owner !== u) fail("无权修改商品");
      p.active = false;
      break;
    }
    case "COMMISSION_PUBLISH": {
      const c = action.data,
        price = amount(c.price);
      if (
        !c.title?.trim() ||
        !c.description?.trim() ||
        !c.location?.trim() ||
        !Number.isInteger(c.hours) ||
        c.hours < 1 ||
        c.hours > 8760
      )
        fail("请补全委托信息和有效期限");
      spend(price);
      s.commissions.unshift({
        ...c,
        price,
        id: id(),
        owner: u,
        status: "OPEN",
      });
      record(`委托预付 · ${c.title}`, -price, "冻结");
      notice("委托已预付并发布到本机演示大厅。");
      break;
    }
    case "ACCEPT": {
      const c = s.commissions.find((c) => c.id === action.id);
      if (!c || c.status !== "OPEN" || isBanned(s, c.owner))
        fail("委托已被接取或已取消");
      if (c.owner === u) fail("不能接取自己的委托");
      c.worker = u;
      c.status = "ACTIVE";
      c.acceptedAt = now;
      notice("接取成功。可在我的委托查看进度。");
      break;
    }
    case "COMMISSION_COMPLETE": {
      const c = s.commissions.find((c) => c.id === action.id);
      if (c?.worker !== u || c.status !== "ACTIVE") fail("委托状态已变化");
      c.status = "COMPLETED";
      c.completedAt = now;
      notice("已提交完成，等待发布者验收。");
      break;
    }
    case "COMMISSION_CONFIRM": {
      const c = s.commissions.find((c) => c.id === action.id);
      if (c?.owner !== u || c.status !== "COMPLETED") fail("暂不可验收");
      c.status = "CONFIRMED";
      s.balances[c.worker] += c.price;
      record(`委托结算 · ${c.title}`, 0, "结算");
      break;
    }
    case "COMMISSION_CANCEL": {
      const c = s.commissions.find((c) => c.id === action.id);
      if (c?.owner !== u || c.status !== "OPEN")
        fail("仅待接取委托可以直接取消");
      c.status = "CANCELLED";
      s.balances[u] += c.price;
      record(`委托退款 · ${c.title}`, c.price, "退款");
      break;
    }
    case "TRANSFER": {
      if (
        !players.some((p) => p.id === action.to) ||
        action.to === u ||
        isBanned(s, action.to)
      )
        fail("请选择有效的其他收款人");
      const cents = amount(action.amount);
      spend(cents);
      s.balances[action.to] += cents;
      record(`转账给 ${players.find((p) => p.id === action.to).name}`, -cents);
      s.records.unshift({
        id: id(),
        user: action.to,
        title: `来自 ${players.find((p) => p.id === u).name} 的转账`,
        amount: cents,
        kind: "收入",
        time: now,
      });
      s.transfers ??= [];
      s.transfers.unshift({
        id: id(),
        from: u,
        to: action.to,
        amount: cents,
        note: action.note || "",
        time: now,
      });
      notice("演示转账已完成。");
      break;
    }
    case "MESSAGE":
      if (!action.text.trim() || action.text.length > 2000)
        fail("消息应为 1–2000 字");
      if (
        action.channel.startsWith("dm:") &&
        !action.channel.split(":").slice(1).includes(u)
      )
        fail("无权发送到该会话");
      s.messages.push({
        id: id(),
        channel: action.channel,
        sender: u,
        text: action.text.trim(),
        reply: action.reply || null,
        time: new Date().toLocaleTimeString("zh-CN", {
          hour: "2-digit",
          minute: "2-digit",
        }),
      });
      if (action.channel === `ai:${u}`)
        s.messages.push({
          id: id(),
          channel: action.channel,
          sender: "assistant",
          text: "这是一条本机示例回复：你可以在玩家市场寻找材料，在委托大厅邀请朋友帮忙，完成后从我的订单跟进。真实 AI 回答将在后端接入后提供。",
          time: "刚刚",
        });
      break;
    case "READ_NOTICES":
      s.notices.filter((n) => n.user === u).forEach((n) => (n.read = true));
      break;
    case "CHAT_DRAFT":
      s.chatDrafts ??= {};
      s.chatDrafts[`${u}:${action.channel}`] = action.text.slice(0, 2000);
      break;
    case "CHAT_READ":
      s.readPositions ??= {};
      s.readPositions[`${u}:${action.channel}`] = action.id;
      break;
    case "ANNOUNCEMENT_CREATE": {
      admin();
      const p = action.data;
      if (
        !p.title.trim() ||
        p.title.length > 100 ||
        !p.summary.trim() ||
        p.summary.length > 300 ||
        !p.body.trim() ||
        p.body.length > 12000
      )
        fail("请填写有效的标题、摘要和正文");
      s.announcements ??= [];
      s.announcements.unshift({
        ...p,
        id: id(),
        title: p.title.trim(),
        status: "DRAFT",
        version: 1,
        updatedAt: now,
        published: null,
      });
      s.audit.unshift({
        id: id(),
        title: `创建公告草稿 ${p.title}`,
        reason: "尚未公开",
        time: now,
      });
      break;
    }
    case "ANNOUNCEMENT_SAVE":
    case "ANNOUNCEMENT_PUBLISH":
    case "ANNOUNCEMENT_WITHDRAW": {
      admin();
      const a = s.announcements?.find((a) => a.id === action.id);
      if (!a) fail("公告不存在");
      if (a.version !== action.expectedVersion)
        fail("公告已被更新，请重新打开后操作");
      if (action.type === "ANNOUNCEMENT_SAVE") {
        const p = action.data;
        if (
          !p.title.trim() ||
          p.title.length > 100 ||
          !p.summary.trim() ||
          p.summary.length > 300 ||
          !p.body.trim() ||
          p.body.length > 12000
        )
          fail("请填写有效的标题、摘要和正文");
        Object.assign(a, p, { hasDraftChanges: Boolean(a.published) });
      } else if (action.type === "ANNOUNCEMENT_PUBLISH") {
        a.published = {
          id: a.id,
          title: a.title,
          summary: a.summary,
          intro: a.summary,
          body: a.body,
          cover: a.cover,
          pinned: a.pinned,
          priority: a.priority,
          publishedAt: now,
          date: dateKey(now),
        };
        a.status = "PUBLISHED";
        a.hasDraftChanges = false;
        a.publishedAt = now;
        for (const p of players)
          s.notices.unshift({
            id: id(),
            user: p.id,
            text: `新公告：${a.title}`,
            time: now,
            read: false,
          });
      } else {
        a.status = "WITHDRAWN";
        a.hasDraftChanges = false;
      }
      a.version++;
      a.updatedAt = now;
      s.audit.unshift({
        id: id(),
        title: `${action.type === "ANNOUNCEMENT_SAVE" ? "保存草稿" : action.type === "ANNOUNCEMENT_PUBLISH" ? "发布公告" : "撤下公告"} ${a.title}`,
        reason: "保留版本与操作记录",
        time: now,
      });
      break;
    }
    case "BAN": {
      admin();
      if (
        action.player === u ||
        players.find((p) => p.id === action.player)?.admin
      )
        fail("不可封禁自己或管理员");
      if (isBanned(s, action.player)) fail("该账号已处于封禁中");
      if (action.reason.trim().length < 2) fail("请填写至少两个字的原因");
      s.bans.unshift({
        id: id(),
        player: action.player,
        reason: action.reason.trim(),
        time: now,
        until: action.days ? Date.now() + action.days * 86400000 : null,
        admin: u,
      });
      s.audit.unshift({
        id: id(),
        title: `封禁 ${action.player}`,
        reason: action.reason,
        time: now,
      });
      break;
    }
    case "UNBAN": {
      admin();
      if (action.reason.trim().length < 2) fail("请填写解封原因");
      const b = s.bans.find((b) => b.id === action.id);
      if (!b || b.revoked) fail("封禁记录已变化");
      b.revoked = now;
      s.audit.unshift({
        id: id(),
        title: `解封 ${b.player}`,
        reason: action.reason,
        time: now,
      });
      break;
    }
    case "PRODUCT_CREATE": {
      admin();
      const p = action.data;
      if (!p.title.trim() || !p.subtitle.trim()) fail("请填写商品标题与简介");
      if (!Number.isInteger(p.stock) || p.stock < 1 || p.stock > 999)
        fail("库存应为 1–999");
      if (!products.some((x) => x.image === p.image)) fail("请选择现有素材");
      s.products.unshift({
        ...p,
        id: id(),
        price: amount(p.price),
        tone: "white",
      });
      s.audit.unshift({
        id: id(),
        title: `发布官方商品 ${p.title}`,
        reason: "创建并发布本机演示商品",
        time: now,
      });
      break;
    }
    case "PRODUCT_SAVE": {
      admin();
      const p = s.products.find((p) => p.id === action.id);
      if (!p) fail("商品不存在");
      if (!action.title.trim()) fail("请输入标题");
      p.title = action.title.trim();
      p.price = amount(action.price);
      if (
        !Number.isInteger(action.stock) ||
        action.stock < 0 ||
        action.stock > 999
      )
        fail("库存无效");
      p.stock = action.stock;
      s.audit.unshift({
        id: id(),
        title: `编辑商品 ${p.title}`,
        reason: "更新商品展示、价格及库存（历史订单不变）",
        time: now,
      });
      break;
    }
    case "CASE": {
      admin();
      const c = s.cases.find((c) => c.id === action.id);
      if (!c) fail("案件不存在");
      if (c.status === "已记录处理意见") fail("处理意见已保存");
      if (action.text.trim().length < 2) fail("请填写处理说明");
      c.status = "已记录处理意见";
      c.history.push(action.text.trim());
      s.audit.unshift({
        id: id(),
        title: `处理案件 ${c.id}`,
        reason: action.text,
        time: now,
      });
      break;
    }
    default:
      fail("未知操作");
  }
  if (scope) s.processed.push({ scope, payload });
  return s;
}
