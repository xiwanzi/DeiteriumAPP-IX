import React, { useEffect, useRef, useState } from "react";
import { confirmNavigation } from "./unsaved-changes.js";
import {
  ShieldCheck,
  Server,
  Package,
  LogOut,
  RefreshCw,
  ArrowRight,
  LockKeyhole,
  Info,
  WifiOff,
  LoaderCircle,
} from "lucide-react";
import AppShell from "./AppShell.jsx";
import ConnectedWallet from "./ConnectedWallet.jsx";
import RemoteCollection from "./RemoteCollection.jsx";
import AuthScreen from "./AuthScreen.jsx";
import OfficialAdmin from "./OfficialAdmin.jsx";
import CatalogManagement from "./CatalogManagement.jsx";
import BusinessPages from "./BusinessPages.jsx";
import ConnectedCart from "./ConnectedCart.jsx";
import ConnectedChat from "./ConnectedChat.jsx";
import ConnectedProfile, {PlayerProfile, ConnectedNotificationSettings} from "./ConnectedProfile.jsx";
import {id} from "./format.js";
import {
  DeuteriumClient,
  createChatConnection,
  normalizeMessage,
  mergeMessages,
  recoveryCursor,
} from "./api.js";
import { Button, Modal, Empty, Avatar, Badge } from "./components.jsx";

export default function ConnectedApp() {
  const [session, setSession] = useState(null),
    [loading, setLoading] = useState(true),
    [failure, setFailure] = useState(""),
    [path, setPath] = useState(location.pathname),
    [search, setSearch] = useState(location.search),
    [dark, setDark] = useState(
      () => localStorage.getItem("deuterium-web-theme") === "dark",
    ),
    [modal, setModal] = useState(null),
    [notice, setNotice] = useState(""),
    [requestedConversation, setRequestedConversation] = useState(null);
  const sessionRef = useRef(null),
    clientRef = useRef(null);
  const currentUrl = useRef(location.href);
  if (!clientRef.current)
    clientRef.current = new DeuteriumClient({
      onSession: (value) => applySession(value),
      onUnauthorized: () => {
        if (sessionRef.current) setNotice("会话已失效，请重新登录。");
        sessionRef.current = null;
        setSession(null);
        setModal(null);
        setRequestedConversation(null);
      },
    });
  const client = clientRef.current;
  const [ownProfile, setOwnProfile] = useState(null);
  useEffect(() => {
    let active = true;
    setOwnProfile(null);
    if (session?.user.playerRef) client.profile(session.user.playerRef).then((r) => { if (active) setOwnProfile(r.data); }).catch(() => {});
    return () => { active = false; };
  }, [client, session?.user.playerRef]);
  function applySession(s) {
    if (!s?.user?.userId) throw new Error("账号响应格式不正确，请重新连接。");
    if (sessionRef.current?.user.userId !== s.user.userId) { setRequestedConversation(null); setModal(null); }
    sessionRef.current = s;
    setSession(s);
    setFailure("");
  }
  const restore = async () => {
    setLoading(true);
    setFailure("");
    try {
      applySession(await client.restore());
    } catch (e) {
      if (e.status !== 401) setFailure(e.message);
    } finally {
      setLoading(false);
    }
  };
  useEffect(() => {
    restore();
    const pop = () => {
      if (!confirmNavigation()) { history.pushState({}, "", currentUrl.current); return; }
      currentUrl.current = location.href;
      setPath(location.pathname);
      setSearch(location.search);
      setModal(null);
    };
    window.addEventListener("popstate", pop);
    return () => {
      window.removeEventListener("popstate", pop);
      client.clear();
    };
  }, []);
  useEffect(() => {
    if (session)
      document.title =
        ({
          "/information": "信息",
          "/admin": "官方管理",
          "/merchant": "商店管理",
          "/me": "我的",
          "/": "商城",
          "/wallet": "钱包",
          "/market": "市场",
          "/commissions": "委托",
        }[path] || "Deuterium") + " · Deuterium";
  }, [path, session]);
  useEffect(() => {
    const key = (e) => {
      if (
        e.key === "/" &&
        session &&
        !e.target.closest("input,textarea,select,[contenteditable=true]")
      ) {
        e.preventDefault();
        if (path === "/information")
          window.dispatchEvent(new Event("deuterium:search"));
        else setModal({ type: "capability" });
      }
    };
    window.addEventListener("keydown", key);
    return () => window.removeEventListener("keydown", key);
  }, [path, session]);
  useEffect(() => {
    document.documentElement.dataset.theme = dark ? "dark" : "light";
    localStorage.setItem("deuterium-web-theme", dark ? "dark" : "light");
  }, [dark]);
  const navigate = (to) => {
    if (!confirmNavigation()) return;
    if (location.pathname + location.search !== to) history.pushState({}, "", to);
    currentUrl.current = location.href;
    setPath(new URL(to, location.origin).pathname);
    setSearch(new URL(to, location.origin).search);
    setModal(null);
    window.scrollTo(0, 0);
  };
  const logout = async () => {
    try {
      await client.logout();
      sessionRef.current = null;
      setSession(null);
      setModal(null);
      setRequestedConversation(null);
      setNotice("你已安全退出。");
    } catch (e) {
      setNotice(e.message);
    }
  };
  if (loading)
    return (
      <div className="connection-empty">
        <LoaderCircle className="spin" size={28} />
        <p>正在恢复 Deuterium ID 会话…</p>
      </div>
    );
  if (failure && !session)
    return (
      <div className="connection-empty">
        <span className="stat-icon blue">
          <WifiOff size={27} />
        </span>
        <h1>暂时无法连接后端</h1>
        <p>{failure}</p>
        <Button onClick={restore}>
          <RefreshCw size={16} />
          重新连接
        </Button>

      </div>
    );
  if (!session)
    return (
      <AuthScreen
        mode="foundation"
        statusMessage={notice}
        dark={dark}
        toggleTheme={() => setDark(!dark)}
        onLogin={async ({ account, password }) => {
          applySession(await client.login(account, password));
          setNotice("");
          navigate("/information");
        }}
        onVerify={(type, fields) => client.requestVerification(type, fields)}
        onComplete={async (type, fields) => { const result = await client.completeVerification(type, fields); if (type === "register") { applySession(result.data); navigate("/information"); } }}
      />
    );
  const user = {
    ...session.user,
    avatar: ownProfile?.avatar || session.user.avatar,
    id: session.user.userId,
    name: session.user.gameId,
    qq: session.user.qq,
    admin: (session.user.permissions || session.permissions || []).some((permission) => ["core.read", "commerce.audit", "announcements.manage", "intervention.manage", "audit.read", "platform.admin", "ADMIN"].includes(permission)),
  };
  const openUnavailable = () => setModal({ type: "capability" });
  return (
    <>
      <AppShell
        key={session.user.userId}
        user={user}
        path={path}
        search={search}
        navigate={navigate}
        mode="foundation"
        dark={dark}
        toggleTheme={() => setDark(!dark)}
        onSearch={() =>
          path === "/information"
            ? window.dispatchEvent(new Event("deuterium:search"))
            : openUnavailable()
        }
        onNotifications={() => navigate("/notifications")}
        onCart={() => navigate("/cart")}
        onAbout={() => setModal({ type: "about" })}
      >
        {path === "/information" ? (
          <ConnectedChat
            key={session.user.userId}
            client={client}
            user={session.user}
            requestedConversation={new URLSearchParams(search).get("conversation") || requestedConversation}
            onProfile={(p) => setModal({ type: "player", player: p })}
            onUnavailable={() => navigate("/announcements")}
          />
        ) : path === "/wallet" ? (
          <ConnectedWallet client={client} user={session.user} />
        ) : path === "/cart" ? (
          <ConnectedCart client={client} user={session.user} navigate={navigate} />
        ) : path === "/orders" ? (
          <BusinessPages client={client} user={session.user} type="ORDER" />
        ) : path === "/commissions" ? (
          <BusinessPages client={client} user={session.user} type="COMMISSION" />
        ) : path === "/me" ? (
          <ConnectedProfile client={client} user={session.user} onLogout={logout} navigate={navigate} onProfileUpdated={setOwnProfile} />
        ) : path === "/notification-settings" ? (
          <ConnectedNotificationSettings client={client} />
        ) : path === "/admin" ? (
          <OfficialAdmin client={client} user={session.user} navigate={navigate} search={search} />
        ) : path === "/merchant" ? (
          <CatalogManagement client={client} user={session.user} />
        ) : (
          <RemoteCollection client={client} path={path} user={session.user} navigate={navigate} onNotificationTarget={(target) => {
            if (target.kind === "CONVERSATION") { setRequestedConversation(target.referenceId); navigate("/information"); }
            else if (target.kind === "ORDER" || target.kind === "REFUND") navigate(`/orders?order=${encodeURIComponent(target.referenceId)}`);
            else if (target.kind === "COMMISSION") navigate(`/commissions?commission=${encodeURIComponent(target.referenceId)}`);
            else navigate(({PUBLIC_CHAT:"/information",WALLET:"/wallet",ANNOUNCEMENT:"/announcements"}[target.kind]) || "/me");
          }} />
        )}
      </AppShell>
      {modal?.type === "about" && (
        <Modal title="Deuterium Web" close={() => setModal(null)}>
          <h2>2.0.7</h2><p className="description">属于我们的世界。与 App 共用 Deuterium ID，连接游戏中的朋友和每一份创造。</p>
          <p className="muted">账号、聊天和交易以服务器记录为准。</p>
        </Modal>
      )}
      {modal?.type === "capability" && (
        <Modal title="服务暂不可用" close={() => setModal(null)}>
          <p className="description">这项服务暂未开放，请稍后再来。</p>
        </Modal>
      )}
      {modal?.type === "player" && (
        <Modal title="玩家资料" close={() => setModal(null)}>
          <PlayerProfile client={client} user={session.user} playerRef={modal.player.id} onMessage={async (player) => {
            const result = await client.createConversation({clientRequestId:id(),otherPlayerRef:player.playerRef});
            setRequestedConversation(result.data.conversationId); navigate("/information");
          }} />
        </Modal>
      )}
      {notice && (
        <div className="toast-region" role="status">
          <div className="toast">
            <Info size={17} />
            <span>{notice}</span>
            <button onClick={() => setNotice("")}>关闭</button>
          </div>
        </div>
      )}
    </>
  );
}
