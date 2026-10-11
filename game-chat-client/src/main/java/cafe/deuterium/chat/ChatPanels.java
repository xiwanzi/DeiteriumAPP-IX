package cafe.deuterium.chat;

import com.google.gson.*;
import com.lowdragmc.lowdraglib2.gui.ui.UIElement;
import com.lowdragmc.lowdraglib2.gui.ui.elements.*;
import com.lowdragmc.lowdraglib2.gui.texture.*;
import net.minecraft.client.Minecraft;
import net.minecraft.network.chat.Component;
import java.util.*;
import java.util.function.Consumer;

final class ChatPanels {
    private static final Map<UIElement,UIElement> OPEN=new WeakHashMap<>();
    private static final Map<UIElement,String> STATES=new WeakHashMap<>();
    static String state(UIElement root){return STATES.getOrDefault(root,"");}
    static void close(UIElement root){UIElement panel=OPEN.remove(root);STATES.remove(root);if(panel!=null)root.removeChild(panel);}
    static void restore(UIElement root,String state,Consumer<String> choose){switch(state){case "stickers"->stickers(root,false,false);case "personal"->stickers(root,true,false);case "manage"->stickers(root,true,true);case "inbox"->inbox(root,choose);case "more"->more(root);case "common"->stickerPage(root,2,false);case "adventure"->stickerPage(root,3,false);case "location"->positionPreview(root);}}
    private static UIElement panel(UIElement root,String title,int width,int height,String state){
        close(root);var p=new PopupSurface();p.lss("position","absolute");p.getLayout().left(WebPixels.contentX()).bottom(WebPixels.bottom()+WebPixels.composer()+8/3f).width(width).height(height).paddingAll(6);
        p.style(s->s.zIndex(50));
        p.addChild(ChatStyle.rect(ChatStyle.label(title,16/3f,ChatStyle.TEXT),7,7,width-30,28/3f));
        p.addChild(ChatStyle.rect(ChatStyle.iconButton("close",()->close(root)),width-16,7,28/3f,28/3f));
        p.addChild(ChatStyle.rect(new UIElement().style(st->st.background(new ColorRectTexture(ChatStyle.YELLOW))),20/3f,-1/3f,10,2/3f));
        root.addChild(p);OPEN.put(root,p);STATES.put(root,state);return p;
    }
    static void stickers(UIElement root){stickers(root,false,false);}
    static void personal(UIElement root){stickers(root,true,false);}
    private static final String[][] EMOJI={
        {"😀","😂","🥹","😊","😍","😎","🥳","🤔","😭","😅","😴","😤","🤯","🫡","👍","👋","🙌","👏","🤝","💪","❤️","✨","🔥","🎉","🌿","🌸","⭐","💎"},
        {"⛏️","🪓","⚔️","🛡️","🏹","🧭","🗺️","💎","🪵","🪨","🔥","🌲","🌻","🍎","🥕","🍞","🐺","🐱","🐷","🐑","🐝","🏠","🏔️","🌅","🌙","☀️","🌧️","❄️"}
    };
    private static void stickers(UIElement root,boolean personal,boolean managing){stickerPage(root,personal?0:1,managing);}
    private static void stickerPage(UIElement root,int category,boolean managing){
        var library=ChatClient.INSTANCE.media.stickers();int cells=library.size()+(library.size()<10&&!managing?1:0),rows=(cells+3)/4;
        int pixels=category==0?Math.max(233,160+rows*73+Math.max(0,rows-1)*7):category==1?440:292;
        String state=category==0?(managing?"manage":"personal"):category==1?"stickers":category==2?"common":"adventure";
        var p=panel(root,"表情",118,(int)Math.ceil(pixels/3f),state);
        String[] titles={"个人","贴纸","常用","冒险"};
        p.addChild(ChatStyle.rect(new UIElement().style(st->st.background(new ColorRectTexture(0x24C9D3D2))),7,94/3f,104,1/3f));
        for(int i=0;i<4;i++){int selected=i;var tab=ChatStyle.button(titles[i],()->stickerPage(root,selected,false));ChatStyle.sound(tab,"card");tab.textStyle(st->st.fontSize(4).textColor(selected==category?ChatStyle.YELLOW:0xFF9FAA9F));
            if(i==0)tab.setId("sticker-personal-tab");else if(i==1)tab.setId("sticker-standard-tab");else tab.setId("emoji-category-"+i);
            p.addChild(ChatStyle.rect(tab,7+i*49/3f,66/3f,8,28/3f));
            if(selected==category)p.addChild(ChatStyle.rect(new UIElement().style(st->st.background(new ColorRectTexture(ChatStyle.YELLOW))),7+i*49/3f,94/3f,8,2/3f));
        }
        if(category>=2){for(int i=0;i<EMOJI[category-2].length;i++){int index=i;String value=EMOJI[category-2][i];var cell=ChatStyle.button("",()->{if(Minecraft.getInstance().screen instanceof ChatScreenFull full){full.view.input.insertText(value);full.view.input.focus();}});cell.noText();cell.setId("emoji-cell-"+i);ChatStyle.sound(cell,"card");
                cell.addChild(ChatStyle.rect(new UIElement().style(st->st.background(SpriteTexture.of(DeuteriumChat.id("textures/emoji/"+(category-2)+"_"+index+".png")))),1,1,32/3f,32/3f));p.addChild(ChatStyle.rect(cell,7+(i%7)*15,107/3f+(i/7)*14,41/3f,38/3f));}return;}
        float gridY=category==0?140/3f:107/3f;
        if(category==0){var count=ChatStyle.label(Minecraft.getInstance().getUser().getName()+" · "+library.size()+"/10",11/3f,0xFFA3B3A9);count.textStyle(st->st.font(DeuteriumChat.id("gilroy")));p.addChild(ChatStyle.rect(count,7,109/3f,75,8));
            var manage=ChatStyle.button(managing?"完成":"管理",()->stickerPage(root,0,!managing));manage.textStyle(st->st.fontSize(11/3f));ChatStyle.sound(manage,"");p.addChild(ChatStyle.rect(manage,92,104/3f,19,9));}
        int count=category==0?library.size():14;
        for(int i=0;i<count;i++){
            int index=i;ChatMessage message=category==1?builtin(i+1):new ChatMessage("own_"+ChatMessage.str(library.get(i).getAsJsonObject(),"mediaId"),"","","world","","",Component.empty(),library.get(i).getAsJsonObject(),java.time.Instant.now(),true,false,"");
            var tile=new StickerTile(()->{if(category==0&&managing)ChatClient.INSTANCE.media.removeSticker(ChatMessage.str(message.media,"mediaId"));else{if(category==1)ChatClient.INSTANCE.media.builtIn(index+1);else ChatClient.INSTANCE.media.sendSticker(message.media);close(root);ChatSounds.play("send");}},()->Minecraft.getInstance().setScreen(new MediaViewer(message,Minecraft.getInstance().screen)));
            tile.setId(category==0?"personal-sticker-"+i:"builtin-sticker-"+(i+1));
            var art=new UIElement(){@Override public void drawBackgroundAdditional(com.lowdragmc.lowdraglib2.gui.ui.rendering.GUIContext ctx){ChatClient.INSTANCE.media.draw(ctx.graphics,message,getPositionX(),getPositionY(),63/3f,63/3f,255,false);}};art.setActive(false);tile.addChild(ChatStyle.rect(art,5/3f,5/3f,21,21));
            if(managing){var remove=ChatStyle.label("×",6,0xFFF0C1C1);tile.addChild(ChatStyle.rect(remove,18,0,6,6));}
            p.addChild(ChatStyle.rect(tile,7+(i%4)*80/3f,gridY+(i/4)*80/3f,73/3f,73/3f));
        }
        if(category==0&&library.size()<10&&!managing){int i=library.size();var add=ChatStyle.button("+",()->ChatClient.INSTANCE.media.choose(true));add.textStyle(st->st.fontSize(10).textColor(0xFFADBDAD));ChatStyle.sound(add,"card");add.style(st->st.overlay(new PixelBorderTexture(0x42AABDAA)));p.addChild(ChatStyle.rect(add,7+(i%4)*80/3f,gridY+(i/4)*80/3f,73/3f,73/3f));}
    }
    private static ChatMessage builtin(int index){var media=new JsonObject();media.addProperty("builtin",index);media.addProperty("mediaId","builtin_"+index);media.addProperty("width",256);media.addProperty("height",256);media.addProperty("type","sticker");return new ChatMessage("builtin_"+index,"","","world","","",Component.empty(),media,java.time.Instant.now(),true,false,"");}
    static void more(UIElement root){
        var p=panel(root,"分享",118,72,"more");
        p.addChild(ChatStyle.rect(new UIElement().style(st->st.background(new ColorRectTexture(0x24D0D0D2))),7,61/3f,104,1/3f));
        for(int i=0;i<2;i++){boolean location=i==0;boolean available=location||ChatStore.INSTANCE.current.equals("world");
            var tile=ChatStyle.button("",()->{if(location)positionPreview(root);else if(ChatClient.INSTANCE.shareItem()){ChatSounds.play("send");close(root);}});
            tile.setId(location?"share-location":"share-item");tile.noText();tile.setActive(available);ChatStyle.sound(tile,"card");tile.buttonStyle(st->st.hoverTexture(IGuiTexture.EMPTY));
            var art=new UIElement().style(st->st.background(new RoundTexture(8/3f,0x18A0A0A2,1/3f,0x3BC9C9CD)));tile.addChild(ChatStyle.rect(art,42.25f/3,7/3f,65/3f,65/3f));
            tile.addEventListener(com.lowdragmc.lowdraglib2.gui.ui.event.UIEvents.MOUSE_ENTER,e->art.style(st->st.background(new RoundTexture(8/3f,0x23C7C7CB,1/3f,0x78DEDEE0))));
            tile.addEventListener(com.lowdragmc.lowdraglib2.gui.ui.event.UIEvents.MOUSE_LEAVE,e->art.style(st->st.background(new RoundTexture(8/3f,0x18A0A0A2,1/3f,0x3BC9C9CD))));
            float size=32/3f;IGuiTexture icon;
            if(location)icon=new MaskTexture("icons/location.png",0xFFD0D0D2);
            else{var player=Minecraft.getInstance().player;var held=player==null?net.minecraft.world.item.ItemStack.EMPTY:player.getMainHandItem();icon=new ItemStackTexture(held.isEmpty()?net.minecraft.world.item.Items.CHEST.getDefaultInstance():held.copy());}
            var texture=icon;art.addChild(ChatStyle.rect(new UIElement().style(st->st.background(texture)),(65/3f-size)/2,(65/3f-size)/2,size,size));
            var caption=ChatStyle.label(location?"当前位置":"当前物品",14/3f,available?0xFFD0D0D2:0xFF929294);caption.textStyle(st->st.textAlignHorizontal(com.lowdragmc.lowdraglib2.gui.ui.data.Horizontal.CENTER));tile.addChild(ChatStyle.rect(caption,0,82/3f,149.5f/3,6));p.addChild(ChatStyle.rect(tile,7+i*162.5f/3,79/3f,149.5f/3,36));
        }
        if(!ChatStore.INSTANCE.current.equals("world"))p.addChild(ChatStyle.rect(ChatStyle.label("当前物品仅支持世界频道",11/3f,0xFFAAAAAE),7,64,104,6));
    }
    private static void positionPreview(UIElement root){
        var p=panel(root,"当前位置",127,116,"location");var player=Minecraft.getInstance().player;
        String world=Minecraft.getInstance().level==null?"":switch(Minecraft.getInstance().level.dimension().location().getPath()){case "overworld"->"主世界";case "the_nether"->"下界";case "the_end"->"末地";default->Minecraft.getInstance().level.dimension().location().getPath();};
        var map=new UIElement().style(st->st.background(new RoundTexture(0,0xFF303032,1/3f,0x36BABABC)));p.addChild(ChatStyle.rect(map,7,24,113,94/3f));map.addChild(ChatStyle.rect(new UIElement().style(st->st.background(new MaskTexture("icons/location.png",0xFFDDDDDF))),32,10,32/3f,32/3f));map.addChild(ChatStyle.rect(ChatStyle.label(world,14/3f,0xFFDDDDDF),48,12,55,7));
        String coords=player==null?"正在读取位置…":"X "+player.getBlockX()+"   Y "+player.getBlockY()+"   Z "+player.getBlockZ();p.addChild(ChatStyle.rect(ChatStyle.label(coords,4,0xFFD0D0D2),7,62,113,9));
        p.addChild(ChatStyle.rect(ChatStyle.label("发送一次，以服务器确认的位置为准。",11/3f,0xFFAAAAAE),7,77,113,8));
        var back=ChatStyle.button("返回",()->more(root));back.textStyle(st->st.fontSize(4));p.addChild(ChatStyle.rect(back,7,96,22,13));var send=ChatStyle.button("发送",()->{ChatClient.INSTANCE.shareLocation();ChatSounds.play("send");close(root);});ChatStyle.sound(send,"");send.textStyle(st->st.textColor(0xFF242426));send.buttonStyle(st->st.baseTexture(new NativeButtonTexture(false)).hoverTexture(new NativeButtonTexture(true)));p.addChild(ChatStyle.rect(send,65,96,164/3f,44/3f));
    }
    static void inbox(UIElement root,Consumer<String> choose){inbox(root,choose,null);}
    private static void inbox(UIElement root,Consumer<String> choose,Set<String> selected){
        close(root);var mc=Minecraft.getInstance();float w=WebPixels.contentWidth(),h=WebPixels.height()-WebPixels.bottom()-WebPixels.topbar()-WebPixels.headingHeight();
        var p=ChatStyle.rect(new UIElement(),WebPixels.contentX(),WebPixels.topbar()+WebPixels.headingHeight(),w,h);p.style(st->st.zIndex(50));root.addChild(p);OPEN.put(root,p);STATES.put(root,selected==null?"inbox":"inbox-select");
        var scroller=new ScrollerView();scroller.viewPort(v->{v.getLayout().paddingAll(0);v.style(st->st.background(IGuiTexture.EMPTY));});p.addChild(ChatStyle.rect(scroller,0,20/3f,w-7/3f,h-20/3f-(selected==null?0:65/3f)));
        for(var c:ChatStore.INSTANCE.orderedContacts()){
            float rw=w-7/3f;var row=new UIElement();row.getLayout().height(31).widthPercent(100).flexShrink(0);
            var card=new UIElement().style(st->st.background(new RoundTexture(2/3f,0x09B4C0C0,1/3f,0x22C5CFCD)));row.addChild(ChatStyle.rect(card,0,0,rw,28));
            card.addChild(ChatStyle.rect(new UIElement().style(st->st.background(SpriteTexture.of(DeuteriumChat.id("textures/controls/bg_mail_strip.png")).setSprite(215,0,477,112).setColor(0x29FFFFFF))),0,0,rw,28));
            Runnable chooseRow=()->{if(selected==null)choose.accept(c.id());else{if(!selected.remove(c.id()))selected.add(c.id());inbox(root,choose,selected);}};
            var open=ChatStyle.button("",chooseRow);open.setId("private-open-"+c.name());open.noText();open.buttonStyle(st->st.hoverTexture(new RoundTexture(2/3f,0x19D9E1D5,1/3f,0x59BDC9BE)));
            if(selected!=null&&selected.contains(c.id()))open.buttonStyle(st->st.baseTexture(new RoundTexture(2/3f,0x10E5EA72,1/3f,0x60E4EB80)));row.addChild(ChatStyle.rect(open,0,0,rw-18,28));
            open.addChild(ChatStyle.rect(new UIElement().style(st->st.background(new SkinFaceTexture(c.name()))),6,20/3f,44/3f,44/3f));
            var name=ChatStyle.label(c.name(),5,0xFFE5B3D9);name.textStyle(st->st.font(DeuteriumChat.id("gilroy")));open.addChild(ChatStyle.rect(name,26,16/3f,rw-65,7));
            float nameW=com.lowdragmc.lowdraglib2.gui.LDLibFonts.font().width(Component.literal(c.name()).withStyle(st->st.withFont(DeuteriumChat.id("gilroy"))))*5/9f;
            var title=ChatStyle.label("",11/3f,0xFFA9B3B8);title.setText(ChatTitles.title(c.ref(),c.name()));title.textStyle(st->st.adaptiveWidth(false).textWrap(com.lowdragmc.lowdraglib2.gui.ui.data.TextWrap.HIDE));title.getStyle().tooltips(ChatTitles.title(c.ref(),c.name()));float titleX=26+Math.min(nameW,rw-95)+7/3f;open.addChild(ChatStyle.rect(title,titleX,18/3f,Math.max(0,rw-42-titleX),6));
            var history=ChatStore.INSTANCE.inScope(c.id());String preview=history.isEmpty()?"":history.getLast().media!=null?("sticker".equals(ChatMessage.str(history.getLast().media,"type"))?"[贴纸]":"[图片]"):history.getLast().content.getString();
            if(!history.isEmpty()){var time=ChatStyle.label(history.getLast().time,10/3f,0xFF98A6A2);time.textStyle(st->st.font(DeuteriumChat.id("numbers")));open.addChild(ChatStyle.rect(time,rw-37,6,20,5));}
            open.addChild(ChatStyle.rect(ChatStyle.label(preview,13/3f,0xFF9CA8A3),26,15,rw-58,7));
            if(selected==null){var pin=ChatStyle.button("",()->{if(!ChatStore.INSTANCE.pinned.remove(c.id()))ChatStore.INSTANCE.pinned.add(c.id());ChatStore.INSTANCE.changed();ChatClient.INSTANCE.savePins();inbox(root,choose);});pin.noText();pin.addChild(ChatStyle.rect(new UIElement().style(st->st.background(new MaskTexture("icons/pin.png",ChatStore.INSTANCE.pinned.contains(c.id())?ChatStyle.YELLOW:0xFF6D7C74))),4,4.5f,17/3f,17/3f));row.addChild(ChatStyle.rect(pin,rw-50/3f,20/3f,14,44/3f));}
            else{var checkbox=ChatStyle.button("",chooseRow);checkbox.noText();boolean checked=selected.contains(c.id());checkbox.buttonStyle(st->st.baseTexture(new RoundTexture(0,checked?ChatStyle.YELLOW:0,1/3f,checked?ChatStyle.YELLOW:0x75A6B39E)));if(checked)checkbox.addChild(ChatStyle.rect(new UIElement().style(st->st.background(new MaskTexture("controls/icon_common_select.png",0xFF242826))),1,1,16/3f,16/3f));row.addChild(ChatStyle.rect(checkbox,rw-46/3f,31/3f,22/3f,22/3f));}
            if(c.unread()>0&&selected==null)row.addChild(ChatStyle.rect(ChatStyle.unread(c.unread()),rw-22-ChatStyle.unreadWidth(c.unread()),10,ChatStyle.unreadWidth(c.unread()),17/3f));scroller.addScrollViewChild(row);
        }
        Runnable apply=()->{if(selected==null)inbox(root,choose,new LinkedHashSet<>(ChatStore.INSTANCE.pinned));else{ChatStore.INSTANCE.pinned.clear();ChatStore.INSTANCE.pinned.addAll(selected);ChatStore.INSTANCE.changed();ChatClient.INSTANCE.savePins();inbox(root,choose);}};
        var batch=ChatStyle.button(selected==null?"批量固定":"应用",apply);batch.setId("private-batch");
        if(selected==null){batch.textStyle(st->st.fontSize(4));p.addChild(ChatStyle.rect(batch,w-69,-19,35,38/3f));p.addChild(ChatStyle.rect(ChatStyle.button("新私聊",()->directory(root,choose)),w-32,-19,32,38/3f));}
        else{p.addChild(ChatStyle.rect(new UIElement().style(st->st.background(new ColorRectTexture(0x24C9D3D2))),0,h-65/3f,w,1/3f));batch.buttonStyle(st->st.baseTexture(new NativeButtonTexture(false)).hoverTexture(new NativeButtonTexture(true)));batch.textStyle(st->st.textColor(0xFF242721));p.addChild(ChatStyle.rect(batch,w-164/3f,h-44/3f,164/3f,44/3f));var cancel=ChatStyle.button("取消",()->inbox(root,choose));ChatStyle.sound(cancel,"");p.addChild(ChatStyle.rect(cancel,w-220/3f,h-44/3f,45/3f,44/3f));}
    }
    private static void directory(UIElement root,Consumer<String> choose){
        ChatClient.INSTANCE.submit(a->a.request("GET","/api/v1/chat/player-directory?limit=100",null),raw->{var p=panel(root,"选择玩家",205,174,"inbox");var list=ChatClient.array(raw);var scroll=new ScrollerView();p.addChild(ChatStyle.rect(scroll,6,24,192,142));
            for(var e:list){var player=e.getAsJsonObject();String ref=ChatMessage.str(player,"playerRef"),name=ChatMessage.str(player,"gameId");
                if(ref.equals(ChatStore.INSTANCE.selfRef))continue;
                var button=ChatStyle.button(name,()->{var b=new JsonObject();b.addProperty("clientRequestId",UUID.randomUUID().toString());b.addProperty("otherPlayerRef",ref);ChatClient.INSTANCE.submit(a->a.request("POST","/api/v1/chat/conversations",b),r->{var c=r.getAsJsonObject();String id="private:"+ChatMessage.str(c,"conversationId");ChatStore.INSTANCE.contacts.put(id,new ChatStore.Contact(id,name,ref,false,0));choose.accept(id);});});button.getLayout().height(19).widthPercent(100);scroll.addScrollViewChild(button);}
        });
    }
}
