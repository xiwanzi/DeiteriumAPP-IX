import React, { useMemo } from "react";
import {
  ArrowUpRight,
  CheckCircle2,
  Pin,
  RefreshCw,
  Search,
  Server,
  ShieldCheck,
  Ticket,
  Gavel,
  ScanEye,
  Clock3,
} from "lucide-react";
import { Badge, Button } from "./components.jsx";
import { adminNavigationGroups } from "./admin-navigation.js";

export default function AdminOverview({
  user,
  navigate,
  signals = {},
  refreshingSignals = false,
  onRefreshSignals,
  pinnedPaths = [],
  onTogglePin,
  recentPaths = [],
  onOpenPalette,
}) {
  const permissions = user?.permissions || [];
  const { groups, flatItems } = useMemo(
    () => adminNavigationGroups(permissions),
    [permissions],
  );

  const pinnedItems = useMemo(
    () => pinnedPaths.map((p) => flatItems.find((i) => i.to === p)).filter(Boolean),
    [pinnedPaths, flatItems],
  );

  const recentItems = useMemo(
    () =>
      recentPaths
        .map((p) => flatItems.find((i) => i.to === p))
        .filter((i) => i && i.section !== "overview")
        .slice(0, 4),
    [recentPaths, flatItems],
  );

  const attentionCards = [
    {
      key: "support",
      visible: permissions.includes("platform.admin") || permissions.includes("support.manage"),
      label: "待处理工单",
      value: signals.supportOpen,
      hint: signals.supportOpen > 0 ? "优先查看玩家来信" : "当前队列已清空",
      to: "/admin?section=support",
      icon: Ticket,
      tone: signals.supportOpen > 0 ? "blue" : "sage",
    },
    {
      key: "whitelist",
      visible: permissions.includes("platform.admin"),
      label: "待审白名单",
      value: signals.whitelistPending,
      hint:
        signals.onlinePlayers != null
          ? `当前在线 ${signals.onlinePlayers} 人 · 有效 ${signals.whitelistActive ?? "—"} 人`
          : "入服申请与通行管理",
      to: "/admin?section=whitelist",
      icon: ShieldCheck,
      tone: signals.whitelistPending > 0 ? "amber" : "sage",
    },
    {
      key: "interventions",
      visible: permissions.includes("platform.admin") || permissions.includes("intervention.manage"),
      label: "平台介入",
      value: signals.interventionsPending,
      hint: signals.interventionsPending > 0 ? "有交易/委托争议待裁决" : "暂无待办争议案件",
      to: "/admin?section=interventions",
      icon: Gavel,
      tone: signals.interventionsPending > 0 ? "red" : "sage",
    },
    {
      key: "image-review",
      visible: permissions.includes("platform.admin"),
      label: "待复核图片",
      value: signals.imageFlagged,
      hint: signals.imageFlagged > 0 ? "超过风险阈值待人工确认" : "暂无高风险图片",
      to: "/admin?section=image-review",
      icon: ScanEye,
      tone: signals.imageFlagged > 0 ? "amber" : "sage",
    },
    {
      key: "core",
      visible: permissions.includes("platform.admin") || permissions.includes("core.read"),
      label: "Core 游戏节点",
      value:
        signals.nodesTotal != null
          ? `${signals.nodesOnline}/${signals.nodesTotal}`
          : null,
      hint:
        signals.nodesTotal != null
          ? signals.nodesOnline === signals.nodesTotal
            ? "全部常驻节点在线"
            : `${signals.nodesTotal - signals.nodesOnline} 个节点当前离线`
          : "查看节点与物品库",
      to: "/admin?section=core",
      icon: Server,
      tone:
        signals.nodesTotal != null && signals.nodesOnline < signals.nodesTotal
          ? "neutral"
          : "sage",
    },
  ].filter((c) => c.visible);

  const totalPending =
    (signals.supportOpen || 0) +
    (signals.whitelistPending || 0) +
    (signals.interventionsPending || 0) +
    (signals.imageFlagged || 0);

  return (
    <section className="adm-overview">
      <header className="adm-overview-hero">
        <div>
          <p className="eyebrow">DEUTERIUM OPERATIONS CONSOLE</p>
          <h1>官方管理工作台</h1>
          <p className="adm-overview-sub">
            {totalPending > 0
              ? `当前有 ${totalPending} 项待处理事务，按业务域分类可一步直达 ${flatItems.length} 项管理能力。`
              : `全部待办已清零 · 共 ${flatItems.length} 项管理能力已按业务域整理就绪。`}
          </p>
        </div>
        <div className="adm-overview-hero-actions">
          <Button secondary disabled={refreshingSignals} onClick={onRefreshSignals}>
            <RefreshCw size={15} className={refreshingSignals ? "adm-spin" : ""} />
            刷新概览
          </Button>
        </div>
      </header>

      {attentionCards.length > 0 && (
        <div className="adm-kpi-grid" aria-label="实时待办与系统状态">
          {attentionCards.map((card) => {
            const Icon = card.icon;
            return (
              <button
                key={card.key}
                type="button"
                className={`adm-kpi-card tone-${card.tone}`}
                onClick={() => navigate(card.to)}
              >
                <div className="adm-kpi-top">
                  <span className="adm-kpi-icon">
                    <Icon size={17} />
                  </span>
                  <span className="adm-kpi-label">{card.label}</span>
                  <ArrowUpRight size={15} className="adm-kpi-arrow" />
                </div>
                <strong className="adm-kpi-value">
                  {card.value ?? "—"}
                </strong>
                <small className="adm-kpi-hint">{card.hint}</small>
              </button>
            );
          })}
        </div>
      )}

      {(pinnedItems.length > 0 || recentItems.length > 0) && (
        <div className="adm-quick-strip">
          {pinnedItems.length > 0 && (
            <section className="adm-quick-block">
              <div className="adm-block-head">
                <h2>
                  <Pin size={14} /> 常用固定入口
                </h2>
                <small>点击任意模块卡片右上角图钉即可自定义</small>
              </div>
              <div className="adm-pinned-row">
                {pinnedItems.map((item) => {
                  const Icon = item.icon;
                  const count = item.badgeKey ? signals[item.badgeKey] : 0;
                  return (
                    <button
                      key={item.to}
                      type="button"
                      className="adm-pinned-chip"
                      onClick={() => navigate(item.to)}
                    >
                      <Icon size={16} />
                      <span>{item.label}</span>
                      {count > 0 && <span className="adm-count-pill">{count}</span>}
                    </button>
                  );
                })}
              </div>
            </section>
          )}

          {recentItems.length > 0 && (
            <section className="adm-quick-block adm-recent-block">
              <div className="adm-block-head">
                <h2>
                  <Clock3 size={14} /> 最近访问
                </h2>
              </div>
              <div className="adm-pinned-row">
                {recentItems.map((item) => {
                  const Icon = item.icon;
                  return (
                    <button
                      key={item.to}
                      type="button"
                      className="adm-pinned-chip is-subtle"
                      onClick={() => navigate(item.to)}
                    >
                      <Icon size={15} />
                      <span>{item.label}</span>
                    </button>
                  );
                })}
              </div>
            </section>
          )}
        </div>
      )}

      <div className="adm-domain-sections">
        {groups.map((group) => (
          <section className="adm-domain-section" key={group.id}>
            <div className="adm-domain-head">
              <div>
                <h2>{group.label}</h2>
                <p>{group.description}</p>
              </div>
              <Badge tone="neutral">{group.items.length} 项</Badge>
            </div>
            <div className="adm-module-grid">
              {group.items.map((item) => {
                const Icon = item.icon;
                const count = item.badgeKey ? signals[item.badgeKey] : 0;
                const pinned = pinnedPaths.includes(item.to);
                return (
                  <div className="adm-module-card" key={item.to}>
                    <button
                      type="button"
                      className="adm-module-main"
                      onClick={() => navigate(item.to)}
                    >
                      <div className="adm-module-icon-row">
                        <span className="adm-module-icon">
                          <Icon size={18} />
                        </span>
                        {count > 0 && (
                          <span className="adm-count-pill is-loud">
                            {count} 待办
                          </span>
                        )}
                      </div>
                      <strong>{item.label}</strong>
                      <p>{item.shortDesc}</p>
                    </button>
                    <button
                      type="button"
                      className={`adm-pin-btn ${pinned ? "is-pinned" : ""}`}
                      title={pinned ? "取消固定" : "固定到常用入口"}
                      aria-label={pinned ? `取消固定${item.label}` : `固定${item.label}到常用入口`}
                      aria-pressed={pinned}
                      onClick={(e) => {
                        e.stopPropagation();
                        onTogglePin?.(item.to);
                      }}
                    >
                      <Pin size={13} />
                    </button>
                  </div>
                );
              })}
            </div>
          </section>
        ))}
      </div>

      <footer className="adm-overview-tip">
        <CheckCircle2 size={15} />
        <span>
          效率提示：在管理后台任意页面按 <kbd>Ctrl</kbd>+<kbd>K</kbd>（或按 <kbd>/</kbd>）可秒开命令面板，支持拼音首字母直达；点击左上角折叠按钮可切换极简图标侧栏。
        </span>
      </footer>
    </section>
  );
}
