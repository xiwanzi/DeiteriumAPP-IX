package cafe.deuterium.chat;

import net.minecraft.network.RegistryFriendlyByteBuf;
import net.minecraft.network.codec.StreamCodec;
import net.minecraft.network.protocol.common.custom.CustomPacketPayload;
import net.minecraft.resources.ResourceLocation;
import java.nio.charset.StandardCharsets;

/** Raw bounded UTF-8 JSON interoperates with Bukkit plugin messaging. */
public record ChatPayload(String json) implements CustomPacketPayload {
    public static final Type<ChatPayload> TYPE=new Type<>(ResourceLocation.parse("deuterium:chat"));
    public static final StreamCodec<RegistryFriendlyByteBuf,ChatPayload> CODEC=StreamCodec.of((b,p)-> {
        byte[] bytes=p.json.getBytes(StandardCharsets.UTF_8);
        if(bytes.length>28000)throw new IllegalArgumentException("Chat payload too large");
        b.writeBytes(bytes);
    }, b -> {
        if(b.readableBytes()>28000)throw new IllegalArgumentException("Chat payload too large");
        return new ChatPayload(b.readCharSequence(b.readableBytes(),StandardCharsets.UTF_8).toString());
    });
    @Override public Type<? extends CustomPacketPayload> type() { return TYPE; }
}
