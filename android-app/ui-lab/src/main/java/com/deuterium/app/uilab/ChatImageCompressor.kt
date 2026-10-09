package com.deuterium.app.uilab

import android.graphics.*
import android.media.ExifInterface
import android.os.Build
import com.bumptech.glide.gifdecoder.GifDecoder
import com.bumptech.glide.gifdecoder.GifHeaderParser
import com.bumptech.glide.gifdecoder.StandardGifDecoder
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.withContext
import java.io.ByteArrayOutputStream
import java.io.File
import kotlin.coroutines.coroutineContext
import kotlin.math.*

internal data class PreparedChatImage(val image:ByteArray,val thumbnail:ByteArray,val width:Int,val height:Int,val animated:Boolean)

internal object ChatImageCompressor {
    data class Size(val width:Int,val height:Int,val long:Boolean)
    fun size(width:Int,height:Int,sticker:Boolean=false,animated:Boolean=false):Size {
        require(width>0&&height>0&&width.toLong()*height<=40_000_000){"图片尺寸过大，请裁剪后再试"}
        val long=height.toDouble()/width>=2.5
        val factor=min(1.0,if(sticker||animated)512.0/max(width,height) else if(long)minOf(1440.0/width,8192.0/height,sqrt(8_000_000.0/(width.toDouble()*height))) else 1920.0/max(width,height))
        return Size(max(1,(width*factor).toInt()),max(1,(height*factor).toInt()),long)
    }
    @Suppress("DEPRECATION")
    private fun encode(bitmap:Bitmap,quality:Int,lossless:Boolean=false):ByteArray {
        val format=if(Build.VERSION.SDK_INT>=30){if(lossless)Bitmap.CompressFormat.WEBP_LOSSLESS else Bitmap.CompressFormat.WEBP_LOSSY}else Bitmap.CompressFormat.WEBP
        return ByteArrayOutputStream().use{out->check(bitmap.compress(format,if(lossless)if(Build.VERSION.SDK_INT>=30)30 else 100 else quality,out));out.toByteArray()}
    }
    private fun resized(b:Bitmap,w:Int,h:Int)=if(b.width==w&&b.height==h)b else Bitmap.createScaledBitmap(b,w,h,true)
    private fun thumbnail(b:Bitmap):ByteArray {
        val scale=min(1.0,320.0/max(b.width,b.height));val small=resized(b,max(1,(b.width*scale).toInt()),max(1,(b.height*scale).toInt()))
        val bytes=encode(small,70);if(small!==b)small.recycle();require(bytes.size<=80*1024){"图片预览未能处理"};return bytes
    }
    suspend fun prepare(file:File,sticker:Boolean):PreparedChatImage=withContext(Dispatchers.Default){
        require(file.length() in 1..20L*1024*1024){"请选择 20MB 以内的图片"}
        val header=ByteArray(12);file.inputStream().use{it.read(header)}
        if(String(header,0,3)=="GIF")return@withContext gif(file.readBytes())
        if(String(header,0,4)=="RIFF") {val bytes=file.readBytes();if(ChatWebp.isAnimated(bytes))return@withContext webpAnimation(bytes)}
        val bounds=BitmapFactory.Options().apply{inJustDecodeBounds=true};BitmapFactory.decodeFile(file.path,bounds)
        require(bounds.outMimeType in setOf("image/jpeg","image/png","image/webp")){"请选择 JPEG、PNG、WebP 或 GIF 图片"}
        val dimensions=size(bounds.outWidth,bounds.outHeight,sticker)
        val options=BitmapFactory.Options().apply{inSampleSize=1;while(bounds.outWidth/(inSampleSize*2)>=dimensions.width&&bounds.outHeight/(inSampleSize*2)>=dimensions.height)inSampleSize*=2;inPreferredConfig=Bitmap.Config.ARGB_8888;inPreferredColorSpace=ColorSpace.get(ColorSpace.Named.SRGB)}
        var bitmap=BitmapFactory.decodeFile(file.path,options) ?: error("无法读取图片")
        try {
            val orientation=runCatching{ExifInterface(file.path).getAttributeInt(ExifInterface.TAG_ORIENTATION,ExifInterface.ORIENTATION_NORMAL)}.getOrDefault(ExifInterface.ORIENTATION_NORMAL)
            val matrix=Matrix().apply{when(orientation){2->setScale(-1f,1f);3->setRotate(180f);4->setScale(1f,-1f);5->{setRotate(90f);postScale(-1f,1f)};6->setRotate(90f);7->{setRotate(-90f);postScale(-1f,1f)};8->setRotate(-90f)}}
            if(!matrix.isIdentity){val rotated=Bitmap.createBitmap(bitmap,0,0,bitmap.width,bitmap.height,matrix,true);if(rotated!==bitmap){bitmap.recycle();bitmap=rotated}}
            val target=size(bitmap.width,bitmap.height,sticker);val scaled=resized(bitmap,target.width,target.height);if(scaled!==bitmap){bitmap.recycle();bitmap=scaled}
            val textLike=target.long||bounds.outMimeType=="image/png";val qualities=if(textLike)listOf(90,88,84) else if(sticker)listOf(85,80,76) else listOf(82,77,72)
            val goal=if(sticker)200*1024 else if(textLike)1024*1024 else 800*1024;val limit=if(sticker)512*1024 else 2*1024*1024
            var best:ByteArray?=null
            for((index,q) in qualities.withIndex()){coroutineContext.ensureActive();val bytes=encode(bitmap,q,textLike&&index==0);if(best==null||bytes.size<best.size)best=bytes;if(best.size<=goal)break}
            require(best!!.size<=limit){"图片压缩后仍然过大，请裁剪后再试"}
            PreparedChatImage(best,thumbnail(bitmap),bitmap.width,bitmap.height,false)
        } finally {bitmap.recycle()}
    }
    private val provider=object:GifDecoder.BitmapProvider {
        override fun obtain(w:Int,h:Int,config:Bitmap.Config)=Bitmap.createBitmap(w,h,config)
        override fun release(bitmap:Bitmap){bitmap.recycle()}
        override fun obtainByteArray(size:Int)=ByteArray(size)
        override fun release(bytes:ByteArray){}
        override fun obtainIntArray(size:Int)=IntArray(size)
        override fun release(array:IntArray){}
    }
    private suspend fun gif(bytes:ByteArray):PreparedChatImage {
        validateGifFrames(bytes)
        val header=GifHeaderParser().setData(bytes).parseHeader()
        require(header.status==GifDecoder.STATUS_OK&&header.width in 1..2048&&header.height in 1..2048&&header.numFrames in 1..600&&header.width.toLong()*header.height*header.numFrames<=160_000_000){"动图尺寸或帧数过大"}
        var sample=1;while(max(header.width,header.height)/(sample*2)>=512)sample*=2
        val decoder=StandardGifDecoder(provider,header,java.nio.ByteBuffer.wrap(bytes),sample)
        try {
            val duration=(0 until decoder.frameCount).sumOf{decoder.getDelay(it).coerceAtLeast(20)};require(duration<=30000){"请选择 30 秒以内的动图"}
            return animation(header.width,header.height,decoder.totalIterationCount,decoder.frameCount,duration,{decoder.resetFrameIndex()}){
                decoder.advance();val bitmap=decoder.nextFrame ?: error("动图内容不完整");if(decoder.status!=GifDecoder.STATUS_OK){bitmap.recycle();error("动图内容不完整")};bitmap to decoder.nextDelay.coerceAtLeast(20)
            }
        } finally {decoder.clear()}
    }
    private fun validateGifFrames(bytes:ByteArray){
        require(bytes.size>=13)
        fun word(p:Int)=ChatWebp.read(bytes,p,2)
        val w=word(6);val h=word(8);require(w in 1..2048&&h in 1..2048)
        var p=13;val packed=bytes[10].toInt() and 255;if(packed and 128!=0)p+=3*(2 shl (packed and 7))
        fun subBlocks(){while(true){require(p<bytes.size);val n=bytes[p++].toInt() and 255;if(n==0)return;require(n<=bytes.size-p);p+=n}}
        var frames=0
        while(p<bytes.size){when(bytes[p++].toInt() and 255){
            0x3b->{require(frames>0);return}
            0x21->{require(p<bytes.size);p++;subBlocks()}
            0x2c->{require(p+9<=bytes.size);val x=word(p);val y=word(p+2);val fw=word(p+4);val fh=word(p+6);val flags=bytes[p+8].toInt() and 255;require(fw>0&&fh>0&&x+fw<=w&&y+fh<=h);p+=9;if(flags and 128!=0)p+=3*(2 shl (flags and 7));require(p<bytes.size);p++;subBlocks();frames++;require(frames<=600&&w.toLong()*h*frames<=160_000_000)}
            else->error("动图内容不完整")
        }};error("动图内容不完整")
    }
    private suspend fun webpAnimation(bytes:ByteArray):PreparedChatImage {
        val source=ChatWebp.animation(bytes);val canvasBitmap=Bitmap.createBitmap(source.width,source.height,Bitmap.Config.ARGB_8888);val canvas=Canvas(canvasBitmap);var index=0;var previous:ChatWebp.Frame?=null
        try {
            return animation(source.width,source.height,source.loops,source.frames.size,source.frames.sumOf{it.duration},{index=0;previous=null;canvasBitmap.eraseColor(Color.TRANSPARENT)}){
                previous?.takeIf{it.flags and 1!=0}?.let{canvas.save();canvas.clipRect(it.x,it.y,it.x+it.width,it.y+it.height);canvas.drawColor(Color.TRANSPARENT,PorterDuff.Mode.CLEAR);canvas.restore()}
                val f=source.frames[index++];val bitmap=BitmapFactory.decodeByteArray(f.bytes,0,f.bytes.size) ?: error("动图内容不完整")
                if(f.flags and 2!=0){canvas.save();canvas.clipRect(f.x,f.y,f.x+f.width,f.y+f.height);canvas.drawColor(Color.TRANSPARENT,PorterDuff.Mode.CLEAR);canvas.restore()}
                canvas.drawBitmap(bitmap,f.x.toFloat(),f.y.toFloat(),null);bitmap.recycle();previous=f
                canvasBitmap.copy(Bitmap.Config.ARGB_8888,false) to f.duration
            }
        } finally {canvasBitmap.recycle()}
    }
    private suspend fun animation(width:Int,height:Int,loops:Int,count:Int,duration:Int,reset:()->Unit,next:()->Pair<Bitmap,Int>):PreparedChatImage {
        val size=size(width,height,animated=true);var best:PreparedChatImage?=null
        val interval=max(50,ceil(duration/300.0).toInt())
        for((attempt,q) in listOf(75,68,62).withIndex()){
            reset();val factor=if(attempt==2).75 else 1.0;val w=max(1,(size.width*factor).toInt());val h=max(1,(size.height*factor).toInt());val frames=mutableListOf<ChatWebp.Frame>();var elapsed=0;var nextAt=0;var lastAt=0;var totalBytes=0;var thumb:ByteArray?=null
            repeat(count){
                coroutineContext.ensureActive();val (bitmap,delay)=next()
                try{if(elapsed>=nextAt||frames.isEmpty()){
                    if(frames.isNotEmpty())frames.last().duration=elapsed-lastAt
                    val small=resized(bitmap,w,h);try{val encoded=encode(small,q);if(thumb==null)thumb=thumbnail(small);frames+=ChatWebp.Frame(encoded,delay);totalBytes+=encoded.size}finally{if(small!==bitmap)small.recycle()}
                    nextAt=elapsed+interval;lastAt=elapsed
                };elapsed+=delay}finally{bitmap.recycle()}
                require(frames.size<=300&&totalBytes<=6*1024*1024){"动图过大，请选择较短的动图"}
            }
            frames.last().duration=elapsed-lastAt;val bytes=ChatWebp.encode(frames,w,h,loops)
            if(best==null||bytes.size<best.image.size)best=PreparedChatImage(bytes,thumb!!,w,h,true)
            if(best.image.size<=1024*1024)break
        }
        require(best!!.image.size<=2*1024*1024){"动图压缩后仍然过大，请选择较短的动图"};return best
    }
}
