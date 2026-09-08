package com.deuterium.app.network

import com.deuterium.app.BuildConfig
import com.deuterium.app.data.AiChatStreamRequest
import com.deuterium.app.data.ApiErrorResponse
import com.deuterium.app.data.ApiResponse
import com.deuterium.app.data.RepoResult
import com.google.gson.Gson
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.job
import kotlinx.coroutines.withContext
import okhttp3.Call
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import okio.BufferedSource
import retrofit2.Response
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory
import java.io.IOException
import java.net.SocketTimeoutException
import java.util.concurrent.TimeUnit

class ApiClient(private val tokenProvider: () -> String?) {
    private val gson = Gson()
    val okHttpClient: OkHttpClient = OkHttpClient.Builder()
        .connectTimeout(12, TimeUnit.SECONDS)
        .readTimeout(20, TimeUnit.SECONDS)
        .writeTimeout(20, TimeUnit.SECONDS)
        .addInterceptor { chain ->
            val original = chain.request()
            val builder = original.newBuilder()
            if (original.header("Accept").isNullOrBlank()) {
                builder.header("Accept", "application/json")
            }
            if (shouldAttachAuthorization(original.url.encodedPath)) {
                tokenProvider()?.takeIf { it.isNotBlank() }?.let { token ->
                    builder.header("Authorization", "Bearer $token")
                }
            }
            chain.proceed(builder.build())
        }
        .build()

    private val aiStreamClient: OkHttpClient = okHttpClient.newBuilder()
        .readTimeout(AI_STREAM_READ_TIMEOUT_SECONDS, TimeUnit.SECONDS)
        .writeTimeout(30, TimeUnit.SECONDS)
        .build()

    val backend: BackendApi = Retrofit.Builder()
        .baseUrl(HTTP_BASE_URL)
        .client(okHttpClient)
        .addConverterFactory(GsonConverterFactory.create(gson))
        .build()
        .create(BackendApi::class.java)

    suspend fun <T> request(block: suspend () -> Response<ApiResponse<T>>): RepoResult<T> {
        return try {
            val response = block()
            if (response.isSuccessful) {
                val data = response.body()?.data
                if (data != null) {
                    RepoResult.Success(data)
                } else {
                    RepoResult.Error("服务器响应缺少必要数据。")
                }
            } else {
                parseError(response)
            }
        } catch (e: CancellationException) {
            throw e
        } catch (e: SocketTimeoutException) {
            RepoResult.Error("连接服务器超时，请稍后再试。")
        } catch (e: IOException) {
            RepoResult.Error("网络连接不可用，请检查网络后重试。")
        } catch (e: Exception) {
            RepoResult.Error("请求处理失败，请稍后再试。")
        }
    }

    fun openChatWebSocket(includeServerEvents: Boolean, listener: WebSocketListener): WebSocket {
        val requestBuilder = Request.Builder()
            .url("$CHAT_WS_URL?includeServerEvents=$includeServerEvents")
        tokenProvider()?.takeIf { it.isNotBlank() }?.let { token ->
            requestBuilder.header("Authorization", "Bearer $token")
        }
        return okHttpClient.newWebSocket(requestBuilder.build(), listener)
    }

    suspend fun streamAiChat(
        body: AiChatStreamRequest,
        onEvent: suspend (AiStreamEvent) -> Unit
    ): RepoResult<Unit> {
        return withContext(Dispatchers.IO) {
            try {
                val request = Request.Builder()
                    .url("${HTTP_BASE_URL}ai/chat/stream")
                    .header("Accept", "text/event-stream")
                    .post(gson.toJson(body).toRequestBody(JSON_MEDIA_TYPE))
                    .build()
                val call = aiStreamClient.newCall(request)
                val cancellationHandle = currentCoroutineContext().job.invokeOnCompletion { cause ->
                    if (cause is CancellationException) call.cancel()
                }
                try {
                    executeAiStreamCall(call, onEvent)
                } finally {
                    cancellationHandle.dispose()
                }
            } catch (e: CancellationException) {
                throw e
            } catch (e: SocketTimeoutException) {
                RepoResult.Error("连接服务器超时，请稍后再试。")
            } catch (e: IOException) {
                if (currentCoroutineContext().job.isCancelled) throw CancellationException("AI stream cancelled", e)
                RepoResult.Error("网络连接不可用，请检查网络后重试。")
            } catch (e: Exception) {
                RepoResult.Error("AI 回复生成失败，请稍后再试。")
            }
        }
    }

    private suspend fun executeAiStreamCall(
        call: Call,
        onEvent: suspend (AiStreamEvent) -> Unit
    ): RepoResult<Unit> {
        call.execute().use { response ->
            if (!response.isSuccessful) {
                return parseError(response.code, response.body?.string())
            }
            val source = response.body?.source()
                ?: return RepoResult.Error("服务器响应缺少流式内容。")
            return consumeAiSseSource(source, gson, onEvent)
        }
    }

    private fun <T> parseError(response: Response<ApiResponse<T>>): RepoResult.Error {
        return parseError(response.code(), response.errorBody()?.string())
    }

    private fun parseError(statusCode: Int, raw: String?): RepoResult.Error {
        val fallback = when (statusCode) {
            401 -> "登录状态已失效，请重新登录。"
            403 -> "当前账号无权执行该操作。"
            404 -> "请求的内容不存在。"
            429 -> "操作过于频繁，请稍后再试。"
            503 -> "服务器连接暂不可用，请稍后再试。"
            else -> "服务器返回异常，请稍后再试。"
        }
        if (raw.isNullOrBlank()) return RepoResult.Error(fallback)

        return runCatching {
            val parsed = gson.fromJson(raw, ApiErrorResponse::class.java)
            val error = parsed.error
            RepoResult.Error(
                message = error?.message?.takeIf { it.isNotBlank() } ?: fallback,
                code = error?.code,
                retryAfterSeconds = error?.retryAfterSeconds
            )
        }.getOrElse {
            RepoResult.Error(fallback)
        }
    }

    private fun shouldAttachAuthorization(path: String): Boolean {
        return PublicAccountPaths.none { path.endsWith(it) }
    }

    companion object {
        val HTTP_BASE_URL: String = BuildConfig.HTTP_BASE_URL
        val CHAT_WS_URL: String = BuildConfig.CHAT_WS_URL
        private const val AI_STREAM_READ_TIMEOUT_SECONDS = 45L
        private val JSON_MEDIA_TYPE = "application/json; charset=utf-8".toMediaType()
        private val PublicAccountPaths = setOf(
            "/account/registration-code",
            "/account/register",
            "/account/login",
            "/account/password-reset-code",
            "/account/password-reset",
            "/app/update-check"
        )
    }
}

internal suspend fun consumeAiSseSource(
    source: BufferedSource,
    gson: Gson = Gson(),
    onEvent: suspend (AiStreamEvent) -> Unit
): RepoResult<Unit> {
    val parser = AiSseParser(gson)
    var terminalEventReceived = false
    suspend fun handleEvent(event: AiStreamEvent) {
        if (event is AiStreamEvent.Done || event is AiStreamEvent.Error) {
            terminalEventReceived = true
        }
        onEvent(event)
    }
    while (true) {
        val line = source.readUtf8Line() ?: break
        parser.accept(line)?.let { handleEvent(it) }
    }
    parser.finish()?.let { handleEvent(it) }
    return if (terminalEventReceived) {
        RepoResult.Success(Unit)
    } else {
        RepoResult.Error("AI 回复连接中断，请稍后重试。", code = "AI_PROVIDER_UNAVAILABLE")
    }
}
