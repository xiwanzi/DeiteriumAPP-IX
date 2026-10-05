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
import java.util.concurrent.atomic.AtomicInteger

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
        val attempts=AtomicInteger()
        val loginAttempts=AtomicInteger();val signedIn=AtomicReference<String?>()
        val resetAttempts=AtomicInteger()
        try {
            context.getSharedPreferences("backend-v2",0).edit().clear().commit()
            val api=BackendApi(context)
            val client=api.http.newBuilder().addInterceptor{chain->
                val request=chain.request()
                var status=200
                val data=if(request.url.encodedPath.endsWith("/account/registration-code")) {
                    val buffer=Buffer();request.body!!.writeTo(buffer)
                    sent.set(JSONObject(buffer.readUtf8()))
                    val attempt=attempts.incrementAndGet()
                    Thread.sleep(2500)
                    if(attempt==1){status=400;JSONObject().put("code","MINECRAFT_PLAYER_NOT_FOUND").put("message","未找到此正版账号 ID，请核对后重新获取验证码。")}
                    else if(attempt==2){status=429;JSONObject().put("code","RATE_LIMITED").put("message","请 2 秒后再试。").put("retryAfterSeconds",2).put("retryScope","RECIPIENT")}
                    else if(attempt==3)throw java.io.IOException("fixture unknown result")
                    else JSONObject().put("verificationToken","fixture-verification").put("resendAfterSeconds",58)
                } else if(request.url.encodedPath.endsWith("/account/login")){
                    when(loginAttempts.incrementAndGet()){
                        1->{status=401;JSONObject().put("code","ACCOUNT_PASSWORD_INVALID").put("message","账号或密码错误。")}
                        2->{status=429;JSONObject().put("code","LOGIN_LOCKED").put("message","登录尝试过多，请 2 秒后再试。").put("retryAfterSeconds",2)}
                        4->{Thread.sleep(2500);status=429;JSONObject().put("code","LOGIN_LOCKED").put("message","原账号请 60 秒后重试。").put("retryAfterSeconds",60)}
                        else->JSONObject().put("token","fixture-login-session").put("user",JSONObject().put("userId","fixture-user").put("playerRef","fixture-player").put("gameId","NewPlayer").put("gameVerified",false))
                    }
                } else if(request.url.encodedPath.endsWith("/account/password-reset-code")){
                    resetAttempts.incrementAndGet();JSONObject().put("verificationToken","fixture-reset").put("resendAfterSeconds",60).put("delivery","EMAIL")
                } else if(request.url.encodedPath.endsWith("/account/register"))JSONObject().put("token","fixture-register-session").put("user",JSONObject().put("userId","fixture-user").put("playerRef","fixture-player").put("gameId","NewPlayer").put("gameVerified",false))
                else JSONObject()
                Response.Builder().request(request).protocol(Protocol.HTTP_1_1).code(status).message("fixture")
                    .body(JSONObject().put(if(status==200)"data" else "error",data).toString().toResponseBody("application/json".toMediaType())).build()
            }.build()
            BackendApi::class.java.getDeclaredField("http").apply{isAccessible=true}.set(api,client)
            instance.set(null,api)
            val activity=test.startActivitySync(Intent(test.targetContext,DeuteriumActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) as DeuteriumActivity
            fun show(theme:Int,glass:Boolean=true){
                test.runOnMainSync{activity.setContent{LabTheme(theme,false,true,GlassParameters()){
                    CompositionLocalProvider(LocalContentColor provides MaterialTheme.colorScheme.onSurface,LocalOverlayGlassEnabled provides glass){AuthPage{signedIn.set(it)}}
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
            fun click(text:String){waitFor{var node=named(text)?:return@waitFor false;while(!node.isClickable&&node.parent!=null)node=node.parent;node.isEnabled&&node.performAction(AccessibilityNodeInfo.ACTION_CLICK)};test.waitForIdleSync()}
            fun clickLoginSubmit(){
                var last:AccessibilityNodeInfo?=null
                fun visit(node:AccessibilityNodeInfo?){if(node==null)return;if(node.text?.toString()=="登录")last=node;for(i in 0 until node.childCount)visit(node.getChild(i))}
                visit(test.uiAutomation.rootInActiveWindow);var node=checkNotNull(last){"Login submit unavailable"};while(!node.isClickable&&node.parent!=null)node=node.parent;check(node.performAction(AccessibilityNodeInfo.ACTION_CLICK));test.waitForIdleSync()
            }
            fun input(label:String,value:String){
                fun field()=find(test.uiAutomation.rootInActiveWindow){node->node.isEditable&&find(node){it.text?.toString()==label||it.contentDescription?.toString()==label}!=null}
                waitFor{field()!=null}
                check(field()!!.performAction(AccessibilityNodeInfo.ACTION_SET_TEXT,Bundle().apply{putCharSequence(AccessibilityNodeInfo.ACTION_ARGUMENT_SET_TEXT_CHARSEQUENCE,value)})){"Could not fill $label"}
                test.waitForIdleSync()
                waitFor{val current=field();value.isNotEmpty()&&named(value)!=null||current!=null&&(if(current.isPassword)current.text?.length==value.length else current.text?.toString()?.contains(value)==true||find(current){it.text?.toString()==value}!=null||value.isEmpty()&&current.text.isNullOrEmpty())}
            }
            fun capture(label:String){Thread.sleep(500);test.uiAutomation.takeScreenshot()?.let{bitmap->test.targetContext.filesDir.resolve("qa-auth-$label.png").outputStream().use{bitmap.compress(Bitmap.CompressFormat.PNG,100,it)};bitmap.recycle()} ?: error("Screenshot unavailable")}
            show(1)
            click("创建账号")
            input("游戏内 ID","NewPlayer")
            input("邮箱","newplayer@example.com")
            click("获取")
            waitFor{find(test.uiAutomation.rootInActiveWindow){it.text?.toString()?.matches(Regex("[0-9]+s"))==true}!=null}
            check(named("正在进入…")==null){"Code lookup blocked the form"}
            input("设置密码（至少8位）","typing-while-querying")
            waitFor{named("未找到此正版账号 ID，请核对后重新获取验证码。")!=null&&named("获取")!=null}
            check(named("验证码邮件已排队，请查收邮箱")==null){"Failed ID reported mail delivery"}
            click("获取")
            waitFor{named("请 2 秒后再试。")!=null}
            check(named("2s")!=null||named("1s")!=null){"Rate limit reopened code button"}
            waitFor{named("获取")!=null}
            click("获取")
            waitFor{named("本次发送结果暂未确认，请查看邮箱；倒计时结束后可重试。")!=null}
            check(named("获取")==null){"Unknown result allowed immediate resend"}
            input("邮箱","corrected@example.com")
            check(named("获取")!=null){"Changing recipient blocked a new email"}
            input("邮箱","newplayer@example.com")
            check(named("获取")==null){"Returning to same recipient lost cooldown"}
            input("邮箱","corrected@example.com")
            click("获取")
            waitFor{sent.get()?.optString("email")=="corrected@example.com"&&find(test.uiAutomation.rootInActiveWindow){node->node.isClickable&&node.isEnabled&&find(node){it.text?.toString()=="创建账号并继续"}!=null}!=null}
            check(attempts.get()==4){"Retry or click guard failed"}
            result.putString("rate_limit_unknown_result_and_recipient_cooldown","PASS")
            result.putString("immediate_countdown_failure_retry_and_editable_form","PASS")
            check(sent.get()!!.keys().asSequence().toSet()==setOf("gameId","email")){"Code request included password"}
            capture("registration-no-password")
            input("验证码","123456")
            input("设置密码（至少8位）","")
            click("创建账号并继续")
            waitFor{named("密码需要 8–64 位")!=null}
            result.putString("code_without_password_and_register_validation","PASS")
            input("设置密码（至少8位）","test-password")
            click("创建账号并继续")
            waitFor{signedIn.get()=="NewPlayer"};signedIn.set(null)
            result.putString("registration_completes_with_email_proof","PASS")
            click("登录")
            input("密码","wrong-password");clickLoginSubmit();waitFor{named("账号或密码错误。")!=null}
            input("密码","test-password");clickLoginSubmit();waitFor{named("登录尝试过多，请 2 秒后再试。")!=null}
            waitFor{find(test.uiAutomation.rootInActiveWindow){it.text?.toString()?.endsWith("s 后重试")==true}==null}
            clickLoginSubmit();waitFor{signedIn.get()=="NewPlayer"}
            check(loginAttempts.get()==3){"Login duplicate submission"}
            result.putString("login_error_limit_and_success","PASS")
            input("邮箱、玩家 ID 或 QQ","PendingPlayer");input("密码","test-password");clickLoginSubmit()
            waitFor{loginAttempts.get()==4};input("邮箱、玩家 ID 或 QQ","ChangedPlayer")
            waitFor{named("正在进入…")==null}
            check(named("原账号请 60 秒后重试。")==null){"Late login error replaced the new account state"}
            check(find(test.uiAutomation.rootInActiveWindow){it.text?.toString()?.endsWith("s 后重试")==true}==null){"Late login limit blocked a different account"}
            input("邮箱、玩家 ID 或 QQ","PendingPlayer")
            waitFor{find(test.uiAutomation.rootInActiveWindow){it.text?.toString()?.endsWith("s 后重试")==true}!=null}
            result.putString("login_late_result_isolation","PASS")
            click("忘记密码？");input("邮箱、玩家 ID 或 QQ","reset-player@example.com");click("获取验证码")
            waitFor{resetAttempts.get()==1&&find(test.uiAutomation.rootInActiveWindow){it.text?.toString()?.endsWith("s 后重发")==true}!=null}
            check(named("游戏内验证码")==null&&named("验证码邮件已排队，请查收邮箱")==null){"Redundant password form copy remains"}
            test.sendKeyDownUpSync(KeyEvent.KEYCODE_BACK);Thread.sleep(300)
            if(named("找回密码")!=null)test.sendKeyDownUpSync(KeyEvent.KEYCODE_BACK)
            waitFor{named("忘记密码？")!=null};click("忘记密码？")
            input("邮箱、玩家 ID 或 QQ","reset-player@example.com")
            waitFor{find(test.uiAutomation.rootInActiveWindow){it.text?.toString()?.endsWith("s 后重发")==true}!=null}
            check(resetAttempts.get()==1){"Reopening password sheet sent another code"}
            result.putString("password_code_cooldown_survives_sheet_close","PASS")
            test.sendKeyDownUpSync(KeyEvent.KEYCODE_BACK);Thread.sleep(300)
            if(named("找回密码")!=null)test.sendKeyDownUpSync(KeyEvent.KEYCODE_BACK)
            waitFor{named("忘记密码？")!=null}
            for((label,theme,glass) in listOf(Triple("light",1,true),Triple("dark",2,true),Triple("glass-off",1,false))) {
                show(theme,glass)
                click("忘记密码？")
                waitFor{named("找回密码")!=null&&named("确认修改")!=null}
                capture("reset-$label")
                if(label=="light") {
                    click("邮箱、玩家 ID 或 QQ")
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
        }catch(error:Throwable){
            fun texts(node:AccessibilityNodeInfo?):List<String>{if(node==null)return emptyList();return listOfNotNull(node.text?.toString()?.takeIf{it.isNotBlank()})+(0 until node.childCount).flatMap{texts(node.getChild(it))}}
            result.putString("ui_failure_text",texts(test.uiAutomation.rootInActiveWindow).joinToString(" | "))
            result.putInt("code_attempts",attempts.get());result.putInt("login_attempts",loginAttempts.get());result.putString("error",error.stackTraceToString());test.finish(1,result)
        }
        finally{instance.set(null,previous)}
    }
}
