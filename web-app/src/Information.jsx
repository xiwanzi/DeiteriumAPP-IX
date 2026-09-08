import React, { useState } from "react";
import { ArrowLeft, MessageCircle } from "lucide-react";
import Messenger from "./Messenger.jsx";
import {
  PublicAnnouncements,
  publishedAnnouncements,
} from "./Announcements.jsx";
import { players } from "./data.js";
import { Avatar, Button, Badge, Modal } from "./components.jsx";
export default function Information({ state, act, open }) {
  const [view, setView] = useState("chat");
  const user = players.find((p) => p.id === state.user);
  const contacts = [
    {
      id: "public",
      channel: "public",
      name: "公共聊天",
      caption: "一起聊聊我们的世界",
    },
    {
      id: "assistant",
      channel: `ai:${state.user}`,
      name: "小祥 AI",
      caption: "你的社区小助手 · 示例回复",
    },
    ...players
      .filter((p) => p.id !== state.user)
      .map((p) => ({
        ...p,
        channel: `dm:${[state.user, p.id].sort().join(":")}`,
        caption: p.bio,
      })),
  ];
  if (view === "announcements")
    return (
      <div className="information-announcements">
        <button className="auth-back" onClick={() => setView("chat")}>
          <ArrowLeft size={16} />
          返回聊天
        </button>
        <PublicAnnouncements state={state} open={open} />
      </div>
    );
  return (
    <Messenger
      user={user}
      contacts={contacts}
      messages={state.messages.map((m) => ({
        ...m,
        senderName: players.find((p) => p.id === m.sender)?.name || m.sender,
      }))}
      drafts={state.chatDrafts}
      readPositions={state.readPositions}
      announcements={publishedAnnouncements(state)}
      onAnnouncement={(item) => open({ type: "announcement", item })}
      onAnnouncements={() => setView("announcements")}
      onSend={async (input) =>
        act({
          type: "MESSAGE",
          ...input,
          reply: input.reply ? { text: input.reply.text } : null,
        })
      }
      onDraft={(channel, text) => {
        if ((state.chatDrafts?.[`${state.user}:${channel}`] || "") !== text)
          act({ type: "CHAT_DRAFT", channel, text });
      }}
      onRead={(channel, id) => {
        if (state.readPositions?.[`${state.user}:${channel}`] !== id)
          act({ type: "CHAT_READ", channel, id });
      }}
      onProfile={(id) => {
        const item = players.find((p) => p.id === id);
        if (item) open({ type: "player", item });
      }}
    />
  );
}
export function PlayerDetail({ item: p, close, navigate }) {
  return (
    <Modal title="玩家资料" close={close}>
      <div className="player-detail">
        <Avatar user={p.id} size="large" />
        <h2>{p.name}</h2>
        <p>{p.bio}</p>
        <Badge tone="neutral">演示玩家</Badge>
      </div>
      <dl className="detail-list">
        <div>
          <dt>游戏 ID</dt>
          <dd>{p.name}</dd>
        </div>
        <div>
          <dt>QQ</dt>
          <dd>{p.qq}</dd>
        </div>
        <div>
          <dt>最近在线</dt>
          <dd>示例状态</dd>
        </div>
      </dl>
      <Button
        onClick={() => {
          close();
          navigate("/information");
        }}
      >
        <MessageCircle size={17} />
        前往消息
      </Button>
    </Modal>
  );
}
