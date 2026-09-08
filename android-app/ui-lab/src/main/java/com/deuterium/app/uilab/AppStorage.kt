package com.deuterium.app.uilab

import android.content.Context
import android.net.Uri
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.File

data class StorageSnapshot(
    val application:Long=0,val images:Long=0,val updates:Long=0,val temporary:Long=0,
    val free:Long=0,val imageBudget:Long=AppImages.DISK_LIMIT,
) {
    val total:Long get()=application+images+updates+temporary
    val cache:Long get()=images+updates+temporary
}

enum class CacheSelection { Images, Updates, Temporary, All }

object AppStorage {
    fun imageWorkFile(context:Context,name:String):File {
        require(name.matches(Regex("[a-zA-Z0-9.-]{1,100}"))&&!name.contains(".."))
        return File(context.cacheDir,"image-work").apply{mkdirs()}.resolve(name)
    }
    fun removeImageWorkFile(context:Context,uri:String){
        val source=Uri.parse(uri)
        if(source.scheme!="file")return
        val file=source.path?.let(::File) ?: return
        val parent=file.canonicalFile.parentFile
        if(parent==File(context.cacheDir,"image-work").canonicalFile)file.delete()
    }
    private fun protectedImages(context:Context):Set<String> = context.getSharedPreferences("ui-lab",Context.MODE_PRIVATE).all.values
        .filterIsInstance<String>().mapNotNull{value->runCatching{Uri.parse(value).takeIf{it.scheme=="file"}?.path?.let{File(it).canonicalPath}}.getOrNull()}.toSet()
    suspend fun measure(context:Context):StorageSnapshot=withContext(Dispatchers.IO){
        val app=context.applicationContext
        val imageCache=AppImages.get(app)
        val images=imageCache.diskBytes()
        val updates=StorageFiles.bytes(File(app.filesDir,"updates/downloads"))
        val temporary=(app.cacheDir.listFiles()?.filter{it.name!="remote-images-v1"}?.sumOf{StorageFiles.bytes(it)} ?: 0)+StorageFiles.legacyImages(app.filesDir,protectedImages(app)).sumOf{it.length()}
        val info=app.applicationInfo
        val binary=(listOf(info.sourceDir)+info.splitSourceDirs.orEmpty()).distinct().sumOf{File(it).length()}
        val privateBytes=StorageFiles.bytes(File(info.dataDir))
        StorageSnapshot((binary+privateBytes-images-updates-temporary).coerceAtLeast(0),images,updates,temporary,app.filesDir.usableSpace,imageCache.diskBudget)
    }

    suspend fun clear(context:Context,selection:CacheSelection,updates:AppUpdates?):Long {
        var freed=0L
        if(selection==CacheSelection.Images||selection==CacheSelection.All){
            val images=AppImages.get(context)
            val before=images.diskBytes()
            images.clear()
            freed+=(before-images.diskBytes()).coerceAtLeast(0)
        }
        if(selection==CacheSelection.Updates||(selection==CacheSelection.All&&updates?.downloadsProtected!=true)){
            if(updates!=null)freed+=updates.clearDownloadCache()
        }
        if(selection==CacheSelection.Temporary||selection==CacheSelection.All)
            freed+=withContext(Dispatchers.IO){StorageFiles.clearChildren(context.cacheDir,setOf("remote-images-v1"))+StorageFiles.clearLegacyImages(context.filesDir,protectedImages(context))}
        return freed
    }
}
