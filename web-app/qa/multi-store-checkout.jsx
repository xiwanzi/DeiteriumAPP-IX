import React, { useState } from "react";
import { createRoot } from "react-dom/client";
import PurchaseFlow, { CreationProgress } from "../src/PurchaseFlow.jsx";
import "../src/styles.css";
import "../src/v2.css";
import "../src/desktop-admin.css";
import "../src/commerce-v209.css";

const lines = [{productId:"a",title:"补给包",unitPrice:"10.00",quantity:1,subtotal:"10.00",productVersion:2},{productId:"b",title:"台灯",unitPrice:"20.00",quantity:1,subtotal:"20.00",productVersion:2}];
const groups = [{storeId:"a",storeName:"EOS Lab旗舰店",totalAmount:"7.00",couponDiscount:"3.00",coupon:{name:"开业礼遇"}},{storeId:"b",storeName:"宜家家居",totalAmount:"20.00",couponDiscount:"0.00"}];
const quote = {quoteId:"quote",version:1,channel:"OFFICIAL_STORE",items:lines,storeOrders:groups,storeName:"多店合并付款",totalAmount:"27.00",originalTotal:"30.00",productDiscount:"0.00",couponDiscount:"3.00",discountTotal:"3.00",expiresAt:new Date(Date.now()+300000).toISOString(),warnings:["按店铺分别生成订单和游戏邮件。"]};
const orders = groups.map((group,i)=>({orderId:`order_${i}`,orderNo:`DTEST${i}`,seller:{displayName:group.storeName},amount:group.totalAmount,fundsStatus:"HELD",pendingOperationId:null}));
const operation = {operationId:"operation_group",status:"COMPLETED",resourceType:"ORDER",resourceId:"order_0",resourceIds:orders.map(order=>order.orderId),amount:"27.00"};
window.checkoutQA = {calls:[],payments:0};
const client = {request:async(path,options={})=>{
  window.checkoutQA.calls.push({path,...options});
  let data;
  if(path.endsWith("/checkout/quotes"))data=quote;
  else if(path.endsWith("/store/orders")){window.checkoutQA.payments++;data={operation,order:orders[0],orders};}
  else if(path.includes("/operations/"))data=operation;
  else data=orders.find(order=>path.endsWith(`/${order.orderId}`));
  if(!data)throw new Error(`Unexpected fixture path: ${path}`);
  return {data,serverTime:new Date().toISOString()};
}};
const user = {userId:"isolated-checkout",gameId:"FixtureSelf"};
function App(){
  const [recover,setRecover]=useState(false),[selected,setSelected]=useState("");
  const entry={userId:user.userId,origin:location.origin,kind:"STORE_PURCHASE",operationId:"operation_group",body:{clientRequestId:"original-group"}};
  return <main style={{maxWidth:760,margin:"24px auto",padding:16}}><h1>跨店结算隔离验收</h1><button onClick={()=>setRecover(!recover)}>模拟断线恢复</button><p data-testid="selected">{selected}</p><section className="panel">{recover?<CreationProgress client={client} user={user} entry={entry} onResource={resource=>setSelected(resource.value.orderId)}/>:<PurchaseFlow client={client} user={user} source="CART" channel="OFFICIAL_STORE" items={lines.map(item=>({productId:item.productId,quantity:1,expectedProductVersion:2}))} previewProducts={lines} onResource={resource=>setSelected(resource.value.orderId)}/>}</section></main>;
}
createRoot(document.getElementById("root")).render(<App/>);
