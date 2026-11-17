import React, { useState } from "react";

export function emojiAssetName(emoji) {
  const value = emoji.includes("\u200d") ? emoji : emoji.replaceAll("\ufe0f", "");
  return [...value].map(character => character.codePointAt(0).toString(16)).join("-");
}

export default function ChatEmoji({ emoji }) {
  const [failed, setFailed] = useState(false);
  return failed ? <span aria-hidden="true">{emoji}</span> : <img className="chat-emoji-art" src={`/media/emoji/${emojiAssetName(emoji)}.svg`} width="32" height="32" alt="" aria-hidden="true" draggable="false" onError={() => setFailed(true)} />;
}
