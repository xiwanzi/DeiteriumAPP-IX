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
    private final RoundTexture bubble=new RoundTexture(3.6f,0);
    private static final Component TITLE=ChatStyle.title();
    private static final Component JUST_NOW=ChatStyle.body(Component.literal("刚刚"));
    private record Row(Component header,Component suffix,Component time,List<FormattedCharSequence> lines,float height,float mediaW,float mediaH,float bodyWidth,float bodyHeight,boolean emoji,List<ChatTextLine> paint){}
    private static boolean emojiOnly(ChatMessage m){String text=m.content.getString();return !m.system&&m.media==null&&!text.isBlank()&&text.codePoints().allMatch(c->Character.isWhitespace(c)||c==0xFE0F||c==0x200D||Character.getType(c)==Character.OTHER_SYMBOL||Character.getType(c)==Character.MODIFIER_SYMBOL);}
    private float inset(ChatMessage m){return m.system?0:(hud?1.8f:(WebPixels.wide()?14:13)/3f);}
    private float bodyScale(){return hud?.6f:(WebPixels.wide()?16:15)/27f;}
    private float lineHeight(){return hud?7.2f:(WebPixels.wide()?8:7.5f);}
    private float fullIndent(){return (WebPixels.wide()?49:44)/3f;}
    private float headerHeight(){return hud?6.6f:((WebPixels.wide()?14:13)*1.3f+3)/3;}
    private float textTop(Hit h){return h.y+(h.message.system?0:headerHeight())+(h.message.reply.isEmpty()?0:8)+(h.message.system||h.row.emoji?0:(hud?1.8f:1.33f));}

    private record Hit(float x,float y,float width,float height,ChatMessage message,Row row){}
    ChatTimeline(boolean hud) {
        this.hud=hud;if(hud) { HUD=this;layout(l->l.widthPercent(100).heightPercent(100)); }
        addEventListener(UIEvents.MOUSE_WHEEL,e->{scrollBy(e.deltaY);e.stopPropagation();});
        addEventListener(UIEvents.MOUSE_DOWN,e->{var point=getLocalMouse(e.x,e.y);click(point.x,point.y,e.button);e.stopPropagation();});
    }
    void saveScroll(){if(!hud&&!scope.isEmpty())ChatStore.INSTANCE.scrolls.put(scope,targetScroll);}
    void resetScroll(){saveScroll();scroll=targetScroll=ChatStore.INSTANCE.scrolls.getOrDefault(ChatStore.INSTANCE.current,0f);scope="";}
    void followLatest(){targetScroll=0;}
    void scrollBy(double amount){targetScroll=Math.clamp(targetScroll+(float)amount*25,0,Math.max(0,totalHeight-40));}
    @Override public void drawBackgroundAdditional(GUIContext ctx) {
        var mc=Minecraft.getInstance();boolean quick=mc.screen instanceof ChatScreen;
        float x=hud?4.8f:getPositionX(),y=hud?0:getPositionY();
        float width=hud?Math.max(80,mc.getWindow().getGuiScaledWidth()*.34f):getSizeWidth();
        float height=hud?mc.getWindow().getGuiScaledHeight()*.7f:getSizeHeight();
        if(height<=0||width<=0)return;
        float bottom=hud?mc.getWindow().getGuiScaledHeight()-24:y+height;
        int available=(int)((width-(hud?10:fullIndent()))/bodyScale());
        var store=ChatStore.INSTANCE;
        if(revision!=store.revision||!scope.equals(store.current)||wrapWidth!=available) {
            boolean sameScope=scope.equals(store.current)&&wrapWidth==available;float oldHeight=totalHeight;
            if(!sameScope&&!hud)scroll=targetScroll=store.scrolls.getOrDefault(store.current,0f);
            if(wrapWidth!=available)cache.clear();
            revision=store.revision;scope=store.current;wrapWidth=available;
            source=hud?store.hud():store.inScope(scope);
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
            if(hud&&++shown>16)break;
            if(cursor<bottom-height&&(hud||scroll<.1f))break;
            drawRow(ctx.graphics,m,r,x,cursor,alpha);
            hits.add(new Hit(x,cursor,width,r.height,m,r));
        }
        ctx.disableScissor();
        if(source.isEmpty()&&!hud)drawText(ctx.graphics,ChatStyle.body(Component.literal(store.status)),x+15,y+15,.6f,0xFF9BA7A2);
        var local=getLocalMouse(ctx.mouseX,ctx.mouseY);hover(ctx.graphics,(int)local.x,(int)local.y);
    }
    private Row row(ChatMessage m,int width) {
        return cache.computeIfAbsent(m.id,id->{
            var font=LDLibFonts.font();var body=ChatStyle.body(m.content);boolean emoji=!hud&&emojiOnly(m);float rowScale=emoji?29/27f:bodyScale(),rowLine=emoji?14.5f:lineHeight();
            var lines=font.split(body,Math.max(20,(int)(width*bodyScale()/rowScale)-(emoji?0:(int)(inset(m)*2/rowScale))));
            float mw=0,mh=0;
            if(m.media!=null){boolean sticker=ChatMessage.str(m.media,"type").equals("sticker");mw=sticker?(hud?16:21.33f):(hud?53.33f:64);
                float iw=m.media.has("width")?m.media.get("width").getAsFloat():1,ih=m.media.has("height")?m.media.get("height").getAsFloat():1;
                mh=sticker?mw:Math.min(hud?32:42.67f,mw*ih/Math.max(1,iw));mw=sticker?mw:Math.min(mw,mh*iw/Math.max(1,ih));}
            float textHeight=(m.media!=null&&(m.content.getString().equals("[图片]")||m.content.getString().equals("[贴纸]")))?0:Math.min(8,lines.size())*rowLine;
            var visible=textHeight==0?List.<FormattedCharSequence>of():lines.subList(0,Math.min(8,lines.size()));
            float bodyWidth=mw;for(var line:visible)bodyWidth=Math.max(bodyWidth,font.width(line)*rowScale);
            float gap=textHeight>0&&mh>0?2:0,bodyHeight=textHeight+mh+gap;
            String location=ChatPresentation.suffix(m.channel,m.server);
            var suffix=ChatStyle.body(Component.literal(location));
            return new Row(ChatStyle.playerName(m),suffix,Component.literal(m.time).withStyle(st->st.withFont(DeuteriumChat.id("numbers"))),visible,
                (m.system?0:headerHeight())+bodyHeight+(m.system||emoji?0:(hud?3.6f:2.67f))+(hud?1.8f:2.67f)+(m.reply.isEmpty()?0:8),mw,mh,bodyWidth,bodyHeight,emoji,visible.stream().map(ChatTextLine::new).toList());

        });
    }
    private void drawRow(GuiGraphics g,ChatMessage m,Row r,float x,float y,int alpha) {
        float textX=x+(hud?7.8f:fullIndent()),cursor=y;
        if(!m.system){
            drawAvatar(g,m,x,y,alpha);float nameScale=hud?.6f:(WebPixels.wide()?14:13)/27f,titleScale=hud?.6f:(WebPixels.wide()?12:11)/27f;
            drawText(g,r.header,textX,y,nameScale,alpha<<24|0xFFFFFF);
            float next=textX+LDLibFonts.font().width(r.header)*nameScale+(hud?1.8f:2.33f);
            drawText(g,TITLE,next,y+.3f,titleScale,alpha<<24|0xA9B3B8);next+=LDLibFonts.font().width(TITLE)*titleScale+(hud?1.8f:2.33f);
            if(!r.suffix.getString().isEmpty()){
                drawText(g,r.suffix,next,y+.3f,titleScale,alpha<<24|0xA9B3B8);next+=LDLibFonts.font().width(r.suffix)*titleScale+(hud?2.4f:4);
            }
            Component time=m.receivedAt!=0&&System.nanoTime()/1_000_000-m.receivedAt<60000?JUST_NOW:r.time;
            drawText(g,time,next,y+.5f,hud?.6f:(WebPixels.wide()?11:10)/27f,alpha<<24|0x9DA8AB);
            cursor+=headerHeight();
        }
        if(!m.reply.isEmpty()){drawText(g,ChatStyle.body(Component.literal("│ "+m.reply)),textX,cursor,.48f,alpha<<24|0x9BA7A2);cursor+=8;}
        float pad=r.emoji?0:inset(m),vertical=m.system||r.emoji?0:(hud?1.8f:1.33f),bodyX=hud?x+6:textX;
        float rowScale=r.emoji?29/27f:bodyScale(),rowLine=r.emoji?14.5f:lineHeight();float textHeight=r.lines.size()*rowLine;
        if(!r.lines.isEmpty()){
            float textWidth=0;for(var line:r.lines)textWidth=Math.max(textWidth,LDLibFonts.font().width(line)*rowScale);
            if(hud)bubble.setColor(Math.min(alpha,128)<<24|0x0E1422).draw(g,0,0,bodyX,cursor,textWidth+pad*2,textHeight+vertical*2,0);
            else if(!m.system&&!r.emoji)NativeBubble.draw(g,bodyX,cursor,textWidth+pad+3.67f,textHeight+vertical*2,alpha);
            cursor+=vertical;
            for(var painted:r.paint){
                float scale=rowScale;g.pose().pushPose();g.pose().translate(bodyX+pad,cursor,0);g.pose().scale(scale,scale,1);painted.draw(g,alpha<<24|0xEEF0EA,hud);g.pose().popPose();cursor+=rowLine;
            }
            cursor+=vertical;
        }
        if(m.media!=null){if(!r.lines.isEmpty())cursor+=2;ChatClient.INSTANCE.media.draw(g,m,hud?bodyX+pad:bodyX,cursor,r.mediaW,r.mediaH,alpha,false);}
        if(m.pending||!m.failure.isEmpty())drawText(g,ChatStyle.body(Component.literal(m.pending?"发送中…":m.failure)),textX,y+r.height-4,.42f,0xFFD7B6BA);
    }
    private void drawAvatar(GuiGraphics g,ChatMessage m,float x,float y,int alpha) {
        SkinFaceTexture.draw(g,m.name,x,y,hud?5.4f:(WebPixels.wide()?35:32)/3f,alpha);
    }
    static void drawText(GuiGraphics g,Component c,float x,float y,float scale,int color) {
        g.pose().pushPose();g.pose().translate(x,y,0);g.pose().scale(scale,scale,1);LDLibFonts.drawText(g,LDLibFonts.font(),c,0,0,color,false);g.pose().popPose();
    }
    private Style styleAt(Hit h,float x,float y) {
        float top=textTop(h);
        int line=(int)((y-top)/(h.row.emoji?14.5f:lineHeight()));
        return y>=top&&line<h.row.lines.size()?LDLibFonts.font().getSplitter().componentStyleAtWidth(h.row.lines.get(line),(int)((x-h.x-(hud?6:fullIndent())-(h.row.emoji?0:inset(h.message)))/(h.row.emoji?29/27f:bodyScale()))):null;
    }
    boolean click(double x,double y,int button) {
        for(var h:hits)if(x>=h.x&&x<h.x+h.width&&y>=h.y&&y<h.y+h.height){
            if(!h.message.failure.isEmpty()){ChatClient.INSTANCE.media.retry(h.message.id);return true;}
            float mediaTop=textTop(h)+h.row.lines.size()*lineHeight()+(h.row.lines.isEmpty()?0:2)+(hud?1.8f:1.33f);
            if(h.message.media!=null&&y>=mediaTop){Minecraft.getInstance().setScreen(new MediaViewer(h.message,Minecraft.getInstance().screen));return true;}
            Style style=styleAt(h,(float)x,(float)y);
            if(style!=null&&style.getClickEvent()!=null)return ChatClient.INSTANCE.click(style);
        }return false;
    }
    void hover(GuiGraphics g,int x,int y){for(var h:hits)if(x>=h.x&&x<h.x+h.width&&y>=h.y&&y<h.y+h.height){var s=styleAt(h,x,y);if(s!=null&&s.getHoverEvent()!=null)g.renderComponentHoverEffect(LDLibFonts.font(),s,x,y);}}
}
