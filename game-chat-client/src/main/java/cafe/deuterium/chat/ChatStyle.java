package cafe.deuterium.chat;

import com.lowdragmc.lowdraglib2.gui.texture.*;
import com.lowdragmc.lowdraglib2.gui.ui.UIElement;
import com.lowdragmc.lowdraglib2.gui.ui.elements.*;
import net.minecraft.network.chat.Component;

final class ChatStyle {
    private static final class ChatButton extends Button {
        private final Runnable action;private String role="click";
        ChatButton(Runnable action){this.action=action;}
        @Override protected void onMouseDown(com.lowdragmc.lowdraglib2.gui.ui.event.UIEvent e){if(e.button==0&&isActive()){ChatSounds.play(role);action.run();setButtonState(State.PRESSED);}}
    }
    static void sound(Button button,String role){if(button instanceof ChatButton b)b.role=role;}
    static final int TEXT=0xFFEEF0EA, MUTED=0xFFA4AFAE, YELLOW=0xFFFFF100;
    static <T extends UIElement> T rect(T e,float x,float y,float w,float h) {
        e.lss("position","absolute");e.getLayout().left(x).top(y).width(w).height(h);return e;
    }
    static Label label(String text,float size,int color) {
        var l=new Label();l.setText(Component.literal(text));
        l.textStyle(s->s.font(DeuteriumChat.id("harmony")).fontSize(size).textColor(color).textShadow(false));return l;
    }
    static Button button(String text,Runnable action) {
        var b=new ChatButton(action);b.setText(text,false).textStyle(s->s.font(DeuteriumChat.id("harmony")).fontSize(5).textColor(TEXT).textShadow(false));
        b.buttonStyle(s->s.baseTexture(IGuiTexture.EMPTY).hoverTexture(new ColorRectTexture(0x246A7370)).pressedTexture(new ColorRectTexture(0x4057615D)));
        return b;
    }
    static Button iconButton(String name,Runnable action) {
        var b=button("",action);b.noText();if(name.equals("close"))sound(b,"close");
        var icon=new UIElement().style(s->s.background(SpriteTexture.of(DeuteriumChat.id("textures/icons/"+name+".png"))));
        b.addChild(rect(icon,3,3,8,8));return b;
    }
    static Button toolButton(String name,Runnable action){
        var b=button("",action);b.noText();b.setId("tool-"+name);sound(b,"panel");
        b.buttonStyle(st->st.baseTexture(IGuiTexture.EMPTY).hoverTexture(new RoundTexture(6,0x16E7EBDE,1/3f,0x48EDF0DC)).pressedTexture(new RoundTexture(6,0x12E9EA64,1/3f,0x70FFF100)));
        String file=switch(name){case "emoji"->"chat-native/icon_sns_chat_emoticon.png";case "share"->"chat-native/icon_plusmark.png";default->"icons/"+name+".png";};
        float w=8,h=8;
        IGuiTexture texture=name.equals("image")?new MaskTexture(file,0xFFABB5B1,6/72f,9/72f,60/72f,54/72f):new MaskTexture(file,name.equals("share")?0xFF33393B:0xFFABB5B1);
        if(name.equals("image"))h=7.2f;
        if(name.equals("share")){b.addChild(rect(new UIElement().style(st->st.background(new RoundTexture(4,0xFFE5E7E2))),2,2,8,8));w=h=14/3f;}
        var icon=new UIElement().style(st->st.background(texture));icon.setId("tool-art-"+name);b.addChild(rect(icon,(12-w)/2,(12-h)/2,w,h));return b;
    }
    static float unreadWidth(int count){var c=Component.literal(Integer.toString(count)).withStyle(st->st.withFont(DeuteriumChat.id("harmony")));return Math.max(17/3f,com.lowdragmc.lowdraglib2.gui.LDLibFonts.font().width(c)*10/27f+8/3f);}
    static Label unread(int count){
        var l=label(Integer.toString(count),10/3f,0xFF242826);
        l.textStyle(st->st.font(DeuteriumChat.id("harmony")).textAlignHorizontal(com.lowdragmc.lowdraglib2.gui.ui.data.Horizontal.CENTER).textAlignVertical(com.lowdragmc.lowdraglib2.gui.ui.data.Vertical.CENTER));
        var outside=new RoundTexture(12/3f,0xFF171D20);var inside=new RoundTexture(10/3f,YELLOW);
        l.style(st->st.background((g,mx,my,x,y,w,h,tick)->{outside.draw(g,mx,my,x-2/3f,y-2/3f,w+4/3f,h+4/3f,tick);inside.draw(g,mx,my,x,y,w,h,tick);}));return l;
    }
    static Component body(Component c) {
        var out=Component.empty().withStyle(s->s.withFont(DeuteriumChat.id("harmony")));
        c.visit((style,text)->{
            var font=style.getFont();
            if(font.equals(net.minecraft.network.chat.Style.DEFAULT_FONT)||font.equals(DeuteriumChat.id("harmony_regular"))||font.equals(DeuteriumChat.id("gilroy")))
                style=style.withFont(DeuteriumChat.id("harmony"));
            out.append(Component.literal(text).setStyle(style));
            return java.util.Optional.empty();
        },net.minecraft.network.chat.Style.EMPTY);
        return out;
    }
    static Component systemBody(Component component){
        var result=Component.empty();
        component.visit((style,text)->{
            var color=style.getColor();
            if(color!=null){
                int rgb=color.getValue(),r=(rgb>>16)&255,g=(rgb>>8)&255,b=rgb&255;
                // Keep bright semantic colors; lift only low-contrast segments against the notice surface.
                double luminance=.2126*linear(r)+.7152*linear(g)+.0722*linear(b);
                if(luminance<.24){
                    for(int step=1;step<=20;step++){double mix=step/20.0;int nr=(int)Math.round(r+(255-r)*mix),ng=(int)Math.round(g+(255-g)*mix),nb=(int)Math.round(b+(255-b)*mix);
                        if(.2126*linear(nr)+.7152*linear(ng)+.0722*linear(nb)>=.35){style=style.withColor((nr<<16)|(ng<<8)|nb);break;}}
                }
            }
            result.append(Component.literal(text).setStyle(style));return java.util.Optional.empty();
        },net.minecraft.network.chat.Style.EMPTY);
        return body(result);
    }
    private static double linear(int channel){double c=channel/255.0;return c<=.04045?c/12.92:Math.pow((c+.055)/1.055,2.4);}
    static Component playerName(ChatMessage m) {
        return Component.literal(m.name).withStyle(s->s.withColor(ChatPresentation.ID_COLOR).withFont(DeuteriumChat.id("harmony")));
    }
    static Component title() {
        return Component.literal("[溯行者]").withStyle(s->s.withColor(0xA9B3B8).withFont(DeuteriumChat.id("harmony")));
    }
}
