import React, { useEffect, useState } from "react";
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
  ChevronRight,
  ArrowUpRight,
  Info,
  LogOut,
} from "lucide-react";
import { Avatar, media } from "./components.jsx";
import { adminNavigation } from "./admin-navigation.js";
export const navigation = [
  ["/", "商城", Store],
  ["/market", "玩家市场", Shapes],
  ["/commissions", "委托大厅", Handshake],
  ["/information", "信息", MessageCircle],
  ["/wallet", "钱包", WalletCards],
  ["/me", "我的", UserRound],
];
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
}) {
  const [mobile, setMobile] = useState(false),
    chat = path === "/information";
  const managing = path === "/admin" || path === "/merchant";
  const links = managing ? adminNavigation(user.permissions) : navigation;
  const section = new URLSearchParams(search).get("section");
  const activePath = path === "/admin" ? links.find(([to]) => to === `/admin?section=${section}`)?.[0] || links.find(([to]) => to.startsWith("/admin?"))?.[0] || "/merchant" : path;
  useEffect(() => {
    setMobile(false);
  }, [path, search]);
  useEffect(() => {
    const h = (e) => {
      if (e.key === "Escape") setMobile(false);
    };
    window.addEventListener("keydown", h);
    return () => window.removeEventListener("keydown", h);
  }, []);
  const link = (e, to) => {
    if (e.metaKey || e.ctrlKey) return;
    e.preventDefault();
    navigate(to);
    setMobile(false);
  };
  return (
    <div className={`app-shell ${chat ? "chat-shell" : ""} ${managing ? "management-shell" : ""}`}>
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
      <aside className={`sidebar ${mobile ? "open" : ""}`}>
        <a
          className="brand"
          href={`${basePath}/`}
          onClick={(e) => link(e, "/")}
          title="Deuterium"
        >
          <img src={media("app-icon.svg")} alt="Deuterium App 图标" />
          <span>
            Deuterium<small>{managing ? "官方管理工作台" : "属于我们的世界"}</small>
          </span>
        </a>
        <div className="sidebar-caption">{managing ? "店铺与运营" : "探索与连接"}</div>
        <nav aria-label={managing ? "管理导航" : "主要导航"}>
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
        <div className="sidebar-bottom">
          <div className="world-card">
            <span className="world-dot" />
            <span>
              创造，始终发生。<small>DEUTERIUM COMMUNITY</small>
            </span>
            <div className="world-lines" />
          </div>
          {managing && <a className="admin-nav" href={`${basePath}/`} onClick={(e) => link(e, "/")}><Store size={19} /><span>返回玩家商城</span><ArrowUpRight size={14} /></a>}
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
          <button
            className="sidebar-about"
            title="帮助与关于"
            aria-label="帮助与关于"
            onClick={onAbout}
          >
            <Info size={18} />
            <span>帮助与关于</span>
          </button>
          <button
            className="sidebar-profile"
            title={user.name}
            onClick={() => navigate("/me")}
          >
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
            <button
              className="icon-button mobile-menu"
              aria-label="打开导航菜单"
              onClick={() => setMobile(!mobile)}
            >
              <Menu size={22} />
            </button>
            <span className="breadcrumb">
              Deuterium <ChevronRight size={13} />
              <strong>
                {links.find((n) => n[0] === activePath)?.[1] || ({"/admin":"官方管理","/announcements":"社区公告","/notifications":"通知","/notification-settings":"通知设置","/merchant":"商店管理","/cart":"购物袋","/orders":"我的订单"}[path]) || "Deuterium"}
              </strong>
            </span>
          </div>
          <div className="topbar-actions">
            <button className="demo-badge" onClick={onAbout}>
              <span />
              Deuterium · 2.0.6
            </button>
            {!managing && <button className="top-search" onClick={onSearch}>
              <Search size={17} />
              <span>{chat ? "搜索消息与会话…" : "搜索好物、委托…"}</span>
              <kbd>/</kbd>
            </button>}
            <button
              className="icon-button"
              aria-label={dark ? "切换浅色模式" : "切换深色模式"}
              onClick={toggleTheme}
            >
              {dark ? <Sun size={19} /> : <Moon size={19} />}
            </button>
            <button
              className="icon-button"
              aria-label="打开通知"
              onClick={onNotifications}
            >
              <Bell size={19} />
              {unread > 0 && <i className="notification-dot" />}
            </button>
            {onCart && (
              <button
                className="icon-button bag-icon"
                aria-label="打开购物袋"
                onClick={onCart}
              >
                <ShoppingBag size={19} />
                {cartCount > 0 && <b>{cartCount}</b>}
              </button>
            )}
          </div>
        </header>
        <main
          id="main-content"
          tabIndex={-1}
          className={`main-content ${chat ? "chat-main" : ""}`}
          key={path}
        >
          {children}
        </main>
        {!chat && (
          <footer className="site-footer">
            <span>
              DEUTERIUM <i />
              让热爱，自在相连。
            </span>
            <span>
              Web 2.0.6 <b>·</b> Deuterium ID
            </span>
          </footer>
        )}
      </div>
    </div>
  );
}
