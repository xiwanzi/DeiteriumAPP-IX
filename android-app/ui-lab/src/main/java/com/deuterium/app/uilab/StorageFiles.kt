package com.deuterium.app.uilab

import java.io.File
import java.nio.file.Files
import java.nio.file.LinkOption
import java.util.Locale

/** File accounting and deletion are restricted to app-owned, regenerable locations. */
object StorageFiles {
    private val legacyImage=Regex("(?:avatar-[0-9]+\\.img|avatar-crop-[0-9]+\\.png|listing-[0-9]+-[0-9]+\\.img)")
    fun legacyImages(root:File,protected:Set<String>,now:Long=System.currentTimeMillis()):List<File> = root.listFiles()?.filter{
        it.isFile&&!Files.isSymbolicLink(it.toPath())&&legacyImage.matches(it.name)&&it.lastModified()<now-24*3600000L&&it.canonicalPath !in protected
    }.orEmpty()
    fun clearLegacyImages(root:File,protected:Set<String>):Long {
        var freed=0L
        legacyImages(root,protected).forEach{file->val size=file.length();if(file.delete())freed+=size}
        return freed
    }
    fun bytes(root:File):Long {
        if(!root.exists()||Files.isSymbolicLink(root.toPath()))return 0
        if(root.isFile)return root.length()
        return root.walkTopDown().onEnter{!Files.isSymbolicLink(it.toPath())}
            .filter{Files.isRegularFile(it.toPath(),LinkOption.NOFOLLOW_LINKS)}.sumOf{it.length()}
    }

    fun clearChildren(root:File,keep:Set<String> = emptySet()):Long {
        val safeRoot=root.canonicalFile
        if(!root.exists())return 0
        require(!Files.isSymbolicLink(root.toPath())){"缓存目录无效"}
        var freed=0L
        fun remove(file:File){
            require(!Files.isSymbolicLink(file.toPath())&&file.canonicalFile.toPath().startsWith(safeRoot.toPath())){"缓存路径无效"}
            if(file.isDirectory)file.listFiles()?.forEach(::remove)
            val size=if(file.isFile)file.length() else 0
            if(file.delete())freed+=size else if(file.exists())throw java.io.IOException("部分缓存暂时无法清理，请稍后重试")
        }
        root.listFiles()?.filter{it.name !in keep}?.forEach(::remove)
        return freed
    }

    fun clearUpdateDownloads(root:File):Long {
        val safeRoot=root.canonicalFile
        val pattern=Regex("(apk|resources)-[a-fA-F0-9]{20}\\.(apk|zip|part)")
        require(!Files.isSymbolicLink(root.toPath())){"更新缓存目录无效"}
        var freed=0L
        root.listFiles()?.filter{it.isFile&&pattern.matches(it.name)}?.forEach{file->
            require(!Files.isSymbolicLink(file.toPath())&&file.canonicalFile.parentFile==safeRoot){"更新缓存路径无效"}
            val size=file.length()
            if(file.delete())freed+=size else if(file.exists())throw java.io.IOException("更新文件正在使用中，请稍后重试")
        }
        return freed
    }

    fun display(bytes:Long):String {
        if(bytes<=0)return "0 KB"
        val units=listOf("B","KB","MB","GB")
        var value=bytes.toDouble();var unit=0
        while(value>=1024&&unit<units.lastIndex){value/=1024;unit++}
        return if(unit==0)"${bytes} B" else String.format(Locale.US,if(value>=100)"%.0f %s" else "%.1f %s",value,units[unit])
    }
}
