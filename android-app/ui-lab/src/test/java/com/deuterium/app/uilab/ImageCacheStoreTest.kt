package com.deuterium.app.uilab

import coil3.decode.DataSource
import coil3.disk.DiskCache
import kotlinx.coroutines.*
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import okio.Buffer
import okio.Path.Companion.toOkioPath
import org.junit.Assert.*
import org.junit.Rule
import org.junit.Test
import org.junit.rules.TemporaryFolder
import java.time.Instant
import java.util.concurrent.TimeUnit

class ImageCacheStoreTest {
    @get:Rule val folder=TemporaryFolder()
    private val bytes="a downloaded image body".toByteArray()
    private val client=OkHttpClient.Builder().readTimeout(3,TimeUnit.SECONDS).build()
    private fun cache()=DiskCache.Builder().directory(folder.newFolder().toOkioPath()).maxSizeBytes(1024*1024).build()
    private fun response()=MockResponse().setBody(Buffer().write(bytes)).setHeader("Content-Type","image/png")
    private fun access(server:MockWebServer)=ImageAccess(server.url("/photo?signature=one").toString(),ImageCacheStore.hash(bytes),"image/png")

    @Test fun concurrentConsumersDownloadOnceAndRotatedSignaturesReuseBytes()=runBlocking {
        MockWebServer().use{server->
            server.enqueue(response().setBodyDelay(100,TimeUnit.MILLISECONDS))
            val disk=cache();val store=ImageCacheStore(disk,client);val access=access(server)
            coroutineScope{(1..8).map{async{store.open("same",access,{true}){access}.use{image->assertArrayEquals(bytes,disk.fileSystem.read(image.file){readByteArray()})}}}.awaitAll()}
            store.open("same",access.copy(url=server.url("/photo?signature=two").toString()),{true}){error("Disk hit must precede authorization")}.use{assertEquals(DataSource.DISK,it.dataSource)}
            assertEquals(1,server.requestCount)
            disk.shutdown()
        }
    }

    @Test fun diskCacheSurvivesLoaderRestartWithoutSignedUrl()=runBlocking {
        MockWebServer().use{server->
            server.enqueue(response());val first=cache();val directory=first.directory
            ImageCacheStore(first,client).open("same",access(server),{true}){access(server)}.close()
            first.shutdown()
            val second=DiskCache.Builder().directory(directory).maxSizeBytes(1024*1024).build()
            ImageCacheStore(second,client).open("same",null,{true}){error("Restart should load offline")}.use{assertEquals(DataSource.DISK,it.dataSource)}
            assertEquals(1,server.requestCount);second.shutdown()
        }
    }

    @Test fun clearingCancelsAnInFlightWriterAndAllowsCleanRedownload()=runBlocking {
        MockWebServer().use{server->
            server.enqueue(response().throttleBody(1,1,TimeUnit.SECONDS));server.enqueue(response())
            val disk=cache();val store=ImageCacheStore(disk,client)
            val writer=async{runCatching{store.open("same",access(server),{true}){access(server)}.close()}}
            withContext(Dispatchers.IO){assertNotNull(server.takeRequest(3,TimeUnit.SECONDS))}
            store.clear()
            assertTrue(withTimeout(4000){writer.await()}.isFailure)
            assertNull(disk.openSnapshot("same"))
            store.open("same",access(server),{true}){access(server)}.close()
            assertEquals(2,server.requestCount);disk.shutdown()
        }
    }

    @Test fun accountAndOriginArePartOfIdentity(){
        val a=FinancialScope("alice","https://one")
        assertNotEquals(ImageCacheStore.key(a,"asset:photo"),ImageCacheStore.key(a.copy(owner="bob"),"asset:photo"))
        assertNotEquals(ImageCacheStore.key(a,"asset:photo"),ImageCacheStore.key(a.copy(origin="https://two"),"asset:photo"))
    }

    @Test fun corruptDiskContentIsReplacedInsteadOfRepeatedlyShowingAnError()=runBlocking {
        MockWebServer().use{server->
            server.enqueue(response());server.enqueue(response())
            val disk=cache();val store=ImageCacheStore(disk,client)
            store.open("same",access(server),{true}){access(server)}.close()
            val path=disk.openSnapshot("same")!!.use{it.data}
            disk.fileSystem.write(path){write(ByteArray(bytes.size))}
            store.open("same",access(server),{true}){access(server)}.use{assertArrayEquals(bytes,disk.fileSystem.read(it.file){readByteArray()})}
            assertEquals(2,server.requestCount);disk.shutdown()
        }
    }

    @Test fun incompleteOrWrongHashDataNeverCommits()=runBlocking {
        MockWebServer().use{server->
            server.enqueue(response());val disk=cache();val store=ImageCacheStore(disk,client)
            val invalid=access(server).copy(sha256="0".repeat(64))
            assertTrue(runCatching{store.open("same",invalid,{true}){invalid}.close()}.isFailure)
            assertNull(disk.openSnapshot("same"));disk.shutdown()
        }
    }

    @Test fun expiredRetiredImageCannotBeResurrectedFromDisk()=runBlocking {
        val disk=cache();var now=Instant.parse("2026-09-08T00:00:00Z");val store=ImageCacheStore(disk,client){now}
        store.seed("same",bytes,ImageAccess("",retainUntil=now.plusSeconds(30)))
        now=now.plusSeconds(31)
        val error=runCatching{store.open("same",null,{true}){error("No request should be needed")}}.exceptionOrNull()
        assertEquals("IMAGE_EXPIRED",(error as ApiFailure).code)
        assertNull(disk.openSnapshot("same"));disk.shutdown()
    }

    @Test fun restoredLiveImageOverridesOldRetirementDeadline()=runBlocking {
        val disk=cache();val start=Instant.parse("2026-09-08T00:00:00Z");var now=start;val store=ImageCacheStore(disk,client){now}
        store.seed("same",bytes,ImageAccess("",retainUntil=start.plusSeconds(30)))
        now=start.plusSeconds(31)
        store.open("same",ImageAccess("",ImageCacheStore.hash(bytes)),{true}){error("Restored unchanged bytes should be reused")}.close()
        disk.shutdown()
    }

    @Test fun privateEvidenceIsTransientAndNeverStoredAsADiskEntry()=runBlocking {
        MockWebServer().use{server->
            server.enqueue(response());val disk=cache();val store=ImageCacheStore(disk,client)
            val access=access(server).copy(privateEvidence=true)
            val result=store.open("private",access,{true}){access};val path=result.file
            assertTrue(disk.fileSystem.exists(path));result.close()
            assertFalse(disk.fileSystem.exists(path));assertNull(disk.openSnapshot("private"));disk.shutdown()
        }
    }

    @Test fun forbiddenDownloadRenewsBusinessAuthorizationOnlyOnce()=runBlocking {
        MockWebServer().use{server->
            server.enqueue(MockResponse().setResponseCode(403));server.enqueue(response())
            val disk=cache();val store=ImageCacheStore(disk,client);val renews=mutableListOf<Boolean>()
            store.open("same",null,{true}){renew->renews+=renew;access(server)}.close()
            assertEquals(listOf(false,true),renews);assertEquals(2,server.requestCount);disk.shutdown()
        }
    }
}
