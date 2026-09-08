package cafe.deuterium.probe;

import cafe.deuterium.core.api.PlayerDataService;
import cafe.deuterium.core.util.CoreFailure;
import org.bukkit.*;
import org.bukkit.entity.Player;
import org.bukkit.inventory.ItemStack;
import java.io.ByteArrayInputStream;
import java.lang.reflect.*;
import java.util.*;

/** Actual server MOD/SQL probes. This class is never packaged in the production Core. */
final class SyncProbe {
    final CoreProbe probe;
    SyncProbe(CoreProbe probe) { this.probe=probe; }
    @SuppressWarnings("unchecked") void channels()throws Exception {
        Class<?> registry=Class.forName("net.neoforged.neoforge.network.registration.NetworkRegistry");
        Field field=registry.getDeclaredField("PAYLOAD_REGISTRATIONS");field.setAccessible(true);
        Map<?,?> map=(Map<?,?>)field.get(null);List<Object> protocols=new ArrayList<>();
        for(var protocol:map.entrySet()){
            List<Object> channels=new ArrayList<>();
            for(Object registration:((Map<?,?>)protocol.getValue()).values()){
                Optional<?> flow=(Optional<?>)call(registration,"flow");
                channels.add(Map.of("id",call(registration,"id").toString(),"version",call(registration,"version"),"optional",call(registration,"optional"),"flow",flow.isPresent()?((Enum<?>)flow.get()).ordinal():-1));
            }
            protocols.add(Map.of("ordinal",((Enum<?>)protocol.getKey()).ordinal(),"name",protocol.getKey().toString(),"channels",channels));
        }
        probe.report("channels",Map.of("protocols",protocols,"purpose","synthetic protocol fixture for isolated server tests"));
    }
    @SuppressWarnings("unchecked") void backpack()throws Exception {
        Player player=probe.player("CoreProbeUser");var runtime=probe.core().runtime();
        CoreProbe.check(runtime.config().storage().database().startsWith("dc_core_game_"),"isolated test database required");
        Class<?> craft=Class.forName("org.bukkit.craftbukkit.inventory.CraftItemStack");
        Class<?> nms=Class.forName("net.minecraft.world.item.ItemStack");
        Class<?> helper=syncClass("com.mohistmc.youermodsync.helper.BackpackStorageHelper");
        ItemStack bukkit=Bukkit.getItemFactory().createItemStack("sophisticatedbackpacks:backpack");
        Object stack=craft.getMethod("asNMSCopy",ItemStack.class).invoke(null,bukkit);
        Object wrapper=Class.forName("net.p3pp3rf1y.sophisticatedbackpacks.backpack.wrapper.BackpackWrapper").getMethod("fromStack",nms).invoke(null,stack);
        Object inventory=call(wrapper,"getInventoryHandler");
        Object diamond=craft.getMethod("asNMSCopy",ItemStack.class).invoke(null,new ItemStack(Material.DIAMOND,7));
        inventory.getClass().getMethod("setStackInSlot",int.class,nms).invoke(inventory,0,diamond);
        bukkit=(ItemStack)craft.getMethod("asBukkitCopy",nms).invoke(null,stack);
        CoreFailure rejected=null;try{runtime.codec.capture("deuterium:external_probe",bukkit,"",runtime.config());}catch(CoreFailure error){rejected=error;}
        CoreProbe.check(rejected!=null&&rejected.code().equals("ITEM_EXTERNAL_STORAGE_UNSUPPORTED"),"external storage alias must not enter library");
        UUID id=((Optional<UUID>)helper.getMethod("getBackpackUUID",nms).invoke(null,stack)).orElseThrow();
        String receipt;
        try(PlayerDataService.Lease lease=runtime.inventoryAccess().acquire(player,runtime.config().inventoryDomain(),UUID.randomUUID())){
            player.getInventory().setItem(0,bukkit);receipt=lease.saveAndConfirm();
        }
        Object expected=((Optional<?>)helper.getMethod("exportBackpack",nms).invoke(null,stack)).orElseThrow();
        byte[] stored=backpackBytes(id);CoreProbe.check(expected.equals(readNbt(stored)),"actual external backpack SQL snapshot mismatch");
        // Force a real external-storage empty state; the UUID-bearing item remains in the player's inventory.
        Class<?> tag=Class.forName("net.minecraft.nbt.CompoundTag");
        Object emptyExport=call(expected,"copy");emptyExport.getClass().getMethod("put",String.class,Class.forName("net.minecraft.nbt.Tag")).invoke(emptyExport,"contents",tag.getConstructor().newInstance());
        try(PlayerDataService.Lease lease=runtime.inventoryAccess().acquire(player,runtime.config().inventoryDomain(),UUID.randomUUID())){
            CoreProbe.check((boolean)helper.getMethod("importBackpack",nms,tag).invoke(null,stack,emptyExport),"empty backpack import failed");
            lease.saveAndConfirm();
        }
        Object persistedEmpty=readNbt(backpackBytes(id));
        Object contents=persistedEmpty.getClass().getMethod("getCompound",String.class).invoke(persistedEmpty,"contents");
        CoreProbe.check((boolean)call(contents,"isEmpty"),"empty external backpack did not overwrite old contents");
        probe.report("backpack",Map.of("passed",true,"backpackUuid",id.toString(),"snapshotReceipt",receipt,"actualModContentSaved",true,"emptyContentReplaced",true,"externalAliasRejected",true));
    }
    void failure()throws Exception {
        Player player=probe.player("CoreProbeUser");var runtime=probe.core().runtime();
        CoreProbe.check(runtime.config().storage().database().startsWith("dc_core_game_"),"isolated test database required");
        byte[] old=runtime.database.read(c->{try(var p=c.prepareStatement("SELECT player_nbt FROM player_data WHERE uuid=?")){p.setString(1,player.getUniqueId().toString());try(var r=p.executeQuery()){CoreProbe.check(r.next(),"previous save missing");return r.getBytes(1);}}});
        runtime.database.read(c->{try(var s=c.createStatement()){s.executeUpdate("CREATE TRIGGER dc_sync_probe_fault BEFORE INSERT ON player_data FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='isolated probe forced save failure'");}return null;});
        UUID operation=UUID.randomUUID();boolean rejected=false;
        try{
            try(PlayerDataService.Lease lease=runtime.inventoryAccess().acquire(player,runtime.config().inventoryDomain(),operation)){
                player.getInventory().setItem(9,new ItemStack(Material.EMERALD,13));lease.saveAndConfirm();
            }
        }catch(Exception expected){rejected=true;}
        finally{runtime.database.read(c->{try(var s=c.createStatement()){s.executeUpdate("DROP TRIGGER dc_sync_probe_fault");}return null;});}
        CoreProbe.check(rejected,"SQL failure was reported as saved");
        Map<String,Object> result=runtime.database.read(c->{
            try(var p=c.prepareStatement("SELECT p.player_nbt,s.phase,s.operation_id,(SELECT COUNT(*) FROM dc_sync_save_receipts WHERE operation_id=?) AS receipts FROM player_data p JOIN dc_sync_sessions s ON s.uuid=p.uuid WHERE p.uuid=?")){
                p.setString(1,operation.toString());p.setString(2,player.getUniqueId().toString());try(var r=p.executeQuery()){
                    CoreProbe.check(r.next(),"failed session missing");CoreProbe.check(Arrays.equals(old,r.getBytes(1)),"failed save changed committed player data");CoreProbe.check(r.getString(2).equals("FAILED")&&r.getString(3).equals(operation.toString())&&r.getInt(4)==0,"failed save not quarantined or false proof produced");
                    return Map.of("passed",true,"operationId",operation.toString(),"phase",r.getString(2),"falseSaveReceipts",r.getInt(4),"committedPlayerDataPreserved",true);
                }
            }
        });probe.report("sync-failure",result);
    }
    void unknown()throws Exception {
        Player player=probe.player("CoreProbeUser");var runtime=probe.core().runtime();
        CoreProbe.check(runtime.config().storage().database().startsWith("dc_core_game_"),"isolated test database required");
        Object service=runtime.inventoryAccess();
        Field storeField=service.getClass().getDeclaredField("store");storeField.setAccessible(true);Object store=storeField.get(service);
        Field factoryField=store.getClass().getDeclaredField("connections");factoryField.setAccessible(true);Object factory=factoryField.get(store);
        Class<?> factoryType=factoryField.getType();Thread gameThread=Thread.currentThread();var commits=new java.util.concurrent.atomic.AtomicInteger();
        Object faulty=Proxy.newProxyInstance(factoryType.getClassLoader(),new Class<?>[]{factoryType},(proxy,method,args)->{
            Object connection;try{connection=method.invoke(factory,args);}catch(InvocationTargetException e){throw e.getCause();}
            if(Thread.currentThread()!=gameThread)return connection;
            return Proxy.newProxyInstance(java.sql.Connection.class.getClassLoader(),new Class<?>[]{java.sql.Connection.class},(p,m,a)->{
                Object result;try{result=m.invoke(connection,a);}catch(InvocationTargetException e){throw e.getCause();}
                if(m.getName().equals("commit")&&commits.incrementAndGet()==2)throw new java.sql.SQLException("Isolated fault: database committed but acknowledgement lost");return result;
            });
        });
        UUID operation=UUID.randomUUID();boolean rejected=false;
        factoryField.set(store,faulty);
        try{
            try(PlayerDataService.Lease lease=runtime.inventoryAccess().acquire(player,runtime.config().inventoryDomain(),operation)){
                player.getInventory().setItem(10,new ItemStack(Material.GOLD_INGOT,11));lease.saveAndConfirm();
            }
        }catch(Exception expected){rejected=true;}
        finally{factoryField.set(store,factory);}
        CoreProbe.check(rejected&&commits.get()>=2,"Commit acknowledgement failure was not injected");
        Map<String,Object> result=runtime.database.read(c->{
            try(var p=c.prepareStatement("SELECT s.phase,s.epoch,r.snapshot_sha256 FROM dc_sync_sessions s JOIN dc_sync_save_receipts r ON r.operation_id=s.operation_id WHERE s.uuid=? AND r.operation_id=?")){
                p.setString(1,player.getUniqueId().toString());p.setString(2,operation.toString());try(var r=p.executeQuery()){
                    CoreProbe.check(r.next()&&r.getString(1).equals("FAILED"),"Unknown committed claim lacks quarantine and durable proof");
                    return Map.of("passed",true,"operationId",operation.toString(),"phase",r.getString(1),"saveReceipt","youermodsync-v1:"+operation+":"+r.getString(2)+":"+r.getString(3),"lostCommitAcknowledgement",true);
                }
            }
        });probe.report("sync-unknown",result);
    }
    void history()throws Exception {
        Player player=probe.player("CoreProbeUser");var runtime=probe.core().runtime();
        CoreProbe.check(runtime.config().storage().database().startsWith("dc_core_game_"),"isolated test database required");
        Class<?> nmsPlayer=Class.forName("net.minecraft.world.entity.player.Player");Object nms=Class.forName("com.mohistmc.youer.api.PlayerAPI").getMethod("getNMSPlayer",Player.class).invoke(null,player);
        Class<?> history=syncClass("com.mohistmc.youermodsync.AutoHistorySyncManager");
        ItemStack before=player.getInventory().getItem(8);before=before==null?null:before.clone();
        Bukkit.dispatchCommand(Bukkit.getConsoleSender(),"yms save CoreProbeUser");
        String timestamp;
        // The existing Sync history UI uses the Sync SQL session's local timestamp, while Core uses UTC.
        try(var c=(java.sql.Connection)syncClass("com.mohistmc.youermodsync.JDBCSetUp").getMethod("getConnection").invoke(null);var p=c.prepareStatement("SELECT DATE_FORMAT(MAX(save_time),'%Y-%m-%d %H:%i:%s') FROM history_data WHERE uuid=? AND data_type='player_nbt'")){
            p.setString(1,player.getUniqueId().toString());try(var r=p.executeQuery()){CoreProbe.check(r.next(),"History checkpoint missing");timestamp=r.getString(1);}
        }
        GameMode old=player.getGameMode();player.setGameMode(GameMode.CREATIVE);player.getInventory().setItem(8,new ItemStack(Material.NETHERITE_INGOT,17));
        try{
            boolean restored=(boolean)history.getMethod("restorePlayerFromHistory",Player.class,String.class).invoke(null,player,timestamp);
            CoreProbe.check(restored,"History restore returned failure");CoreProbe.check(Objects.equals(before,player.getInventory().getItem(8)),"History inventory was not restored");
            CoreProbe.check(player.getGameMode()==GameMode.CREATIVE&&player.getAllowFlight(),"Preserved .2 local game mode/abilities regression");
        }finally{player.setGameMode(old);}
        probe.report("sync-history",Map.of("passed",true,"timestamp",timestamp,"inventoryRestored",true,"localGameModeAndAbilitiesPreserved",true));
    }
    byte[] backpackBytes(UUID id)throws Exception{return probe.core().runtime().database.read(c->{try(var p=c.prepareStatement("SELECT backpack_nbt FROM backpack_data WHERE uuid=?")){p.setString(1,id.toString());try(var r=p.executeQuery()){CoreProbe.check(r.next(),"backpack row missing");return r.getBytes(1);}}});}
    Object readNbt(byte[] bytes)throws Exception{Class<?> account=Class.forName("net.minecraft.nbt.NbtAccounter");Object bound=account.getConstructor(long.class,int.class).newInstance(32L*1024*1024,64);return Class.forName("net.minecraft.nbt.NbtIo").getMethod("readCompressed",java.io.InputStream.class,account).invoke(null,new ByteArrayInputStream(bytes),bound);}
    Class<?> syncClass(String name)throws Exception{return Class.forName(name,true,Bukkit.getPluginManager().getPlugin("YouerModSync").getClass().getClassLoader());}
    static Object call(Object object,String name)throws Exception{Method method=object.getClass().getMethod(name);method.setAccessible(true);return method.invoke(object);}
}
