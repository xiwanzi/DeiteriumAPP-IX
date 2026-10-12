import { Store, Gavel, Megaphone, ChartNoAxesCombined, Mail, Server, Sparkles, Users, Smartphone, Ticket, Monitor, ShieldCheck, Boxes, ScanEye } from "lucide-react";

export function adminNavigation(permissions = []) {
  const all = permissions.includes("platform.admin");
  if (!all && permissions.includes("support.manage")) return [["/admin?section=support", "工单系统", Ticket]];
  return [
    ...(all ? [["/admin?section=support", "工单系统", Ticket]] : []),
    ...(all ? [["/admin?section=image-review", "图片审核", ScanEye]] : []),
    ["/merchant", "商店管理", Store],
    ...(all ? [["/admin?section=coupons", "优惠券", Ticket], ["/admin?section=desktop-launcher", "启动器", Monitor]] : []),
    ...(all || permissions.includes("intervention.manage") ? [["/admin?section=interventions", "平台介入", Gavel]] : []),
    ...(all || permissions.includes("announcements.manage") ? [["/admin?section=announcements", "公告管理", Megaphone]] : []),
    ...(all || permissions.includes("audit.read") ? [["/admin?section=audit", "管理审计", ChartNoAxesCombined]] : []),
    ...(all ? [["/admin?section=email", "邮件系统", Mail]] : []),
    ...(all ? [["/admin?section=ai", "小祥设置", Sparkles], ["/admin?section=accounts", "账号与权限", Users]] : []),
    ...(all ? [["/admin?section=whitelist", "白名单管理", ShieldCheck]] : []),
    ...(all ? [["/admin?section=ysm", "模型管理", Boxes], ["/admin?section=titles", "头衔管理", ShieldCheck]] : []),
    ...(all ? [["/admin?section=launcher-icon", "应用图标", Smartphone]] : []),
    ...(all || permissions.includes("core.read") ? [["/admin?section=core", "Core 管理", Server]] : []),
  ];
}
