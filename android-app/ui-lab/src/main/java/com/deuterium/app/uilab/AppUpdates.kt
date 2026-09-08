package com.deuterium.app.uilab

import android.content.*
import android.app.Application
import android.content.pm.PackageManager
import android.graphics.BitmapFactory
import android.net.Uri
import android.os.Build
import android.provider.Settings
import androidx.compose.runtime.*
import androidx.core.content.FileProvider
import androidx.lifecycle.AndroidViewModel
import kotlinx.coroutines.*
import org.json.*
import java.io.*
import java.net.HttpURLConnection
import java.net.URL
import java.net.URLEncoder
import java.security.MessageDigest
import java.util.zip.ZipFile

enum class UpdateKind { Apk, Resources }
enum class UpdateStage { Idle, Downloading, Verifying, Ready, Applying, Applied, Failed }
data class AppRelease(val id:String,val kind:UpdateKind,val versionName:String,val versionCode:Long,val resourceVersion:Int,
    val notes:String,val url:String,val sizeBytes:Long,val sha256:String,val minApp:Long=1,val maxApp:Long=Long.MAX_VALUE,val asset:String?=null)

/** One verified transfer at a time; no application or server credentials are sent to artifact URLs. */
class AppUpdates(application:Application):AndroidViewModel(application) {
    private val context=application
    private lateinit var prefs:android.content.SharedPreferences
    private val scope=CoroutineScope(SupervisorJob()+Dispatchers.Main.immediate)
    private var job:Job?=null
    var releases by mutableStateOf<List<AppRelease>>(emptyList());private set
    var active by mutableStateOf<AppRelease?>(null);private set
    var stage by mutableStateOf(UpdateStage.Idle);private set
    var bytes by mutableLongStateOf(0);private set
    var message by mutableStateOf("");private set
    var checking by mutableStateOf(false);private set
    var checkError by mutableStateOf<String?>(null);private set
    var demoMode by mutableStateOf(false);private set
    var lastChecked by mutableLongStateOf(0);private set
    var resourceVersion by mutableIntStateOf(0);private set
    var resourceDirectory by mutableStateOf<String?>(null);private set
    val resourceStrings=mutableStateMapOf<String,String>()
    var currentVersionCode=0L;private set
    var currentVersionName="";private set
    private var cleaningDownloads by mutableStateOf(false)
    private var installing by mutableStateOf(false)
    private var installerLaunched=false
    val busy:Boolean get()=cleaningDownloads||installing||stage in listOf(UpdateStage.Downloading,UpdateStage.Verifying,UpdateStage.Applying)
    val downloadsProtected:Boolean get()=busy||pendingInstall
    val hasUpdates:Boolean get()=releases.any{isNew(it)}
    private var readyFile:File?=null
    private var pendingInstall=false
    private fun isNew(item:AppRelease)=if(item.kind==UpdateKind.Apk)item.versionCode>currentVersionCode else item.resourceVersion>resourceVersion&&currentVersionCode in item.minApp..item.maxApp
    private fun cacheRoot()=File(context.filesDir,"updates/downloads").apply{mkdirs()}
    private fun partial(item:AppRelease)=File(cacheRoot(),"${item.kind.name.lowercase()}-${item.sha256.take(20)}.part")
    private fun completeFile(item:AppRelease)=File(cacheRoot(),"${item.kind.name.lowercase()}-${item.sha256.take(20)}.${if(item.kind==UpdateKind.Apk)"apk" else "zip"}")
    private fun version(info:android.content.pm.PackageInfo)=if(Build.VERSION.SDK_INT>=28)info.longVersionCode else @Suppress("DEPRECATION") info.versionCode.toLong()
    fun initialize(){
        if(this::prefs.isInitialized)return
        prefs=context.getSharedPreferences("app-updates-v2",Context.MODE_PRIVATE)
        val installed=context.packageManager.getPackageInfo(context.packageName,0);currentVersionCode=version(installed);currentVersionName=installed.versionName ?: ""
        resourceVersion=prefs.getInt("resourceVersion",0)
        runCatching{prefs.getString("resourceDirectory",null)?.let{path->val file=File(path);require(file.canonicalFile.toPath().startsWith(File(context.filesDir,"updates/resources").canonicalFile.toPath()));val strings=readResourceStrings(file);resourceDirectory=path;resourceStrings.putAll(strings)}}.onFailure{resourceVersion=0;message="资源记录无法读取，已使用随包资源"}
        prefs.getString("activeRelease",null)?.let{text->runCatching{decode(JSONObject(text),allowAsset=true)}.getOrNull()?.let restoreRelease@{item->if(!isNew(item)){prefs.edit().remove("activeRelease").remove("ready").apply();return@restoreRelease};active=item;val file=completeFile(item);if(prefs.getBoolean("ready",false)&&file.exists()){readyFile=file;stage=UpdateStage.Ready;bytes=file.length()}else if(partial(item).exists()){stage=UpdateStage.Failed;bytes=partial(item).length();message="上次下载已中断，可继续下载"}}}
        check()
    }
    private fun endpoint():String=context.getString(R.string.update_manifest_url).ifBlank{BuildConfig.API_BASE_URL+"/api/v1/app/update-check"}
    private fun networkUrl(value:String):URL {
        val url=URL(value);val debug=context.applicationInfo.flags and android.content.pm.ApplicationInfo.FLAG_DEBUGGABLE!=0
        require(url.userInfo==null&&(url.protocol=="https"||(debug&&url.protocol=="http"&&url.host in setOf("10.0.2.2","localhost","127.0.0.1","47.103.99.34")))){"更新地址必须使用 HTTPS"}
        return url
    }
    private fun open(value:String,offset:Long=0):HttpURLConnection {
        var url=networkUrl(value)
        repeat(5){
            val connection=(url.openConnection() as HttpURLConnection).apply{connectTimeout=15000;readTimeout=20000;instanceFollowRedirects=false;setRequestProperty("Accept-Encoding","identity");if(offset>0)setRequestProperty("Range","bytes=$offset-")}
            val code=connection.responseCode
            if(code in setOf(301,302,303,307,308)){val location=connection.getHeaderField("Location") ?: error("更新地址跳转无效");connection.disconnect();url=networkUrl(URL(url,location).toString())}
            else return connection
        };error("更新地址跳转过多")
    }
    private fun readLimited(input:InputStream,max:Long):ByteArray {val output=ByteArrayOutputStream();val buffer=ByteArray(8192);while(true){val count=input.read(buffer);if(count<0)break;require(output.size()+count<=max){"响应内容过大"};output.write(buffer,0,count)};return output.toByteArray()}
    fun check(){
        if(checking||busy)return
        checking=true;checkError=null
        scope.launch{
            runCatching{withContext(Dispatchers.IO){
                val base=endpoint();demoMode=false
                if(base.isBlank())error("尚未配置更新服务")
                else {
                    val query="versionCode=$currentVersionCode&versionName=${URLEncoder.encode(currentVersionName,"UTF-8")}&resourceVersion=$resourceVersion&packageName=${URLEncoder.encode(context.packageName,"UTF-8")}&channel=stable"
                    val connection=open(base+(if(base.contains('?'))"&" else "?")+query)
                    try{require(connection.responseCode==200){"更新服务暂不可用（${connection.responseCode}）"};val root=JSONObject(connection.inputStream.use{readLimited(it,1024*1024).toString(Charsets.UTF_8)}).getJSONObject("data");listOfNotNull(root.optJSONObject("apkUpdate")?.let{decode(it)},root.optJSONObject("resourceUpdate")?.let{decode(it)})}finally{connection.disconnect()}
                }
            }}.onSuccess{items->releases=items.filter{isNew(it)};lastChecked=System.currentTimeMillis();
                if(active!=null&&!busy&&releases.none{it.id==active?.id}){active=null;readyFile=null;stage=UpdateStage.Idle;prefs.edit().remove("activeRelease").remove("ready").apply()}
                if(!demoMode){val announced=prefs.getStringSet("announced",emptySet()).orEmpty();val fresh=releases.filter{it.id !in announced};if(fresh.isNotEmpty()){LabNotifications(context).post(DemoNotice(System.currentTimeMillis(),"有更新可用",fresh.joinToString("、"){it.versionName},"about",NotificationTopic.Updates));prefs.edit().putStringSet("announced",announced+fresh.map{it.id}).apply()}};if(stage==UpdateStage.Idle||stage==UpdateStage.Applied)message=if(releases.isEmpty())"当前已是最新版本" else "有更新可用"}
                .onFailure{checkError=it.message ?: "暂时无法检查更新";message=checkError!!}
            checking=false
        }
    }
    fun start(item:AppRelease){
        if(busy)return
        active=item;stage=UpdateStage.Downloading;message="正在下载";readyFile=null
        prefs.edit().putString("activeRelease",encode(item).toString()).putBoolean("ready",false).apply()
        job=scope.launch{
            try{
                val file=withContext(Dispatchers.IO){download(item)}
                stage=UpdateStage.Verifying;message="正在验证完整性"
                withContext(Dispatchers.IO){UpdateFiles.verify(file,item.sizeBytes,item.sha256);if(item.kind==UpdateKind.Apk)verifyApk(file,item)}
                val ready=completeFile(item)
                withContext(Dispatchers.IO){if(ready.exists())require(ready.delete());require(file.renameTo(ready)){"无法保存更新包"}}
                readyFile=ready;bytes=item.sizeBytes;prefs.edit().putBoolean("ready",true).apply()
                if(item.kind==UpdateKind.Resources)applyResources(item,ready) else {stage=UpdateStage.Ready;message="下载完成，可以安装更新"}
            }catch(cancelled:CancellationException){throw cancelled}
            catch(error:Exception){val invalid=stage==UpdateStage.Verifying;stage=UpdateStage.Failed;message=error.message ?: "下载失败，请重试";if(invalid||message.contains("校验")||message.contains("大小不匹配"))withContext(Dispatchers.IO){partial(item).delete()}}
        }
    }
    private suspend fun download(item:AppRelease):File {
        val file=partial(item)
        if(file.length()>item.sizeBytes)file.delete()
        var offset=file.length();withContext(Dispatchers.Main){bytes=offset}
        if(offset==item.sizeBytes)return file
        var connection:HttpURLConnection?=null
        try {
        val stream=if(item.asset!=null){offset=0;context.assets.open(item.asset)}else{
            connection=open(item.url,offset)
            if(connection.responseCode==206){val range=connection.getHeaderField("Content-Range").orEmpty();require(range.startsWith("bytes $offset-")){"下载续传位置不匹配"}}
            else {require(connection.responseCode==200){"下载失败（${connection.responseCode}）"};offset=0}
            connection.inputStream
        }
        stream.use{input->FileOutputStream(file,offset>0).use{output->val buffer=ByteArray(32768);var total=offset;var last=0L
            while(true){currentCoroutineContext().ensureActive();val count=input.read(buffer);if(count<0)break;total+=count;require(total<=item.sizeBytes){"更新包超过声明大小"};output.write(buffer,0,count)
                val now=System.currentTimeMillis();if(now-last>=80||total==item.sizeBytes){withContext(Dispatchers.Main){bytes=total};last=now}
                if(item.asset!=null)delay(12)
            }
            output.fd.sync()
        }}
        }finally{connection?.disconnect()}
        return file
    }
    fun cancel(){if(stage!=UpdateStage.Downloading)return;val item=active;job?.cancel();job=null;stage=UpdateStage.Idle;message="下载已取消";active=null;readyFile=null;prefs.edit().remove("activeRelease").remove("ready").apply();scope.launch(Dispatchers.IO){item?.let{partial(it).delete()}}}
    private fun certificateHashes(info:android.content.pm.PackageInfo):Set<String>{
        val signatures=if(Build.VERSION.SDK_INT>=28)info.signingInfo?.apkContentsSigners else @Suppress("DEPRECATION") info.signatures
        return signatures.orEmpty().map{signature->MessageDigest.getInstance("SHA-256").digest(signature.toByteArray()).joinToString(""){"%02x".format(it)}}.toSet()
    }
    private fun verifyApk(file:File,item:AppRelease){
        val flags=if(Build.VERSION.SDK_INT>=28)PackageManager.GET_SIGNING_CERTIFICATES else @Suppress("DEPRECATION") PackageManager.GET_SIGNATURES
        val archive=context.packageManager.getPackageArchiveInfo(file.absolutePath,flags) ?: error("不是有效的 Android 安装包")
        val installed=context.packageManager.getPackageInfo(context.packageName,flags)
        require(archive.packageName==context.packageName){"安装包不属于本应用"}
        require(version(archive)==item.versionCode&&version(archive)>version(installed)){"安装包版本不匹配或不是更新版本"}
        val current=certificateHashes(installed);require(current.isNotEmpty()&&certificateHashes(archive)==current){"安装包签名与当前应用不一致"}
    }
    fun install(){
        if(busy)return
        val item=active ?: return;val file=readyFile ?: return
        if(item.kind!=UpdateKind.Apk||stage!=UpdateStage.Ready)return
        if(!context.packageManager.canRequestPackageInstalls()){
            pendingInstall=true;message="请允许此来源安装应用，然后返回"
            context.startActivity(Intent(Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES,Uri.parse("package:${context.packageName}")).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK));return
        }
        pendingInstall=false;installing=true
        scope.launch{
            runCatching{withContext(Dispatchers.IO){UpdateFiles.verify(file,item.sizeBytes,item.sha256);verifyApk(file,item)};val uri=FileProvider.getUriForFile(context,context.packageName+".updates",file);installerLaunched=true;context.startActivity(Intent(Intent.ACTION_VIEW).setDataAndType(uri,"application/vnd.android.package-archive").addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_GRANT_READ_URI_PERMISSION))}.onFailure{installing=false;installerLaunched=false;stage=UpdateStage.Failed;message=it.message ?: "无法打开系统安装程序"}
        }
    }
    fun onResume(){if(installerLaunched){installing=false;installerLaunched=false};if(pendingInstall){if(context.packageManager.canRequestPackageInstalls())install()else pendingInstall=false};if(System.currentTimeMillis()-lastChecked>4*3600000L)check()}
    suspend fun clearDownloadCache():Long {
        check(!downloadsProtected){"更新文件正在使用中，请稍后清理"}
        cleaningDownloads=true
        try {
            val freed=withContext(Dispatchers.IO){StorageFiles.clearUpdateDownloads(cacheRoot())}
            active=null;readyFile=null;bytes=0;stage=UpdateStage.Idle;message="更新缓存已清理，可重新下载"
            prefs.edit().remove("activeRelease").remove("ready").apply()
            return freed
        } finally {cleaningDownloads=false}
    }
    fun applyReadyResources(){val item=active ?: return;val file=readyFile ?: return;if(busy||stage!=UpdateStage.Ready||item.kind!=UpdateKind.Resources)return;stage=UpdateStage.Verifying;job=scope.launch{try{withContext(Dispatchers.IO){UpdateFiles.verify(file,item.sizeBytes,item.sha256)};applyResources(item,file)}catch(error:Exception){stage=UpdateStage.Failed;message=error.message ?: "资源更新失败"}}}
    private fun readResourceStrings(directory:File):Map<String,String>{
        val file=File(directory,"strings.json");require(file.length() in 2..65536){"资源文案无效"}
        val json=JSONObject(file.readText());val allowed=setOf("commissions.subtitle","updates.resourceNote","announcement.footer")
        return json.keys().asSequence().associateWith{key->require(key in allowed){"资源包含不支持的配置项"};json.get(key).let{value->require(value is String&&value.length<=500){"资源文案类型或长度无效"};value}}
    }
    private suspend fun applyResources(item:AppRelease,file:File){
        stage=UpdateStage.Applying;message="正在更新资源"
        val installed=withContext(Dispatchers.IO){
            val manifest=ZipFile(file).use{zip->val entry=zip.getEntry("manifest.json") ?: error("缺少资源清单");JSONObject(zip.getInputStream(entry).use{readLimited(it,65536).toString(Charsets.UTF_8)})}
            require(manifest.getInt("schemaVersion")==1&&manifest.getInt("resourceVersion")==item.resourceVersion&&currentVersionCode in manifest.getLong("minAppVersionCode")..manifest.getLong("maxAppVersionCode")){"资源包与应用版本不兼容"}
            val entries=manifest.getJSONArray("entries").let{array->(0 until array.length()).map{index->val entry=array.getJSONObject(index);ResourceEntry(entry.getString("path"),entry.getLong("sizeBytes"),entry.getString("sha256"))}}
            val directory=UpdateFiles.extractVerified(file,File(context.filesDir,"updates/resources"),item.resourceVersion,entries)
            try{
                val strings=readResourceStrings(directory)
                directory.walkTopDown().filter{it.isFile&&it.extension.lowercase() in setOf("png","webp","jpg","jpeg")}.forEach{image->val options=BitmapFactory.Options().apply{inJustDecodeBounds=true};BitmapFactory.decodeFile(image.absolutePath,options);require(options.outWidth in 1..8192&&options.outHeight in 1..8192){"资源图片无法读取或尺寸过大"}}
                require(prefs.edit().putString("resourceDirectory",directory.absolutePath).putInt("resourceVersion",item.resourceVersion).remove("activeRelease").remove("ready").commit()){"无法保存资源版本"}
                directory to strings
            }catch(error:Exception){directory.deleteRecursively();throw error}
        }
        resourceDirectory=installed.first.absolutePath;resourceStrings.clear();resourceStrings.putAll(installed.second);resourceVersion=item.resourceVersion
        stage=UpdateStage.Applied;message="资源已更新，无需重新安装 App";releases=releases.filter{isNew(it)}
    }
    private fun decode(json:JSONObject,allowAsset:Boolean=false):AppRelease {
        val kind=when(json.getString("kind")){"APK"->UpdateKind.Apk;"RESOURCES"->UpdateKind.Resources;else->error("更新类型不支持")}
        val asset:String?=null
        val url=json.optString("downloadUrl");if(asset==null)networkUrl(url)
        val item=AppRelease(json.getString("releaseId"),kind,json.getString("versionName"),json.optLong("versionCode",0),json.optInt("resourceVersion",0),json.getString("releaseNotes"),url,json.getLong("sizeBytes"),json.getString("sha256"),json.optLong("minAppVersionCode",1),json.optLong("maxAppVersionCode",Long.MAX_VALUE),asset)
        require(item.id.length in 1..100&&item.notes.length<=10000&&item.versionName.length<=80&&item.sha256.matches(Regex("[a-fA-F0-9]{64}"))&&item.sizeBytes in 1..(if(kind==UpdateKind.Apk)200L*1024*1024 else 20L*1024*1024)){"更新清单无效"}
        require(item.minApp>0&&item.maxApp>=item.minApp&&(if(kind==UpdateKind.Apk)item.versionCode>0 else item.resourceVersion>0)){"更新版本无效"}
        return item
    }
    private fun encode(item:AppRelease)=JSONObject().put("releaseId",item.id).put("kind",if(item.kind==UpdateKind.Apk)"APK" else "RESOURCES").put("versionName",item.versionName).put("versionCode",item.versionCode).put("resourceVersion",item.resourceVersion).put("releaseNotes",item.notes).put("downloadUrl",item.url).put("sizeBytes",item.sizeBytes).put("sha256",item.sha256).put("minAppVersionCode",item.minApp).put("maxAppVersionCode",item.maxApp).apply{item.asset?.let{put("asset",it)}}
    override fun onCleared(){scope.cancel()}
}

val LocalAppUpdates=staticCompositionLocalOf<AppUpdates?>{null}
