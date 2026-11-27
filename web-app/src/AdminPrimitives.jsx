import React, { useEffect, useMemo, useRef, useState } from "react";
import {
  Check,
  Copy,
  CornerDownLeft,
  Moon,
  MoreHorizontal,
  PanelLeftClose,
  PanelLeftOpen,
  Pin,
  Search,
  Store,
  Sun,
  X,
} from "lucide-react";
import {
  OVERVIEW_NAV_ITEM,
  adminNavigationGroups,
  matchAdminSearch,
} from "./admin-navigation.js";

/**
 * Compact inline copy button with transient check feedback.
 */
export function CopyButton({ value, label = "复制", compact = false, className = "" }) {
  const [copied, setCopied] = useState(false);
  const timer = useRef(null);
  useEffect(() => () => clearTimeout(timer.current), []);
  const copy = async (e) => {
    e.stopPropagation();
    if (!value) return;
    try {
      await navigator.clipboard.writeText(String(value));
      setCopied(true);
      clearTimeout(timer.current);
      timer.current = setTimeout(() => setCopied(false), 1500);
    } catch {
      // Fallback for restricted clipboard contexts
      const input = document.createElement("textarea");
      input.value = String(value);
      input.style.position = "fixed";
      input.style.opacity = "0";
      document.body.appendChild(input);
      input.select();
      try {
        document.execCommand("copy");
        setCopied(true);
        clearTimeout(timer.current);
        timer.current = setTimeout(() => setCopied(false), 1500);
      } catch {}
      document.body.removeChild(input);
    }
  };
  return (
    <button
      type="button"
      className={`adm-copy-btn ${compact ? "is-compact" : ""} ${copied ? "is-copied" : ""} ${className}`}
      onClick={copy}
      title={copied ? "已复制" : `复制${label}`}
      aria-label={copied ? "已复制" : `复制${label}`}
    >
      {copied ? <Check size={13} /> : <Copy size={13} />}
      {!compact && <span>{copied ? "已复制" : label}</span>}
    </button>
  );
}

/**
 * Compact row action overflow menu so data tables don't explode with 6 buttons per row.
 */
export function ActionMenu({ items = [], label = "更多操作", disabled = false }) {
  const [open, setOpen] = useState(false);
  const rootRef = useRef(null);
  useEffect(() => {
    if (!open) return;
    const onPointer = (e) => {
      if (!rootRef.current?.contains(e.target)) setOpen(false);
    };
    const onKey = (e) => {
      if (e.key === "Escape") setOpen(false);
    };
    document.addEventListener("mousedown", onPointer);
    window.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onPointer);
      window.removeEventListener("keydown", onKey);
    };
  }, [open]);

  const visibleItems = items.filter(Boolean);
  if (!visibleItems.length) return null;

  return (
    <div className={`adm-menu ${open ? "is-open" : ""}`} ref={rootRef}>
      <button
        type="button"
        className="adm-menu-trigger"
        disabled={disabled}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label={label}
        onClick={(e) => {
          e.stopPropagation();
          setOpen((v) => !v);
        }}
      >
        <MoreHorizontal size={16} />
        <span>更多</span>
      </button>
      {open && (
        <div className="adm-menu-popover" role="menu">
          {visibleItems.map((entry, idx) =>
            entry.divider ? (
              <div className="adm-menu-divider" key={`div-${idx}`} role="separator" />
            ) : entry.heading ? (
              <div className="adm-menu-heading" key={`head-${idx}`}>
                {entry.heading}
              </div>
            ) : (
              <button
                key={entry.key || entry.label}
                type="button"
                role="menuitem"
                disabled={entry.disabled}
                className={`adm-menu-item ${entry.danger ? "is-danger" : ""}`}
                onClick={(e) => {
                  e.stopPropagation();
                  setOpen(false);
                  entry.onClick?.();
                }}
              >
                {entry.icon && <entry.icon size={15} />}
                <span>{entry.label}</span>
              </button>
            ),
          )}
        </div>
      )}
    </div>
  );
}

/**
 * Keyboard-first Command Palette (Ctrl+K / Cmd+K / '/') for instant jumps across all 18 admin modules.
 */
export function AdminCommandPalette({
  open,
  onClose,
  permissions,
  activePath,
  pinnedPaths = [],
  recentPaths = [],
  badges = {},
  onSelect,
  dark,
  toggleTheme,
  sidebarCollapsed,
  toggleSidebar,
}) {
  const [query, setQuery] = useState("");
  const [cursor, setCursor] = useState(0);
  const inputRef = useRef(null);

  const { flatItems, hasOverview } = useMemo(
    () => adminNavigationGroups(permissions),
    [permissions],
  );

  const allNavItems = useMemo(
    () => (hasOverview ? [OVERVIEW_NAV_ITEM, ...flatItems] : flatItems),
    [hasOverview, flatItems],
  );

  const quickActions = useMemo(
    () => [
      {
        id: "action:storefront",
        label: "返回玩家商城",
        shortDesc: "切换到 Deuterium 玩家社区前台",
        icon: Store,
        keywords: ["storefront", "player", "back", "商城", "玩家", "前台", "返回"],
        run: () => onSelect("/"),
      },
      {
        id: "action:theme",
        label: dark ? "切换为浅色外观" : "切换为深色外观",
        shortDesc: "即时切换管理后台明暗主题",
        icon: dark ? Sun : Moon,
        keywords: ["theme", "dark", "light", "主题", "深色", "浅色", "夜间"],
        run: () => {
          toggleTheme?.();
          onClose();
        },
      },
      {
        id: "action:sidebar",
        label: sidebarCollapsed ? "展开左侧导航栏" : "收起左侧导航栏（极简图标模式）",
        shortDesc: "切换宽屏工作视野",
        icon: sidebarCollapsed ? PanelLeftOpen : PanelLeftClose,
        keywords: ["sidebar", "nav", "collapse", "expand", "侧边栏", "导航", "收起", "展开"],
        run: () => {
          toggleSidebar?.();
          onClose();
        },
      },
    ],
    [dark, sidebarCollapsed, toggleTheme, toggleSidebar, onSelect, onClose],
  );

  const sections = useMemo(() => {
    const trimmed = query.trim();
    if (!trimmed) {
      const result = [];
      const recents = recentPaths
        .map((p) => allNavItems.find((i) => i.to === p))
        .filter(Boolean)
        .slice(0, 4);
      if (recents.length) result.push({ title: "最近访问", items: recents });

      const pinned = pinnedPaths
        .map((p) => allNavItems.find((i) => i.to === p))
        .filter((item) => item && !recents.some((r) => r.to === item.to))
        .slice(0, 5);
      if (pinned.length) result.push({ title: "常用固定", items: pinned });

      result.push({ title: "全部管理功能", items: allNavItems });
      result.push({ title: "快捷操作", items: quickActions });
      return result;
    }

    const navMatches = allNavItems.filter((item) => matchAdminSearch(item, trimmed));
    const actionMatches = quickActions.filter((item) => matchAdminSearch(item, trimmed));
    const result = [];
    if (navMatches.length) result.push({ title: "匹配功能", items: navMatches });
    if (actionMatches.length) result.push({ title: "快捷操作", items: actionMatches });
    return result;
  }, [query, allNavItems, recentPaths, pinnedPaths, quickActions]);

  const flatSelectable = useMemo(
    () => sections.flatMap((s) => s.items),
    [sections],
  );

  useEffect(() => {
    if (!open) return;
    setQuery("");
    setCursor(0);
    const frame = requestAnimationFrame(() => inputRef.current?.focus());
    return () => cancelAnimationFrame(frame);
  }, [open]);

  useEffect(() => {
    setCursor(0);
  }, [query]);

  if (!open) return null;

  const activate = (entry) => {
    if (!entry) return;
    if (entry.run) {
      entry.run();
    } else if (entry.to) {
      onSelect(entry.to);
      onClose();
    }
  };

  const onKeyDown = (e) => {
    if (e.key === "Escape") {
      e.preventDefault();
      onClose();
    } else if (e.key === "ArrowDown") {
      e.preventDefault();
      if (flatSelectable.length) setCursor((c) => (c + 1) % flatSelectable.length);
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      if (flatSelectable.length)
        setCursor((c) => (c - 1 + flatSelectable.length) % flatSelectable.length);
    } else if (e.key === "Enter") {
      e.preventDefault();
      activate(flatSelectable[cursor]);
    }
  };

  let runningIndex = -1;

  return (
    <div
      className="adm-palette-backdrop"
      role="dialog"
      aria-modal="true"
      aria-label="管理后台快速跳转"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div className="adm-palette" onKeyDown={onKeyDown}>
        <div className="adm-palette-input-row">
          <Search size={18} />
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="搜索功能名称、拼音缩写或英文（如：白名单 / bmd / 补给 / 审计 / 键位）…"
            aria-label="搜索管理后台功能"
          />
          {query && (
            <button
              type="button"
              className="adm-palette-clear"
              onClick={() => setQuery("")}
              aria-label="清空搜索"
            >
              <X size={15} />
            </button>
          )}
          <kbd className="adm-kbd">Esc</kbd>
        </div>

        <div className="adm-palette-results">
          {!flatSelectable.length ? (
            <div className="adm-palette-empty">
              <p>没有找到与“{query}”匹配的管理功能</p>
              <small>试试输入：工单、白名单、商店、补给、账号、头衔、小祥、Core</small>
            </div>
          ) : (
            sections.map((section) => (
              <div className="adm-palette-group" key={section.title}>
                <div className="adm-palette-group-title">{section.title}</div>
                {section.items.map((item) => {
                  runningIndex += 1;
                  const idx = runningIndex;
                  const Icon = item.icon;
                  const isActive = idx === cursor;
                  const isCurrent = item.to && item.to === activePath;
                  const count = item.badgeKey ? badges[item.badgeKey] : 0;
                  const isPinned = item.to && pinnedPaths.includes(item.to);
                  return (
                    <button
                      key={`${section.title}:${item.to || item.id}`}
                      type="button"
                      className={`adm-palette-item ${isActive ? "is-highlighted" : ""} ${isCurrent ? "is-current" : ""}`}
                      onMouseEnter={() => setCursor(idx)}
                      onClick={() => activate(item)}
                    >
                      <span className="adm-palette-icon">
                        <Icon size={17} />
                      </span>
                      <span className="adm-palette-copy">
                        <strong>
                          {item.label}
                          {isPinned && <Pin size={11} className="adm-inline-pin" />}
                        </strong>
                        <small>{item.shortDesc}</small>
                      </span>
                      {item.groupLabel && (
                        <span className="adm-palette-tag">{item.groupLabel}</span>
                      )}
                      {count > 0 && <span className="adm-count-pill">{count}</span>}
                      {isCurrent && <span className="adm-current-tag">当前</span>}
                      {isActive && <CornerDownLeft size={14} className="adm-enter-hint" />}
                    </button>
                  );
                })}
              </div>
            ))
          )}
        </div>

        <div className="adm-palette-footer">
          <span>
            <kbd>↑</kbd> <kbd>↓</kbd> 选择
          </span>
          <span>
            <kbd>Enter</kbd> 跳转
          </span>
          <span>
            <kbd>Ctrl</kbd>+<kbd>K</kbd> 随时唤起
          </span>
        </div>
      </div>
    </div>
  );
}
