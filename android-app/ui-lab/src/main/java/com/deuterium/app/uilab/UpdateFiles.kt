package com.deuterium.app.uilab

import java.io.File
import java.io.InputStream
import java.security.MessageDigest
import java.nio.file.Files
import java.nio.file.StandardCopyOption
import java.util.UUID
import java.util.zip.ZipFile

data class ResourceEntry(val path:String,val sizeBytes:Long,val sha256:String)

/** Files are verified in a new private directory before an active-resource pointer can change. */
object UpdateFiles {
    const val MAX_RESOURCE_BYTES=100L*1024*1024
    fun sha256(file:File)=file.inputStream().use{sha256(it)}
    fun sha256(input:InputStream):String {val digest=MessageDigest.getInstance("SHA-256");val buffer=ByteArray(32768);while(true){val count=input.read(buffer);if(count<0)break;digest.update(buffer,0,count)};return digest.digest().joinToString(""){"%02x".format(it)}}
    fun verify(file:File,sizeBytes:Long,sha256:String){require(sizeBytes>0&&file.length()==sizeBytes){"更新包大小不匹配"};require(sha256.matches(Regex("[0-9a-fA-F]{64}"))&&UpdateFiles.sha256(file).equals(sha256,true)){"更新包校验失败，请重新下载"}}
    fun safePath(path:String):Boolean=path.isNotBlank()&&path.length<=180&&!path.contains('\\')&&!path.contains(':')&&!path.startsWith('/')&&path.split('/').all{it.isNotBlank()&&it!="."&&it!=".."}&&path.substringAfterLast('.').lowercase() in setOf("json","png","webp","jpg","jpeg")
    fun extractVerified(zip:File,parent:File,version:Int,entries:List<ResourceEntry>):File {
        require(version>0&&entries.size in 1..500){"资源清单无效"}
        require(entries.all{safePath(it.path)&&it.path!="manifest.json"&&it.sizeBytes in 1..MAX_RESOURCE_BYTES&&it.sha256.matches(Regex("[0-9a-fA-F]{64}"))}){"资源包含不允许的路径或类型"}
        require(entries.map{it.path.lowercase()}.distinct().size==entries.size){"资源清单包含重复路径"}
        require(entries.sumOf{it.sizeBytes}<=MAX_RESOURCE_BYTES){"资源解压大小超出限制"}
        parent.mkdirs();val staging=File(parent,"staging-${UUID.randomUUID()}").apply{mkdirs()}
        require(staging.canonicalFile.toPath().startsWith(parent.canonicalFile.toPath()))
        try {
            val expected=entries.associateBy{it.path};val seen=mutableSetOf<String>();var extracted=0L
            ZipFile(zip).use{archive->
                val iterator=archive.entries()
                while(iterator.hasMoreElements()){
                    val item=iterator.nextElement()
                    require(!item.isDirectory){"资源包不应包含额外目录条目"}
                    if(item.name=="manifest.json"){require(seen.add(item.name)&&item.size in 1..65536){"资源清单重复或过大"};continue}
                    val spec=expected[item.name] ?: error("资源包包含清单外的文件")
                    require(seen.add(item.name)&&safePath(item.name)){"资源路径重复或不安全"}
                    val target=File(staging,item.name)
                    require(target.canonicalFile.toPath().startsWith(staging.canonicalFile.toPath())){"资源路径越界"}
                    target.parentFile?.mkdirs()
                    archive.getInputStream(item).use{input->target.outputStream().use{output->val buffer=ByteArray(32768);var written=0L;while(true){val count=input.read(buffer);if(count<0)break;written+=count;extracted+=count;require(written<=spec.sizeBytes&&extracted<=MAX_RESOURCE_BYTES){"资源实际大小超出清单"};output.write(buffer,0,count)}}}
                    verify(target,spec.sizeBytes,spec.sha256)
                }
            }
            require(seen==expected.keys+"manifest.json"){"资源包缺少必要文件"}
            val ready=File(parent,"resources-$version-${UUID.randomUUID()}")
            Files.move(staging.toPath(),ready.toPath(),StandardCopyOption.ATOMIC_MOVE)
            return ready
        }catch(error:Exception){staging.deleteRecursively();throw error}
    }
}
