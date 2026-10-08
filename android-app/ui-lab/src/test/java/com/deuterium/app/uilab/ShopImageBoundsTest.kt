package com.deuterium.app.uilab

import org.junit.Assert.*
import org.junit.Test

class ShopImageBoundsTest {
    @Test fun studioPaddingIsRemovedWithoutClippingTallObjects() {
        val pixels=IntArray(100*100){0xfff2f3f5.toInt()}
        for(y in 12..87)for(x in 44..55)pixels[y*100+x]=0xff385897.toInt()
        val b=shopImageBounds(100,100,pixels)
        assertTrue(b.x<44&&b.y<12&&b.x+b.width>55&&b.y+b.height>87)
        assertTrue(b.width<35&&b.height<100)
    }
    @Test fun whiteItemsAndBlackItemsOnTransparencyRemainComplete() {
        for((bg,fg) in listOf(0xfff2f3f5.toInt() to 0xffffffff.toInt(),0 to 0xff000000.toInt())) {
            val pixels=IntArray(80*80){bg};for(y in 20..59)for(x in 16..63)pixels[y*80+x]=fg
            val b=shopImageBounds(80,80,pixels)
            assertTrue(b.x<=16&&b.x+b.width>=64&&b.y<=20&&b.y+b.height>=60)
            assertTrue(b.width<80)
        }
    }
    @Test fun scenesAndBlankImagesDoNotAcquireAnInventedCrop() {
        val blank=IntArray(64*64){0xffeeeeee.toInt()}
        assertEquals(64,shopImageBounds(64,64,blank).width)
        blank[0]=0xffef1010.toInt();blank[63]=0xff10e010.toInt()
        val b=shopImageBounds(64,64,blank)
        assertEquals(ShopImageBounds(0,0,64,64,blank[0]),b)
    }
}
