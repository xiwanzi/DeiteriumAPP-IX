#include <jni.h>
#include <stdint.h>
#include <stdlib.h>
#include <webp/encode.h>
#include <webp/demux.h>

typedef struct { uint8_t* bytes; WebPAnimDecoder* decoder; WebPAnimInfo info; int stamp; } Reader;
static void fail(JNIEnv* e,const char* text){jclass c=(*e)->FindClass(e,"java/io/IOException");if(c)(*e)->ThrowNew(e,c,text);}
JNIEXPORT jbyteArray JNICALL Java_cafe_deuterium_chat_WebpCodec_encodeNative(JNIEnv* e,jclass cls,jintArray pixels,jint w,jint h,jfloat quality,jboolean lossless){
 (void)cls;
 if(w<1||h<1||w>8192||h>8192||(int64_t)w*h>8000000||(*e)->GetArrayLength(e,pixels)!=(int64_t)w*h){fail(e,"Invalid image dimensions");return NULL;}
 size_t count=(size_t)w*h;uint8_t* rgba=malloc(count*4);if(!rgba){fail(e,"Image memory limit");return NULL;}
 jint* in=(*e)->GetIntArrayElements(e,pixels,NULL);if(!in){free(rgba);return NULL;}
 for(size_t i=0;i<count;i++){uint32_t p=in[i];rgba[i*4]=p>>16;rgba[i*4+1]=p>>8;rgba[i*4+2]=p;rgba[i*4+3]=p>>24;}
 (*e)->ReleaseIntArrayElements(e,pixels,in,JNI_ABORT);
 uint8_t* out=NULL;size_t len=lossless?WebPEncodeLosslessRGBA(rgba,w,h,w*4,&out):WebPEncodeRGBA(rgba,w,h,w*4,quality,&out);free(rgba);
 if(!len||len>32*1024*1024){if(out)WebPFree(out);fail(e,"WebP encoding failed");return NULL;}
 jbyteArray result=(*e)->NewByteArray(e,(jsize)len);if(result)(*e)->SetByteArrayRegion(e,result,0,(jsize)len,(jbyte*)out);WebPFree(out);return result;
}
JNIEXPORT jlong JNICALL Java_cafe_deuterium_chat_WebpCodec_openNative(JNIEnv* e,jclass cls,jbyteArray encoded){
 (void)cls;jsize len=(*e)->GetArrayLength(e,encoded);if(len<12||len>20*1024*1024){fail(e,"Invalid image size");return 0;}
 Reader* r=calloc(1,sizeof(Reader));if(!r){fail(e,"Image memory limit");return 0;}
 r->bytes=malloc(len);if(!r->bytes){free(r);fail(e,"Image memory limit");return 0;}
 (*e)->GetByteArrayRegion(e,encoded,0,len,(jbyte*)r->bytes);
 WebPBitstreamFeatures features;if(WebPGetFeatures(r->bytes,len,&features)!=VP8_STATUS_OK||features.width<1||features.height<1||features.width>16384||features.height>16384||(int64_t)features.width*features.height>16000000){free(r->bytes);free(r);fail(e,"Image dimensions exceed budget");return 0;}
 WebPData data={r->bytes,(size_t)len};WebPAnimDecoderOptions options;WebPAnimDecoderOptionsInit(&options);options.color_mode=MODE_RGBA;options.use_threads=0;
 r->decoder=WebPAnimDecoderNew(&data,&options);
 if(!r->decoder||!WebPAnimDecoderGetInfo(r->decoder,&r->info)||r->info.frame_count>1000){if(r->decoder)WebPAnimDecoderDelete(r->decoder);free(r->bytes);free(r);fail(e,"Invalid WebP animation");return 0;}
 return (jlong)(intptr_t)r;
}
JNIEXPORT jintArray JNICALL Java_cafe_deuterium_chat_WebpCodec_infoNative(JNIEnv* e,jclass cls,jlong handle){
 (void)cls;Reader*r=(Reader*)(intptr_t)handle;if(!r)return NULL;
 jint values[]={r->info.canvas_width,r->info.canvas_height,r->info.frame_count,r->info.loop_count};jintArray out=(*e)->NewIntArray(e,4);if(out)(*e)->SetIntArrayRegion(e,out,0,4,values);return out;
}
JNIEXPORT jintArray JNICALL Java_cafe_deuterium_chat_WebpCodec_nextNative(JNIEnv* e,jclass cls,jlong handle){
 (void)cls;Reader*r=(Reader*)(intptr_t)handle;if(!r||!WebPAnimDecoderHasMoreFrames(r->decoder))return NULL;
 uint8_t* buffer;int stamp;if(!WebPAnimDecoderGetNext(r->decoder,&buffer,&stamp)){fail(e,"Invalid WebP frame");return NULL;}
 size_t count=(size_t)r->info.canvas_width*r->info.canvas_height;jintArray result=(*e)->NewIntArray(e,(jsize)count+3);if(!result)return NULL;
 jint* out=(*e)->GetIntArrayElements(e,result,NULL);if(!out)return NULL;
 out[0]=r->info.canvas_width;out[1]=r->info.canvas_height;out[2]=stamp-r->stamp;r->stamp=stamp;
 for(size_t i=0;i<count;i++)out[i+3]=((uint32_t)buffer[i*4+3]<<24)|((uint32_t)buffer[i*4]<<16)|((uint32_t)buffer[i*4+1]<<8)|buffer[i*4+2];
 (*e)->ReleaseIntArrayElements(e,result,out,0);return result;
}
JNIEXPORT void JNICALL Java_cafe_deuterium_chat_WebpCodec_closeNative(JNIEnv* e,jclass cls,jlong handle){
 (void)e;(void)cls;Reader*r=(Reader*)(intptr_t)handle;if(r){WebPAnimDecoderDelete(r->decoder);free(r->bytes);free(r);}
}
