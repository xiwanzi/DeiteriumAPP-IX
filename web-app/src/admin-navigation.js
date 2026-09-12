import { Store, Gavel, Megaphone, ChartNoAxesCombined, Mail, Server, Sparkles, Users, Smartphone, Ticket, Monitor } from "lucide-react";

export function adminNavigation(permissions = []) {
  const all = permissions.includes("platform.admin");
  return [
    ["/merchant", "商店管理", Store],
    ...(all ? [["/admin?section=coupons", "优惠券", Ticket], ["/admin?section=desktop-launcher", "启动器", Monitor]] : []),
    ...(all || permissions.includes("intervention.manage") ? [["/admin?section=interventions", "平台介入", Gavel]] : []),
    ...(all || permissions.includes("announcements.manage") ? [["/admin?section=announcements", "公告管理", Megaphone]] : []),
    ...(all || permissions.includes("audit.read") ? [["/admin?section=audit", "管理审计", ChartNoAxesCombined]] : []),
    ...(all ? [["/admin?section=email", "邮件提醒", Mail]] : []),
    ...(all ? [["/admin?section=ai", "小祥设置", Sparkles], ["/admin?section=accounts", "账号与权限", Users]] : []),
    ...(all ? [["/admin?section=launcher-icon", "应用图标", Smartphone]] : []),
    ...(all || permissions.includes("core.read") ? [["/admin?section=core", "Core 管理", Server]] : []),
  ];
}
