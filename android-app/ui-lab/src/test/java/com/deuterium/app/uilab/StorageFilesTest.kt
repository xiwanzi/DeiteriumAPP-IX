package com.deuterium.app.uilab

import org.junit.Assert.*
import org.junit.Rule
import org.junit.Test
import org.junit.rules.TemporaryFolder
import java.io.File

class StorageFilesTest {
    @get:Rule val folder=TemporaryFolder()
    @Test fun cacheCleaningPreservesImageStoreAndBusinessFiles(){
        val data=folder.newFolder();val cache=File(data,"cache").apply{mkdirs()}
        File(data,"account.json").writeText("account")
        val active=File(cache,"remote-images-v1").apply{mkdirs()};File(active,"current").writeBytes(ByteArray(21))
        File(cache,"unused").writeBytes(ByteArray(37))
        assertEquals(37,StorageFiles.clearChildren(cache,setOf("remote-images-v1")))
        assertEquals(21,StorageFiles.bytes(active));assertEquals("account",File(data,"account.json").readText())
    }
    @Test fun updateCleaningOnlyRemovesRecognizedDownloads(){
        val updates=folder.newFolder();val downloads=File(updates,"downloads").apply{mkdirs()}
        File(downloads,"apk-${"a".repeat(20)}.apk").writeBytes(ByteArray(15))
        File(downloads,"resources-${"b".repeat(20)}.part").writeBytes(ByteArray(6))
        File(downloads,"upload-draft.jpg").writeText("draft")
        val active=File(updates,"resources/current").apply{mkdirs()};File(active,"strings.json").writeText("current")
        assertEquals(21,StorageFiles.clearUpdateDownloads(downloads))
        assertEquals("draft",File(downloads,"upload-draft.jpg").readText())
        assertEquals("current",File(active,"strings.json").readText())
        assertEquals(0,StorageFiles.clearUpdateDownloads(downloads))
    }
    @Test fun storageAccountingUsesActualBytes(){
        val root=folder.newFolder();File(root,"one").writeBytes(ByteArray(12))
        val nested=File(root,"nested").apply{mkdirs()};File(nested,"two").writeBytes(ByteArray(23))
        assertEquals(35,StorageFiles.bytes(root));assertEquals("1.0 MB",StorageFiles.display(1024*1024));assertEquals("0 KB",StorageFiles.display(0))
    }
    @Test fun legacyCleanupOnlySelectsOldUnreferencedImageScratchFiles(){
        val root=folder.newFolder();val old=System.currentTimeMillis()-48*3600000L
        val disposable=File(root,"listing-123-0.img").apply{writeBytes(ByteArray(13));setLastModified(old)}
        val avatar=File(root,"avatar-crop-456.png").apply{writeText("current avatar");setLastModified(old)}
        val recent=File(root,"avatar-789.img").apply{writeText("editing")}
        val business=File(root,"orders.json").apply{writeText("orders");setLastModified(old)}
        assertEquals(13,StorageFiles.clearLegacyImages(root,setOf(avatar.canonicalPath)))
        assertFalse(disposable.exists());assertTrue(avatar.exists());assertTrue(recent.exists());assertEquals("orders",business.readText())
    }
}
