package com.deuterium.app.network

import com.deuterium.app.data.RepoResult
import kotlinx.coroutines.runBlocking
import okio.Buffer
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class AiSseParserTest {
    @Test
    fun parsesMetaDeltaDoneEvents() {
        val parser = AiSseParser()
        val lines = listOf(
            "event: meta",
            "data: {\"conversationId\":\"conv-1\",\"assistantMessageId\":\"msg-a\",\"quota\":{\"remaining\":19,\"limit\":20}}",
            "",
            "event: delta",
            "data: {\"delta\":\"你好\"}",
            "",
            "event: done",
            "data: {\"message\":{\"messageId\":\"msg-a\",\"role\":\"assistant\",\"content\":\"你好\"},\"quota\":{\"remaining\":19,\"limit\":20}}",
            ""
        )

        val events = lines.mapNotNull(parser::accept) + listOfNotNull(parser.finish())

        assertEquals(3, events.size)
        assertEquals("conv-1", (events[0] as AiStreamEvent.Meta).data.conversationId)
        assertEquals("你好", (events[1] as AiStreamEvent.Delta).text)
        assertEquals("msg-a", (events[2] as AiStreamEvent.Done).data.message?.messageId)
    }

    @Test
    fun parsesErrorEventWithUnifiedBody() {
        val parser = AiSseParser()

        val event = listOf(
            "event: error",
            "data: {\"error\":{\"code\":\"AI_QUOTA_EXCEEDED\",\"message\":\"额度已用完\",\"retryAfterSeconds\":30}}",
            ""
        ).mapNotNull(parser::accept).single()

        assertTrue(event is AiStreamEvent.Error)
        val error = (event as AiStreamEvent.Error).error
        assertEquals("AI_QUOTA_EXCEEDED", error.code)
        assertEquals(30L, error.retryAfterSeconds)
    }

    @Test
    fun parsesMultilineDeltaEvent() {
        val parser = AiSseParser()

        val events = listOf(
            "event: delta",
            "data: 你",
            "data: 好",
            "",
            "event: delta",
            "data: {\"text\":\"，世界\"}",
            ""
        ).mapNotNull(parser::accept) + listOfNotNull(parser.finish())

        assertEquals(2, events.size)
        assertEquals("你\n好", (events[0] as AiStreamEvent.Delta).text)
        assertEquals("，世界", (events[1] as AiStreamEvent.Delta).text)
    }

    @Test
    fun ignoresHeartbeatAndParsesStatusAndSources() {
        val parser = AiSseParser()

        val events = listOf(
            ": keep-alive",
            "",
            "event: status",
            "data: {\"status\":\"searching_knowledge\"}",
            "",
            "event: sources",
            "data: [{\"title\":\"常见问题\",\"sourceUrl\":\"https://wiki.deuterium.cafe/zh/Q&A\",\"score\":9}]",
            ""
        ).mapNotNull(parser::accept) + listOfNotNull(parser.finish())

        assertEquals(2, events.size)
        assertEquals("searching_knowledge", (events[0] as AiStreamEvent.Status).data.status)
        assertEquals("常见问题", (events[1] as AiStreamEvent.Sources).sources.first().title)
    }

    @Test
    fun streamConsumerFailsWhenConnectionEndsWithoutTerminalEvent() = runBlocking {
        val source = Buffer().writeUtf8(
            """
            event: delta
            data: {"delta":"半截回复"}

            """.trimIndent()
        )
        val events = mutableListOf<AiStreamEvent>()

        val result = consumeAiSseSource(source) { events += it }

        assertTrue(result is RepoResult.Error)
        assertEquals("AI_PROVIDER_UNAVAILABLE", (result as RepoResult.Error).code)
        assertEquals("半截回复", (events.single() as AiStreamEvent.Delta).text)
        assertFalse(events.any { it is AiStreamEvent.Done || it is AiStreamEvent.Error })
    }
}
