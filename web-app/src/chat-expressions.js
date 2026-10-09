export const EMOJI_GROUPS = [
  {name: "表情", symbol: "☺", items: "😀 😃 😄 😁 😆 😅 😂 🤣 🥹 😊 😇 🙂 🙃 😉 😌 😍 🥰 😘 😋 😛 😜 🤪 🤨 🧐 🤓 😎 🥳 😏 😒 😔 😞 😢 😭 😤 😠 😡 🤯 😳 🥺 😱 😨 😰 😥 🤔 🤭 🫢 🫣 🤫 😶 😐 😑 🙄 😮 😲 🥱 😴 🤤 😷 🤒 🤕 🤢 🤮 🤧 🥵 🥶 🥴 😵 🤠 🥸 😈 👻 🤖 💩".split(" ")},
  {name: "手势", symbol: "✌", items: "👍 👎 👊 ✊ 🤛 🤜 🤞 ✌️ 🤟 🤘 👌 🤌 🤏 🫶 👐 🙌 👏 🤝 🙏 💪 👋 🤚 🖐️ ✋ 🖖 👈 👉 👆 👇 ☝️ 🫵 ✍️ 💅 👀".split(" ")},
  {name: "心情", symbol: "♡", items: "❤️ 🧡 💛 💚 💙 💜 🖤 🤍 🤎 💔 ❤️‍🔥 💕 💞 💓 💗 💖 💘 💝 💯 💢 💥 💫 💦 💤 ✨ 🎉 🎊 🎈 🎁 🎀 💐 🌹".split(" ")},
  {name: "自然", symbol: "❀", items: "🐱 🐶 🐭 🐹 🐰 🦊 🐻 🐼 🐨 🐯 🦁 🐷 🐸 🐵 🐣 🐧 🐦 🦆 🦋 🐝 🐢 🐬 🐳 🐟 🌿 🍀 🌱 🌴 🌵 🌸 🌼 🌻 🌙 ☀️ ⭐ 🌈 ☁️ ❄️ 🔥 💧".split(" ")},
  {name: "美食", symbol: "☕", items: "🍎 🍊 🍋 🍉 🍇 🍓 🫐 🍒 🍑 🥭 🍍 🥥 🥑 🍅 🌽 🥕 🍞 🥐 🥨 🧀 🍔 🍟 🍕 🌮 🍜 🍲 🍣 🍙 🍚 🥟 🍰 🎂 🍦 🍪 🍫 🍬 ☕ 🫖 🧋 🥤".split(" ")},
  {name: "活动", symbol: "⚑", items: "🎮 🕹️ 🎲 ♟️ 🎯 ⚽ 🏀 🏈 🎾 🏓 🏸 🏆 🥇 🎸 🎹 🎵 🎶 🎧 🎤 🎬 📷 📱 💻 💡 📚 ✏️ 🏡 🚗 🚲 ✈️ 🚀 ⛏️ 💎 🔑 🛠️ ✅ ❌ ❓ ❗ 🔔".split(" ")},
];
export const CHAT_EMOJI = EMOJI_GROUPS.flatMap(group => group.items);

// Percentages keep a circular corner proportional to the short edge at every display size.
export function stickerRadius(width, height) {
  const ratio = Math.max(1, width || 1) / Math.max(1, height || 1);
  return `${14 * Math.min(1, 1 / ratio)}% / ${14 * Math.min(1, ratio)}%`;
}

export function deleteChatGrapheme(text, start, end = start) {
  if (start !== end) return {text: text.slice(0, start) + text.slice(end), caret: start};
  if (!start) return {text, caret: 0};
  const before = text.slice(0, start);
  const parts = typeof Intl.Segmenter === "function" ? [...new Intl.Segmenter(undefined, {granularity: "grapheme"}).segment(before)].map(part => part.segment) : Array.from(before);
  const caret = start - (parts.at(-1)?.length || 0);
  return {text: text.slice(0, caret) + text.slice(end), caret};
}
