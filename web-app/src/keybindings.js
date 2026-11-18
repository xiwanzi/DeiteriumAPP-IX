const special = {
  unknown: "未绑定",
  space: "Space",
  enter: "Enter",
  escape: "Esc",
  backspace: "Backspace",
  tab: "Tab",
  up: "↑",
  down: "↓",
  left: "←",
  right: "→",
  home: "Home",
  end: "End",
  insert: "Insert",
  delete: "Delete",
  "page.up": "Page Up",
  "page.down": "Page Down",
  "left.shift": "左 Shift",
  "right.shift": "右 Shift",
  "left.control": "左 Ctrl",
  "right.control": "右 Ctrl",
  "left.alt": "左 Alt",
  "right.alt": "右 Alt",
  "left.win": "左 Win / Super",
  "right.win": "右 Win / Super",
  "caps.lock": "Caps Lock",
  "num.lock": "Num Lock",
  "scroll.lock": "Scroll Lock",
  "print.screen": "Print Screen",
  menu: "Menu",
  pause: "Pause",
  apostrophe: "'",
  backslash: "\\",
  comma: ",",
  equal: "=",
  "grave.accent": "`",
  "left.bracket": "[",
  "right.bracket": "]",
  minus: "-",
  period: ".",
  semicolon: ";",
  slash: "/",
  "world.1": "World 1",
  "world.2": "World 2",
};
const series = (n, fn) => Array.from({ length: n }, (_, i) => fn(i));
const keyboard = [
  ...Object.keys(special),
  ..."abcdefghijklmnopqrstuvwxyz0123456789",
  ...series(25, (i) => "f" + (i + 1)),
  ...[
    ..."0123456789",
    "add",
    "subtract",
    "multiply",
    "divide",
    "decimal",
    "enter",
    "equal",
  ].map((n) => "keypad." + n),
];
export const choices = [
  ...keyboard.map((k) => "key.keyboard." + k),
  ...["left", "right", "middle", "4", "5", "6", "7", "8"].map(
    (k) => "key.mouse." + k,
  ),
];
export const unbound = "key.keyboard.unknown";
export function keyName(value) {
  const [key, mod] = value.split(":");
  let name;
  if (key.startsWith("key.mouse.")) {
    const button = key.slice(10);
    name =
      { left: "鼠标左键", right: "鼠标右键", middle: "鼠标中键" }[button] ||
      "鼠标 " + button;
  } else if (key.startsWith("scancode.")) name = "扫描码 " + key.slice(9);
  else {
    const code = key.replace(/^key\.keyboard\./, "");
    name =
      special[code] ||
      (code.startsWith("keypad.")
        ? "小键盘 " + code.slice(7).toUpperCase()
        : code.toUpperCase());
  }
  return mod
    ? ({ SHIFT: "Shift", CONTROL: "Ctrl", ALT: "Alt" }[mod] || mod) +
        " + " +
        name
    : name;
}
export function keyStatus(row) {
  if (row.missing) return "当前不可用";
  if (row.conflict) return "修改冲突";
  return row.value === row.observed ? "已生效" : "待生效";
}
export function keyClashes(rows, action, value) {
  if (value === unbound) return [];
  const [key, mod] = value.split(":");
  const context = rows[action]?.context;
  const modifierOf = (k) =>
    ({
      "key.keyboard.left.shift": "SHIFT",
      "key.keyboard.right.shift": "SHIFT",
      "key.keyboard.left.alt": "ALT",
      "key.keyboard.right.alt": "ALT",
      "key.keyboard.left.control": "CONTROL",
      "key.keyboard.right.control": "CONTROL",
    })[k];
  return Object.entries(rows)
    .filter(([id, row]) => {
      if (
        id === action ||
        row.missing ||
        row.value === unbound ||
        (context === "game" && row.context === "gui") ||
        (context === "gui" && row.context === "game")
      )
        return false;
      const [other, otherMod] = row.value.split(":");
      if (
        (mod && mod === modifierOf(other)) ||
        (otherMod && otherMod === modifierOf(key))
      )
        return true;
      return (
        key === other &&
        (mod === otherMod ||
          ((!mod || !otherMod) && context !== "gui" && row.context !== "gui"))
      );
    })
    .map(([, row]) => row.label);
}
