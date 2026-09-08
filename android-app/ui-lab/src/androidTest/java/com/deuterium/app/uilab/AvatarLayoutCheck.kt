package com.deuterium.app.uilab

import android.app.Instrumentation
import android.content.Intent
import android.graphics.Bitmap
import android.os.Bundle
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.*
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.ui.Modifier
import androidx.compose.ui.layout.onGloballyPositioned
import androidx.compose.ui.unit.dp
import java.util.concurrent.ConcurrentHashMap

/** Real decoded image layout check; no account, network or business writes. */
object AvatarLayoutCheck {
    fun run(test:Instrumentation) {
        val result=Bundle();val name="avatar_layout_fixture"
        val file=test.targetContext.cacheDir.resolve("avatar-layout-fixture.png")
        val sizes=ConcurrentHashMap<String,Pair<Int,Int>>()
        try {
            Bitmap.createBitmap(640,640,Bitmap.Config.ARGB_8888).apply{eraseColor(0xff357ed3.toInt())}.let { bitmap->file.outputStream().use{bitmap.compress(Bitmap.CompressFormat.PNG,100,it)};bitmap.recycle() }
            val activity=test.startActivitySync(Intent(test.targetContext,MainActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) as MainActivity
            test.runOnMainSync {
                Players.add(PlayerProfile(name,avatar=file.toURI().toString()))
                activity.setContent { MaterialTheme { CompositionLocalProvider(LocalAccountAvatar provides PlayerProfile("avatar_self_fixture",avatar=file.toURI().toString())) { Column(Modifier.fillMaxSize().padding(20.dp)) {
                    for((label,dp) in listOf("default" to 42,"small" to 21,"profile" to 98,"self" to 42)) {
                        Row(Modifier.fillMaxWidth().height(112.dp)) {
                            val requested=if(label=="default"||label=="self")Modifier else Modifier.size(dp.dp)
                            PlayerAvatar(if(label=="self")"avatar_self_fixture" else name,requested.onGloballyPositioned{sizes[label]=it.size.width to it.size.height})
                            Text("  $label: ${dp}dp")
                        }
                    }
                } } } }
            }
            fun loaded(node:android.view.accessibility.AccessibilityNodeInfo?):Int {
                if(node==null)return 0
                return (if(node.contentDescription?.toString()?.endsWith("的头像")==true)1 else 0)+(0 until node.childCount).sumOf{loaded(node.getChild(it))}
            }
            repeat(30){if(loaded(test.uiAutomation.rootInActiveWindow)<4)Thread.sleep(100)}
            check(loaded(test.uiAutomation.rootInActiveWindow)==4){"Not all four decoded avatars were displayed"}
            test.waitForIdleSync()
            val density=test.targetContext.resources.displayMetrics.density
            for((label,dp) in listOf("default" to 42,"small" to 21,"profile" to 98,"self" to 42)) {
                val expected=(dp*density+.5f).toInt();val actual=sizes[label]
                check(actual==(expected to expected)){"$label expected ${expected}px, got $actual"}
                result.putString(label,"${dp}dp = ${actual!!.first}x${actual.second}px PASS")
            }
            test.uiAutomation.takeScreenshot()?.let{bitmap->test.targetContext.filesDir.resolve("qa-avatar-layout.png").outputStream().use{bitmap.compress(Bitmap.CompressFormat.PNG,100,it)};bitmap.recycle()}
            result.putString("status","PASS");test.finish(-1,result)
        }catch(error:Throwable){result.putString("error",error.message);test.finish(1,result)}
        finally {test.runOnMainSync{Players.removeAll{it.name==name}};file.delete()}
    }
}
