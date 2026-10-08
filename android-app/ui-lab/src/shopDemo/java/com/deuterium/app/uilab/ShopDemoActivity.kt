package com.deuterium.app.uilab

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.enableEdgeToEdge
import androidx.activity.compose.*
import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.rememberGraphicsLayer
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.unit.dp
import androidx.core.view.WindowCompat
import androidx.lifecycle.lifecycleScope
import kotlinx.coroutines.launch
import okhttp3.OkHttpClient

class ShopDemoActivity:ComponentActivity() {
    override fun onCreate(savedInstanceState:Bundle?) {
        super.onCreate(savedInstanceState);enableEdgeToEdge()
        var state by mutableStateOf<LabState?>(null)
        lifecycleScope.launch {
            val local=OfflineShop(this@ShopDemoActivity)
            val api=BackendApi(this@ShopDemoActivity,client=OkHttpClient.Builder().addInterceptor(local).build())
            api.login("DemoPlayer","offline")
            val imageScope=BackendApi.get(this@ShopDemoActivity).financialScope()
            val images=AppImages.get(this@ShopDemoActivity)
            for((id,file) in local.media) {
                RemoteImageUrls.remember(local.image(id),imageScope)
                images.seed("asset:$id",assets.open("shop-demo/$file").use{it.readBytes()})
            }
            state=LabState(lifecycleScope,userName="DemoPlayer",api=api).apply{profileDisplayName="演示玩家";profileBio="离线商城演示 · 不连接业务服务";balanceKnown=true}
        }
        setContent {
            val prefs=remember{getSharedPreferences("demo-appearance",MODE_PRIVATE)}
            var theme by rememberSaveable{mutableIntStateOf(prefs.getInt("theme",0))}
            var motion by rememberSaveable{mutableStateOf(prefs.getBoolean("motion",true))}
            var glass by rememberSaveable{mutableStateOf(prefs.getBoolean("glass",true))}
            var materials by remember{mutableStateOf(GlassMaterials())}
            var overlayGlass by rememberSaveable{mutableStateOf(true)}
            var tilt by rememberSaveable{mutableStateOf(true)}
            val session=remember{LabSession().apply{launchFinished=true}}
            LabTheme(theme,motion,true,materials.overlay) {
                val dark=MaterialTheme.colorScheme.background.red<.5f;val background=MaterialTheme.colorScheme.background
                SideEffect{window.setBackgroundDrawable(android.graphics.drawable.ColorDrawable(background.toArgb()));WindowCompat.getInsetsController(window,window.decorView).apply{isAppearanceLightStatusBars=!dark;isAppearanceLightNavigationBars=!dark}}
                val current=state
                if(current==null)Box(Modifier.fillMaxSize().background(background),contentAlignment=Alignment.Center){Text("正在准备离线商城…",color=MaterialTheme.colorScheme.onSurface)}
                else CompositionLocalProvider(LocalHeaderGlassParameters provides materials.header,LocalOverlayGlassEnabled provides overlayGlass,LocalContentColor provides MaterialTheme.colorScheme.onSurface,LocalDeviceTilt provides rememberDeviceTilt(tilt&&motion&&(glass||overlayGlass))) {
                    LabApp(theme,motion,glass,materials,null,"DemoPlayer",
                        {theme=it;prefs.edit().putInt("theme",it).apply()},
                        {motion=it;prefs.edit().putBoolean("motion",it).apply()},
                        {glass=it;prefs.edit().putBoolean("glass",it).apply()},
                        {materials=it},{},{},emptySet(),{},tilt,{tilt=it},overlayGlass,{overlayGlass=it},null,{},session,providedState=current)
                }
            }
        }
    }
}
