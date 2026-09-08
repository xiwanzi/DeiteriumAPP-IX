package com.deuterium.app.uilab

import org.junit.Assert.*
import org.junit.Test
import java.io.StringReader

class AiStreamTest {
    @Test fun parsesSemanticEventsAndIgnoresReasoning(){
        val seen=mutableListOf<String>();val answer=StringBuilder()
        val input=": heartbeat\r\n\r\nevent: meta\r\ndata: {\"conversationId\":\"c\"}\r\n\r\nevent: response.reasoning_text.delta\r\ndata: {\"delta\":\"private reasoning\"}\r\n\r\nevent: delta\r\ndata: {\"content\":\"正文\"}\r\n\r\nevent: sources\r\ndata: {\"sources\":[{\"title\":\"来源\",\"url\":\"https://example.test\",\"origin\":\"provider_text\"}]}\r\n\r\nevent: done\r\ndata: {\"message\":{\"content\":\"正文\"}}\r\n\r\nevent: delta\r\ndata: {\"content\":\"must not append after done\"}\r\n\r\n"
        readAiEvents(StringReader(input)){event,data->seen+=event;if(event=="delta")answer.append(data.getString("content"))}
        assertEquals(listOf("meta","delta","sources","done"),seen);assertEquals("正文",answer.toString())
    }
    @Test fun endOfFileDoesNotCompletePartialAnswer(){
        val error=assertThrows(ApiFailure::class.java){readAiEvents(StringReader("event: delta\ndata: {\"content\":\"partial\"}\n\n")){_,_->}}
        assertEquals("AI_STREAM_INTERRUPTED",error.code)
    }
    @Test fun explicitFailureRetainsStableServerCode(){
        val error=assertThrows(ApiFailure::class.java){readAiEvents(StringReader("event: error\ndata: {\"error\":{\"code\":\"AI_RESPONSE_INCOMPLETE\",\"message\":\"partial\"}}\n\n")){event,data->if(event=="error")throw ApiFailure(data.getJSONObject("error").getString("code"),"partial")}}
        assertEquals("AI_RESPONSE_INCOMPLETE",error.code)
    }
}
