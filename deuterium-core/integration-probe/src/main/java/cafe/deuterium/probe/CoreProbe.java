package cafe.deuterium.probe;

import cafe.deuterium.core.DeuteriumCorePlugin;
import cafe.deuterium.core.api.PlayerDataService;
import cafe.deuterium.core.items.InventoryGrant;
import cafe.deuterium.core.compat.LocalSaveProof;
import cafe.deuterium.core.util.Json;
import net.kyori.adventure.text.Component;
import net.milkbowl.vault.economy.Economy;
import org.bukkit.*;
import org.bukkit.command.*;
import org.bukkit.entity.Player;
import org.bukkit.inventory.*;
import org.bukkit.inventory.meta.*;
import org.bukkit.persistence.PersistentDataType;
import org.bukkit.plugin.java.JavaPlugin;
import java.math.BigDecimal;
import java.nio.file.*;
import java.util.*;

/** Test-only plugin with a hard loopback/workspace gate. */
public final class CoreProbe extends JavaPlugin {
    @Override public void onEnable(){
        String root=Path.of(".").toAbsolutePath().normalize().toString().replace('\\','/').toLowerCase(Locale.ROOT);
        if(!Bukkit.getIp().equals("127.0.0.1")||!root.contains("/.tools/core-")){getLogger().severe("Probe refused non-isolated server");Bukkit.getPluginManager().disablePlugin(this);return;}
        Objects.requireNonNull(getCommand("dcprobe")).setExecutor((sender,command,label,args)->{
            if(sender instanceof Player||args.length!=1)return true;
            try {switch(args[0]){case "items"->items();case "trchat"->new ChatProbe(this).run();case "funds"->funds();case "pay"->pay();case "cache"->cache();case "channels"->new SyncProbe(this).channels();case "backpack"->new SyncProbe(this).backpack();case "sync-failure"->new SyncProbe(this).failure();case "sync-unknown"->new SyncProbe(this).unknown();case "sync-history"->new SyncProbe(this).history();case "mail"->new MailProbe(this).claim(false);case "mail-recovery"->new MailProbe(this).claim(true);case "status"->report("status",Map.of("localSaveSupported",LocalSaveProof.supported(),"plugins",Arrays.stream(Bukkit.getPluginManager().getPlugins()).map(p->p.getName()+":"+p.getDescription().getVersion()).toList()));default->throw new IllegalArgumentException("unknown probe");}
            }catch(Throwable error){getLogger().log(java.util.logging.Level.SEVERE,"PROBE_FAILED "+args[0],error);report(args[0],Map.of("passed",false,"failure",error.toString()));}
            sender.sendMessage("Probe finished; see isolated plugins/DeuteriumCoreProbe results and console.");return true;
        });
    }
    DeuteriumCorePlugin core(){return (DeuteriumCorePlugin)Bukkit.getPluginManager().getPlugin("DeuteriumCore");}
    Player player(String name){Player p=Bukkit.getPlayerExact(name);if(p==null)throw new IllegalStateException("test player offline: "+name);return p;}
    void items()throws Exception{
        var runtime=core().runtime();Player p=player("CoreProbeUser");check(!p.isDead()&&p.isValid(),"probe player must be alive");p.getInventory().clear();
        ItemStack sword=new ItemStack(Material.DIAMOND_SWORD);ItemMeta meta=sword.getItemMeta();meta.displayName(Component.text("星海测试 · 完整物品"));meta.lore(List.of(Component.text("NBT/PDC 保留")));meta.addEnchant(org.bukkit.enchantments.Enchantment.UNBREAKING,3,true);((Damageable)meta).setDamage(173);meta.getPersistentDataContainer().set(new NamespacedKey(this,"payload"),PersistentDataType.BYTE_ARRAY,new byte[]{0,1,2,-1,127});sword.setItemMeta(meta);
        var captured=runtime.codec.capture("deuterium:probe_sword",sword,"",runtime.config());var v1=runtime.catalog.save(captured,"integration-probe");ItemStack restored=runtime.codec.decode(v1,runtime.config());check(restored.isSimilar(sword),"item metadata roundtrip");
        ItemStack box=new ItemStack(Material.SHULKER_BOX);BlockStateMeta container=(BlockStateMeta)box.getItemMeta();var state=(org.bukkit.block.ShulkerBox)container.getBlockState();state.getInventory().setItem(4,sword.clone());container.setBlockState(state);box.setItemMeta(container);
        var nested=runtime.catalog.save(runtime.codec.capture("deuterium:probe_box",box,"Nested",runtime.config()),"integration-probe");check(runtime.codec.decode(nested,runtime.config()).isSimilar(box),"nested container roundtrip");
        p.getInventory().setItemInMainHand(restored);String receipt;
        try(PlayerDataService.Lease lease=runtime.inventoryAccess().acquire(p,runtime.config().inventoryDomain(),UUID.randomUUID())){lease.verifyCurrent();lease.verifyItems(List.of(restored));receipt=lease.saveAndConfirm();}
        check(receipt!=null&&!receipt.isEmpty(),"missing save receipt");
        ItemStack[] before=p.getInventory().getStorageContents();boolean rejected=false;try{InventoryGrant.apply(p.getInventory(),sword,37);}catch(RuntimeException expected){rejected=true;}check(rejected&&Arrays.equals(before,p.getInventory().getStorageContents()),"inventory overflow altered contents");
        p.setOp(true);p.performCommand("dc save probe_command CommandSave");
        Bukkit.getScheduler().runTaskLater(this,()->{try{runtime.catalog.find("deuterium:probe_command",0);p.performCommand("dc get probe_command 1");}catch(Throwable error){getLogger().log(java.util.logging.Level.SEVERE,"PROBE_COMMAND_FAILED",error);}},20);
        report("items",Map.of("passed",true,"server",Bukkit.getVersion(),"swordSha256",v1.payloadSha256(),"containerSha256",nested.payloadSha256(),"localSaveReceipt",receipt,"nestedNbtPreserved",true,"overflowPreserved",true));
    }
    @SuppressWarnings("unchecked") void funds()throws Exception{
        Player payer=player("CoreProbeUser"),payee=player("CoreProbePeer");Economy vault=Bukkit.getServicesManager().load(Economy.class);check(vault!=null,"Vault missing");
        // These are fresh test players in a dedicated test economy database.
        check(vault.hasAccount(payer)&&vault.hasAccount(payee),"test economy accounts missing");check(vault.depositPlayer(payer,100).transactionSuccess(),"Vault deposit failed");
        double before=vault.getBalance(payer);check(vault.withdrawPlayer(payer,1.25).transactionSuccess(),"Vault withdrawal failed");check(decimal(vault.getBalance(payer)).equals(decimal(before-1.25)),"Vault returned before commit");
        var runtime=core().runtime();runtime.economy().initializeSystemAccounts();String op="live_"+UUID.randomUUID();var payload=Json.tree(Map.of("fromUuid",payer.getUniqueId().toString(),"toUuid",payee.getUniqueId().toString(),"amount","2.50"));
        var first=runtime.economy().execute(op,"wallet.transfer",payload);runtime.economy().execute(op,"wallet.transfer",payload);check(decimal(vault.getBalance(payer)).equals(decimal(before-3.75)),"Core replay debited twice or Vault read stale");check("COMPLETED".equals(first.get("status").getAsString()),"missing commit receipt");
        Class<?> api=Class.forName("me.yic.xconomy.api.XConomyAPI",true,Bukkit.getPluginManager().getPlugin("XConomy").getClass().getClassLoader());Object nativeAPI=api.getConstructor().newInstance();
        Object result=api.getMethod("changePlayerBalance",UUID.class,String.class,BigDecimal.class,Boolean.class,String.class).invoke(nativeAPI,payer.getUniqueId(),payer.getName(),new BigDecimal("0.75"),true,"DeuteriumCoreProbe");check(result.equals(0),"native API rejected deposit");
        check(decimal(vault.getBalance(payer)).equals(decimal(before-3.00)),"native API was not committed");
        report("funds",Map.of("passed",true,"vaultProvider",vault.getName(),"operationId",op,"payerUuid",payer.getUniqueId().toString(),"payeeUuid",payee.getUniqueId().toString(),"payerBalance",decimal(vault.getBalance(payer)),"payeeBalance",decimal(vault.getBalance(payee))));
    }
    void pay(){Player payer=player("CoreProbeUser"),payee=player("CoreProbePeer");Economy vault=Bukkit.getServicesManager().load(Economy.class);double from=vault.getBalance(payer),to=vault.getBalance(payee);payer.performCommand("pay CoreProbePeer 1.10");Bukkit.getScheduler().runTaskLater(this,()->{try{check(decimal(vault.getBalance(payer)).equals(decimal(from-1.10)),"native pay debit");check(decimal(vault.getBalance(payee)).equals(decimal(to+1.10)),"native pay credit");report("pay",Map.of("passed",true,"payerBalance",decimal(vault.getBalance(payer)),"payeeBalance",decimal(vault.getBalance(payee))));}catch(Throwable failure){getLogger().log(java.util.logging.Level.SEVERE,"PROBE_PAY_FAILED",failure);report("pay",Map.of("passed",false,"failure",failure.toString()));}},20);}
    void cache(){
        var runtime=core().runtime();check(runtime.config().storage().database().startsWith("dc_core_game_"),"cache probe requires dedicated test schema");
        var historical=Bukkit.getOfflinePlayerIfCached("CoreProbePeer");check(historical!=null&&!historical.isOnline(),"historical test player must be cached and offline");
        runtime.workers.submit(()->{
            runtime.database.read(c->{try(var s=c.prepareStatement("DELETE FROM dc_players WHERE player_uuid=?")){s.setString(1,historical.getUniqueId().toString());s.executeUpdate();}return null;});
            var reply=runtime.command(Json.tree(Map.of("operationId","cache_"+UUID.randomUUID(),"command","player.resolve","expiresAt",java.time.Instant.now().plusSeconds(30).toString(),"payload",Map.of("gameId","CoreProbePeer"))));
            check(reply.get("status").getAsString().equals("COMPLETED"),"cached resolution failed: "+reply);check(reply.getAsJsonObject("data").get("playerUuid").getAsString().equals(historical.getUniqueId().toString()),"cached UUID changed");
            report("cache",Map.of("passed",true,"resolved",reply.getAsJsonObject("data"),"coreDirectoryRowRemoved",true));return null;
        }).exceptionally(error->{report("cache",Map.of("passed",false,"failure",error.toString()));return null;});
    }
    static String decimal(double value){return BigDecimal.valueOf(value).setScale(2,java.math.RoundingMode.HALF_UP).toPlainString();}
    static void check(boolean condition,String message){if(!condition)throw new IllegalStateException(message);}
    void report(String phase,Map<String,Object> data){try{Files.createDirectories(getDataFolder().toPath());Files.writeString(getDataFolder().toPath().resolve(phase+".json"),Json.GSON.toJson(data));getLogger().info("PROBE "+phase+" "+Json.GSON.toJson(data));}catch(Exception failure){throw new IllegalStateException(failure);}}
}
