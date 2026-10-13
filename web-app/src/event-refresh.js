// One task per changed resource. No idle timer; only failed work backs off.
export class EventRefresh {
  tasks = new Map();
  closed = false;
  request(key, action) {
    if (this.closed) return;
    const existing = this.tasks.get(key);
    if (existing) { existing.dirty = true; return existing.promise; }
    const task = { dirty: true, attempts: 0, cancelDelay: null };
    this.tasks.set(key, task);
    task.promise = Promise.resolve().then(async () => {
      try {
        while (!this.closed) {
          task.dirty = false;
          try { await action(); task.attempts = 0; }
          catch {
            if (this.closed) break;
            task.attempts++;
            await new Promise(resolve => {
              const timer = setTimeout(resolve, Math.min(60_000, 1000 * 2 ** Math.min(task.attempts, 6)));
              task.cancelDelay = () => { clearTimeout(timer); resolve(); };
            });
            task.cancelDelay = null;
            continue;
          }
          if (!task.dirty) break;
        }
      } finally { this.tasks.delete(key); }
    });
    return task.promise;
  }
  close() { this.closed = true; for (const task of this.tasks.values()) task.cancelDelay?.(); }
}
