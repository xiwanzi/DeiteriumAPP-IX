import {
  LayoutGrid,
  Store,
  Gavel,
  Megaphone,
  ChartNoAxesCombined,
  Mail,
  Server,
  Sparkles,
  Users,
  Smartphone,
  Ticket,
  Monitor,
  ShieldCheck,
  Boxes,
  ScanEye,
  Keyboard,
  Award,
  Gift,
  Layers,
} from "lucide-react";

const ADMIN_DOMAINS = [
  {
    id: "support_risk",
    label: "客服与风控",
    description: "日常玩家来信、入服审核、争议处理与内容安全",
    items: [
      {
        to: "/admin?section=support",
        section: "support",
        label: "工单系统",
        shortDesc: "处理玩家邮件工单与回复跟进",
        icon: Ticket,
        badgeKey: "supportOpen",
        keywords: ["ticket", "support", "help", "gd", "gongdan", "kefu", "工单", "客服", "邮件支持"],
        allow: (all, p) => all || p.includes("support.manage"),
      },
      {
        to: "/admin?section=whitelist",
        section: "whitelist",
        label: "白名单管理",
        shortDesc: "审批入服申请、增删通行权限",
        icon: ShieldCheck,
        badgeKey: "whitelistPending",
        keywords: ["whitelist", "admission", "bmd", "baimingdan", "rufu", "白名单", "入服", "通行", "审批"],
        allow: (all) => all,
      },
      {
        to: "/admin?section=interventions",
        section: "interventions",
        label: "平台介入",
        shortDesc: "裁决交易与委托争议、执行资金处理",
        icon: Gavel,
        badgeKey: "interventionsPending",
        keywords: ["intervention", "dispute", "jr", "ptjr", "shensu", "介入", "争议", "申诉", "退款"],
        allow: (all, p) => all || p.includes("intervention.manage"),
      },
      {
        to: "/admin?section=image-review",
        section: "image-review",
        label: "图片审核",
        shortDesc: "复核高风险聊天图片与调整识别阈值",
        icon: ScanEye,
        badgeKey: "imageFlagged",
        keywords: ["image", "review", "moderation", "tpsh", "tupian", "shenhe", "图片", "审核", "风控"],
        allow: (all) => all,
      },
    ],
  },
  {
    id: "commerce",
    label: "商城与运营",
    description: "官方商城目录、武器补给卡池、优惠与社区公告",
    items: [
      {
        to: "/merchant",
        section: "merchant",
        label: "商店管理",
        shortDesc: "维护官方商品、分类层级与交付模板",
        icon: Store,
        keywords: ["store", "shop", "catalog", "merchant", "sd", "shangdian", "shangpin", "商店", "商品", "商城"],
        allow: (all, p) =>
          all ||
          !p.includes("support.manage") ||
          p.some((x) => ["store.manage", "store.members.manage", "commerce.manage"].includes(x)),
      },
      {
        to: "/admin?section=supply",
        section: "supply",
        label: "补给管理",
        shortDesc: "配置卡池、素材、票券账本与抽卡记录",
        icon: Gift,
        keywords: ["supply", "gacha", "pool", "bg", "buji", "kachi", "chouka", "补给", "卡池", "抽卡", "武器"],
        allow: (all) => all,
      },
      {
        to: "/admin?section=coupons",
        section: "coupons",
        label: "优惠券",
        shortDesc: "起草、批量发放与管理商城优惠活动",
        icon: Ticket,
        keywords: ["coupon", "discount", "promo", "yhq", "youhui", "zhekou", "优惠券", "折扣", "满减"],
        allow: (all) => all,
      },
      {
        to: "/admin?section=announcements",
        section: "announcements",
        label: "公告管理",
        shortDesc: "起草、置顶与发布社区官方公告",
        icon: Megaphone,
        keywords: ["announcement", "notice", "news", "gg", "gonggao", "公告", "通知", "置顶"],
        allow: (all, p) => all || p.includes("announcements.manage"),
      },
    ],
  },
  {
    id: "players_game",
    label: "玩家与游戏",
    description: "社区账号权限、游戏头衔、模型云库与云端键位",
    items: [
      {
        to: "/admin?section=accounts",
        section: "accounts",
        label: "账号与权限",
        shortDesc: "查找成员、调整权限、处理封禁与改名",
        icon: Users,
        keywords: ["account", "user", "role", "ban", "zh", "zhanghao", "quanxian", "账号", "用户", "权限", "封禁"],
        allow: (all) => all,
      },
      {
        to: "/admin?section=titles",
        section: "titles",
        label: "头衔管理",
        shortDesc: "配置炫彩头衔、显示权重与权限节点",
        icon: Award,
        keywords: ["title", "prefix", "tx", "touxian", "chenghao", "头衔", "称号", "渐变"],
        allow: (all) => all,
      },
      {
        to: "/admin?section=ysm",
        section: "ysm",
        label: "模型管理",
        shortDesc: "发布 YSM 模型版本、管理玩家与 NPC 外观",
        icon: Layers,
        keywords: ["ysm", "model", "npc", "mx", "moxing", "waiguan", "模型", "外观"],
        allow: (all) => all,
      },
      {
        to: "/admin?section=keybindings",
        section: "keybindings",
        label: "玩家键位",
        shortDesc: "远程查看并协助修复玩家游戏按键冲突",
        icon: Keyboard,
        keywords: ["keybinding", "keys", "hotkey", "jw", "jianwei", "anjian", "键位", "按键", "快捷键"],
        allow: (all) => all,
      },
    ],
  },
  {
    id: "client_ai",
    label: "客户端与 AI",
    description: "桌面启动器发布、App 图标切换与小祥 AI 配置",
    items: [
      {
        to: "/admin?section=desktop-launcher",
        section: "desktop-launcher",
        label: "启动器",
        shortDesc: "同步 McPatch 资源、维护轮播与启动器版本",
        icon: Monitor,
        keywords: ["launcher", "mcpatch", "oss", "qdq", "qidongqi", "zhengbao", "启动器", "整包", "更新"],
        allow: (all) => all,
      },
      {
        to: "/admin?section=launcher-icon",
        section: "launcher-icon",
        label: "应用图标",
        shortDesc: "远程切换 App 日常或节日活动图标",
        icon: Smartphone,
        keywords: ["icon", "app", "appearance", "tb", "tubiao", "yingyong", "图标", "外观", "双子节"],
        allow: (all) => all,
      },
      {
        to: "/admin?section=ai",
        section: "ai",
        label: "小祥设置",
        shortDesc: "配置 AI 套餐、模型参数、提示词与知识库",
        icon: Sparkles,
        keywords: ["ai", "saki", "prompt", "xx", "xiaoxiang", "taocan", "小祥", "提示词", "知识库", "套餐"],
        allow: (all) => all,
      },
    ],
  },
  {
    id: "system_audit",
    label: "系统与审计",
    description: "游戏核心节点监控、邮件收发通道与全服审计追溯",
    items: [
      {
        to: "/admin?section=core",
        section: "core",
        label: "Core 管理",
        shortDesc: "监控游戏节点健康度与已发布物品库",
        icon: Server,
        keywords: ["core", "node", "server", "item", "hx", "jiedian", "fuwuqi", "节点", "物品库", "服务器"],
        allow: (all, p) => all || p.includes("core.read"),
      },
      {
        to: "/admin?section=email",
        section: "email",
        label: "邮件系统",
        shortDesc: "配置 DM/企业邮箱通道、收件同步与模板",
        icon: Mail,
        keywords: ["email", "mail", "smtp", "imap", "dm", "yj", "youjian", "邮件", "邮箱", "模板"],
        allow: (all) => all,
      },
      {
        to: "/admin?section=audit",
        section: "audit",
        label: "管理审计",
        shortDesc: "查询全服资金流水、订单与后台操作日志",
        icon: ChartNoAxesCombined,
        keywords: ["audit", "ledger", "log", "trade", "sj", "shenji", "liushui", "审计", "流水", "日志", "订单"],
        allow: (all, p) => all || p.includes("audit.read"),
      },
    ],
  },
];

export const OVERVIEW_NAV_ITEM = {
  to: "/admin?section=overview",
  section: "overview",
  label: "工作台总览",
  shortDesc: "实时待办提醒、常用固定入口与全部功能导航",
  icon: LayoutGrid,
  keywords: ["overview", "home", "dashboard", "zl", "zonglan", "shouye", "总览", "工作台", "首页"],
};

export const DEFAULT_PINNED_PATHS = [
  "/admin?section=support",
  "/admin?section=whitelist",
  "/merchant",
  "/admin?section=accounts",
  "/admin?section=supply",
];

/**
 * Legacy flat tuple helper preserved for existing callers & unit tests.
 * Returns [[to, label, Icon], ...].
 */
export function adminNavigation(permissions = []) {
  const all = permissions.includes("platform.admin");
  if (!all && permissions.includes("support.manage")) {
    return [["/admin?section=support", "工单系统", Ticket]];
  }
  return [
    ...(all ? [["/admin?section=support", "工单系统", Ticket]] : []),
    ...(all ? [["/admin?section=image-review", "图片审核", ScanEye]] : []),
    ["/merchant", "商店管理", Store],
    ...(all ? [["/admin?section=supply", "补给管理", Boxes]] : []),
    ...(all ? [["/admin?section=coupons", "优惠券", Ticket], ["/admin?section=desktop-launcher", "启动器", Monitor]] : []),
    ...(all || permissions.includes("intervention.manage") ? [["/admin?section=interventions", "平台介入", Gavel]] : []),
    ...(all || permissions.includes("announcements.manage") ? [["/admin?section=announcements", "公告管理", Megaphone]] : []),
    ...(all || permissions.includes("audit.read") ? [["/admin?section=audit", "管理审计", ChartNoAxesCombined]] : []),
    ...(all ? [["/admin?section=email", "邮件系统", Mail]] : []),
    ...(all ? [["/admin?section=ai", "小祥设置", Sparkles], ["/admin?section=accounts", "账号与权限", Users]] : []),
    ...(all ? [["/admin?section=whitelist", "白名单管理", ShieldCheck]] : []),
    ...(all ? [["/admin?section=ysm", "模型管理", Boxes], ["/admin?section=titles", "头衔管理", ShieldCheck], ["/admin?section=keybindings", "玩家键位", Keyboard]] : []),
    ...(all ? [["/admin?section=launcher-icon", "应用图标", Smartphone]] : []),
    ...(all || permissions.includes("core.read") ? [["/admin?section=core", "Core 管理", Server]] : []),
  ];
}

export function adminNavigationGroups(permissions = []) {
  const all = permissions.includes("platform.admin");
  const supportOnly = !all && permissions.includes("support.manage");
  const groups = ADMIN_DOMAINS.map((group) => ({
    ...group,
    items: group.items.filter((item) => {
      if (supportOnly) return item.section === "support";
      return item.allow(all, permissions);
    }),
  })).filter((group) => group.items.length > 0);

  const flatItems = groups.flatMap((group) =>
    group.items.map((item) => ({ ...item, groupId: group.id, groupLabel: group.label })),
  );
  const hasOverview = !supportOnly && flatItems.length >= 3;
  return { groups, flatItems, hasOverview, supportOnly };
}

export function findAdminNavItem(activePath, permissions = []) {
  const { groups, flatItems, hasOverview } = adminNavigationGroups(permissions);
  if (hasOverview && activePath === OVERVIEW_NAV_ITEM.to) {
    return { item: OVERVIEW_NAV_ITEM, group: null };
  }
  const match = flatItems.find((item) => item.to === activePath);
  if (!match) return { item: null, group: null };
  const group = groups.find((g) => g.id === match.groupId) || null;
  return { item: match, group };
}

export function matchAdminSearch(item, rawQuery) {
  const q = String(rawQuery || "").trim().toLowerCase();
  if (!q) return true;
  const tokens = q.split(/\s+/).filter(Boolean);
  const haystack = [
    item.label,
    item.shortDesc || "",
    item.section || "",
    item.groupLabel || "",
    ...(item.keywords || []),
  ]
    .join(" ")
    .toLowerCase();
  return tokens.every((token) => haystack.includes(token));
}
