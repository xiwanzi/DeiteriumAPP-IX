package cafe.deuterium.chat;

import com.lowdragmc.lowdraglib2.gui.ui.*;
import com.lowdragmc.lowdraglib2.gui.ui.elements.*;
import com.lowdragmc.lowdraglib2.gui.texture.*;
import net.minecraft.client.Minecraft;
import net.minecraft.network.chat.Component;
import java.util.*;

final class ChatView implements AutoCloseable {
    final UIElement root=new UIElement();
    final ModularUI ui=ModularUI.of(UI.of(root,size->WebPixels.size())).shouldCloseOnEsc(false).shouldCloseOnKeyInventory(false);
    final TextField input=new TextField(){@Override public net.minecraft.client.gui.Font getFont(){return ChatTextLine.hasColor(getValue())?Minecraft.getInstance().font:com.lowdragmc.lowdraglib2.gui.LDLibFonts.font();}@Override protected void onBlur(com.lowdragmc.lowdraglib2.gui.ui.event.UIEvent event){/* Keep the selection while a composer tool owns focus. */}};
    final ChatTimeline timeline=new ChatTimeline(false);
    private final Label heading=ChatStyle.label("世界",25/3f,ChatStyle.TEXT);
    private final UIElement rail=new UIElement(),line=new UIElement();
    private final ScrollerView contacts=new ScrollerView();
    private final UIElement tools=new UIElement(),divider=new UIElement();
    private final BrandMark brand=new BrandMark();
    private Button close,textSize,back,pinCurrent;private Label esc;
    private final Label textSizeLabel=ChatStyle.label(ChatTextSize.label(),12/3f,ChatStyle.MUTED);
    private final Label privateTitle=ChatStyle.label("[溯行者]",ChatPresentation.DETAIL_PX/3f,0xFFA9B3B8);
    private final Button send=ChatStyle.button("发送",this::send);
    private final ShaderTexture latestArrow=new ShaderTexture(DeuteriumChat.id("latest_arrow")).setColor(ChatStyle.MUTED);
    final Button latest=ChatStyle.button("",()->{timeline.jumpToLatest();input.focus();});
    private final AutoCloseable subscription;
    private String scope=ChatStore.INSTANCE.current;
    private int width=-1,height=-1,textSizeLevel=-1;
    private long contactsRevision=-1;
    private final Map<String,Button> nav=new HashMap<>(),toolButtons=new HashMap<>(),contactButtons=new HashMap<>();
    private String lastPanel="initial";
    private final Map<String,UIElement> navIcons=new HashMap<>();
    private final Map<String,Label> navLabels=new HashMap<>();

    ChatView() {
        WebPixels.apply(root);
        root.addChild(ChatStyle.rect(brand,(WebPixels.wide()?40:32)/3f,WebPixels.topbar()/2-3.5f,135/3f,7));
        close=ChatStyle.button("",()->{ChatSounds.play("close");Minecraft.getInstance().setScreen(null);});ChatStyle.sound(close,"");close.noText();
        close.lss("position","absolute");close.getLayout().right(12).top(WebPixels.topbar()/2-6.5f).width(13).height(13);close.addChild(ChatStyle.rect(new UIElement().style(st->st.background(SpriteTexture.of(DeuteriumChat.id("textures/controls/window-close.png")))),0,0,13,13));root.addChild(close);
        esc=ChatStyle.label("Esc",11/3f,0xFFF2F2ED);esc.textStyle(st->st.font(DeuteriumChat.id("harmony")).textAlignHorizontal(com.lowdragmc.lowdraglib2.gui.ui.data.Horizontal.CENTER).textAlignVertical(com.lowdragmc.lowdraglib2.gui.ui.data.Vertical.CENTER));esc.style(st->st.background(new RoundTexture(1,0x60171A1B,1/3f,0x66DDDDDD)));esc.lss("position","absolute");esc.getLayout().right(86/3f).top(WebPixels.topbar()/2-23/6f).width(32/3f).height(23/3f);root.addChild(esc);
        contacts.viewPort(v->{v.getLayout().paddingAll(0);v.style(st->st.background(IGuiTexture.EMPTY));});
        root.addChildren(rail,contacts,heading,line,timeline,input,tools,send);
        divider.style(st->st.background(new ColorRectTexture(0x24C9D3D2)));root.addChild(divider);
        textSize=ChatStyle.iconButton("settings",()->{ChatTextSize.cycle();resize(WebPixels.width(),WebPixels.height());});textSize.setId("chat-text-size");textSize.lss("position","absolute");textSize.getLayout().left(11).bottom(32/3f).width(44/3f).height(44/3f);
        textSizeLabel.setId("chat-text-size-label");textSizeLabel.lss("position","absolute");textSizeLabel.textStyle(st->st.textAlignHorizontal(com.lowdragmc.lowdraglib2.gui.ui.data.Horizontal.CENTER));root.addChildren(textSize,textSizeLabel);
        back=ChatStyle.iconButton("back",()->switchScope("inbox"));back.setId("private-back");pinCurrent=ChatStyle.iconButton("pin",()->{if(!ChatStore.INSTANCE.pinned.remove(scope))ChatStore.INSTANCE.pinned.add(scope);ChatStore.INSTANCE.changed();ChatClient.INSTANCE.savePins();updateHeading();});root.addChildren(back,privateTitle,pinCurrent);
        line.style(s->s.background(new PixelRuleTexture(0x707D8C87)));
        String[] names={"世界","附近","团队","私聊"},ids={"world","nearby","team","inbox"};
        for(int i=0;i<ids.length;i++) {
            String id=ids[i];var b=ChatStyle.button("",()->switchScope(id));b.noText();
            var icon=new UIElement();icon.style(s2->s2.background(SpriteTexture.of(DeuteriumChat.id("textures/icons/"+id+".png"))));b.addChild(ChatStyle.rect(icon,25/3f,12/3f,8,8));
            var label=ChatStyle.label(names[i],4,ChatStyle.MUTED);label.textStyle(st->st.textAlignHorizontal(com.lowdragmc.lowdraglib2.gui.ui.data.Horizontal.CENTER));b.addChild(ChatStyle.rect(label,0,44/3f,74/3f,6));nav.put(id,b);navIcons.put(id,icon);navLabels.put(id,label);
            b.textStyle(s->s.fontSize(5.2f));ChatStyle.rect(b,0,i*79/3f,74/3f,73/3f);rail.addChild(b);
        }
        input.textFieldStyle(s->s.font(DeuteriumChat.id("harmony")).fontSize(5).textColor(ChatStyle.TEXT).textShadow(false).focusOverlay(IGuiTexture.EMPTY).placeholder(Component.literal("输入消息…")));
        input.style(s->s.background(new ColorRectTexture(0x0BC5D2CC)).overlay(new PixelBorderTexture(0x30A8B5B5)));
        input.getLayout().paddingLeft(5).paddingRight(5).paddingTop(4).paddingBottom(4);
        input.setText(ChatStore.INSTANCE.drafts.getOrDefault(scope,""));
        input.setTextResponder(s->{ChatStore.INSTANCE.drafts.put(scope,s);updateSend();});
        String[] toolNames={"emoji","image","share"};Runnable[] actions={()->ChatPanels.personal(root),()->ChatClient.INSTANCE.media.choose(false),()->ChatPanels.more(root)};
        for(int i=0;i<toolNames.length;i++){var tool=ChatStyle.toolButton(toolNames[i],actions[i]);toolButtons.put(toolNames[i],tool);tools.addChild(ChatStyle.rect(tool,i*16,0,12,12));}
        latest.noText();latest.setId("chat-jump-latest");latest.setDisplay(false);
        latest.buttonStyle(st->st.baseTexture(IGuiTexture.EMPTY).hoverTexture(new RoundTexture(6,0x16E7EBDE,1/3f,0x48EDF0DC)).pressedTexture(new RoundTexture(6,0x26E7EBDE)));
        latest.addChild(ChatStyle.rect(new UIElement().style(st->st.background(latestArrow)),2,2,8,8));
        tools.addChild(ChatStyle.rect(latest,48,0,12,12));
        send.setId("chat-send");input.setId("chat-draft");ChatStyle.sound(send,"");
        send.buttonStyle(s->s.baseTexture(new NativeButtonTexture(false)).hoverTexture(new NativeButtonTexture(true)).pressedTexture(new NativeButtonTexture(false,0xFFC7C7C7)));
        send.textStyle(s->s.textColor(0xFF202624).fontSize(5));
        subscription=ChatStore.INSTANCE.listen(this::refreshContacts);
        refreshContacts();highlight();updateSend();
        updateHeading();if(scope.equals("inbox"))switchScope("inbox");
    }
    void resize(int w,int h) {
        if(w==width&&h==height&&textSizeLevel==ChatTextSize.level())return;width=w;height=h;textSizeLevel=ChatTextSize.level();
        float start=WebPixels.contentX(),content=WebPixels.contentWidth(),top=WebPixels.topbar(),head=WebPixels.headingHeight(),bottom=WebPixels.bottom();boolean wide=WebPixels.wide();
        ChatStyle.rect(brand,(wide?40:32)/3f,top/2-3.5f,45,7);close.getLayout().top(top/2-6.5f);esc.getLayout().top(top/2-23/6f);textSize.getLayout().left((wide?42:33)/3f).bottom(bottom);
        textSizeLabel.setText(Component.literal(ChatTextSize.label()));textSizeLabel.getLayout().left((wide?42:33)/3f).bottom(Math.max(2,bottom-7)).width(44/3f).height(5);
        textSize.getStyle().tooltips(ChatStyle.body(Component.literal("文字大小")));
        ChatStyle.rect(divider,(wide?38:32)/3f,top+(wide?365:329)/3f,(wide?52:46)/3f,1/3f);
        float railX=(wide?24:18)/3f,railW=(wide?80:74)/3f,step=(wide?88:79)/3f,contactsY=top+(wide?378:342)/3f;
        ChatStyle.rect(rail,railX,top,railW,step*4-2);ChatStyle.rect(contacts,railX,contactsY,railW,h-bottom-56/3f-contactsY);
        heading.textStyle(st->st.fontSize((wide?28:25)/3f));
        ChatStyle.rect(heading,start+2/3f,top+(head-5-(wide?11.2f:10))/2,content-4/3f,wide?11.2f:10);ChatStyle.rect(line,start,top+head-1/3f,content,1/3f);
        float composerY=h-bottom-WebPixels.composer();
        ChatStyle.rect(timeline,start,top+head+14/3f,content-14/3f,composerY-16/3f-top-head-14/3f);
        ChatStyle.rect(tools,start,composerY+9.5f/3f,100,12);
        float draftH=(wide?54:49)/3f;input.textFieldStyle(st->st.fontSize(ChatTextSize.pixels(ChatPresentation.BODY_PX)/3f));
        ChatStyle.rect(input,start,h-bottom-draftH,content-179/3f,draftH);ChatStyle.rect(send,start+content-164/3f,h-bottom-draftH/2-22/3f,164/3f,44/3f);
        contactsRevision=-1;refreshContacts();updateHeading();
        for(int i=0;i<4;i++){String id=new String[]{"world","nearby","team","inbox"}[i];ChatStyle.rect(nav.get(id),0,i*step,railW,step-2);float iw=(id.equals("team")?38:28)/3f,ih=id.equals("team")?iw*70/123:28/3f;float iconY=(step-2-18)/2;ChatStyle.rect(navIcons.get(id),(railW-iw)/2,iconY+(28/3f-ih)/2,iw,ih);ChatStyle.rect(navLabels.get(id),0,iconY+12,railW,6);navLabels.get(id).textStyle(st->st.fontSize(14/3f));}

    }
    void switchScope(String id) {
        ChatStore.INSTANCE.current=id;scope=id;
        String title=switch(id) {case "world"->"世界";case "nearby"->"附近";case "team"->"团队";case "inbox"->"私聊";default->ChatStore.INSTANCE.contacts.getOrDefault(id,new ChatStore.Contact(id,"私聊","",false,0)).name();};
        heading.setText(Component.literal(title));input.setText(ChatStore.INSTANCE.drafts.getOrDefault(id,""));timeline.resetScroll();
        boolean inbox=id.equals("inbox");input.setDisplay(!inbox);tools.setDisplay(!inbox);send.setDisplay(!inbox);timeline.setDisplay(!inbox);
        if(inbox)ChatPanels.inbox(root,this::switchScope);else ChatPanels.close(root);
        ChatClient.INSTANCE.loadScope(id);refreshContacts();highlight();updateSend();
        updateHeading();
    }
    private void updateHeading(){
        boolean priv=scope.startsWith("private:");String text=switch(scope){case "world"->"世界";case "nearby"->"附近";case "team"->"团队";case "inbox"->"私聊";default->ChatStore.INSTANCE.contacts.getOrDefault(scope,new ChatStore.Contact(scope,"私聊","",false,0)).name();};
        heading.setText(Component.literal(text));heading.textStyle(st->st.font(DeuteriumChat.id("harmony")).fontSize((priv?23:WebPixels.wide()?28:25)/3f).textColor(priv?0xFFE5B3D9:ChatStyle.TEXT));
        float x=WebPixels.contentX(),w=WebPixels.contentWidth(),top=WebPixels.topbar(),h=WebPixels.headingHeight();float y=top+(h-5-(WebPixels.wide()?11.2f:10))/2;
        ChatStyle.rect(heading,x+(priv?46:2)/3f,y,w-(priv?90:4)/3f,WebPixels.wide()?11.2f:10);
        back.setDisplay(priv);pinCurrent.setDisplay(priv);privateTitle.setDisplay(priv);
        if(priv){privateTitle.setText(ChatTitles.title("",text));privateTitle.textStyle(st->st.adaptiveWidth(false).textWrap(com.lowdragmc.lowdraglib2.gui.ui.data.TextWrap.HIDE));privateTitle.getStyle().tooltips(ChatTitles.title("",text));ChatStyle.rect(back,x,top+1,32/3f,32/3f);ChatStyle.rect(pinCurrent,x+w-40/3f,top,40/3f,40/3f);float nameWidth=com.lowdragmc.lowdraglib2.gui.LDLibFonts.font().width(Component.literal(text).withStyle(st->st.withFont(DeuteriumChat.id("harmony"))))*23/27f;ChatStyle.rect(privateTitle,x+56/3f+Math.min(nameWidth,w-70),y+3,Math.max(0,w-40/3f-56/3f-Math.min(nameWidth,w-70)-8/3f),6);pinCurrent.getChildren().getLast().style(st->st.background(new MaskTexture("icons/pin.png",ChatStore.INSTANCE.pinned.contains(scope)?ChatStyle.YELLOW:0xFFA6AFAD)));}
    }
    private static IGuiTexture channelIcon(String id,boolean selected){String art=switch(id){case "team"->"chat-native/icon_sns_chat_11.png";case "inbox"->"chat-native/sns_icon_chat.png";default->"icons/"+id+".png";};return new MaskTexture(art,selected?ChatStyle.YELLOW:0xFFBDC5C2);}

    private void highlight(){String active=scope.startsWith("private:")?"":scope;for(var entry:nav.entrySet()){boolean selected=entry.getKey().equals(active);entry.getValue().buttonStyle(s->s.baseTexture(selected?new ChannelSelectionTexture():IGuiTexture.EMPTY));entry.getValue().style(s->s.overlay(IGuiTexture.EMPTY));navIcons.get(entry.getKey()).style(s->s.background(channelIcon(entry.getKey(),selected)));navLabels.get(entry.getKey()).textStyle(s->s.textColor(selected?ChatStyle.YELLOW:ChatStyle.MUTED));}for(var entry:contactButtons.entrySet())entry.getValue().buttonStyle(st->st.baseTexture(entry.getKey().equals(scope)?new ChannelSelectionTexture():IGuiTexture.EMPTY));}
    private void updateSend(){boolean enabled=!input.getValue().isBlank();send.setActive(enabled);send.textStyle(s->s.textColor(enabled?0xFF242721:0xFF929B98));send.buttonStyle(s->s.baseTexture(new NativeButtonTexture(false,enabled?0xFFF7F7F7:0xFF454545)));}
    void refreshPanelTools(){
        boolean showLatest=!scope.equals("inbox")&&timeline.awayFromLatest();if(latest.isDisplayed()!=showLatest)latest.setDisplay(showLatest);
        latestArrow.setColor(latest.getState()==Button.State.DEFAULT?0xFFABB5B1:ChatStyle.TEXT);
        String panel=ChatPanels.state(root);if(panel.equals(lastPanel))return;lastPanel=panel;
        for(var e:toolButtons.entrySet()){boolean active=e.getKey().equals("emoji")&&(panel.equals("personal")||panel.equals("manage")||panel.equals("stickers")||panel.equals("common")||panel.equals("adventure"))||e.getKey().equals("share")&&panel.equals("more");
            e.getValue().buttonStyle(st->st.baseTexture(active?new RoundTexture(6,0x12E9EA64,1/3f,0x70FFF100):IGuiTexture.EMPTY));
            var art=e.getValue().getChildren().getLast();String name=e.getKey();int color=active?ChatStyle.YELLOW:0xFFABB5B1;
            if(name.equals("share")){art.transform(t->t.rotation(active?45:0));art.style(st->st.background(new MaskTexture("chat-native/icon_plusmark.png",active?0xFFEEED68:0xFF33393B)));e.getValue().getChildren().getFirst().style(st->st.background(new RoundTexture(4,active?0xFF414746:0xFFE5E7E2)));}
            else art.style(st->st.background(name.equals("image")?new MaskTexture("icons/image.png",color,6/72f,9/72f,60/72f,54/72f):new MaskTexture("chat-native/icon_sns_chat_emoticon.png",color)));
        }
    }
    void send() {
        String value=input.getValue().strip();if(value.isEmpty())return;
        if(ChatClient.INSTANCE.send(value)){input.setText("");timeline.followLatest();ChatSounds.play("send");}
    }
    boolean enter(){
        if(!input.isDisplayed()||!input.isActive())return false;
        if(!input.isFocused())input.focus();else send();
        return true;
    }
    private void refreshContacts() {
        long revision=ChatTitles.revision+31L*ChatStore.INSTANCE.contacts.hashCode()+ChatStore.INSTANCE.pinned.hashCode();for(var c:ChatStore.INSTANCE.contacts.values()){var list=ChatStore.INSTANCE.inScope(c.id());if(!list.isEmpty())revision=31*revision+list.getLast().id.hashCode();}if(contactsRevision==revision)return;contactsRevision=revision;updateHeading();
        if(ChatPanels.state(root).equals("inbox"))ChatPanels.inbox(root,this::switchScope);
        contacts.clearAllScrollViewChildren();contactButtons.clear();int i=0;
        var ordered=ChatStore.INSTANCE.orderedContacts();
        for(var c:ordered) {
            if(!ChatStore.INSTANCE.pinned.contains(c.id()))continue;
            var b=ChatStyle.button("",()->switchScope(c.id()));b.noText();contactButtons.put(c.id(),b);
            b.addChild(ChatStyle.rect(new UIElement().style(st->st.background(new SkinFaceTexture(c.name()))),(WebPixels.wide()?25:22)/3f,(WebPixels.wide()?13:9)/3f,10,10));
            var name=ChatStyle.label(c.name(),11/3f,0xFFE5B3D9);name.textStyle(st->st.font(DeuteriumChat.id("harmony")).textAlignHorizontal(com.lowdragmc.lowdraglib2.gui.ui.data.Horizontal.CENTER));
            b.addChild(ChatStyle.rect(name,0,(WebPixels.wide()?49:45)/3f,(WebPixels.wide()?80:74)/3f,5));
            if(c.unread()>0)b.addChild(ChatStyle.rect(ChatStyle.unread(c.unread()),(WebPixels.wide()?75:69)/3f-ChatStyle.unreadWidth(c.unread()),5/3f,ChatStyle.unreadWidth(c.unread()),17/3f));
            b.getLayout().width((WebPixels.wide()?80:74)/3f).height((WebPixels.wide()?76:68)/3f).flexShrink(0);contacts.addScrollViewChild(b);i++;
        }
        highlight();
    }
    @Override public void close() { timeline.saveScroll();latestArrow.close();try {subscription.close();}catch(Exception ignored){} }
}
