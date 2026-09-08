package cafe.deuterium.core.mail;

import cafe.deuterium.core.api.PlayerDataService;
import cafe.deuterium.core.config.CoreConfig;
import cafe.deuterium.core.storage.CatalogStore;
import cafe.deuterium.core.storage.OutboxStore;
import cafe.deuterium.mail.api.MailPlayerDataBarrier;
import org.bukkit.Bukkit;
import org.bukkit.plugin.Plugin;
import org.bukkit.plugin.ServicePriority;
import java.util.function.Supplier;

public final class MailboxSupport {
    private MailboxSupport() { }
    public static CoreMailbox attach(Plugin owner, Supplier<CoreConfig> config, CatalogStore catalog,
                                     OutboxStore outbox, Supplier<PlayerDataService> dataService) {
        if (!Bukkit.getPluginManager().isPluginEnabled("DeuteriumMail")) return new UnavailableMailbox("独立邮箱未安装或未启用。");
        try {
            Bukkit.getServicesManager().register(MailPlayerDataBarrier.class, new MailBarrierAdapter(dataService), owner, ServicePriority.Normal);
            return new MailboxAdapter(config, catalog, outbox);
        } catch (LinkageError mismatch) { return new UnavailableMailbox("独立邮箱 API 版本不匹配，需要 0.6.0 的商城对接接口。"); }
    }
}
