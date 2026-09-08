package com.deuterium.app.uilab

import android.net.Uri
import android.content.Intent
import android.os.Build
import android.Manifest
import android.hardware.Sensor
import android.hardware.SensorManager
import android.provider.Settings
import androidx.compose.ui.platform.LocalContext
import androidx.compose.material.icons.automirrored.outlined.ArrowBack
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.lifecycleScope
import androidx.lifecycle.compose.LocalLifecycleOwner
import android.os.Bundle
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.activity.compose.*
import androidx.activity.enableEdgeToEdge
import androidx.activity.result.PickVisualMediaRequest
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.animation.*
import androidx.compose.animation.core.*
import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.Chat
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.layout.onGloballyPositioned
import androidx.compose.ui.layout.positionInRoot
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.input.nestedscroll.nestedScroll
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.ui.graphics.rememberGraphicsLayer
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.core.view.WindowCompat
import kotlinx.coroutines.*

class MainActivity : ComponentActivity() {
    private var incomingRoute by mutableStateOf<String?>(null)
    private var nativeLaunchReady by mutableStateOf(false)
    private fun syncSystemTheme(theme:Int) {
        if(Build.VERSION.SDK_INT>=31)getSystemService(android.app.UiModeManager::class.java).setApplicationNightMode(
            when(theme){1->android.app.UiModeManager.MODE_NIGHT_NO;2->android.app.UiModeManager.MODE_NIGHT_YES;else->android.app.UiModeManager.MODE_NIGHT_AUTO})
    }
    override fun onNewIntent(intent:Intent) { super.onNewIntent(intent);setIntent(intent);incomingRoute=intent.getStringExtra("route") }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        nativeLaunchReady=Build.VERSION.SDK_INT<31||savedInstanceState!=null
        if(Build.VERSION.SDK_INT>=31)splashScreen.setOnExitAnimationListener { splash ->splash.remove();nativeLaunchReady=true}
        incomingRoute=intent.getStringExtra("route")
        val prefs = getSharedPreferences("ui-lab", MODE_PRIVATE)
        syncSystemTheme(prefs.getInt("theme",1))
        prefs.edit().remove("dynamic").remove("accent").apply()
        prefs.edit().remove("demoSignedIn").remove("demoUser").apply()
        val api=BackendApi.get(this)
        val images=AppImages.get(this)
        if(api.signedIn)prefs.getString("avatar:${api.userName}",null)?.let{source->lifecycleScope.launch{runCatching{images.bitmap(source,256)}}}
        val materialStore=GlassSettingsStore(prefs)
        val session = androidx.lifecycle.ViewModelProvider(this)[LabSession::class.java]
        val updates=androidx.lifecycle.ViewModelProvider(this)[AppUpdates::class.java]
        updates.initialize()
        setContent {
            var theme by rememberSaveable { mutableIntStateOf(prefs.getInt("theme", 1)) }
            var motion by rememberSaveable { mutableStateOf(prefs.getBoolean("motion", true)) }
            var tilt by rememberSaveable { mutableStateOf(prefs.getBoolean("tilt",true)) }
            var overlayGlass by rememberSaveable{mutableStateOf(prefs.getBoolean("overlayGlass",true))}
            var glass by rememberSaveable { mutableStateOf(prefs.getBoolean("glass", true)) }
            var cropSource by rememberSaveable { mutableStateOf<String?>(null) }
            var avatar by remember { mutableStateOf(prefs.getString("avatar:${api.userName}",null).takeIf{api.signedIn}) }
            val userName = api.userName
            val signedIn = api.signedIn
            LaunchedEffect(signedIn,userName){
                if(!signedIn)avatar=null else {
                    avatar=prefs.getString("avatar:$userName",null)
                    runCatching{api.request("GET","/players/${api.playerRef}")}.onSuccess{profile->avatar=profile.optJSONObject("avatar")?.optString("assetId")?.takeIf{it.isNotBlank()}?.let{"asset:$it"};prefs.edit().putString("avatar:$userName",avatar).apply()}
                }
            }
            LaunchedEffect(signedIn) { if(!signedIn)session.clearSession() }
            var params by remember { mutableStateOf(materialStore.load()) }
            var paymentAssetsReady by remember { mutableStateOf(FacePayAssets.movie!=null) }
            LaunchedEffect(Unit) {
                PaymentSound.prepare(this@MainActivity)
                runCatching { FacePayAssets.load(this@MainActivity) }
                paymentAssetsReady=true
            }
            val scope = rememberCoroutineScope()
            val picker = rememberLauncherForActivityResult(ActivityResultContracts.PickVisualMedia()) { uri ->
                if(uri != null) scope.launch {
                    val copied = withContext(Dispatchers.IO) {
                        runCatching {
                            val target = AppStorage.imageWorkFile(this@MainActivity,"avatar-${System.currentTimeMillis()}.img")
                            contentResolver.openInputStream(uri)?.use { input ->
                                target.outputStream().use { output ->
                                    val buffer = ByteArray(8192); var total = 0L
                                    while(true) { val count = input.read(buffer); if(count < 0) break; total += count; check(total <= 20L*1024*1024) { "图片不能超过 20 MB" }; output.write(buffer,0,count) }
                                }
                            } ?: error("无法读取图片")
                            Uri.fromFile(target).toString()
                        }
                    }
                    copied.onSuccess { cropSource = it }
                        .onFailure { Toast.makeText(this@MainActivity,it.message ?: "图片读取失败",Toast.LENGTH_SHORT).show() }
                }
            }
            LabTheme(theme, motion, true, params.overlay) {
                val dark = MaterialTheme.colorScheme.background.red < .5f
                SideEffect { WindowCompat.getInsetsController(window,window.decorView).apply { isAppearanceLightStatusBars = !dark; isAppearanceLightNavigationBars = !dark } }
                CompositionLocalProvider(LocalHeaderGlassParameters provides params.header,LocalAppUpdates provides updates,LocalContentColor provides MaterialTheme.colorScheme.onSurface, LocalOverlayGlassEnabled provides overlayGlass,LocalDeviceTilt provides rememberDeviceTilt(tilt && motion && (glass||overlayGlass))) {
                    Box(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.background)) {
                    if(!signedIn) AuthPage { name ->
                        session.clearSession()
                    } else LabApp(
                        theme, motion, glass, params, avatar, userName,
                        { theme = it; prefs.edit().putInt("theme",it).apply();syncSystemTheme(it) },
                        { motion = it; prefs.edit().putBoolean("motion",it).apply() },
                        { glass = it; prefs.edit().putBoolean("glass",it).apply() },
                        { params = it; materialStore.save(it) },
                        { picker.launch(PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageOnly)) },
                        { scope.launch { runCatching{api.logout()}.onSuccess{session.clearSession()}.onFailure{Toast.makeText(this@MainActivity,it.message ?: "退出失败，请重试",Toast.LENGTH_LONG).show()} } },
                        prefs.getStringSet("followed:$userName",emptySet())?.toSet() ?: emptySet(),
                        { prefs.edit().putStringSet("followed:$userName",it).apply() },
                        tilt,{tilt=it;prefs.edit().putBoolean("tilt",it).apply()},overlayGlass,{overlayGlass=it;prefs.edit().putBoolean("overlayGlass",it).apply()},incomingRoute,{incomingRoute=null;intent.removeExtra("route")},session
                    )
                    cropSource?.let { source ->
                        fun releaseSource(){cropSource=null;scope.launch(Dispatchers.IO){AppStorage.removeImageWorkFile(this@MainActivity,source)}}
                        AvatarCropper(source,::releaseSource){avatar=it;prefs.edit().putString("avatar:$userName",it).apply();releaseSource()}
                    }
                    AppLaunchOverlay(paymentAssetsReady&&(!signedIn||session.readyFor(userName)),motion,session,nativeLaunchReady)
                    }
                }
            }
        }
    }
}

private enum class Destination(val title:String,val icon:ImageVector) {
    Shop("商城",DeuteriumIcons.Shop), Market("市场",DeuteriumIcons.Market),
    Info("信息",DeuteriumIcons.Messages), Profile("我的",DeuteriumIcons.Person)
}

@Composable
private fun LabApp(theme:Int,motion:Boolean,glass:Boolean,parameters:GlassMaterials,avatar:String?,userName:String,
    onTheme:(Int)->Unit,onMotion:(Boolean)->Unit,onGlass:(Boolean)->Unit,onParameters:(GlassMaterials)->Unit,onAvatar:()->Unit,onLogout:()->Unit,
    initialFollowed:Set<String>,saveFollowed:(Set<String>)->Unit,tilt:Boolean,onTilt:(Boolean)->Unit,overlayGlass:Boolean,onOverlayGlass:(Boolean)->Unit,incomingRoute:String?,consumeRoute:()->Unit,session:LabSession) {
    val context=LocalContext.current;val scope=rememberCoroutineScope();val notifications=remember{LabNotifications(context.applicationContext)}
    val state=remember(userName){session.get(userName,initialFollowed,saveFollowed,notifications::post,context.applicationContext)}
    LaunchedEffect(state.storageMessage){state.storageMessage?.let{Toast.makeText(context,it,Toast.LENGTH_LONG).show()}}
    if(state.restoring){Box(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.background));return}
    var destination by rememberSaveable{mutableStateOf(Destination.Shop)};var stack by rememberSaveable{mutableStateOf(listOf<String>())}
    val route=stack.lastOrNull();val page=route ?: destination.name
    var transfer by rememberSaveable{mutableStateOf(false)};var transferTo by rememberSaveable{mutableStateOf<String?>(null)}
    var people by remember{mutableStateOf(false)};var record by remember{mutableStateOf<LedgerEntry?>(null)};var tuner by remember{mutableStateOf(false)};var resetPassword by remember{mutableStateOf(false)}
    var pendingNotification by remember{mutableStateOf<String?>(null)};var notificationAllowed by remember{mutableStateOf(notifications.allowed())}
    val notificationPermission=rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()){granted->notificationAllowed=granted&&notifications.allowed();pendingNotification=null;Toast.makeText(context,if(granted)"系统通知已开启" else "未开启系统通知，仍可查看应用内提醒",Toast.LENGTH_LONG).show()}
    val updates=LocalAppUpdates.current
    val lifecycle=LocalLifecycleOwner.current
    DisposableEffect(lifecycle){val observer=LifecycleEventObserver{_,event->notificationAllowed=notifications.allowed();if(event==androidx.lifecycle.Lifecycle.Event.ON_RESUME)updates?.onResume()};lifecycle.lifecycle.addObserver(observer);onDispose{lifecycle.lifecycle.removeObserver(observer)}}
    val sensorManager=remember{context.getSystemService(android.content.Context.SENSOR_SERVICE) as SensorManager}
    val sensorAvailable=remember{listOf(Sensor.TYPE_GAME_ROTATION_VECTOR,Sensor.TYPE_ROTATION_VECTOR,Sensor.TYPE_ACCELEROMETER).any{sensorManager.getDefaultSensor(it)!=null}}
    val saved=rememberSaveableStateHolder();val atmosphere=rememberGraphicsLayer();val backdrop=rememberGraphicsLayer()
    val density=LocalDensity.current;val ime=WindowInsets.ime;val keyboard=ime.getBottom(density)>0
    val statusTop=WindowInsets.statusBars.asPaddingValues().calculateTopPadding();val compactTop=statusTop+48.dp
    val compactFadeBottom=compactTop+24.dp
    val secondaryContentTop=compactFadeBottom+12.dp
    val navBottom=WindowInsets.navigationBars.asPaddingValues().calculateBottomPadding()
    val conversationInsets=ime.union(WindowInsets(bottom=navBottom+4.dp))
    val shopChrome=rememberPageChrome();val marketChrome=rememberPageChrome();val infoChrome=rememberPageChrome();val profileChrome=rememberPageChrome()
    fun chromeFor(name:String)=when(name){"Shop"->shopChrome;"Market"->marketChrome;"Info"->infoChrome;else->profileChrome}
    val chrome=chromeFor(destination.name);val connection=remember(chrome,density.density){chrome.connection(density.density)}
    val focus=remember{FocusRequester()};val keyboardController=LocalSoftwareKeyboardController.current
    var bagCenter by remember{mutableStateOf(Offset.Zero)};var flight by remember{mutableStateOf<Pair<ShopProduct,Offset>?>(null)}
    val flightProgress=remember{Animatable(1f)}
    LaunchedEffect(flight){if(flight!=null){flightProgress.snapTo(0f);if(motion)flightProgress.animateTo(1f,tween(650,easing=FastOutSlowInEasing))else flightProgress.snapTo(1f);flight=null}}
    fun open(name:String){keyboardController?.hide();if(stack.lastOrNull()!=name)stack=stack+name}
    fun back(){keyboardController?.hide();if(stack.isNotEmpty())stack=stack.dropLast(1)else destination=Destination.Shop}
    fun openTransfer(name:String?=null){transferTo=name;transfer=true}
    fun showOrder(id:String){stack=stack.filterNot{it=="bag"||it.startsWith("order:")||it.startsWith("refund:")}+"order:$id"}
    fun goToNotice(target:String){destination=if(target=="wallet")Destination.Profile else Destination.Info;stack=listOf(target);state.notice=null}
    LaunchedEffect(incomingRoute){incomingRoute?.let{if(it=="wallet"||it=="public"||it.startsWith("order:")||it.startsWith("refund:")||it.startsWith("commission:")||it.startsWith("dm:")||it=="announcement"||it=="about"||it=="storage")goToNotice(it);consumeRoute()}}
    BackHandler(!keyboard&&(stack.isNotEmpty()||destination!=Destination.Shop)&&!transfer&&!people&&record==null&&!tuner&&!resetPassword){back()}
    LaunchedEffect(state.notice?.id){if(state.notice!=null){delay(4200);state.notice=null}}
    val title=when {
        route=="storage"->"存储空间"
        route=="wallet"->"我的钱包";route?.startsWith("bills:")==true->"历史账单";route=="public"->"公共聊天";route=="ai"->"小祥 AI";route?.startsWith("dm:")==true->route.removePrefix("dm:")
        route=="announcement"->"官方公告";route=="event"->"委托大厅";route=="bag"->"购物袋";route=="orders"->"我的订单";route?.startsWith("order:")==true->"订单详情"
        route?.startsWith("refund:")==true->"退款详情";route=="trade-notices"->"交易通知";route=="commissions"->"委托大厅";route=="my-commissions"->"我的委托";route=="publish-commission"->"发布委托";route?.startsWith("commission:")==true->"委托详情";route=="events"->"委托大厅";route?.startsWith("event:")==true->"活动详情";route?.startsWith("player:")==true->"玩家资料";route=="bio"->"个人简介";route?.startsWith("edit-listing:")==true->"重新上架";route=="notifications"->"通知";route=="appearance"->"外观";route=="account"->"账号与安全";route=="about"->"软件更新";route=="publish"->"发布商品";route=="listings"->"我发布的"
        route?.startsWith("product:")==true->ShopCatalog.find{it.id==route.removePrefix("product:")}?.name ?: "商品详情"
        route?.startsWith("market-product:")==true->"商品详情";else->destination.title
    }
    CompositionLocalProvider(LocalAccountAvatar provides PlayerProfile(userName,avatar=avatar),LocalGlassBackdrop provides atmosphere,LocalOverlayBackdrop provides backdrop) {
        Box(Modifier.fillMaxSize().then(if(route==null)Modifier.nestedScroll(connection) else Modifier)) {
            Box(Modifier.fillMaxSize().recordGlassBackdrop(backdrop)) {
                GlassAtmosphere(Modifier.recordGlassBackdrop(atmosphere))
                AnimatedContent(page,modifier=Modifier.fillMaxSize(),transitionSpec={if(motion)(fadeIn(tween(180,25))+slideInHorizontally(tween(260,easing=FastOutSlowInEasing)){it/14}) togetherWith fadeOut(tween(130)) else EnterTransition.None togetherWith ExitTransition.None},label="pages"){current->
                    saved.SaveableStateProvider(current) {
                        val inset=if(current in Destination.entries.map{it.name})statusTop+184.dp-chromeFor(current).offset.dp else secondaryContentTop
                        when {
                            current=="Shop"->ShopPage(state,{open("product:$it")},inset,"")
                            current=="Market"->MarketPage(state.commerce,"",inset,{open("market-product:$it")})
                            current=="Info"->InfoPage(state,inset,"",::open)
                            current=="Profile"->ProfilePage(state,avatar,userName,inset,"",::open,onAvatar)
                            current=="appearance"->AppearancePage(theme,motion,glass,tilt,inset,onTheme,onMotion,onGlass,onTilt,{tuner=true},sensorAvailable,overlayGlass,onOverlayGlass)
                            current=="account"->AccountPage(avatar,userName,inset,onAvatar,{resetPassword=true},onLogout){open("bio")}
                            current=="bio"->BioEditorPage(state,inset){back()}
                            current.startsWith("player:")->PlayerProfilePage(current.removePrefix("player:"),state,avatar,inset,{open("dm:${current.removePrefix("player:")}")},{openTransfer(current.removePrefix("player:"))})
                            current.startsWith("search:")->SearchPage(current.removePrefix("search:"),state,{back()},::open,active=current==page)
                            current=="about"->UpdatePage(state,inset)
                            current=="storage"->StoragePage(inset)
                            current=="wallet"->WalletPage(state,{openTransfer()},{record=it},inset){open("bills:$it")}
                            current.startsWith("bills:")->BillHistoryPage(state,current.removePrefix("bills:"),inset){record=it}
                            current=="public"->Box(Modifier.fillMaxSize().windowInsetsPadding(conversationInsets)){ChatPage(state,secondaryContentTop,{open("player:$it")}){openTransfer(it)}}
                            current=="ai"||current.startsWith("dm:")->Box(Modifier.fillMaxSize().windowInsetsPadding(conversationInsets)){DirectChatPage(state,if(current=="ai")"AI 助手" else current.removePrefix("dm:"),secondaryContentTop){open("player:$it")}}
                            current=="announcement"->CommunityDetail(false,state,inset)
                            current=="commissions"||current=="events"||current=="event"->CommissionHallPage(state,inset,{open("commission:$it")})
                            current=="my-commissions"->CommissionHallPage(state,inset,{open("commission:$it")},mine=true)
                            current=="publish-commission"->PublishCommissionPage(state,inset){id->saved.removeState(current);stack=stack.dropLast(1)+"commission:$id"}
                            current.startsWith("commission:")->CommissionDetailPage(state,current.removePrefix("commission:"),inset){open("dm:$it")}
                            current.startsWith("event:")->EventDetailPage(state,current.removePrefix("event:"),inset)
                            current.startsWith("product:")->ShopCatalog.find{it.id==current.removePrefix("product:")}?.let{ProductPage(it,state,inset,{open("bag")}){product,origin->flight=product to origin}}
                            current=="bag"->ShoppingBag(state,inset,{back()},::showOrder)
                            current.startsWith("market-product:")->MarketProductPage(state.commerce,current.removePrefix("market-product:"),inset,{open("dm:$it")},::showOrder){open("edit-listing:$it")}
                            current=="publish"||current.startsWith("edit-listing:")->PublishListingPage(state.commerce,inset,{id->saved.removeState(current);val parent=stack.dropLast(1);val target="market-product:$id";stack=if(parent.lastOrNull()==target)parent else parent+target},if(current.startsWith("edit-listing:"))state.commerce.listings.find{it.id==current.removePrefix("edit-listing:")} else null)
                            current=="listings"->MarketPage(state.commerce,"",inset,{open("market-product:$it")},true){open("publish")}
                            current=="orders"->OrdersPage(state.commerce,inset,::showOrder)
                            current.startsWith("order:")->OrderDetailPage(state.commerce,current.removePrefix("order:"),inset,{open("dm:$it")}){open("refund:$it")}
                            current.startsWith("refund:")->RefundStatusPage(state.commerce,current.removePrefix("refund:"),inset,{open("dm:$it")},::showOrder)
                            current=="trade-notices"->TradeNoticesPage(state.commerce,inset,notificationAllowed,{if(Build.VERSION.SDK_INT>=33)notificationPermission.launch(Manifest.permission.POST_NOTIFICATIONS)},::open,state::readNotice)
                            current=="notifications"->NotificationPage(inset,notificationAllowed,userName,{if(Build.VERSION.SDK_INT>=33)notificationPermission.launch(Manifest.permission.POST_NOTIFICATIONS)else context.startActivity(Intent(Settings.ACTION_APP_NOTIFICATION_SETTINGS).putExtra(Settings.EXTRA_APP_PACKAGE,context.packageName))},{context.startActivity(Intent(Settings.ACTION_APP_NOTIFICATION_SETTINGS).putExtra(Settings.EXTRA_APP_PACKAGE,context.packageName))})
                        }
                    }
                }
            }
            if(route==null)CollapsingChrome(title,when(destination){Destination.Shop->"搜索商品";Destination.Market->"搜索物品或玩家";Destination.Info->"搜索联系人";Destination.Profile->"搜索设置"},chrome,statusTop,backdrop,{open("search:${destination.name}")}) {
                if(chrome.searchProgress>.75f)IconButton({open("search:${destination.name}")}){Icon(Icons.Outlined.Search,"搜索",tint=MaterialTheme.colorScheme.primary)}
                if(destination==Destination.Shop)Box(Modifier.onGloballyPositioned{bagCenter=it.positionInRoot()+Offset(it.size.width/2f,it.size.height/2f)}){ShoppingBagButton(state.cart.values.sum()){open("bag")}}
                if(destination==Destination.Market)IconButton({open("publish")}){Icon(Icons.Outlined.Add,"发布商品",tint=MaterialTheme.colorScheme.primary)}
            } else if(!route.startsWith("search:")) {
                GradientGlassHeader(backdrop,Modifier.fillMaxWidth().height(compactFadeBottom))
                Row(Modifier.fillMaxWidth().padding(top=statusTop).height(48.dp).padding(horizontal=10.dp),verticalAlignment=Alignment.CenterVertically){
                    IconButton({back()}){Icon(Icons.Outlined.ChevronLeft,"返回",Modifier.size(29.dp),tint=MaterialTheme.colorScheme.primary)}
                    if(route in listOf("commissions","events","event","my-commissions"))Spacer(Modifier.width(48.dp))
                    Text(title,Modifier.weight(1f),textAlign=androidx.compose.ui.text.style.TextAlign.Center,style=MaterialTheme.typography.titleMedium,maxLines=1)
                    when {
                        route in listOf("commissions","events","event","my-commissions")->Row{IconButton({open("my-commissions")}){Icon(Icons.Outlined.Assignment,"我的委托",tint=MaterialTheme.colorScheme.primary)};IconButton({open("publish-commission")}){Icon(Icons.Outlined.Add,"发布委托",tint=MaterialTheme.colorScheme.primary)}}
                        route=="public"->IconButton({people=true}){Icon(Icons.Outlined.PeopleOutline,"在线玩家",tint=MaterialTheme.colorScheme.primary)}
                        route.startsWith("product:")->Box(Modifier.onGloballyPositioned{bagCenter=it.positionInRoot()+Offset(it.size.width/2f,it.size.height/2f)}){ShoppingBagButton(state.cart.values.sum()){open("bag")}}
                        route=="listings"->IconButton({open("publish")}){Icon(Icons.Outlined.Add,"发布商品",tint=MaterialTheme.colorScheme.primary)}
                        route.startsWith("dm:")->IconButton({open("player:${route.removePrefix("dm:")}")}){Icon(Icons.Outlined.PersonOutline,"查看玩家资料",tint=MaterialTheme.colorScheme.primary)}
                        else->Spacer(Modifier.width(48.dp))
                    }
                }
            }
            if(route==null)LiquidGlass(Modifier.align(Alignment.BottomCenter).navigationBarsPadding().padding(horizontal=18.dp,vertical=9.dp).fillMaxWidth().height(70.dp).graphicsLayer{alpha=if(keyboard)0f else 1f},radius=30.dp,backdrop=backdrop,enabled=glass,parameters=parameters.bottomBar){MovingGlassNav(Destination.entries.map{it.title},Destination.entries.map{it.icon},destination.ordinal,!keyboard,if(updates?.hasUpdates==true)setOf(Destination.Profile.ordinal) else emptySet()){destination=Destination.entries[it]}}
            flight?.let{(product,start)->Image(painterResource(product.image),null,Modifier.size(72.dp).graphicsLayer{val t=flightProgress.value;val current=start+(bagCenter-start)*t+Offset(-90f*kotlin.math.sin(t*Math.PI).toFloat(),-100f*kotlin.math.sin(t*Math.PI).toFloat());translationX=current.x-size.width/2;translationY=current.y-size.height/2;scaleX=1f-.78f*t;scaleY=scaleX;alpha=1f-.4f*t}.clip(androidx.compose.foundation.shape.RoundedCornerShape(16.dp)),contentScale=ContentScale.Crop)}
            AnimatedVisibility(state.notice!=null,modifier=Modifier.align(Alignment.TopCenter).padding(top=compactTop+8.dp,start=16.dp,end=16.dp),enter=fadeIn()+slideInVertically{-it/2},exit=fadeOut()+slideOutVertically{-it/2}) {
                state.notice?.let{notice->LiquidGlass(Modifier.fillMaxWidth(),backdrop=backdrop,onClick={goToNotice(notice.route)}){Row(Modifier.padding(15.dp),verticalAlignment=Alignment.CenterVertically){Icon(Icons.Outlined.NotificationsNone,null,tint=MaterialTheme.colorScheme.primary);Column(Modifier.weight(1f).padding(horizontal=10.dp)){Text(notice.title,style=MaterialTheme.typography.titleMedium);Text(notice.body,style=MaterialTheme.typography.bodySmall,maxLines=2)};IconButton({state.notice=null},Modifier.size(36.dp)){Icon(Icons.Outlined.Close,"关闭提醒",Modifier.size(18.dp))}}}}
            }
        }
        if(transfer)TransferSheet(state,transferTo){transfer=false}
        if(tuner)GlassTuner(parameters,onParameters){tuner=false}
        if(resetPassword)ResetPasswordSheet(userName){resetPassword=false}
        if(people)IosSheet({people=false}){Column(Modifier.padding(horizontal=24.dp).padding(bottom=24.dp)){Text("在线玩家",style=MaterialTheme.typography.titleLarge);Players.filter{it.online}.forEach{person->Row(Modifier.fillMaxWidth().padding(vertical=8.dp),verticalAlignment=Alignment.CenterVertically){PlayerAvatar(person.name);Text(person.name,Modifier.weight(1f).padding(start=12.dp));PlainButton({people=false;open("dm:${person.name}")}){Text("私聊")};PlainButton({people=false;state.mentionPlayer(person.name)}){Text("@ 提及")}}}}}
        record?.let{entry->IosSheet({record=null}){Column(Modifier.padding(24.dp)){Text((if(entry.amount>0)"+" else "−")+credit(kotlin.math.abs(entry.amount)),style=MaterialTheme.typography.headlineLarge);DetailRow("交易状态","已完成");DetailRow("往来玩家",entry.name);DetailRow("备注",entry.detail);DetailRow("时间",entry.at.format(java.time.format.DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm")));DetailRow("流水号","DT${entry.id.toString().padStart(10,'0')}");MotionButton({record=null},Modifier.fillMaxWidth().padding(top=18.dp)){Text("完成")}}}}
    }
}

@Composable
private fun ShoppingBagButton(count:Int,onClick:()->Unit){
    Box(Modifier.size(48.dp)){
        IconButton(onClick,Modifier.matchParentSize()) { Icon(Icons.Outlined.ShoppingBag,"购物袋",Modifier.size(23.dp)) }
        if(count>0)Box(Modifier.align(Alignment.TopEnd).padding(top=1.dp,end=1.dp).defaultMinSize(minWidth=19.dp,minHeight=19.dp).background(MaterialTheme.colorScheme.error,CircleShape).padding(horizontal=4.dp,vertical=3.dp),contentAlignment=Alignment.Center){
            Text(if(count>99)"99+" else count.toString(),fontSize=10.sp,lineHeight=12.sp,fontWeight=FontWeight.Bold,color=MaterialTheme.colorScheme.onError,maxLines=1)
        }
    }
}

@Composable
fun DetailRow(label:String,value:String){Row(Modifier.fillMaxWidth().padding(vertical=9.dp),horizontalArrangement=Arrangement.SpaceBetween){
    Text(label,style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)
    Text(value,style=MaterialTheme.typography.bodyMedium,fontWeight=FontWeight.Medium,modifier=Modifier.padding(start=16.dp).weight(1f),textAlign=androidx.compose.ui.text.style.TextAlign.End)
}}
