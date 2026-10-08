import React from "react";
import { createRoot } from "react-dom/client";
import { CategoryFields, ShopBannerSettings } from "../src/ShopNavigationSettings.jsx";
import ProductEditor from "../src/ProductEditor.jsx";
import "../src/styles.css";
import "../src/v2.css";
import "../src/desktop-admin.css";
import "../src/commerce-v209.css";

const categories = [{categoryId:"root",name:"家具家居",active:true,sortOrder:0,version:1},{categoryId:"child",name:"床与寝具",active:true,sortOrder:2,version:4}];
const cover = {assetId:"cover",url:"/media/flower-shop.png"};
const home = {version:3,intro:"保留原介绍",brandIds:["brand"],categoryIds:["child"],sections:[{sectionId:"existing",title:"原商品区",layout:"PRODUCT_GRID",productIds:["product"],bannerAssetId:null,sortOrder:3},{sectionId:"banner",title:"夏日庭院",layout:"BANNER",productIds:[],bannerAssetId:"cover",sortOrder:0}],banners:[{sectionId:"banner",image:cover}]};
window.shopQA={calls:[]};
const client={request:async(path,options={})=>{window.shopQA.calls.push({path,...options});if(path.endsWith("/homepage"))return {data:options.body?{...home,...options.body,version:4}:home};return {data:{productId:"product",version:5,draft:options.body?.content||{},...options.body}}}};
const draft={title:"花园摆件",subtitle:"隔离配置验证",description:"保存原有内容和标签",brandId:"brand",categoryId:"child",price:"100.00",includedItems:["摆件 × 1"],deliveryTemplateRef:"template",inventoryPolicy:"UNLIMITED",stock:0,limitPerOrder:64,badges:["手作"],contentBlocks:[],galleryAssetIds:["cover"],galleryAltTexts:["摆件"]};
createRoot(document.getElementById("root")).render(<main style={{maxWidth:1100,margin:"24px auto",padding:20}}><h1>商城配置 · 隔离验收</h1><section className="panel"><h2>分类层级</h2><CategoryFields client={client} storeId="store" kind="category" categories={categories} initial={categories[1]} onSaved={()=>{}}/></section><section className="panel"><h2>首页活动图</h2><ShopBannerSettings client={client} storeId="store" onSaved={()=>{}}/></section><section className="panel"><h2>商品推荐标签</h2><ProductEditor client={client} storeId="store" initial={{productId:"product",version:4,draft,draftImages:[cover]}} brands={[{brandId:"brand",name:"测试品牌",active:true}]} categories={categories} templates={[{templateRef:"template",name:"游戏邮件",summary:"摆件 × 1",active:true}]} onSaved={()=>{}}/></section></main>);
