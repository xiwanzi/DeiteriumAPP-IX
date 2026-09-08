package com.deuterium.app.uilab

import org.junit.Assert.*
import org.junit.Rule
import org.junit.Test
import org.junit.rules.TemporaryFolder
import java.io.File
import java.util.zip.ZipEntry
import java.util.zip.ZipOutputStream

class UpdateFilesTest {
    @get:Rule val folder=TemporaryFolder()
    private fun zip(files:Map<String,ByteArray>):File=folder.newFile().also{file->ZipOutputStream(file.outputStream()).use{zip->(mapOf("manifest.json" to "{}".toByteArray())+files).forEach{(name,bytes)->zip.putNextEntry(ZipEntry(name));zip.write(bytes);zip.closeEntry()}}}
    private fun entry(path:String,bytes:ByteArray)=ResourceEntry(path,bytes.size.toLong(),UpdateFiles.sha256(bytes.inputStream()))
    private fun fails(block:()->Unit){try{block();fail("Expected validation failure")}catch(_:IllegalArgumentException){}catch(_:IllegalStateException){}}
    @Test fun validPackActivatesInNewDirectory(){val data="{\"title\":\"new\"}".toByteArray();val parent=folder.newFolder();val archive=zip(mapOf("strings.json" to data));val target=UpdateFiles.extractVerified(archive,parent,2,listOf(entry("strings.json",data)));assertArrayEquals(data,File(target,"strings.json").readBytes());assertTrue(target.canonicalPath.startsWith(parent.canonicalPath));assertFalse(parent.listFiles()!!.any{it.name.startsWith("staging-")})}
    @Test fun corruptedPackIsRejectedByWholeFileDigest(){val file=folder.newFile().apply{writeText("content")};fails{UpdateFiles.verify(file,file.length(),"0".repeat(64))}}
    @Test fun declaredSizeMustMatch(){val file=folder.newFile().apply{writeText("content")};fails{UpdateFiles.verify(file,file.length()+1,UpdateFiles.sha256(file))}}
    @Test fun traversalAndExecutablePathsAreRejected(){listOf("../outside.json","/absolute.json","C:/outside.json","images\\outside.png","images/../outside.png","x.dex","x.jar","x.so","index.html").forEach{assertFalse(it,UpdateFiles.safePath(it))}}
    @Test fun extraZipEntryDoesNotTouchExistingResources(){val parent=folder.newFolder();val old=File(parent,"old").apply{mkdirs()};File(old,"strings.json").writeText("stable");val data="{}".toByteArray();fails{UpdateFiles.extractVerified(zip(mapOf("strings.json" to data,"extra.json" to data)),parent,2,listOf(entry("strings.json",data)))};assertEquals("stable",File(old,"strings.json").readText());assertEquals(listOf("old"),parent.list()!!.toList())}
    @Test fun missingManifestEntryIsRejected(){val data="{}".toByteArray();fails{UpdateFiles.extractVerified(zip(emptyMap()),folder.newFolder(),2,listOf(entry("strings.json",data)))}}
    @Test fun fileDigestFailureRollsBackStaging(){val data="{}".toByteArray();val parent=folder.newFolder();fails{UpdateFiles.extractVerified(zip(mapOf("strings.json" to data)),parent,2,listOf(entry("strings.json",data).copy(sha256="0".repeat(64))))};assertTrue(parent.list()!!.isEmpty())}
    @Test fun duplicateManifestPathsAreRejected(){val data="{}".toByteArray();fails{UpdateFiles.extractVerified(zip(mapOf("strings.json" to data)),folder.newFolder(),2,listOf(entry("strings.json",data),entry("strings.json",data)))}}
    @Test fun oversizedExpandedFileIsRejected(){val data=ByteArray(100);fails{UpdateFiles.extractVerified(zip(mapOf("strings.json" to data)),folder.newFolder(),2,listOf(entry("strings.json",data).copy(sizeBytes=50)))}}
    @Test fun invalidResourceVersionIsRejected(){val data="{}".toByteArray();fails{UpdateFiles.extractVerified(zip(mapOf("strings.json" to data)),folder.newFolder(),0,listOf(entry("strings.json",data)))}}
}
