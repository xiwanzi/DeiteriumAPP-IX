package cafe.deuterium.core.supply;

import cafe.deuterium.core.util.CoreFailure;
import java.lang.reflect.InvocationTargetException;
import java.util.List;
import java.util.UUID;
import org.bukkit.Bukkit;
import org.bukkit.inventory.ItemStack;

/** Uses Mail's installed public MMO processor so every delivered gun has its own committed identity. */
public final class SupplyItemPreparation {
    public record Prepared(boolean ready, ItemStack item, String message) { }

    private Class<?> contract() throws ReflectiveOperationException {
        var mail = Bukkit.getPluginManager().getPlugin("DeuteriumMail");
        if (mail == null || !mail.isEnabled()) throw new CoreFailure("SUPPLY_ITEMS_UNAVAILABLE", "邮箱物品准备接口尚未就绪");
        return Class.forName("cafe.deuterium.mail.api.MailItemProcessor", true, mail.getClass().getClassLoader());
    }

    public Prepared prepare(String delivery, UUID recipient, ItemStack template) throws Exception {
        if (!Bukkit.isPrimaryThread()) throw new IllegalStateException("Supply item preparation requires the game thread");
        Class<?> api = contract();
        Object processor = Bukkit.getServicesManager().load(api);
        if (processor == null) throw new CoreFailure("SUPPLY_ITEMS_UNAVAILABLE", "MMO 物品准备服务尚未就绪");
        try {
            Object result = api.getMethod("prepare", String.class, UUID.class, List.class)
                    .invoke(processor, "supply/" + delivery, recipient, List.of(template));
            Class<?> resultType = api.getMethod("prepare", String.class, UUID.class, List.class).getReturnType();
            boolean ready = (boolean) resultType.getMethod("ready").invoke(result);
            String message = (String) resultType.getMethod("message").invoke(result);
            List<?> items = (List<?>) resultType.getMethod("items").invoke(result);
            if (!ready) return new Prepared(false, null, message);
            if (items.size() != 1 || !(items.getFirst() instanceof ItemStack item) || item.getType().isAir() || item.getAmount() != 1)
                throw new CoreFailure("SUPPLY_ITEMS_INVALID", "奖励准备结果不符合已发布物品数量");
            return new Prepared(true, item.clone(), "");
        } catch (InvocationTargetException error) {
            throw new CoreFailure("SUPPLY_ITEMS_UNAVAILABLE", "奖励准备尚未确认，请等待服务恢复", error.getCause());
        }
    }

    public void delivered(String delivery, UUID recipient) throws Exception {
        Class<?> api = contract(); Object processor = Bukkit.getServicesManager().load(api);
        if (processor != null) api.getMethod("delivered", String.class, UUID.class)
                .invoke(processor, "supply/" + delivery, recipient);
    }
}
