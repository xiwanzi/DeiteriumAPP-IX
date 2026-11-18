package cafe.deuterium.chat;

import net.minecraft.client.gui.GuiGraphics;
import net.minecraft.client.gui.components.Button;
import net.minecraft.client.gui.screens.Screen;
import net.minecraft.network.chat.Component;

final class KeybindingScreen extends Screen {
    private final Screen parent;

    KeybindingScreen(Screen parent) {
        super(Component.literal("云键位"));
        this.parent = parent;
    }

    @Override
    protected void init() {
        var keys = CloudKeybindings.INSTANCE;
        keys.prepare();
        int x = width / 2 - 100, y = height / 2 - 24;
        addRenderableWidget(
                Button.builder(
                                assist(),
                                b -> {
                                    keys.toggle();
                                    b.setMessage(assist());
                                })
                        .bounds(x, y, 200, 20)
                        .build());
        var undo =
                addRenderableWidget(
                        Button.builder(
                                        Component.literal("撤销最近一次远程改键"),
                                        b -> {
                                            keys.undo();
                                            b.active = false;
                                        })
                                .bounds(x, y + 24, 200, 20)
                                .build());
        undo.active = keys.canUndo();
        addRenderableWidget(
                Button.builder(Component.translatable("gui.done"), b -> onClose())
                        .bounds(x, y + 52, 200, 20)
                        .build());
    }

    private Component assist() {
        return Component.literal(
                "允许管理员远程调键 · " + (CloudKeybindings.INSTANCE.allowed() ? "已开启" : "已关闭"));
    }

    @Override
    public void render(GuiGraphics graphics, int mouseX, int mouseY, float delta) {
        super.render(graphics, mouseX, mouseY, delta);
        graphics.drawCenteredString(font, title, width / 2, height / 2 - 64, 0xffffff);
        graphics.drawCenteredString(
                font, CloudKeybindings.INSTANCE.status, width / 2, height / 2 - 45, 0xc8c8c8);
    }

    @Override
    public void onClose() {
        minecraft.setScreen(parent);
    }
}
