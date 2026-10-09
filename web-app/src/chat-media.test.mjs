import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { outputSize, webpChunks, animatedWebp } from "./chat-media-codec.js";
import { normalizeMessage, mergeMessages, DeuteriumClient } from "./api.js";
import { acquireChatMedia } from "./chat-media.js";
import { deleteChatGrapheme } from "./chat-expressions.js";

test("expression backspace removes full joined emoji and preserves text around the caret",()=>{
  for(const emoji of ["❤️‍🔥","👨‍👩‍👧‍👦","👍🏽","🇨🇳","1️⃣"]){
    assert.deepEqual(deleteChatGrapheme("前"+emoji+"后",1+emoji.length),{text:"前后",caret:1});
  }
  assert.deepEqual(deleteChatGrapheme("你好世界",1,3),{text:"你界",caret:1});
  assert.deepEqual(deleteChatGrapheme("你好",0),{text:"你好",caret:0});
});

test("chat media sizing keeps long screenshot text readable and never enlarges small stickers",()=>{
  assert.deepEqual(outputSize(4000,3000),{width:1920,height:1440,long:false});
  assert.deepEqual(outputSize(1080,6500),{width:1080,height:6500,long:true});
  assert.deepEqual(outputSize(160,120,true),{width:160,height:120,long:false});
  const long=outputSize(1440,18000);assert.ok(long.height<=8192&&long.width*long.height<=8_000_000);
  assert.throws(()=>outputSize(20000,20000));
});
test("animation container keeps timing and excludes static metadata chunks",()=>{
  const bytes=readFileSync(new URL("../../backend-next/internal/objectstorage/testdata/sticker.webp",import.meta.url));
  const animated=animatedWebp([{bytes,duration:80},{bytes,duration:120}],512,512,0),chunks=webpChunks(animated);
  assert.deepEqual(chunks.map(c=>c.type),["VP8X","ANIM","ANMF","ANMF"]);
  assert.equal(chunks[2].data[12],80);assert.equal(chunks[3].data[12],120);assert.equal(chunks[2].data[15],2);
  assert.throws(()=>animatedWebp([{bytes,duration:40000}],512,512));
  assert.throws(()=>webpChunks(animated.slice(0,-1)));
});
test("authoritative media receipt reconciles optimistic message without losing attachment",()=>{
  const incoming=normalizeMessage({messageId:"m",clientMessageId:"c",content:"[图片]",kind:"public_chat",sentAt:"2026-10-09T00:00:00Z",sender:{playerRef:"p"},media:{mediaId:"image",width:512,height:512}},{playerRef:"p"});
  const rows=mergeMessages([{id:"pending:c",clientMessageId:"c",channel:"public",mine:true}], [incoming]);
  assert.equal(rows.length,1);assert.equal(rows[0].media.mediaId,"image");
});
test("background media controller survives route detach and stops on account switch",()=>{
  const client={request:async()=>({data:{stickers:[]}})};
  const first=acquireChatMedia(client,{playerRef:"alice"},()=>{},()=>{});
  first.update=()=>{};first.accepted=()=>{};
  assert.equal(acquireChatMedia(client,{playerRef:"alice"},()=>{},()=>{}),first);
  const next=acquireChatMedia(client,{playerRef:"bob"},()=>{},()=>{});
  assert.notEqual(first,next);assert.equal(first.stopped,true);next.stop();
});
test("media requests carry expected browser identity and clearing session stops outbox",async()=>{
  let headers,stopped=false,unauthorized=false;
  const client=new DeuteriumClient({fetchImpl:async(_,options)=>{headers=options.headers;return new Response(JSON.stringify(unauthorized?{error:{code:"UNAUTHORIZED",message:"expired"}}:{data:{stickers:[]}}),{status:unauthorized?401:200,headers:{"Content-Type":"application/json"}})}});
  await client.request("/api/v1/chat/stickers",{expectedPlayerRef:"alice"});assert.equal(headers["X-Deuterium-Actor"],"alice");
  client.chatMediaSession={stop(){stopped=true}};client.clear();assert.equal(stopped,true);
  stopped=false;unauthorized=true;client.chatMediaSession={stop(){stopped=true}};await assert.rejects(client.request("/api/v1/chat/stickers"));assert.equal(stopped,true);
});
