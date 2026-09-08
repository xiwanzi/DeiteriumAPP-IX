package com.deuterium.app.uilab

import coil3.decode.DataSource
import coil3.disk.DiskCache
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.Semaphore
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.sync.withPermit
import kotlinx.coroutines.withContext
import okhttp3.Call
import okhttp3.Callback
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okio.Path
import org.json.JSONObject
import java.io.IOException
import java.security.MessageDigest
import java.time.Instant
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.atomic.AtomicLong
import kotlin.coroutines.resumeWithException

internal data class ImageAccess(
    val url: String,
    val sha256: String = "",
    val mimeType: String? = null,
    val retainUntil: Instant? = null,
    val expired: Boolean = false,
    val privateEvidence: Boolean = false,
)

internal class CachedImage(
    val file: Path,
    val mimeType: String?,
    val dataSource: DataSource,
    private val release: () -> Unit,
) : AutoCloseable {
    override fun close() = release()
}

/** Coil owns the byte-budgeted LRU. This adapter adds signed access, integrity and single-flight reads. */
internal class ImageCacheStore(
    val disk: DiskCache,
    private val http: OkHttpClient,
    private val now: () -> Instant = Instant::now,
) {
    companion object {
        const val MAX_IMAGE_BYTES = 20L * 1024 * 1024
        fun key(scope: FinancialScope, source: String): String = hash("${scope.origin}\n${scope.owner}\n$source".toByteArray())
        fun hash(bytes: ByteArray): String = hex(MessageDigest.getInstance("SHA-256").digest(bytes))
        private fun hex(bytes: ByteArray) = bytes.joinToString("") { "%02x".format(it) }
    }

    private val locks = Array(32) { Mutex() }
    private val downloads = Semaphore(3)
    private val calls = ConcurrentHashMap.newKeySet<Call>()
    private val commits = Any()
    val generation = AtomicLong()
    private fun lock(key: String) = locks[(key.hashCode() and Int.MAX_VALUE) % locks.size]
    private fun checkAccess(access: ImageAccess?) {
        if (access?.expired == true || access?.retainUntil?.isAfter(now()) == false)
            throw ApiFailure("IMAGE_EXPIRED", "图片已过期")
    }

    suspend fun open(
        key: String,
        known: ImageAccess?,
        validSession: () -> Boolean,
        resolve: suspend (renew: Boolean) -> ImageAccess,
    ): CachedImage = withContext(Dispatchers.IO) {
        val epoch = generation.get()
        fun current() {
            if (generation.get() != epoch || !validSession()) throw CancellationException("Image request superseded")
        }
        lock(key).withLock {
            current()
            checkAccess(known)
            if (known?.privateEvidence != true) read(key, known)?.let { return@withLock it }
            downloads.withPermit {
                current()
                var access = resolve(false)
                checkAccess(access)
                var response = request(access.url)
                if (response.code == 401 || response.code == 403 || response.code == 404) {
                    response.close()
                    access = resolve(true)
                    checkAccess(access)
                    current()
                    response = request(access.url)
                }
                response.use { reply ->
                    if (!reply.isSuccessful) throw ApiFailure("IMAGE_UNAVAILABLE", "图片暂时无法加载", reply.code)
                    val body = reply.body ?: throw IOException("图片内容为空")
                    require(body.contentLength() <= MAX_IMAGE_BYTES) { "图片过大" }
                    val editor = if (!access.privateEvidence) runCatching { disk.openEditor(key) }.getOrNull() else null
                    val temporary = if (editor == null) disk.directory / "transient-${java.util.UUID.randomUUID()}.tmp" else null
                    val path = editor?.data ?: temporary!!
                    try {
                        val digest = MessageDigest.getInstance("SHA-256")
                        var total = 0L
                        disk.fileSystem.createDirectories(disk.directory)
                        body.byteStream().use { input ->
                            disk.fileSystem.sink(path).use { output ->
                                val bytes = ByteArray(32768)
                                while (true) {
                                    currentCoroutineContext().ensureActive()
                                    current()
                                    val count = input.read(bytes)
                                    if (count < 0) break
                                    total += count
                                    require(total <= MAX_IMAGE_BYTES) { "图片过大" }
                                    digest.update(bytes, 0, count)
                                    val buffer = okio.Buffer().write(bytes, 0, count)
                                    output.write(buffer, count.toLong())
                                }
                            }
                        }
                        val sha = hex(digest.digest())
                        require(total > 0 && (body.contentLength() < 0 || total == body.contentLength())) { "图片下载不完整" }
                        require(access.sha256.isBlank() || access.sha256.equals(sha, true)) { "图片完整性校验失败" }
                        val mime = access.mimeType ?: body.contentType()?.toString()?.substringBefore(';')
                        val metadata = JSONObject().put("size", total).put("sha256", sha).put("mime", mime)
                            .put("retainUntil", access.retainUntil?.toString())
                        if (editor != null) {
                            disk.fileSystem.write(editor.metadata) { writeUtf8(metadata.toString()) }
                            val snapshot = synchronized(commits) { current(); editor.commitAndOpenSnapshot() }
                            if (snapshot != null) return@withPermit CachedImage(snapshot.data, mime, DataSource.NETWORK, snapshot::close)
                            throw IOException("图片缓存暂时不可用，请重试")
                        }
                        current()
                        CachedImage(path, mime, DataSource.NETWORK) { disk.fileSystem.delete(path, mustExist = false) }
                    } catch (failure: Throwable) {
                        runCatching { editor?.abort() }
                        temporary?.let { runCatching { disk.fileSystem.delete(it, mustExist = false) } }
                        throw failure
                    }
                }
            }
        }
    }

    private fun read(key: String, known: ImageAccess?): CachedImage? {
        val snapshot = runCatching { disk.openSnapshot(key) }.getOrNull() ?: return null
        try {
            val metadata = JSONObject(disk.fileSystem.read(snapshot.metadata) { readUtf8() })
            val expiry = metadata.optString("retainUntil").takeIf { it.isNotBlank() }?.let(Instant::parse)
            if (known == null && expiry?.isAfter(now()) == false) {
                snapshot.close(); disk.remove(key)
                throw ApiFailure("IMAGE_EXPIRED", "图片已过期")
            }
            val size = disk.fileSystem.metadata(snapshot.data).size ?: 0
            require(size in 1..MAX_IMAGE_BYTES && size == metadata.getLong("size"))
            val sha = metadata.getString("sha256")
            require(known?.sha256.isNullOrBlank() || known.sha256.equals(sha, true))
            val digest = MessageDigest.getInstance("SHA-256")
            disk.fileSystem.source(snapshot.data).use { source ->
                val buffer = okio.Buffer()
                while (source.read(buffer, 32768) != -1L) digest.update(buffer.readByteArray())
            }
            require(hex(digest.digest()) == sha)
            return CachedImage(snapshot.data, metadata.optString("mime").takeIf { it.isNotBlank() }, DataSource.DISK, snapshot::close)
        } catch (failure: Exception) {
            snapshot.close()
            disk.remove(key)
            if (failure is ApiFailure) throw failure
            return null
        }
    }

    suspend fun seed(key: String, bytes: ByteArray, access: ImageAccess?) = withContext(Dispatchers.IO) {
        if (access?.privateEvidence == true || bytes.isEmpty() || bytes.size > MAX_IMAGE_BYTES) return@withContext
        val epoch = generation.get()
        lock(key).withLock {
            checkAccess(access)
            val sha = hash(bytes)
            require(access?.sha256.isNullOrBlank() || access.sha256.equals(sha, true))
            val editor = disk.openEditor(key) ?: return@withLock
            try {
                disk.fileSystem.write(editor.data) { write(bytes) }
                disk.fileSystem.write(editor.metadata) {
                    writeUtf8(JSONObject().put("size", bytes.size).put("sha256", sha).put("mime", access?.mimeType)
                        .put("retainUntil", access?.retainUntil?.toString()).toString())
                }
                synchronized(commits) {
                    if (epoch != generation.get()) throw CancellationException("Image cache cleared")
                    editor.commit()
                }
            } catch (failure: Throwable) { runCatching { editor.abort() }; throw failure }
        }
    }

    /** Generation advances before cancelling IO, so late callbacks cannot repopulate a cleared cache. */
    fun invalidateRequests() {
        synchronized(commits) { generation.incrementAndGet() }
        calls.forEach(Call::cancel)
    }

    suspend fun clear() = withContext(Dispatchers.IO) {
        invalidateRequests()
        synchronized(commits) { disk.clear() }
    }

    private suspend fun request(url: String): Response = suspendCancellableCoroutine { continuation ->
        val call = http.newCall(Request.Builder().url(url).build())
        calls.add(call)
        continuation.invokeOnCancellation { call.cancel(); calls.remove(call) }
        call.enqueue(object : Callback {
            override fun onFailure(call: Call, e: IOException) {
                calls.remove(call)
                if (continuation.isActive) continuation.resumeWithException(e)
            }
            override fun onResponse(call: Call, response: Response) {
                // Keep the call cancellable while its body is streamed, not merely until headers arrive.
                val body = response.body
                if(body==null)calls.remove(call)
                val tracked = if (body == null) response else response.newBuilder().body(object : okhttp3.ResponseBody() {
                    override fun contentType() = body.contentType()
                    override fun contentLength() = body.contentLength()
                    override fun source() = body.source()
                    override fun close() { try { body.close() } finally { calls.remove(call) } }
                }).build()
                continuation.resume(tracked) { _, value, _ -> value.close() }
            }
        })
    }
}
