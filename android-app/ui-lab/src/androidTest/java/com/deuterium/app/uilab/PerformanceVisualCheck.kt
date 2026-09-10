package com.deuterium.app.uilab

import android.app.Instrumentation
import android.content.Intent
import android.graphics.Bitmap
import android.os.Bundle
import android.os.Handler
import android.os.Looper
import android.view.PixelCopy
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.rememberGraphicsLayer
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.unit.*
import kotlinx.coroutines.*
import org.json.JSONObject
import java.time.LocalDateTime
import java.time.LocalDate
import java.time.ZoneId
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicInteger

/** Same fixture on the original and candidate APK. PixelCopy excludes the system-bar surfaces.
 * No login, requests, purchases, notifications or production records are part of this check. */
object PerformanceVisualCheck {
    fun run(test:Instrumentation,label:String) {
        require(label.matches(Regex("[a-zA-Z0-9-]{1,40}")))
        val result=Bundle()
        val scope=CoroutineScope(SupervisorJob()+Dispatchers.Main.immediate)
        var host:DeuteriumActivity?=null
        val output=test.targetContext.filesDir.resolve("performance-$label").apply{mkdirs()}
        try {
            val photo=test.targetContext.filesDir.resolve("performance-photo.png")
            val bitmap=Bitmap.createBitmap(400,300,Bitmap.Config.ARGB_8888)
            android.graphics.Canvas(bitmap).apply {
                drawColor(android.graphics.Color.rgb(216,230,241))
                val paint=android.graphics.Paint(android.graphics.Paint.ANTI_ALIAS_FLAG)
                paint.color=android.graphics.Color.rgb(36,86,128);drawCircle(200f,150f,94f,paint)
                paint.color=android.graphics.Color.rgb(229,193,116);drawRect(185f,32f,215f,268f,paint)
            }
            photo.outputStream().use{bitmap.compress(Bitmap.CompressFormat.PNG,100,it)};bitmap.recycle()
            val source=android.net.Uri.fromFile(photo).toString()
            val time=LocalDate.now(ZoneId.of("Asia/Shanghai")).atTime(12,0)
            lateinit var state:LabState
            host=test.startActivitySync(Intent(test.targetContext,DeuteriumActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) as DeuteriumActivity
            test.runOnMainSync {
                state=LabState(scope,initialFollowed=setOf("玩家01"),userName="FixtureSelf")
                state.balance=987654321;state.balanceKnown=true;state.profileBio="同样的画面与交互"
                Players.clear();ShopCatalog.clear()
                repeat(24){i->val name="玩家"+(i+1).toString().padStart(2,'0')
                    Players.add(PlayerProfile(name,qq="100000${i}",bio="一起建造世界",online=i%3==0,playerRef="fixture_$i",avatar=source))
                    state.conversation(name).add(ChatLine(i.toLong(),name,"固定的私聊预览 ${i+1}",false,"12:00",remoteId="message_$i",serverAt=(24-i).toLong()))
                    if(i%4==0)state.conversationUnread[name]=1
                }
                repeat(8){i->
                    ShopCatalog.add(ShopProduct("product_$i","探索者补给 ${i+1}",if(i%2==0)"补给" else "装备","来自游戏世界的精选物品",123450+i*100L,Color(0xFFD8E6F1),photos=listOf(source),brand="Deuterium",stock=20))
                    state.commerce.listings.add(MarketListing("listing_$i","玩家商品 ${i+1}","固定的简介","商品说明","补给",50000+i*100L,10,"玩家01","1000000",setOf(DeliveryMethod.Pickup),"主城",imageUri=source,createdAt=time.minusHours(i.toLong())))
                    state.ledger.add(LedgerEntry(i+1L,"玩家01","固定交易记录",if(i%2==0)123450 else -987654321,"12:00",time.minusDays(i.toLong())))
                    state.commerce.orders.add(CommerceOrder("order_$i","key_$i",OrderChannel.Official,"FixtureSelf","Deuterium 商店","1000000",listOf(OrderLine("product_$i","探索者补给 ${i+1}","固定简介",123450,2,imageUri=source)),DeliveryMethod.Mailbox,"",time.minusHours(i.toLong()),OrderStage.AwaitingClaim))
                    state.commissions.entries.add(Commission("commission_$i","key_$i","玩家01",CommissionDraft("一起建造主城 ${i+1}","固定说明","主城",123450,24,Urgency.Normal,source),time.minusHours(i.toLong())))
                }
            }
            val scenes=listOf("shop-0","shop-34","shop-98","shop-128","market","info","wallet","bills","orders","commission","controls","glass","flight-0","flight-25","flight-50","flight-75","flight-100")
            var captured=0
            for(theme in listOf(1,2))for(font in listOf(1f,1.4f))for(scene in scenes){
                test.runOnMainSync { host!!.setContent {
                    key(scene,theme,font){LabTheme(theme,false){
                        val density=LocalDensity.current
                        val backdrop=rememberGraphicsLayer()
                        val chrome=remember{PageChromeState(scene.substringAfter("shop-","0").toFloatOrNull() ?: 0f)}
                        val tilt=remember{mutableStateOf(Offset(.23f,-.17f))}
                        CompositionLocalProvider(LocalDensity provides Density(density.density,font),LocalContentColor provides MaterialTheme.colorScheme.onSurface,
                            LocalOverlayBackdrop provides backdrop,LocalDeviceTilt provides tilt) {
                            Box(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.background)) {
                                Box(Modifier.fillMaxSize().recordGlassBackdrop(backdrop)) {
                                    GlassAtmosphere()
                                    when {
                                        scene.startsWith("shop-")->ShopPage(state,{},208.dp-chrome.offset.dp,"")
                                        scene=="market"->MarketPage(state.commerce,"",64.dp,{})
                                        scene=="info"->InfoPage(state,24.dp,""){}
                                        scene=="wallet"->WalletPage(state,{},{},24.dp)
                                        scene=="bills"->BillHistoryPage(state,"all",24.dp){}
                                        scene=="orders"->OrdersPage(state.commerce,24.dp){}
                                        scene=="commission"->CommissionHallPage(state,24.dp,{})
                                        scene=="controls"->Column(Modifier.padding(24.dp),verticalArrangement=Arrangement.spacedBy(18.dp)) {
                                            Text("原效果控件",style=MaterialTheme.typography.headlineLarge)
                                            IosSwitch(false,{});IosSwitch(true,{});IosSwitch(true,{},enabled=false)
                                            SegmentedControl(listOf("全部","进行中","已完成"),1,{})
                                            IosSlider(2.37f,{},1f..4f,Modifier.fillMaxWidth())
                                            MotionButton({}){Text("确认操作")}
                                            AdaptiveMoney("123,456,789,012.34",Modifier.fillMaxWidth())
                                            MarkdownBody("**保持原样** · ~~旧价格~~\n\n|项目|内容|\n|---|---|\n|状态|已确认|\n\n`固定代码`")
                                        }
                                        scene=="glass"->Column(Modifier.padding(24.dp),verticalArrangement=Arrangement.spacedBy(18.dp)) {
                                            repeat(6){i->Surface(color=if(i%2==0)Color(0xFF007AFF) else Color(0xFFFFB83F),modifier=Modifier.fillMaxWidth().height(82.dp)){Text("动态材质背景 $i",Modifier.padding(18.dp))}}
                                        }
                                    }
                                }
                                if(scene.startsWith("shop-"))CollapsingChrome("商城","搜索商品",chrome,24.dp,backdrop,{}){}
                                if(scene=="glass") {
                                    GradientGlassHeader(backdrop,Modifier.fillMaxWidth().height(160.dp))
                                    LiquidGlass(Modifier.align(Alignment.Center).padding(horizontal=24.dp).fillMaxWidth().height(140.dp),backdrop=backdrop){Text("柔光玻璃",Modifier.align(Alignment.Center),style=MaterialTheme.typography.headlineSmall)}
                                }
                                if(scene.startsWith("flight-"))ShoppingBagFlight(ShopCatalog.first(),Offset(120f,650f),Offset(570f,90f),scene.substringAfter('-').toFloat()/100f)
                                if(scene.startsWith("shop-")||scene=="market"||scene=="info")LiquidGlass(Modifier.align(Alignment.BottomCenter).padding(horizontal=18.dp,vertical=9.dp).fillMaxWidth().height(70.dp),radius=30.dp,backdrop=backdrop,parameters=GlassMaterials().bottomBar){MovingGlassNav(listOf("商城","市场","信息","我的"),listOf(DeuteriumIcons.Shop,DeuteriumIcons.Market,DeuteriumIcons.Messages,DeuteriumIcons.Person),if(scene=="market")1 else if(scene=="info")2 else 0,true){}}
                            }
                        }
                    }}
                }}
                test.waitForIdleSync();Thread.sleep(400)
                val window=host!!.window
                val view=window.decorView
                val pixels=Bitmap.createBitmap(view.width,view.height,Bitmap.Config.ARGB_8888)
                val copied=CountDownLatch(1);val code=AtomicInteger(-1)
                PixelCopy.request(window,pixels,{value->code.set(value);copied.countDown()},Handler(Looper.getMainLooper()))
                check(copied.await(5,TimeUnit.SECONDS)&&code.get()==PixelCopy.SUCCESS){"PixelCopy failed for $scene: ${code.get()}"}
                output.resolve("$scene-t$theme-f${(font*10).toInt()}.png").outputStream().use{pixels.compress(Bitmap.CompressFormat.PNG,100,it)}
                pixels.recycle();captured++
            }
            output.resolve("result.json").writeText(JSONObject().put("screenshots",captured).put("label",label).put("fixture","local-only").put("pixelSource","application-window").toString(2))
            result.putInt("screenshots",captured);result.putString("performanceVisual","PASS");test.finish(-1,result)
        } catch(failure:Throwable){result.putString("failure",failure.stackTraceToString());test.finish(1,result)}
        finally{scope.cancel();host?.let{test.runOnMainSync{it.finish()}}}
    }
}
