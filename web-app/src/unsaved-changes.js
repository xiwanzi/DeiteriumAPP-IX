import { useEffect, useRef } from "react";

const guards = new Map();
export function confirmNavigation() {
  for (const guard of guards.values()) if (!guard()) return false;
  return true;
}

export function useUnsavedChanges(value, busy = false) {
  const initial = useRef(JSON.stringify(value)), current = useRef(null), token = useRef(Symbol("editor"));
  current.current = { dirty: JSON.stringify(value) !== initial.current, busy };
  useEffect(() => {
    const allow = () => {
      if (current.current.busy) return false;
      return !current.current.dirty || window.confirm("还有未保存的修改。确认放弃这些修改并离开？");
    };
    guards.set(token.current, allow);
    const beforeUnload = (event) => { if (current.current.dirty || current.current.busy) { event.preventDefault(); event.returnValue = ""; } };
    window.addEventListener("beforeunload", beforeUnload);
    return () => { guards.delete(token.current); window.removeEventListener("beforeunload", beforeUnload); };
  }, []);
  return { dirty: current.current.dirty, markSaved: (nextValue = value) => { initial.current = JSON.stringify(nextValue); current.current.dirty = false; } };
}
