package com.deuterium.app.uilab

import org.junit.Assert.*
import org.junit.Test
import java.time.Instant
import java.time.ZoneId
import java.util.Locale
import java.util.UUID

class AiReplyTemplateTest {
    @Test fun streamingSnapshotsKeepIdentityTimeSourcesAndExactGrowingMarkdown() {
        val template=aiReplyTemplate("reply-id","客服小祥","2026-09-11T04:05:06Z",ZoneId.of("Asia/Shanghai"),Locale.US)
        val sources=listOf(AiSource("来源","https://example.com/source","annotation"))
        var text=""
        for(delta in listOf("**你","好**\n\n","|列|值|\n|---|---|\n","|A|B|")){
            text+=delta;val line=template.copy(text=text,aiStatus="streaming",sources=sources)
            assertEquals(text,line.text);assertEquals("12:05",line.time);assertEquals("reply-id",line.remoteId)
            assertEquals(UUID.nameUUIDFromBytes("reply-id".toByteArray()).mostSignificantBits,line.id)
            assertEquals(Instant.parse("2026-09-11T04:05:06Z").toEpochMilli(),line.serverAt)
            assertEquals(sources,line.sources);assertFalse(line.mine);assertFalse(line.searchUsed)
        }
        assertEquals("unknown",template.copy(text=text,aiStatus="unknown").aiStatus)
    }

    @Test fun changedTimezoneUsesTheSameInstantAndTheCorrectLocalDisplayTime() {
        val local=aiReplyTemplate("id","小祥","2026-09-11T04:05:06Z",ZoneId.of("Asia/Shanghai"),Locale.US)
        val utc=aiReplyTemplate("id","小祥","2026-09-11T04:05:06Z",ZoneId.of("UTC"),Locale.US)
        assertEquals(local.id,utc.id);assertEquals(local.serverAt,utc.serverAt)
        assertEquals("12:05",local.time);assertEquals("04:05",utc.time)
    }
}
