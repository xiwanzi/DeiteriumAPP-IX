import test from "node:test";
import assert from "node:assert/strict";
import { EventRefresh } from "./event-refresh.js";

test("events arriving during a request coalesce into one follow-up; unrelated resources proceed", async () => {
  const refresh = new EventRefresh(); let release, runs = 0, other = 0;
  const blocked = new Promise(resolve => { release = resolve; });
  const work = async () => { runs++; if(runs === 1) await blocked; };
  const done = refresh.request("one", work); await Promise.resolve();
  for(let i=0;i<100;i++) refresh.request("one",work);
  await refresh.request("two",async()=>{other++;});
  assert.equal(other,1); assert.equal(runs,1);
  release(); await done;
  assert.equal(runs,2); assert.equal(refresh.tasks.size,0);
  refresh.close(); refresh.request("one",work); assert.equal(refresh.tasks.size,0);
});

test("closing cancels failure backoff without starting another attempt", async () => {
  const refresh=new EventRefresh();let calls=0;
  const done=refresh.request("failed",async()=>{calls++;throw new Error("offline");});
  await Promise.resolve();await Promise.resolve();
  refresh.close();await done;
  assert.equal(calls,1);assert.equal(refresh.tasks.size,0);
});
