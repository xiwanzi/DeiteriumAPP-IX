import launcherIcon from "./assets/launcher-icons/default.svg";
import React, { useCallback, useContext, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import {
  Store,
  Shapes,
  Handshake,
  MessageCircle,
  WalletCards,
  UserRound,
  ShieldCheck,
  Search,
  Bell,
  ShoppingBag,
  Moon,
  Sun,
  Menu,
  PanelLeftClose,
  PanelLeftOpen,
  ChevronRight,
  ChevronDown,
  ArrowUpRight,
  Info,
  Pin,
} from "lucide-react";
import { Avatar } from "./components.jsx";
import { ShoppingCartContext } from "./ShoppingCartContext.jsx";
import {
  OVERVIEW_NAV_ITEM,
  adminNavigation,
  adminNavigationGroups,
  findAdminNavItem,
} from "./admin-navigation.js";
import { AdminCommandPalette } from "./AdminPrimitives.jsx";

export const navigation = [
  ["/", "商城", Store],
  ["/market", "玩家市场", Shapes],
  ["/commissions", "委托大厅", Handshake],
  ["/information", "信息", MessageCircle],
  ["/wallet", "钱包", WalletCards],
  ["/me", "我的", UserRound],
];

const COLLAPSED_GROUPS_KEY = "deuterium-admin-collapsed-groups";
const RAIL_COLLAPSED_KEY = "deuterium-admin-rail-collapsed";

function readStorageJson(key, fallback) {
  try {
    const raw = localStorage.getItem(key);
    return raw ? JSON.parse(raw) : fallback;
  } catch {
    return fallback;
  }
}

function CartButton({ count, size, onClick, active }) {
  const description = count === null ? undefined : count > 0 ? `购物袋中有 ${count} 件商品` : "购物袋为空";
  return (
    <button
      type="button"
      className="icon-button bag-icon"
      aria-label="打开购物袋"
      aria-description={description}
      title={description ?? "打开购物袋"}
      aria-current={active ? "page" : undefined}
      onClick={onClick}
    >
      <ShoppingBag size={size} />
      {count > 0 && (
        <span className="bag-count" aria-hidden="true">
          {count > 99 ? "99+" : count}
        </span>
      )}
    </button>
  );
}

export default function AppShell({
  user,
  path,
  search = "",
  navigate,
  children,
  dark,
  toggleTheme,
  onSearch,
  onNotifications,
  onCart,
  onAbout,
  cartCount = 0,
  unread = 0,
  mode = "connected",
  basePath = "",
  adminSignals = {},
  pinnedPaths = [],
  onTogglePin,
  recentPaths = [],
  paletteOpen = false,
  setPaletteOpen,
}) {
  const cart = useContext(ShoppingCartContext);
  const bagCount = cart ? cart.count : cartCount;
  const [mobile, setMobile] = useState(false);
  const chat = path === "/information";
  const searchLabel = chat
    ? "搜索消息与会话"
    : {
        "/wallet": "筛选账单",
        "/cart": "搜索商城商品",
        "/me": "搜索商城商品",
        "/notification-settings": "搜索商城商品",
      }[path] || "搜索好物、委托";
  const managing = path === "/admin" || path === "/merchant";
  const adminMeta = useMemo(
    () => (managing ? adminNavigationGroups(user.permissions) : null),
    [managing, user.permissions],
  );
  const supportOnly = Boolean(adminMeta?.supportOnly);
  const links = managing ? adminNavigation(user.permissions) : navigation;
  const section = new URLSearchParams(search).get("section");

  const activePath = useMemo(() => {
    if (!managing) return path;
    if (path === "/merchant") return "/merchant";
    if (section === "overview" && adminMeta?.hasOverview) return OVERVIEW_NAV_ITEM.to;
    if (section) {
      const exact = adminMeta?.flatItems.find((i) => i.to === `/admin?section=${section}`);
      if (exact) return exact.to;
    }
    if (adminMeta?.hasOverview) return OVERVIEW_NAV_ITEM.to;
    return links.find(([to]) => to.startsWith("/admin?"))?.[0] || "/merchant";
  }, [managing, path, section, adminMeta, links]);

  const support = path === "/admin" && activePath === "/admin?section=support";
  const [supportNavExpanded, setSupportNavExpanded] = useState(false);
  useEffect(() => {
    setSupportNavExpanded(false);
  }, [support]);

  const [railCollapsed, setRailCollapsed] = useState(() =>
    Boolean(readStorageJson(RAIL_COLLAPSED_KEY, false)),
  );
  const toggleRail = useCallback(() => {
    if (support) {
      setSupportNavExpanded((v) => !v);
      return;
    }
    setRailCollapsed((old) => {
      const next = !old;
      try {
        localStorage.setItem(RAIL_COLLAPSED_KEY, JSON.stringify(next));
      } catch {}
      return next;
    });
  }, [support]);

  const effectiveRailCollapsed = managing && (support ? !supportNavExpanded : railCollapsed);

  const [collapsedGroups, setCollapsedGroups] = useState(() =>
    readStorageJson(COLLAPSED_GROUPS_KEY, {}),
  );

  const toggleGroup = (groupId) => {
    setCollapsedGroups((old) => {
      const next = { ...old, [groupId]: !old[groupId] };
      try {
        localStorage.setItem(COLLAPSED_GROUPS_KEY, JSON.stringify(next));
      } catch {}
      return next;
    });
  };

  const activeNavMeta = useMemo(
    () => (managing ? findAdminNavItem(activePath, user.permissions) : { item: null, group: null }),
    [managing, activePath, user.permissions],
  );

  // Auto-expand the group containing the currently active page
  useEffect(() => {
    if (!managing || !activeNavMeta.group?.id) return;
    setCollapsedGroups((old) => {
      if (!old[activeNavMeta.group.id]) return old;
      const next = { ...old, [activeNavMeta.group.id]: false };
      try {
        localStorage.setItem(COLLAPSED_GROUPS_KEY, JSON.stringify(next));
      } catch {}
      return next;
    });
  }, [managing, activeNavMeta.group?.id]);

  const [internalPaletteOpen, setInternalPaletteOpen] = useState(false);
  const isPaletteOpen = setPaletteOpen ? paletteOpen : internalPaletteOpen;
  const setPalette = setPaletteOpen || setInternalPaletteOpen;

  const pendingStoreSearch = useRef(false);
  const openSearch = useCallback(() => {
    if (managing) {
      setPalette(true);
      return;
    }
    if (["/cart", "/me", "/notification-settings"].includes(path)) {
      pendingStoreSearch.current = true;
      navigate("/");
    } else onSearch?.();
  }, [managing, path, navigate, onSearch, setPalette]);

  useEffect(() => {
    setMobile(false);
    if (pendingStoreSearch.current && path === "/") window.dispatchEvent(new Event("deuterium:search"));
    pendingStoreSearch.current = false;
  }, [path, search]);

  useEffect(() => {
    const h = (e) => {
      if (e.key === "Escape") setMobile(false);
      if (managing && (e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        setPalette((v) => !v);
        return;
      }
      if (
        e.key === "/" &&
        !e.defaultPrevented &&
        !e.ctrlKey &&
        !e.metaKey &&
        !e.altKey &&
        !e.target.closest?.("input,textarea,select,[contenteditable=true],dialog") &&
        !document.querySelector("dialog[open]")
      ) {
        e.preventDefault();
        openSearch();
      }
    };
    window.addEventListener("keydown", h);
    return () => window.removeEventListener("keydown", h);
  }, [managing, openSearch, setPalette]);

  const content = useRef(null);
  const previousChat = useRef(chat);
  useLayoutEffect(() => {
    if (previousChat.current === chat) return;
    previousChat.current = chat;
    if (matchMedia("(prefers-reduced-motion: reduce)").matches) return;
    const motion = content.current?.animate(
      [
        { opacity: 0.45, transform: "translateY(4px)" },
        { opacity: 1, transform: "translateY(0)" },
      ],
      { duration: 200, easing: "ease-out" },
    );
    return () => motion?.cancel();
  }, [chat]);

  const link = (e, to) => {
    if (e.metaKey || e.ctrlKey) return;
    e.preventDefault();
    navigate(to);
    setMobile(false);
  };

  const pinnedNavItems = useMemo(() => {
    if (!managing || !adminMeta || supportOnly) return [];
    return pinnedPaths
      .map((p) => adminMeta.flatItems.find((i) => i.to === p))
      .filter(Boolean);
  }, [managing, adminMeta, supportOnly, pinnedPaths]);

  const renderAdminNavItem = (item, { showPin = true } = {}) => {
    const Icon = item.icon;
    const count = item.badgeKey ? adminSignals[item.badgeKey] : 0;
    const pinned = pinnedPaths.includes(item.to);
    const active = activePath === item.to;
    return (
      <div className="adm-nav-row" key={item.to}>
        <a
          href={`${basePath}${item.to}`}
          title={item.label}
          aria-label={item.label}
          aria-current={active ? "page" : undefined}
          className={active ? "active" : ""}
          onClick={(e) => link(e, item.to)}
        >
          <Icon size={17} />
          <span>{item.label}</span>
          {count > 0 && <span className="adm-nav-badge">{count > 99 ? "99+" : count}</span>}
        </a>
        {showPin && onTogglePin && !supportOnly && (
          <button
            type="button"
            className={`adm-nav-pin ${pinned ? "is-pinned" : ""}`}
            title={pinned ? "取消固定" : "固定到顶部常用"}
            aria-label={pinned ? `取消固定${item.label}` : `固定${item.label}`}
            aria-pressed={pinned}
            onClick={(e) => {
              e.stopPropagation();
              onTogglePin(item.to);
            }}
          >
            <Pin size={12} />
          </button>
        )}
      </div>
    );
  };

  return (
    <div
      className={`app-shell ${chat ? "chat-shell" : path === "/" ? "store-shell" : ""} ${managing ? "management-shell" : ""} ${effectiveRailCollapsed ? "adm-rail-collapsed" : ""} ${support ? `support-shell ${supportNavExpanded ? "nav-expanded" : "nav-collapsed"}` : ""}`}
    >
      <a className="skip-link" href="#main-content">
        跳到主要内容
      </a>
      {mobile && (
        <button
          className="mobile-backdrop"
          aria-label="关闭导航菜单"
          onClick={() => setMobile(false)}
        />
      )}
      <aside id="app-navigation" className={`sidebar ${mobile ? "open" : ""}`}>
        <a
          className="brand"
          href={managing && adminMeta?.hasOverview ? `${basePath}${OVERVIEW_NAV_ITEM.to}` : `${basePath}/`}
          onClick={(e) => link(e, managing && adminMeta?.hasOverview ? OVERVIEW_NAV_ITEM.to : "/")}
          title={managing ? "官方管理工作台" : "Deuterium"}
          aria-label={managing ? "官方管理工作台" : "Deuterium 玩家首页"}
        >
          <img src={launcherIcon} alt="Deuterium App 图标" />
          <span>
            Deuterium
            <small>{managing ? (supportOnly ? "玩家支持工作台" : "官方管理工作台") : "属于我们的世界"}</small>
          </span>
        </a>

        {managing ? (
          <>
            {!supportOnly && (
              <button
                type="button"
                className="adm-sidebar-quick-jump"
                onClick={() => setPalette(true)}
                title="快速跳转功能 (Ctrl+K)"
                aria-label="快速跳转管理功能"
              >
                <Search size={14} />
                <span>快速跳转…</span>
                <kbd>Ctrl K</kbd>
              </button>
            )}
            <div className="sidebar-scroll">
              <nav aria-label="管理导航">
                {adminMeta?.hasOverview && renderAdminNavItem(OVERVIEW_NAV_ITEM, { showPin: false })}

                {pinnedNavItems.length > 0 && (
                  <div className="adm-nav-group adm-pinned-section">
                    <div className="adm-nav-group-toggle" style={{ cursor: "default" }}>
                      <Pin size={12} />
                      <span>常用固定</span>
                    </div>
                    <div className="adm-nav-group-items">
                      {pinnedNavItems.map((item) => renderAdminNavItem(item, { showPin: true }))}
                    </div>
                  </div>
                )}

                {adminMeta?.groups.map((group) => {
                  const isCollapsed = Boolean(collapsedGroups[group.id]) && !effectiveRailCollapsed;
                  const groupPending = group.items.reduce(
                    (sum, item) => sum + (item.badgeKey ? adminSignals[item.badgeKey] || 0 : 0),
                    0,
                  );
                  return (
                    <div className="adm-nav-group" key={group.id}>
                      {!supportOnly && (
                        <button
                          type="button"
                          className="adm-nav-group-toggle"
                          aria-expanded={!isCollapsed}
                          onClick={() => toggleGroup(group.id)}
                        >
                          <ChevronDown size={13} />
                          <span>{group.label}</span>
                          {isCollapsed && groupPending > 0 && (
                            <span className="adm-nav-group-count">{groupPending}</span>
                          )}
                        </button>
                      )}
                      {!isCollapsed && (
                        <div className="adm-nav-group-items">
                          {group.items.map((item) => renderAdminNavItem(item, { showPin: true }))}
                        </div>
                      )}
                    </div>
                  );
                })}
              </nav>
            </div>
          </>
        ) : (
          <>
            <div className="sidebar-caption">探索与连接</div>
            <nav aria-label="主要导航">
              {links.map(([to, label, Icon]) => (
                <a
                  key={to}
                  href={`${basePath}${to}`}
                  title={label}
                  aria-label={label}
                  aria-current={activePath === to ? "page" : undefined}
                  className={activePath === to ? "active" : ""}
                  onClick={(e) => link(e, to)}
                >
                  <Icon size={21} />
                  <span>{label}</span>
                  {to === "/information" && unread > 0 && <i className="nav-dot" />}
                </a>
              ))}
            </nav>
          </>
        )}

        <div className="sidebar-bottom">
          {chat && (
            <div className="chat-rail-actions">
              <button
                className="icon-button"
                title={dark ? "切换浅色模式" : "切换深色模式"}
                aria-label={dark ? "切换浅色模式" : "切换深色模式"}
                onClick={toggleTheme}
              >
                {dark ? <Sun size={21} /> : <Moon size={21} />}
              </button>
              <button className="icon-button" title="打开通知" aria-label="打开通知" onClick={onNotifications}>
                <Bell size={21} />
                {unread > 0 && <i className="notification-dot" />}
              </button>
              {onCart && <CartButton count={bagCount} size={21} onClick={onCart} />}
            </div>
          )}
          <div className="world-card">
            <span className="world-dot" />
            <span>
              创造，始终发生。<small>DEUTERIUM COMMUNITY</small>
            </span>
            <div className="world-lines" />
          </div>
          {managing && (
            <a
              className="admin-nav"
              title="返回玩家商城"
              aria-label="返回玩家商城"
              href={`${basePath}/`}
              onClick={(e) => link(e, "/")}
            >
              <Store size={17} />
              <span>返回玩家商城</span>
              <ArrowUpRight size={14} />
            </a>
          )}
          {user.admin && !managing && (
            <a
              title="官方管理"
              aria-label="官方管理"
              className={`admin-nav ${path === "/admin" ? "active" : ""}`}
              href={`${basePath}/admin`}
              onClick={(e) => link(e, "/admin")}
            >
              <ShieldCheck size={19} />
              <span>官方管理</span>
              <ArrowUpRight size={14} />
            </a>
          )}
          <button className="sidebar-about" title="帮助与关于" aria-label="帮助与关于" onClick={onAbout}>
            <Info size={17} />
            <span>帮助与关于</span>
          </button>
          <button className="sidebar-profile" title={user.name} onClick={() => navigate("/me")}>
            <Avatar user={user} />
            <span>
              <strong>{user.name}</strong>
              <small>Deuterium ID</small>
            </span>
            <ChevronRight size={16} />
          </button>
        </div>
      </aside>
      <div className="main-shell">
        <header className="topbar">
          <div className="topbar-left">
            {managing && (
              <button
                type="button"
                className={`icon-button ${support ? "support-nav-toggle" : "adm-rail-toggle"}`}
                aria-label={effectiveRailCollapsed ? "展开左侧导航" : "收起左侧导航"}
                title={effectiveRailCollapsed ? "展开左侧导航" : "收起左侧导航"}
                aria-expanded={!effectiveRailCollapsed}
                aria-controls="app-navigation"
                onClick={toggleRail}
              >
                {effectiveRailCollapsed ? <PanelLeftOpen size={18} /> : <PanelLeftClose size={18} />}
              </button>
            )}
            <button
              className="icon-button mobile-menu"
              aria-label="打开导航菜单"
              onClick={() => setMobile(!mobile)}
            >
              <Menu size={22} />
            </button>
            {managing ? (
              <span className="breadcrumb" aria-label="当前位置">
                {adminMeta?.hasOverview ? (
                  <button
                    type="button"
                    className="adm-crumb-btn"
                    onClick={() => navigate(OVERVIEW_NAV_ITEM.to)}
                  >
                    官方管理
                  </button>
                ) : (
                  <span>官方管理</span>
                )}
                {activeNavMeta.group && (
                  <>
                    <ChevronRight size={13} />
                    <span className="adm-crumb-domain">{activeNavMeta.group.label}</span>
                  </>
                )}
                <ChevronRight size={13} />
                <strong>{activeNavMeta.item?.label || "官方管理"}</strong>
              </span>
            ) : (
              <span className="breadcrumb">
                Deuterium <ChevronRight size={13} />
                <strong>
                  {links.find((n) => n[0] === activePath)?.[1] ||
                    {
                      "/admin": "官方管理",
                      "/announcements": "社区公告",
                      "/notifications": "通知",
                      "/notification-settings": "通知设置",
                      "/merchant": "商店管理",
                      "/cart": "购物袋",
                      "/orders": "我的订单",
                    }[path] ||
                    "Deuterium"}
                </strong>
              </span>
            )}
            {path === "/" && <div id="shop-feedback-host" className="shop-feedback-host" />}
          </div>
          <div className="topbar-actions">
            {managing && !supportOnly && (
              <button
                type="button"
                className="adm-topbar-palette"
                onClick={() => setPalette(true)}
                aria-label="快速跳转管理功能"
              >
                <Search size={15} />
                <span>快速跳转功能…</span>
                <kbd>Ctrl K</kbd>
              </button>
            )}
            <button className="demo-badge" onClick={onAbout}>
              <span />
              Deuterium · 2.0.46
            </button>
            {!managing && path !== "/" && (
              <button className="top-search" onClick={openSearch} aria-label={searchLabel}>
                <Search size={17} />
                <span>{searchLabel}…</span>
                <kbd>/</kbd>
              </button>
            )}
            <button
              className="icon-button"
              aria-label={dark ? "切换浅色模式" : "切换深色模式"}
              title={dark ? "切换浅色模式" : "切换深色模式"}
              onClick={toggleTheme}
            >
              {dark ? <Sun size={19} /> : <Moon size={19} />}
            </button>
            <button
              className="icon-button"
              aria-label="打开通知"
              title="打开通知"
              onClick={onNotifications}
            >
              <Bell size={19} />
              {unread > 0 && <i className="notification-dot" />}
            </button>
            {onCart && !managing && <CartButton count={bagCount} size={19} active={path === "/cart"} onClick={onCart} />}
          </div>
        </header>
        <main
          ref={content}
          id="main-content"
          tabIndex={-1}
          className={`main-content ${chat ? "chat-main" : ""}`}
          key={managing ? activePath : path}
        >
          {children}
        </main>
        {!chat && !support && (
          <footer className="site-footer">
            <span>
              DEUTERIUM <i />
              让热爱，自在相连。
            </span>
            <span>
              Web 2.0.46 <b>·</b> Deuterium ID
            </span>
          </footer>
        )}
      </div>

      {managing && !supportOnly && (
        <AdminCommandPalette
          open={isPaletteOpen}
          onClose={() => setPalette(false)}
          permissions={user.permissions}
          activePath={activePath}
          pinnedPaths={pinnedPaths}
          recentPaths={recentPaths}
          badges={adminSignals}
          onSelect={(to) => navigate(to)}
          dark={dark}
          toggleTheme={toggleTheme}
          sidebarCollapsed={effectiveRailCollapsed}
          toggleSidebar={toggleRail}
        />
      )}
    </div>
  );
}
