export async function readAiEvents(response, onEvent) {
  if (!response.body) throw new Error("回复连接没有返回数据。");
  const reader = response.body.getReader(), decoder = new TextDecoder(); let buffer = "", completed = false;
  const consume = async (block) => {
    let event = "message"; const data = [];
    for (const line of block.split(/\r?\n/)) {
      if (line.startsWith("event:")) event = line.slice(6).trim();
      else if (line.startsWith("data:")) data.push(line.slice(5).replace(/^ /, ""));
    }
    if (!data.length) return;
    const payload = JSON.parse(data.join("\n"));
    await onEvent(event, payload); if (event === "done") completed = true;
  };
  try {
    while (true) {
      const chunk = await reader.read(); if (chunk.done) break;
      buffer += decoder.decode(chunk.value, { stream: true });
      if (buffer.length > 1024 * 1024) throw new Error("回复数据过大，请重新查看会话。");
      let match;
      while ((match = /\r?\n\r?\n/.exec(buffer))) { const block = buffer.slice(0, match.index); buffer = buffer.slice(match.index + match[0].length); await consume(block); }
    }
    buffer += decoder.decode(); if (buffer.trim()) await consume(buffer);
    if (!completed) throw new Error("回复连接已中断，请继续查看同一次回复。");
  } finally { if (!completed) await reader.cancel().catch(() => {}); reader.releaseLock(); }
}

export function aiSources(sources) {
  return (Array.isArray(sources) ? sources : []).flatMap((source) => {
    try { const url = new URL(source.url); if (!["https:", "http:"].includes(url.protocol) || url.username || url.password) return []; return [{ title: source.title || url.hostname, url: url.href, origin: source.origin === "provider_text" ? "provider_text" : "annotation" }]; }
    catch { return []; }
  });
}
