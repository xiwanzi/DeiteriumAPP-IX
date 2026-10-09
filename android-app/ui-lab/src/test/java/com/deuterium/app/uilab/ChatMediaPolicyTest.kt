package com.deuterium.app.uilab

import org.junit.Assert.*
import org.junit.Test

class ChatMediaPolicyTest {
    @Test fun longScreenshotRetainsReadableWidthAndSmallStickerIsNotEnlarged(){
        assertEquals(ChatImageCompressor.Size(1920,1440,false),ChatImageCompressor.size(4000,3000))
        assertEquals(ChatImageCompressor.Size(1080,6500,true),ChatImageCompressor.size(1080,6500))
        assertEquals(ChatImageCompressor.Size(128,64,false),ChatImageCompressor.size(128,64,true))
        val extreme=ChatImageCompressor.size(1440,18000)
        assertTrue(extreme.height<=8192&&extreme.width.toLong()*extreme.height<=8_000_000)
    }
    @Test fun oversizedInputIsRejectedBeforeAllocation(){
        assertThrows(IllegalArgumentException::class.java){ChatImageCompressor.size(20000,20000)}
        assertThrows(IllegalArgumentException::class.java){ChatImageCompressor.size(0,10)}
    }
    @Test fun malformedRiffIsRejectedBeforeChunksAreRead(){
        assertThrows(IllegalArgumentException::class.java){ChatWebp.chunks("RIFFxxxxWEBP".toByteArray())}
        assertThrows(IllegalArgumentException::class.java){ChatWebp.chunks(byteArrayOf(1,2,3))}
    }
}
