package cafe.deuterium.chat;

import com.lowdragmc.lowdraglib2.gui.holder.ModularUIScreen;
import com.lowdragmc.lowdraglib2.gui.ui.*;
import com.lowdragmc.lowdraglib2.gui.ui.elements.*;
import net.minecraft.client.Minecraft;
import net.minecraft.client.gui.GuiGraphics;
import net.minecraft.client.gui.screens.Screen;
import net.minecraft.network.chat.Component;
import org.lwjgl.glfw.GLFW;

final class MediaViewer extends ModularUIScreen {
    private final ChatMessage message;private final Screen previous;private final String previousPanel;
    private final UIElement root;private final Button minus,plus,fit,collect,close;private final Label percentage;
    private float zoom=1,panX,panY;private long clickAt;private int lastPercent;private boolean collected;
    private final Runnable confirm,dismiss;private boolean submitted,disposed;
    MediaViewer(ChatMessage message,Screen previous){this(message,previous,null,null);}
    MediaViewer(ChatMessage message,Screen previous,Runnable confirm,Runnable dismiss){this(message,previous,new UIElement(),confirm,dismiss);}
    private MediaViewer(ChatMessage message,Screen previous,UIElement root,Runnable confirm,Runnable dismiss){
        super(ModularUI.of(UI.of(root.layout(l->l.widthPercent(100).heightPercent(100)),size->WebPixels.size())),Component.literal("图片"));
        ChatSounds.play("card");WebPixels.apply(root);this.message=message;this.previous=QuickChatHooks.returnTo(previous);this.root=root;this.confirm=confirm;this.dismiss=dismiss;previousPanel=previous instanceof ChatScreenFull full?ChatPanels.state(full.view.root):"";
        minus=ChatStyle.button("−",()->setZoom(zoom/1.3f));plus=ChatStyle.button("+",()->setZoom(zoom*1.3f));
        fit=ChatStyle.button("适应窗口",()->{setZoom(1);panX=panY=0;});collect=ChatStyle.button(confirm==null?"收藏为贴纸":"发送",()->{if(confirm==null)ChatClient.INSTANCE.media.collect(message.media);else{submitted=true;confirm.run();onClose();}});close=ChatStyle.iconButton("close",this::onClose);
        ChatStyle.sound(minus,"");ChatStyle.sound(plus,"");ChatStyle.sound(fit,"");ChatStyle.sound(close,"");ChatStyle.sound(collect,confirm==null?"card":"send");minus.textStyle(st->st.fontSize(19/3f));plus.textStyle(st->st.fontSize(19/3f));fit.textStyle(st->st.fontSize(11/3f));
        minus.setId("viewer-minus");plus.setId("viewer-plus");fit.setId("viewer-fit");collect.setId("viewer-collect");close.setId("viewer-close");
        percentage=ChatStyle.label("100%",11/3f,ChatStyle.TEXT);root.addChildren(minus,percentage,plus,fit,collect,close);root.addChild(ChatStyle.rect(ChatStyle.label("sticker".equals(ChatMessage.str(message.media,"type"))?"贴纸":"图片",5,ChatStyle.TEXT),32/3f,8,100,8));
        collect.textStyle(st->st.fontSize(4));collect.buttonStyle(st->st.baseTexture(new RoundTexture(1,0x08C8D9BE,1/3f,0x3BBDCEBC)).hoverTexture(new RoundTexture(1,0x14C8D9BE,1/3f,0x77E0E9CC)));
        for(var control:java.util.List.of(minus,plus))control.buttonStyle(st->st.baseTexture(new RoundTexture(1,0,1/3f,0x2BC3D0BB)).hoverTexture(new RoundTexture(1,0x14C7D5BB,1/3f,0x67D8E1C8)));
        var notice=ChatStyle.label("滚轮缩放 · 拖动查看",4,0xFFB8CFB8);notice.lss("position","absolute");notice.getLayout().left(32/3f).bottom(7).width(160).height(6);root.addChild(notice);
    }
    @Override public void init(){WebPixels.apply(root);super.init();int width=WebPixels.width(),height=WebPixels.height();float start=width-248/3f;
        ChatStyle.rect(minus,start,height-46.5f/3f,11,11);ChatStyle.rect(percentage,start+49/3f,height-36/3f,14,5);ChatStyle.rect(plus,start+107/3f,height-46.5f/3f,11,11);ChatStyle.rect(fit,start+156/3f,height-46.5f/3f,20,11);
        ChatStyle.rect(collect,width-187/3f,17/3f,100/3f,34/3f);ChatStyle.rect(close,width-70/3f,5,38/3f,38/3f);}
    private void setZoom(float value){zoom=Math.clamp(value,1,6);}
    @Override public void render(GuiGraphics g,int x,int y,float tick){
        ChatBackdrop.draw(g,0,0,width,height,3,1);g.fill(0,0,width,height,0xFC10181C);
        g.pose().pushPose();g.pose().scale(3f/(float)Minecraft.getInstance().getWindow().getGuiScale(),3f/(float)Minecraft.getInstance().getWindow().getGuiScale(),1);int width=WebPixels.width(),height=WebPixels.height();
        float iw=message.media.get("width").getAsFloat(),ih=message.media.get("height").getAsFloat();float areaW=width-56/3f,areaH=height-184/3f;
        float gui=3;float maxPixels="sticker".equals(ChatMessage.str(message.media,"type"))?320:Math.max(iw,ih);float natural=Math.min(1,maxPixels/Math.max(iw,ih))/gui;float scale=Math.min(natural,Math.min(areaW/iw,areaH/ih));float w=iw*scale*zoom,h=ih*scale*zoom;
        panX=Math.clamp(panX,-Math.max(0,(w-areaW)/2),Math.max(0,(w-areaW)/2));panY=Math.clamp(panY,-Math.max(0,(h-areaH)/2),Math.max(0,(h-areaH)/2));
        g.enableScissor(0,68/3,width,height-20);ChatClient.INSTANCE.media.draw(g,message,(width-w)/2+panX,96/3f+(areaH-h)/2+panY,w,h,255,true);g.disableScissor();
        int percent=Math.round(zoom*100);if(percent!=lastPercent){percentage.setText(Component.literal(percent+"%"));lastPercent=percent;}
        boolean saved=false;String id=ChatMessage.str(message.media,"mediaId");for(var e:ChatClient.INSTANCE.media.stickers())if(ChatMessage.str(e.getAsJsonObject(),"mediaId").equals(id))saved=true;
        if(confirm==null&&saved!=collected){collected=saved;collect.setText(saved?"已收藏":"收藏为贴纸",false);collect.setActive(!saved);}
        g.fill(0,height-20,width,height,0x8811191B);g.pose().pushPose();g.pose().scale(1/3f,1/3f,1);g.fill(0,68,width*3,69,0x1BC6D7C3);g.fill(0,height*3-60,width*3,height*3-59,0x19C6D7C3);g.pose().popPose();g.pose().popPose();minus.setActive(zoom>1);plus.setActive(zoom<6);super.render(g,x,y,tick);
    }
    @Override public boolean mouseScrolled(double x,double y,double dx,double dy){setZoom((float)(zoom*Math.pow(1.15,dy)));return true;}
    @Override public boolean mouseDragged(double x,double y,int button,double dx,double dy){float unit=(float)Minecraft.getInstance().getWindow().getGuiScale()/3; if(button==0&&y*unit>=68/3f&&y*unit<WebPixels.height()-20){panX+=dx*unit;panY+=dy*unit;return true;}return super.mouseDragged(x,y,button,dx,dy);}
    @Override public boolean mouseClicked(double x,double y,int button){
        float unit=(float)Minecraft.getInstance().getWindow().getGuiScale()/3;if(y*unit<68/3f||y*unit>=WebPixels.height()-20)return super.mouseClicked(x,y,button);
        if(button==0){long now=System.currentTimeMillis();if(now-clickAt<280){setZoom(zoom>1?1:2);panX=panY=0;}clickAt=now;return true;}return false;
    }
    @Override public boolean keyPressed(int key,int scan,int mods){if(key==GLFW.GLFW_KEY_ESCAPE){ChatSounds.play("close");onClose();return true;}return super.keyPressed(key,scan,mods);}
    @Override public void onClose(){if(previous instanceof ChatScreenFull){var full=new ChatScreenFull(false);Minecraft.getInstance().setScreen(full);ChatPanels.restore(full.view.root,previousPanel,full.view::switchScope);}else Minecraft.getInstance().setScreen(previous);}
    @Override public void removed(){if(!disposed){disposed=true;if(dismiss!=null&&!submitted)dismiss.run();}super.removed();}
}
