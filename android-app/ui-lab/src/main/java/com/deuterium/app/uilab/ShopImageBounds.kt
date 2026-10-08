package com.deuterium.app.uilab

import kotlin.math.*

internal data class ShopImageBounds(val x:Int,val y:Int,val width:Int,val height:Int,val background:Int)

/** Only uniform studio backgrounds are trimmed. Scene/photos retain the whole image. */
internal fun shopImageBounds(w:Int,h:Int,pixels:IntArray):ShopImageBounds {
    require(w>0&&h>0&&pixels.size==w*h)
    val corners=listOf(pixels[0],pixels[w-1],pixels[(h-1)*w],pixels.last())
    fun distance(a:Int,b:Int):Int {
        if((a ushr 24)<32&&(b ushr 24)<32)return 0
        if(abs((a ushr 24)-(b ushr 24))>20)return 255
        return maxOf(abs((a shr 16 and 255)-(b shr 16 and 255)),abs((a shr 8 and 255)-(b shr 8 and 255)),abs((a and 255)-(b and 255)))
    }
    val bg=corners.first()
    if(corners.any{distance(it,bg)>10})return ShopImageBounds(0,0,w,h,bg)
    var left=w;var top=h;var right=-1;var bottom=-1
    for(y in 0 until h)for(x in 0 until w){val p=pixels[y*w+x];if((p ushr 24)>32&&distance(p,bg)>10){left=min(left,x);right=max(right,x);top=min(top,y);bottom=max(bottom,y)}}
    if(right-left<w/12||bottom-top<h/12)return ShopImageBounds(0,0,w,h,bg)
    val margin=(max(right-left,bottom-top)*.065f).roundToInt().coerceAtLeast(2)
    left=(left-margin).coerceAtLeast(0);right=(right+margin).coerceAtMost(w-1)
    top=(top-margin).coerceAtLeast(0);bottom=(bottom+margin).coerceAtMost(h-1)
    return ShopImageBounds(left,top,right-left+1,bottom-top+1,bg)
}

