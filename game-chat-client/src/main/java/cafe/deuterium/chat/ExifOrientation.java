package cafe.deuterium.chat;

import java.awt.image.BufferedImage;
import java.nio.*;

final class ExifOrientation {
    private ExifOrientation(){}
    static int orientation(byte[] b){
        try{if(b.length<4||(b[0]&255)!=255||(b[1]&255)!=216)return 1;
            for(int p=2;p+4<b.length;){if((b[p]&255)!=255)break;int marker=b[p+1]&255;if(marker==218||marker==217)break;int len=(b[p+2]&255)*256+(b[p+3]&255);if(len<2||p+2+len>b.length)break;
                if(marker==225&&ChatImageCompressor.tag(b,p+4,4).equals("Exif")){
                    int start=p+10;var v=ByteBuffer.wrap(b).order(b[start]=='I'?ByteOrder.LITTLE_ENDIAN:ByteOrder.BIG_ENDIAN);int ifd=start+v.getInt(start+4);int count=Short.toUnsignedInt(v.getShort(ifd));
                    for(int i=0;i<Math.min(count,256);i++){int at=ifd+2+i*12;if(at+12>p+len+2)break;if(Short.toUnsignedInt(v.getShort(at))==0x112&&v.getShort(at+2)==3&&v.getInt(at+4)==1)return Short.toUnsignedInt(v.getShort(at+8));}
                }p+=len+2;
            }
        }catch(RuntimeException ignored){}return 1;
    }
    static BufferedImage apply(BufferedImage image,byte[] source){int o=orientation(source);if(o<2||o>8)return image;int w=image.getWidth(),h=image.getHeight();var out=new BufferedImage(o>=5?h:w,o>=5?w:h,BufferedImage.TYPE_INT_ARGB);
        for(int y=0;y<h;y++)for(int x=0;x<w;x++){int nx=x,ny=y;switch(o){case 2->nx=w-1-x;case 3->{nx=w-1-x;ny=h-1-y;}case 4->ny=h-1-y;case 5->{nx=y;ny=x;}case 6->{nx=h-1-y;ny=x;}case 7->{nx=h-1-y;ny=w-1-x;}case 8->{nx=y;ny=w-1-x;}}out.setRGB(nx,ny,image.getRGB(x,y));}image.flush();return out;
    }
}
