package com.deuterium.app.uilab

import android.content.Context
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Base64
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONObject
import java.io.IOException
import java.security.KeyStore
import java.util.concurrent.TimeUnit
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec

class ApiFailure(val code: String, message: String, val status: Int = 0) : IOException(message)

/** All asset and identity authority stays on the backend. No offline-success fallback. */
class BackendApi internal constructor(context: Context, origin: String = BuildConfig.API_BASE_URL) {
    private val appContext=context.applicationContext
    private val prefs = context.getSharedPreferences("backend-v2", Context.MODE_PRIVATE)
    private val pendingWrites=PendingPreferences(prefs)
    val baseUrl: String = origin.trimEnd('/')
    val http = OkHttpClient.Builder().connectTimeout(12, TimeUnit.SECONDS)
        .readTimeout(20, TimeUnit.SECONDS).callTimeout(25, TimeUnit.SECONDS)
        .retryOnConnectionFailure(false).followRedirects(false).build()
    var token: String? by mutableStateOf(readToken()); private set
    var user: JSONObject? by mutableStateOf(prefs.getString("user", null)?.let { runCatching { JSONObject(it) }.getOrNull() }); private set
    val signedIn: Boolean get() = !token.isNullOrBlank() && user != null
    val userName: String get() = user?.optString("gameId").orEmpty()
    val playerRef: String get() = user?.optString("playerRef").orEmpty()

    private fun key(): SecretKey {
        val store = KeyStore.getInstance("AndroidKeyStore").apply { load(null) }
        (store.getKey("deuterium-session-v2", null) as? SecretKey)?.let { return it }
        return KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, "AndroidKeyStore").apply {
            init(KeyGenParameterSpec.Builder("deuterium-session-v2", KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT)
                .setBlockModes(KeyProperties.BLOCK_MODE_GCM).setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE).build())
        }.generateKey()
    }
    private fun readToken(): String? = runCatching {
        if(prefs.getString("sessionOrigin",null)!=baseUrl)return@runCatching null
        val saved = prefs.getString("session", null) ?: return@runCatching null
        val parts = saved.split(':')
        Cipher.getInstance("AES/GCM/NoPadding").apply {
            init(Cipher.DECRYPT_MODE, key(), GCMParameterSpec(128, Base64.decode(parts[0], Base64.NO_WRAP)))
        }.doFinal(Base64.decode(parts[1], Base64.NO_WRAP)).toString(Charsets.UTF_8)
    }.getOrNull()
    private fun rememberSession(data: JSONObject) {
        val nextToken = data.getString("token")
        val nextUser = data.getJSONObject("user")
        require(nextToken.isNotBlank() && nextUser.getString("gameId").isNotBlank())
        val cipher = Cipher.getInstance("AES/GCM/NoPadding").apply { init(Cipher.ENCRYPT_MODE, key()) }
        val encrypted = cipher.doFinal(nextToken.toByteArray())
        prefs.edit().putString("session", Base64.encodeToString(cipher.iv, Base64.NO_WRAP) + ":" + Base64.encodeToString(encrypted, Base64.NO_WRAP))
            .putString("sessionOrigin",baseUrl)
            .putString("user", nextUser.toString()).apply()
        user = nextUser; token = nextToken
    }
    fun forgetSession() { token = null; user = null; AppImages.sessionEnded(); prefs.edit().remove("session").remove("user").apply() }
    suspend fun login(account: String, password: String): String {
        val data = request("POST", "/account/login", JSONObject().put("account", account).put("password", password), authenticated = false)
        rememberSession(data); return userName
    }
    suspend fun register(gameId: String, qq: String, password: String, verificationToken: String, code: String): String {
        val data = request("POST", "/account/register", JSONObject()
            .put("password", password).put("verificationToken", verificationToken).put("code", code), authenticated = false)
        rememberSession(data); return userName
    }
    suspend fun verifySession() {
        val data = request("GET", "/account/me")
        user = data.getJSONObject("user")
        prefs.edit().putString("user", user.toString()).apply()
    }
    suspend fun logout() { request("POST", "/account/logout", JSONObject()); forgetSession() }

    suspend fun request(method: String, path: String, body: JSONObject? = null, authenticated: Boolean = true,idempotencyKey:String?=null): JSONObject {
        val imageScope=financialScope()
        val requestToken=if(authenticated)token ?: throw ApiFailure("UNAUTHORIZED","请重新登录",401) else null
        return withContext(Dispatchers.IO) {
        if(authenticated){pendingWrites.awaitCommitted();imageScope.verifyCurrent(financialScope())}
        val builder = Request.Builder().url("$baseUrl/api/v1$path").header("Accept", "application/json")
        if(authenticated) builder.header("Authorization", "Bearer $requestToken")
        idempotencyKey?.let{builder.header("Idempotency-Key",it)}
        val payload = body?.toString()?.toRequestBody("application/json; charset=utf-8".toMediaType())
        builder.method(method, if(method in listOf("POST", "PUT", "PATCH")) payload ?: "{}".toRequestBody("application/json".toMediaType()) else null)
        try {
            http.newCall(builder.build()).execute().use { response ->
                val text = response.body?.string().orEmpty()
                val root = runCatching { JSONObject(text) }.getOrNull()
                if(!response.isSuccessful) {
                    val error = root?.optJSONObject("error")
                    if(response.code == 401 && authenticated) withContext(Dispatchers.Main) { if(token==requestToken)forgetSession() }
                    throw ApiFailure(error?.optString("code") ?: "HTTP_${response.code}",
                        error?.optString("message")?.takeIf { it.isNotBlank() } ?: "服务暂不可用（${response.code}）", response.code)
                }
                val result=root?.optJSONObject("data") ?: root?.optJSONArray("data")?.let { JSONObject().put("items", it) }
                    ?: throw ApiFailure("INVALID_RESPONSE", "服务器响应格式不正确")
                root?.optJSONObject("page")?.let{result.put("_page",it)}
                root?.optString("serverTime")?.takeIf{it.isNotBlank()}?.let{result.put("_serverTime",it)}
                redactErasedAccounts(result,erasedPlayerRefs())
                if(!authenticated||token==requestToken)RemoteImageUrls.remember(result,imageScope,refreshPath=path.takeIf{method=="GET"})
                result
            }
        } catch(error: ApiFailure) { throw error }
        catch(error: IOException) { throw ApiFailure("NETWORK_UNAVAILABLE", "连接暂不可用，请稍后重试") }
        }
    }

    internal fun financialScope()=FinancialScope(if(signedIn)playerRef else "",baseUrl)
    internal fun erasedPlayerRefs():Set<String> = prefs.getStringSet("erased-player-refs:$baseUrl",emptySet()).orEmpty().toSet()
    internal fun accountDeletionCursor():Long=prefs.getLong("account-deletion-cursor:$baseUrl",0)
    internal fun rememberAccountDeletions(refs:Set<String>,cursor:Long){
        val all=erasedPlayerRefs()+refs
        val edit=prefs.edit().putStringSet("erased-player-refs:$baseUrl",all).putLong("account-deletion-cursor:$baseUrl",cursor)
        prefs.all.keys.filter{key->(key.startsWith("pending-direct:")||key.startsWith("pending-forward:"))&&refs.any{key.endsWith(":$it")}}.forEach(edit::remove)
        edit.apply()
    }
    internal suspend fun clearErasedPresentation(sources:Set<String>,names:Set<String>){
        AppImages.get(appContext).evictSources(sources)
        LabNotifications(appContext).clearAccounts(names)
    }
    fun pendingTransfer(): JSONObject? = (prefs.getString("pendingTransfer:$baseUrl:$playerRef", null) ?: prefs.getString("pendingTransfer",null))?.let { runCatching { JSONObject(it) }.getOrNull() }
        ?.takeIf { it.optString("owner") == playerRef }
    internal suspend fun saveTransfer(value: JSONObject?,scope:FinancialScope=financialScope()) {
        val key="pendingTransfer:${scope.origin}:${scope.owner}"
        pendingWrites.save(key,value?.let{JSONObject(it.toString()).put("owner",scope.owner).put("origin",scope.origin).toString()})
        if(value!=null)scope.verifyCurrent(financialScope())
    }
    fun pendingChat(): JSONObject? = prefs.getString("pendingChat", null)?.let { runCatching { JSONObject(it) }.getOrNull() }
        ?.takeIf { it.optString("owner") == playerRef }
    suspend fun savePendingChat(value: JSONObject?) {
        val scope=financialScope()
        pendingWrites.save("pendingChat",value?.put("owner",scope.owner)?.toString())
        if(value!=null)scope.verifyCurrent(financialScope())
    }
    fun pendingDirect(recipient:String):JSONObject? = prefs.getString("pending-direct:$playerRef:$recipient",null)?.let { runCatching { JSONObject(it) }.getOrNull() }
    suspend fun savePendingDirect(recipient:String,value:JSONObject?) {
        if(value!=null&&recipient in erasedPlayerRefs())throw ApiFailure("ACCOUNT_DELETED","该账号已注销")
        val scope=financialScope();val key="pending-direct:${scope.owner}:$recipient"
        pendingWrites.save(key,value?.toString())
        if(value!=null&&recipient in erasedPlayerRefs()){pendingWrites.save(key,null);throw ApiFailure("ACCOUNT_DELETED","该账号已注销")}
        if(value!=null)scope.verifyCurrent(financialScope())
    }
    fun pendingForward(recipient:String):JSONObject? = prefs.getString("pending-forward:$playerRef:$recipient",null)?.let { runCatching { JSONObject(it) }.getOrNull() }
    suspend fun savePendingForward(recipient:String,value:JSONObject?) {
        if(value!=null&&recipient in erasedPlayerRefs())throw ApiFailure("ACCOUNT_DELETED","该账号已注销")
        val scope=financialScope();val key="pending-forward:${scope.owner}:$recipient"
        pendingWrites.save(key,value?.toString())
        if(value!=null&&recipient in erasedPlayerRefs()){pendingWrites.save(key,null);throw ApiFailure("ACCOUNT_DELETED","该账号已注销")}
        if(value!=null)scope.verifyCurrent(financialScope())
    }
    fun shownNotices():Set<String> = prefs.getStringSet("shown-notices:$playerRef",emptySet()).orEmpty().toSet()
    fun saveShownNotices(ids:Set<String>){prefs.edit().putStringSet("shown-notices:$playerRef",ids.toList().takeLast(200).toSet()).apply()}
    internal fun couponReceipts(scope:FinancialScope):JSONObject = prefs.getString("coupon-receipts:${scope.origin}:${scope.owner}",null)?.let{runCatching{JSONObject(it)}.getOrNull()} ?: JSONObject()
    internal fun saveCouponReceipts(scope:FinancialScope,value:JSONObject) {
        scope.verifyCurrent(financialScope())
        prefs.edit().putString("coupon-receipts:${scope.origin}:${scope.owner}",value.toString()).apply()
    }
    fun uploadState(key:String):JSONObject? = prefs.getString("upload:$playerRef:$key",null)?.let{runCatching{JSONObject(it)}.getOrNull()}
    suspend fun saveUploadState(key:String,value:JSONObject){val scope=financialScope();pendingWrites.save("upload:${scope.owner}:$key",value.toString());scope.verifyCurrent(financialScope())}
    suspend fun clearUploadState(key:String){pendingWrites.save("upload:$playerRef:$key",null)}
    fun pendingAI(owner:String):JSONObject?=prefs.getString("ai-pending:$owner",null)?.let{runCatching{JSONObject(it)}.getOrNull()}
    suspend fun savePendingAI(owner:String,value:JSONObject?){val scope=financialScope();pendingWrites.save("ai-pending:$owner",value?.toString());if(value!=null)scope.verifyCurrent(financialScope())}
    fun pendingOperation(kind:String):JSONObject? = (prefs.getString("operation:$baseUrl:$playerRef:$kind",null) ?: prefs.getString("operation:$playerRef:$kind",null))?.let{runCatching{JSONObject(it)}.getOrNull()}
    internal suspend fun saveOperation(kind:String,value:JSONObject?,scope:FinancialScope=financialScope()) {
        val key="operation:${scope.origin}:${scope.owner}:$kind"
        pendingWrites.save(key,value?.let{JSONObject(it.toString()).put("owner",scope.owner).put("origin",scope.origin).toString()})
        if(value!=null)scope.verifyCurrent(financialScope())
    }
    suspend fun listAll(path:String,itemField:String="items"):List<JSONObject>{
        val result=mutableListOf<JSONObject>();var cursor:String?=null;val seen=mutableSetOf<String>()
        repeat(50){
            val response=request("GET",path+(if(path.contains('?'))"&" else "?")+"limit=100"+(cursor?.let{"&cursor="+java.net.URLEncoder.encode(it,"UTF-8")} ?: ""))
            val items=response.getJSONArray(itemField);for(index in 0 until items.length())result.add(items.getJSONObject(index))
            val page=response.optJSONObject("_page");cursor=page?.optString("nextCursor")?.takeUnless{it.isBlank()||it=="null"}
            check(page?.optBoolean("hasMore")!=true||cursor!=null){"服务器分页结果不完整，请稍后重试"}
            if(cursor==null)return result
            check(seen.add(cursor!!)){"服务器分页游标重复"}
        }
        error("记录过多，请缩小筛选范围")
    }

    companion object {
        @Volatile private var instance: BackendApi? = null
        fun get(context: Context): BackendApi = instance ?: synchronized(this) {
            instance ?: BackendApi(context.applicationContext).also { instance = it }
        }
    }
}
