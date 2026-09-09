async (page) => {
  await page.unroute("**/*");
  await page.addInitScript(() => {localStorage.setItem("deuterium-web-theme", "light");window.__v204Confirmations=0;window.confirm=()=>{window.__v204Confirmations++;return false;};});
  const media = "http://127.0.0.1:5244/media/store_watch.webp", now = "2026-09-09T02:00:00Z";
  const content = {title:"石材建造礼包",subtitle:"主城建造常用材料",description:"提供石材与照明材料，通过游戏内邮箱领取。",price:"12.30",brandId:"brand_one",categoryId:"category_one",deliveryTemplateRef:"template_one",deliverySummary:"游戏内邮箱领取",estimatedDelivery:"付款后自动发放",inventoryPolicy:"FINITE",stock:30,limitPerOrder:9,posterTone:"LIGHT",accentColor:"#4C78BD",sortOrder:0,galleryAssetIds:["asset_one"],coverAssetId:"asset_one",galleryAltTexts:["石材"],includedItems:["石材 × 64"],contentBlocks:[],badges:[]};
  let announcements=[{announcementId:"ann_one",title:"服务器维护通知",summary:"维护时间与安排",contentBlocks:[{blockId:"one",type:"PARAGRAPH",text:"维护公告正文。"}],priority:"NORMAL",pinned:false,status:"PUBLISHED",version:2}], product={productId:"product_one",storeId:"store_one",version:1,draft:content,draftImages:[{assetId:"asset_one",url:media}],visibility:"ACTIVE"};
  let settings={enabled:false,host:"smtp.example.com",port:587,security:"STARTTLS",username:"sender@example.com",from:"sender@example.com",recipients:["admin@example.com"],version:1,passwordConfigured:true};
  const order={orderId:"order_new",orderNo:"IX2040001",channel:"PLAYER_MARKET",status:"CONFIRMED",fundsStatus:"SETTLED",amount:"12.30",createdAt:now,updatedAt:now,buyer:{displayName:"青梧",playerRef:"player_buyer"},seller:{displayName:"白石",playerRef:"player_seller"},items:[{productId:"listing_one",title:"石材建造礼包",quantity:1,unitPrice:"12.30"}],delivery:{method:"PICKUP",location:"主城仓库"},availableActions:[],readOnly:true,images:[]};
  const events=[];
  await page.route("**/web-config.json",route=>route.fulfill({status:200,contentType:"application/json",body:JSON.stringify({mode:"connected",version:"2.0.4",configured:true})}));
  await page.route("**/api/v1/**",async route=>{
    const request=route.request(),path="/api/v1"+request.url().split("/api/v1")[1].split("?")[0],body=request.postDataJSON(),method=request.method();
    const json=(data,pageInfo={nextCursor:null,hasMore:false})=>route.fulfill({status:200,contentType:"application/json",body:JSON.stringify({data,page:pageInfo})});
    if(path==="/api/v1/web/session")return json({user:{userId:"fixture-admin",playerRef:"player_admin",gameId:"AuditAdmin",qq:"100001",permissions:["platform.admin"]},csrfToken:"fixture-csrf",expiresAt:"2030-01-01T00:00:00Z"});
    if(path==="/api/v1/admin/interventions/summary")return json({pending:2,submitted:1});
    if(path==="/api/v1/admin/interventions")return json([{caseId:"case_one",status:"SUBMITTED",applicant:{displayName:"青梧"},respondent:{displayName:"白石"},description:"申请核对商品交付情况与交易约定。",updatedAt:now}]);
    if(path==="/api/v1/admin/announcements"&&method==="GET")return json(announcements);
    if(path.endsWith("/announcements/ann_one/delete")){announcements=[];return json({announcementId:"ann_one",deleted:true});}
    if(path==="/api/v1/admin/email-settings"&&method==="GET")return json({settings,secretStorageReady:true,delivery:{pending:events.filter(e=>e.status!=="SENT").length,retrying:0,events}});
    if(path==="/api/v1/admin/email-settings"&&method==="PUT"){settings={...settings,...body,version:settings.version+1,passwordConfigured:true};delete settings.password;delete settings.clientRequestId;delete settings.expectedVersion;return json(settings);}
    if(path==="/api/v1/admin/email-settings/test"){events.push({eventId:"smtp_test_one",caseId:null,status:"SENT",attempts:1,createdAt:now,lastError:""});return json({eventId:"smtp_test_one",status:"PENDING"});}
    if(path==="/api/v1/admin/players")return json([{playerRef:"player_buyer",uuid:"10000000-0000-0000-0000-000000000001",gameId:"青梧",qq:"100002",registered:true,status:"active"},{playerRef:"player_seller",uuid:"10000000-0000-0000-0000-000000000002",gameId:"白石",qq:"100003",registered:true,status:"active"}]);
    if(path==="/api/v1/admin/transactions")return json([{recordId:"econ_9",player:{gameId:"青梧"},title:"游戏内支出",note:"游戏内支出",source:"GAME",direction:"expense",amount:"10.00",afterBalance:"900.00",occurredAt:now},{recordId:"econ_8",player:{gameId:"白石"},title:"玩家转账",otherPlayer:{gameId:"青梧"},direction:"income",amount:"12.30",afterBalance:"50.00",occurredAt:"2026-09-09T01:00:00Z"}]);
    if(path==="/api/v1/admin/orders")return json([order]);
    if(path==="/api/v1/admin/orders/order_new")return json(order);
    if(path==="/api/v1/admin/products")return json([{productId:"product_one",kind:"product",title:content.title,price:content.price,stock:30,state:"UNLISTED",createdAt:now}]);
    if(path==="/api/v1/admin/products/product_one")return json({...product,readOnly:true});
    if(path==="/api/v1/admin/audit-events")return json([{eventId:"123",actorId:"fixture-admin",action:"announcement.delete",resourceId:"ann_one",createdAt:now}]);
    if(path==="/api/v1/merchant/me")return json({storeIds:["store_one"]});
    if(path==="/api/v1/merchant/stores/store_one")return json({storeId:"store_one",name:"主城物资商店",intro:"常用物资与建造材料",contactQq:"100001",version:1});
    if(path.endsWith("/store_one/brands"))return json([{brandId:"brand_one",name:"Deuterium",active:true}]);
    if(path.endsWith("/store_one/categories"))return json([{categoryId:"category_one",name:"建筑材料",active:true}]);
    if(path.endsWith("/store_one/delivery-templates"))return json([{templateRef:"template_one",name:"石材模板",summary:"石材 × 64",active:true}]);
    if(path.endsWith("/store_one/products")&&method==="GET")return json([product]);
    if((path.endsWith("/store_one/products")&&method==="POST")||(path.endsWith("/products/product_one")&&method==="PUT")){product={...product,draft:body.content,version:product.version+1};return json(product);}
    if(path.endsWith("/products/product_one/publish")){product={...product,visibility:"ACTIVE",version:product.version+1};return json(product);}
    if(path==="/api/v1/assets/uploads")return json({uploadId:"upload_one",assetId:"asset_one",status:"READY",asset:{assetId:"asset_one",url:media,status:"READY"}});
    if(path==="/api/v1/notifications")return json([]);
    if(path==="/api/v1/notifications/preferences")return json({version:1});
    return route.fulfill({status:404,contentType:"application/json",body:JSON.stringify({error:{code:"NOT_FOUND",message:"隔离夹具未提供此接口"}})});
  });
  await page.goto("http://127.0.0.1:5244/admin");
  return {mode:"isolated browser contract fixtures; no production requests",url:page.url()};
}
