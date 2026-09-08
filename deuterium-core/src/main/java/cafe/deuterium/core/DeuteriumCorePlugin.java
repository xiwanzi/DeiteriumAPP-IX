package cafe.deuterium.core;

import cafe.deuterium.core.commands.CoreCommand;
import cafe.deuterium.core.config.CoreConfig;
import org.bukkit.plugin.java.JavaPlugin;

public final class DeuteriumCorePlugin extends JavaPlugin {
    private CoreRuntime runtime;
    @Override public void onEnable() {
        saveDefaultConfig();
        try {
            runtime = new CoreRuntime(this, CoreConfig.load(getConfig()));
            CoreCommand command = new CoreCommand(this, runtime);
            java.util.Objects.requireNonNull(getCommand("dc")).setExecutor(command);
            getCommand("dc").setTabCompleter(command);
            getLogger().info("Core 已启用：/dc help；节点 " + runtime.config().nodeId() + "，物品库 " + runtime.config().storage().kind() + "。");
        } catch (Exception | LinkageError failure) {
            getLogger().severe("Core 启动失败（" + failure.getClass().getSimpleName() + "）：请检查配置、数据库与依赖版本。不会自动切换存储或启用发放。");
            getServer().getPluginManager().disablePlugin(this);
        }
    }
    @Override public void onDisable() { if (runtime != null) { runtime.close(); runtime = null; } }
    public CoreRuntime runtime() { return runtime; }
}
