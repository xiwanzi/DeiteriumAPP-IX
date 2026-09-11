package com.deuterium.app.uilab

import android.app.Instrumentation
import android.content.ContextWrapper
import android.content.Intent
import android.graphics.Bitmap
import android.os.Bundle
import android.view.KeyEvent
import android.view.accessibility.AccessibilityNodeInfo
import androidx.activity.compose.setContent
import androidx.compose.material3.LocalContentColor
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.CompositionLocalProvider
import okhttp3.Protocol
import okhttp3.Response
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.ResponseBody.Companion.toResponseBody
import okio.Buffer
import org.json.JSONObject
import java.util.concurrent.atomic.AtomicReference

/** Isolated preferences and intercepted HTTP: no real account, OTP delivery or server writes. */
object AuthRegistrationCheck {
    fun run(test: Instrumentation) {
        val result=Bundle()
        val instance=BackendApi::class.java.getDeclaredField("instance").apply{isAccessible=true}
        val previous=instance.get(null)
        val context=object:ContextWrapper(test.targetContext){
            override fun getSharedPreferences(name:String,mode:Int)=super.getSharedPreferences("qa-auth-registration-$name",mode)
        }
        val sent=AtomicReference<JSONObject?>()
        try {
            val api=BackendApi(context)
            val client=api.http.newBuilder().addInterceptor{chain->
                val request=chain.request()
                val data=if(request.url.encodedPath.endsWith("/account/registration-code")) {
                    val buffer=Buffer();request.body!!.writeTo(buffer)
                    sent.set(JSONObject(buffer.readUtf8()))
                    JSONObject().put("verificationToken","fixture-verification").put("resendAfterSeconds",60)
                } else JSONObject()
                Response.Builder().request(request).protocol(Protocol.HTTP_1_1).code(200).message("OK")
                    .body(JSONObject().put("data",data).toString().toResponseBody("application/json".toMediaType())).build()
            }.build()
            BackendApi::class.java.getDeclaredField("http").apply{isAccessible=true}.set(api,client)
            instance.set(null,api)
            val activity=test.startActivitySync(Intent(test.targetContext,DeuteriumActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) as DeuteriumActivity
            fun show(theme:Int,glass:Boolean=true){
                test.runOnMainSync{activity.setContent{LabTheme(theme,false,true,GlassParameters()){
                    CompositionLocalProvider(LocalContentColor provides MaterialTheme.colorScheme.onSurface,LocalOverlayGlassEnabled provides glass){AuthPage{error("Unexpected registration")}}
                }}}
                test.waitForIdleSync()
            }
            fun find(node:AccessibilityNodeInfo?,predicate:(AccessibilityNodeInfo)->Boolean):AccessibilityNodeInfo? {
                if(node==null)return null
                if(predicate(node))return node
                for(i in 0 until node.childCount)find(node.getChild(i),predicate)?.let{return it}
                return null
            }
            fun named(text:String)=find(test.uiAutomation.rootInActiveWindow){it.text?.toString()==text||it.contentDescription?.toString()==text}
            fun waitFor(predicate:()->Boolean){repeat(80){if(predicate())return;Thread.sleep(100)};error("Auth UI did not reach expected state")}
            fun click(text:String){waitFor{named(text)!=null};var node=named(text)!!;while(!node.isClickable&&node.parent!=null)node=node.parent;check(node.performAction(AccessibilityNodeInfo.ACTION_CLICK)){"Could not click $text"};test.waitForIdleSync()}
            fun input(label:String,value:String){
                fun field()=find(test.uiAutomation.rootInActiveWindow){node->node.isEditable&&find(node){it.text?.toString()==label||it.contentDescription?.toString()==label}!=null}
                waitFor{field()!=null}
                check(field()!!.performAction(AccessibilityNodeInfo.ACTION_SET_TEXT,Bundle().apply{putCharSequence(AccessibilityNodeInfo.ACTION_ARGUMENT_SET_TEXT_CHARSEQUENCE,value)})){"Could not fill $label"}
                test.waitForIdleSync()
            }
            fun capture(label:String){Thread.sleep(500);test.uiAutomation.takeScreenshot()?.let{bitmap->test.targetContext.filesDir.resolve("qa-auth-$label.png").outputStream().use{bitmap.compress(Bitmap.CompressFormat.PNG,100,it)};bitmap.recycle()} ?: error("Screenshot unavailable")}
            show(1)
            click("创建账号")
            input("游戏内 ID","NewPlayer")
            input("QQ 号","10009")
            click("获取")
            waitFor{sent.get()!=null&&named("验证码已发送至游戏内，请在服务器中查看")!=null}
            check(sent.get()!!.keys().asSequence().toSet()==setOf("gameId","qq")){"Code request included password"}
            capture("registration-no-password")
            input("游戏内验证码","123456")
            click("创建账号并继续")
            waitFor{named("密码需要 8–64 位")!=null}
            result.putString("code_without_password_and_register_validation","PASS")
            click("登录")
            for((label,theme,glass) in listOf(Triple("light",1,true),Triple("dark",2,true),Triple("glass-off",1,false))) {
                show(theme,glass)
                click("忘记密码？")
                waitFor{named("重置密码")!=null&&named("确认修改")!=null}
                capture("reset-$label")
                if(label=="light") {
                    click("玩家 ID 或 QQ")
                    Thread.sleep(700)
                    capture("reset-keyboard")
                    test.sendKeyDownUpSync(KeyEvent.KEYCODE_BACK)
                    Thread.sleep(200)
                }
                test.sendKeyDownUpSync(KeyEvent.KEYCODE_BACK)
                waitFor{named("忘记密码？")!=null}
            }
            result.putString("reset_light_dark_glass_off_keyboard_back","PASS")
            test.finish(-1,result)
        }catch(error:Throwable){result.putString("error",error.stackTraceToString());test.finish(1,result)}
        finally{instance.set(null,previous)}
    }
}
