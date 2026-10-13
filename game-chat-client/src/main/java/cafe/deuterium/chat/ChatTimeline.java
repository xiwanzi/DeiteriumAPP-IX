package cafe.deuterium.chat;

import com.lowdragmc.lowdraglib2.gui.LDLibFonts;
import com.lowdragmc.lowdraglib2.gui.texture.SDFRectTexture;
import com.lowdragmc.lowdraglib2.gui.ui.UIElement;
import com.lowdragmc.lowdraglib2.gui.ui.event.UIEvents;
import com.lowdragmc.lowdraglib2.gui.ui.rendering.GUIContext;
import net.minecraft.client.Minecraft;
import net.minecraft.client.gui.GuiGraphics;
import net.minecraft.client.gui.screens.ChatScreen;
import net.minecraft.network.chat.*;
import net.minecraft.util.FormattedCharSequence;
import java.util.*;

/** Retained text layouts and viewport-only drawing: appending does not recreate old rows. */
final class ChatTimeline extends UIElement {
    static ChatTimeline HUD;
    private final boolean hud;
    private final Map<String,Row> cache=new HashMap<>();
    private List<ChatMessage> source=List.of();
    private long revision=-1;
    private String scope="";
    private int wrapWidth=-1;
    private float scroll,targetScroll,totalHeight;
    private long previousFrame=System.nanoTime();
    private final List<Hit> hits=new ArrayList<>();
    private RoundTexture bubble=new RoundTexture(3.6f,0);
    private static final Component JUST_NOW=ChatStyle.body(Component.literal("刚刚"));
    private record Row(ChatMessage source,Component header,Component suffix,Component time,List<FormattedCharSequence> lines,float height,float mediaW,float mediaH,float bodyWidth,float bodyHeight,boolean emoji,List<ChatTextLine> paint){}
    private static boolean emojiOnly(ChatMessage m){String text=m.content.getString();return !m.system&&m.media==null&&!text.isBlank()&&text.codePoints().allMatch(c->Character.isWhitespace(c)||c==0xFE0F||c==0x200D||Character.getType(c)==Character.OTHER_SYMBOL||Character.getType(c)==Character.MODIFIER_SYMBOL);}
    private float density(){return hud?(float)Minecraft.getInstance().getWindow().getGuiScale():3;}
    private float px(float value){return value/density();}
    private float textPx(int value){return px(ChatTextSize.pixels(value));}
    private float inset(ChatMessage m){return textPx(hud?10:12);}
    private float bodyScale(){return textPx(ChatPresentation.BODY_PX)/9;}
    private float lineHeight(){return textPx(ChatPresentation.LINE_PX);}
    private float fullIndent(){return px(hud?36:48);}
    private float headerHeight(){return textPx(ChatPresentation.HEADER_PX);}
    private float vertical(ChatMessage m,Row row){return row.emoji?0:textPx(6);}
    private float rowLine(Row row){return row.emoji?textPx(40):lineHeight();}
    private float rowScale(Row row){return row.emoji?textPx(32)/9:bodyScale();}
    private float leading(Row row){return (rowLine(row)-9*rowScale(row))/2;}
    private float bodyTop(Hit h){return h.y+(h.message.system?0:headerHeight())+(h.message.reply.isEmpty()?0:textPx(22));}
    private float textTop(Hit h){return bodyTop(h)+vertical(h.message,h.row)+leading(h.row);}
    private int densityStamp=-1;

    private record Hit(float x,float y,float width,float height,ChatMessage message,Row row){}
    ChatTimeline(boolean hud) {
        setId(hud?"chat-hud-timeline":"chat-timeline");
        this.hud=hud;if(hud) { HUD=this;layout(l->l.widthPercent(100).heightPercent(100)); }
        addEventListener(UIEvents.MOUSE_WHEEL,e->{scrollBy(e.deltaY);e.stopPropagation();});
        addEventListener(UIEvents.MOUSE_DOWN,e->{var point=getLocalMouse(e.x,e.y);click(point.x,point.y,e.button);e.stopPropagation();});
    }
    void saveScroll(){if(!hud&&!scope.isEmpty())ChatStore.INSTANCE.scrolls.put(scope,targetScroll);}
    void resetScroll(){saveScroll();scroll=targetScroll=ChatStore.INSTANCE.scrolls.getOrDefault(ChatStore.INSTANCE.current,0f);scope="";}
    void followLatest(){targetScroll=0;}
    boolean awayFromLatest(){return Math.max(scroll,targetScroll)>.5f;}
    void jumpToLatest(){scroll=targetScroll=0;if(!scope.isEmpty())ChatStore.INSTANCE.scrolls.put(scope,0f);}
    void scrollBy(double amount){targetScroll=Math.clamp(targetScroll+(float)amount*25,0,Math.max(0,totalHeight-40));}
    @Override public void drawBackgroundAdditional(GUIContext ctx) {
        var mc=Minecraft.getInstance();boolean quick=mc.screen instanceof ChatScreen;
        float x=hud?px(12):getPositionX(),y=hud?0:getPositionY();
        float width=hud?px(Math.min(540,mc.getWindow().getWidth()*.44f)):getSizeWidth();
        float height=hud?px(Math.min(WebPixels.compact()?400:600,mc.getWindow().getHeight()*.65f)):getSizeHeight();
        if(height<=0||width<=0)return;
        float bottom=hud?mc.getWindow().getGuiScaledHeight()-px(quick?Math.max(48,(float)mc.getWindow().getGuiScale()*16+12):48):y+height;
        int available=(int)((width-fullIndent())/bodyScale());
        int nextDensity=Math.round(density()*100)+(WebPixels.compact()?10000:0)+ChatTextSize.level()*100000;
        if(nextDensity!=densityStamp){densityStamp=nextDensity;bubble=new RoundTexture(px(10),0);cache.clear();revision=-1;}
        var store=ChatStore.INSTANCE;
        if(revision!=store.revision||!scope.equals(store.current)||wrapWidth!=available) {
            boolean sameScope=scope.equals(store.current)&&wrapWidth==available;float oldHeight=totalHeight;
            if(!sameScope&&!hud)scroll=targetScroll=store.scrolls.getOrDefault(store.current,0f);
            if(wrapWidth!=available)cache.clear();
            revision=store.revision;scope=store.current;wrapWidth=available;
            source=hud?store.liveHud():store.inScope(scope);
            var live=new HashSet<String>();totalHeight=0;
            for(var m:source){live.add(m.id);totalHeight+=row(m,available).height;}
            if(sameScope&&targetScroll>.1f){targetScroll+=totalHeight-oldHeight;scroll+=totalHeight-oldHeight;}
            cache.keySet().retainAll(live);
        }
        long now=System.nanoTime();float dt=Math.min(.05f,(now-previousFrame)/1e9f);previousFrame=now;
        targetScroll=Math.clamp(targetScroll,0,Math.max(0,totalHeight-height));
        scroll+=(targetScroll-scroll)*(1-(float)Math.exp(-24*dt));if(Math.abs(scroll-targetScroll)<.05f)scroll=targetScroll;
        if(hud&&!quick)scroll=targetScroll=0;
        hits.clear();ctx.enableScissor(x,hud?Math.max(0,bottom-height):y,width,height);
        float cursor=bottom+scroll;int shown=0;
        for(int i=source.size()-1;i>=0;i--) {
            var m=source.get(i);int alpha=hud&&!quick?ChatPresentation.hudAlpha(now/1_000_000-m.receivedAt):255;
            if(alpha<20)continue;Row r=row(m,available);cursor-=r.height;
            if(cursor+r.height<bottom-height)break;
            if(cursor>=bottom)continue;
            if(hud&&++shown>(WebPixels.compact()?6:10))break;
            if(cursor<bottom-height&&(hud||scroll<.1f))break;
            drawRow(ctx.graphics,m,r,x,cursor,width,alpha);
            hits.add(new Hit(x,cursor,width,r.height,m,r));
        }
        ctx.disableScissor();
        if(source.isEmpty()&&!hud)drawText(ctx.graphics,ChatStyle.body(Component.literal(store.status)),x+15,y+15,.6f,0xFF9BA7A2);
        var local=getLocalMouse(ctx.mouseX,ctx.mouseY);hover(ctx.graphics,(int)local.x,(int)local.y);
    }
    private Row row(ChatMessage m,int width) {
        var existing=cache.get(m.id);if(existing!=null&&existing.source()!=m)cache.remove(m.id);
        return cache.computeIfAbsent(m.id,id->{
            var font=LDLibFonts.font();var body=m.system?ChatStyle.systemBody(m.content):ChatStyle.body(m.content);boolean emoji=!hud&&emojiOnly(m);
            float scale=emoji?textPx(32)/9:bodyScale(),line=emoji?textPx(40):lineHeight();
            var lines=font.split(body,Math.max(20,(int)((width*bodyScale()-(emoji?0:inset(m)*2))/scale)));
            float mw=0,mh=0;
            if(m.media!=null){boolean sticker=ChatMessage.str(m.media,"type").equals("sticker");mw=Math.min(px(sticker?72:hud?180:216),width*bodyScale());
                float iw=m.media.has("width")?m.media.get("width").getAsFloat():1,ih=m.media.has("height")?m.media.get("height").getAsFloat():1;
                mh=sticker?mw:Math.min(px(hud?110:150),mw*ih/Math.max(1,iw));mw=sticker?mw:Math.min(mw,mh*iw/Math.max(1,ih));}
            boolean onlyMedia=m.media!=null&&(m.content.getString().equals("[图片]")||m.content.getString().equals("[贴纸]"));
            var visible=onlyMedia?List.<FormattedCharSequence>of():lines.subList(0,Math.min(8,lines.size()));
            float textHeight=visible.size()*line,bodyWidth=mw;for(var item:visible)bodyWidth=Math.max(bodyWidth,font.width(item)*scale);
            float padding=emoji||visible.isEmpty()?0:2*textPx(6),gap=textHeight>0&&mh>0?px(8):0,bodyHeight=textHeight+padding+mh+gap;
            String suffixText=ChatPresentation.suffix(m.channel,m.server);
            var suffix=ChatStyle.body(Component.literal(hud&&m.channel.equals("world")&&!suffixText.isEmpty()?"["+suffixText+"]":suffixText));
            return new Row(m,ChatStyle.playerName(m),suffix,ChatStyle.body(Component.literal(m.time)),visible,
                (m.system?0:headerHeight())+bodyHeight+textPx(WebPixels.compact()?14:12)+(m.reply.isEmpty()?0:textPx(22)),mw,mh,bodyWidth,bodyHeight,emoji,visible.stream().map(ChatTextLine::new).toList());
        });
    }
    private void drawRow(GuiGraphics g,ChatMessage m,Row r,float x,float y,float width,int alpha) {
        float textX=x+(m.system?0:fullIndent()),cursor=y;
        if(!m.system){
            SkinFaceTexture.draw(g,m.name,x,y,px(hud?26:36),alpha);
            float nameScale=textPx(ChatPresentation.NAME_PX)/9,detailScale=textPx(ChatPresentation.DETAIL_PX)/9;
            drawText(g,r.header,textX,y,nameScale,alpha<<24|0xFFFFFF,hud);
            float next=textX+LDLibFonts.font().width(r.header)*nameScale+px(8),right=x+width;
            Component TITLE=ChatTitles.title(m);
            if(next+LDLibFonts.font().width(TITLE)*detailScale+px(8)<right){drawText(g,TITLE,next,y+px(2),detailScale,alpha<<24|0xB4BEBC,hud);next+=LDLibFonts.font().width(TITLE)*detailScale+px(8);}
            if(!r.suffix.getString().isEmpty()&&next+LDLibFonts.font().width(r.suffix)*detailScale+px(8)<right){drawText(g,r.suffix,next,y+px(2),detailScale,alpha<<24|0xB4BEBC,hud);next+=LDLibFonts.font().width(r.suffix)*detailScale+px(10);}
            Component time=m.receivedAt!=0&&System.nanoTime()/1_000_000-m.receivedAt<60000?JUST_NOW:r.time;
            if(next+LDLibFonts.font().width(time)*detailScale<=right)drawText(g,time,next,y+px(2),detailScale,alpha<<24|0xADB7B8,hud);
            cursor+=headerHeight();
        }
        if(!m.reply.isEmpty()){drawText(g,ChatStyle.body(Component.literal("│ "+m.reply)),textX,cursor,textPx(14)/9,alpha<<24|0xA9B3B1);cursor+=textPx(22);}
        float pad=r.emoji?0:inset(m),v=vertical(m,r),scale=rowScale(r),line=rowLine(r),textHeight=r.lines.size()*line;
        if(!r.lines.isEmpty()){
            float textWidth=0;for(var item:r.lines)textWidth=Math.max(textWidth,LDLibFonts.font().width(item)*scale);
            if(m.system&&hud)bubble.setColor(Math.min(alpha,224)<<24|0x17181C).draw(g,0,0,textX,cursor,textWidth+pad*2,textHeight+v*2,0);
            else if(hud)bubble.setColor(Math.min(alpha,166)<<24|0x0E1422).draw(g,0,0,textX,cursor,textWidth+pad*2,textHeight+v*2,0);
            else if(!m.system&&!r.emoji)NativeBubble.draw(g,textX,cursor,textWidth+pad*2,textHeight+v*2,alpha);
            cursor+=v;
            for(var painted:r.paint){g.pose().pushPose();g.pose().translate(textX+pad,cursor+leading(r),0);g.pose().scale(scale,scale,1);snapText(g);painted.draw(g,alpha<<24|0xF1F3EF,false);g.pose().popPose();cursor+=line;}
            cursor+=v;
        }
        if(m.media!=null){if(!r.lines.isEmpty())cursor+=px(8);ChatClient.INSTANCE.media.draw(g,m,textX,cursor,r.mediaW,r.mediaH,alpha,false);}
        if(m.pending||!m.failure.isEmpty())drawText(g,ChatStyle.body(Component.literal(m.pending?"发送中…":m.failure)),textX,y+r.height-px(12),px(12)/9,0xFFD7B6BA);
    }
    private static void snapText(GuiGraphics g){var m=g.pose().last().pose();float d=(float)Minecraft.getInstance().getWindow().getGuiScale();m.m30(Math.round(m.m30()*d)/d);m.m31(Math.round(m.m31()*d)/d);}
    static void drawText(GuiGraphics g,Component c,float x,float y,float scale,int color) {drawText(g,c,x,y,scale,color,false);}
    private static void drawText(GuiGraphics g,Component c,float x,float y,float scale,int color,boolean shadow) {
        g.pose().pushPose();g.pose().translate(x,y,0);g.pose().scale(scale,scale,1);snapText(g);LDLibFonts.drawText(g,LDLibFonts.font(),c,0,0,color,shadow);g.pose().popPose();
    }
    private Style styleAt(Hit h,float x,float y) {
        float top=textTop(h);int line=(int)((y-top)/rowLine(h.row));
        float left=h.x+(h.message.system?0:fullIndent())+(h.row.emoji?0:inset(h.message));
        return y>=top&&x>=left&&line<h.row.lines.size()?LDLibFonts.font().getSplitter().componentStyleAtWidth(h.row.lines.get(line),(int)((x-left)/rowScale(h.row))):null;
    }
    boolean click(double x,double y,int button) {
        for(var h:hits)if(x>=h.x&&x<h.x+h.width&&y>=h.y&&y<h.y+h.height){
            if(!h.message.failure.isEmpty()){ChatClient.INSTANCE.media.retry(h.message.id);return true;}
            float mediaTop=bodyTop(h)+(h.row.lines.isEmpty()?0:h.row.lines.size()*rowLine(h.row)+vertical(h.message,h.row)*2+px(8));
            if(h.message.media!=null&&y>=mediaTop){Minecraft.getInstance().setScreen(new MediaViewer(h.message,Minecraft.getInstance().screen));return true;}
            Style style=styleAt(h,(float)x,(float)y);
            if(style!=null&&style.getClickEvent()!=null)return ChatClient.INSTANCE.click(style);
        }return false;
    }
    void hover(GuiGraphics g,int x,int y){for(var h:hits)if(x>=h.x&&x<h.x+h.width&&y>=h.y&&y<h.y+h.height){var s=styleAt(h,x,y);if(s!=null&&s.getHoverEvent()!=null)g.renderComponentHoverEffect(LDLibFonts.font(),s,x,y);}}
}
