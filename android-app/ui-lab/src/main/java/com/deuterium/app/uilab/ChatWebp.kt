package com.deuterium.app.uilab

import java.io.ByteArrayOutputStream

/** RIFF container only. Android's WebP encoder/decoder handles actual pixels. */
internal object ChatWebp {
    data class Chunk(val kind:String,val data:ByteArray)
    data class Frame(val bytes:ByteArray,var duration:Int,val x:Int=0,val y:Int=0,val width:Int=0,val height:Int=0,val flags:Int=2)
    data class Animation(val width:Int,val height:Int,val frames:List<Frame>,val loops:Int)
    fun read(b:ByteArray,p:Int,n:Int):Int {require(p>=0&&p+n<=b.size);var v=0;repeat(n){v=v or ((b[p+it].toInt() and 255) shl (it*8))};return v}
    private fun le(v:Int,n:Int)=ByteArray(n){(v ushr (it*8)).toByte()}
    private fun chunk(name:String,data:ByteArray)=name.toByteArray(Charsets.US_ASCII)+le(data.size,4)+data+ByteArray(data.size%2)
    fun chunks(b:ByteArray):List<Chunk> {
        require(b.size>=12&&String(b,0,4,Charsets.US_ASCII)=="RIFF"&&String(b,8,4,Charsets.US_ASCII)=="WEBP"&&read(b,4,4)==b.size-8){"图片格式不正确"}
        val out=mutableListOf<Chunk>();var p=12
        while(p<b.size){require(p+8<=b.size);val n=read(b,p+4,4);require(n>=0&&n<=b.size-p-8);val end=p+8+n;require(end+n%2<=b.size);out+=Chunk(String(b,p,4,Charsets.US_ASCII),b.copyOfRange(p+8,end));p=end+n%2;require(out.size<=1024)}
        return out
    }
    fun isAnimated(b:ByteArray)=runCatching{chunks(b).any{it.kind=="ANIM"}}.getOrDefault(false)
    private fun riff(body:ByteArray)="RIFF".toByteArray()+le(body.size,4)+body
    fun still(data:ByteArray,width:Int,height:Int):ByteArray {
        val alpha=if(data.size>=4&&String(data,0,4,Charsets.US_ASCII)=="ALPH")0x10 else 0
        val x=byteArrayOf(alpha.toByte(),0,0,0)+le(width-1,3)+le(height-1,3)
        return riff("WEBP".toByteArray()+chunk("VP8X",x)+data)
    }
    fun encode(frames:List<Frame>,width:Int,height:Int,loops:Int):ByteArray {
        require(frames.isNotEmpty()&&frames.size<=300&&width in 1..512&&height in 1..512&&frames.sumOf{it.duration}<=30000)
        val body=ByteArrayOutputStream();body.write("WEBP".toByteArray());body.write(chunk("VP8X",byteArrayOf(0x12,0,0,0)+le(width-1,3)+le(height-1,3)));body.write(chunk("ANIM",ByteArray(4)+le(loops.coerceIn(0,65535),2)))
        frames.forEach{f->
            require(f.duration in 10..30000)
            val payload=ByteArrayOutputStream();payload.write(ByteArray(6)+le(width-1,3)+le(height-1,3)+le(f.duration,3)+byteArrayOf(2))
            chunks(f.bytes).filter{it.kind in setOf("ALPH","VP8 ","VP8L")}.forEach{payload.write(chunk(it.kind,it.data))}
            body.write(chunk("ANMF",payload.toByteArray()))
        };return riff(body.toByteArray())
    }
    fun animation(bytes:ByteArray):Animation {
        val parts=chunks(bytes);val x=parts.first{it.kind=="VP8X"}.data;require(x.size==10)
        val width=read(x,4,3)+1;val height=read(x,7,3)+1;require(width in 1..2048&&height in 1..2048&&width.toLong()*height<=4_000_000)
        val anim=parts.first{it.kind=="ANIM"}.data;require(anim.size==6)
        val frames=parts.filter{it.kind=="ANMF"}.map{c->
            val d=c.data;require(d.size>=24);val fx=read(d,0,3)*2;val fy=read(d,3,3)*2;val w=read(d,6,3)+1;val h=read(d,9,3)+1;val flags=d[15].toInt() and 255
            require(fx+w<=width&&fy+h<=height&&flags and 252==0)
            Frame(still(d.copyOfRange(16,d.size),w,h),read(d,12,3).coerceAtLeast(20),fx,fy,w,h,flags)
        }
        require(frames.isNotEmpty()&&frames.size<=600&&frames.sumOf{it.duration}<=30000&&width.toLong()*height*frames.size<=160_000_000)
        return Animation(width,height,frames,read(anim,4,2))
    }
}
