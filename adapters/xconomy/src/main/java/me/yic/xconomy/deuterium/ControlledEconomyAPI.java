package me.yic.xconomy.deuterium;

import me.yic.xconomy.XConomy;
import me.yic.xconomy.XConomyLoad;
import me.yic.xconomy.data.DataCon;
import me.yic.xconomy.data.DataFormat;
import me.yic.xconomy.data.DataLink;
import me.yic.xconomy.data.caches.Cache;
import me.yic.xconomy.data.sql.SQL;
import me.yic.xconomy.data.syncdata.PlayerData;
import me.yic.xconomy.utils.UUIDMode;
import org.bukkit.Bukkit;
import java.math.BigDecimal;
import java.util.*;

/** GPL-3.0-or-later. Public, thread-safe API owned by the patched XConomy plugin.
 * Input maps contain strings only. No credentials or SQL cross this boundary. */
public final class ControlledEconomyAPI {
    private static volatile FundsEngine engine;
    private static FundsEngine engine(){
        FundsEngine ready=engine;if(ready!=null)return ready;
        synchronized(ControlledEconomyAPI.class){
            if(engine==null){
                if(!XConomyLoad.DConfig.isMySQL()||!XConomyLoad.DConfig.EnableConnectionPool||DataFormat.isint||DataFormat.maxNumber==null)
                    throw new FundsFailure("ECONOMY_CONFIGURATION_UNSUPPORTED","持久化资金 API 要求 MySQL/MariaDB 连接池与两位小数经济配置。");
                FundsEngine created=new FundsEngine(()->SQL.database.getConnectionAndCheck(),SQL.tableName,ControlledEconomyAPI::refresh,DataFormat.maxNumber);
                created.initialize();engine=created;
            }return engine;
        }
    }
    public static int apiVersion(){engine();return 1;}
    public static Map<String,Object> initializeSystemAccounts(){return engine().initializeSystemAccounts();}
    public static Map<String,Object> balance(UUID player){return engine().balance(player);}
    public static Map<String,Object> execute(String operation,String command,Map<String,Object> payload){
        if(command.equals("wallet.escrow.query")){
            if(payload.size()!=2||!(payload.get("escrowRef") instanceof String ref)||!(payload.get("businessRef") instanceof String business))throw new FundsFailure("INVALID_REQUEST","无效担保查询。");
            return engine().hold(ref,business);
        }
        if(!Set.of("wallet.transfer","wallet.escrow.reserve","wallet.escrow.bind","wallet.escrow.settle","wallet.escrow.refund").contains(command))throw new FundsFailure("COMMAND_NOT_ALLOWED","未开放该受控资金操作。");
        return engine().execute(operation,command,payload);
    }
    public static Map<String,Object> queryOperation(String id){return engine().operation(id);}
    public static BigDecimal nativeChange(UUID id,BigDecimal amount,Boolean add,String type,String command){return engine().nativeChange(id,DataFormat.formatBigDecimal(amount),add,type,command);}
    public static void nativePay(UUID from,UUID to,BigDecimal debit,BigDecimal credit,String command){engine().nativeTransfer(from,to,DataFormat.formatBigDecimal(debit),DataFormat.formatBigDecimal(credit),command);}
    public static void nativeBulk(Collection<UUID> ids,BigDecimal amount,Boolean add,String type,String command){engine().nativeBulk(ids,DataFormat.formatBigDecimal(amount),add,type,command);}
    public static void deleteAccount(UUID id){engine().deleteAccount(id,XConomyLoad.Config.UUIDMODE.equals(UUIDMode.SEMIONLINE)?SQL.tableUUIDName:null);}
    public static boolean protectedName(String name){return name!=null&&(name.equalsIgnoreCase("DIMA")||name.equalsIgnoreCase("DaoYu"));}
    public static boolean isSystemIdentity(UUID uuid,String name){return engine().isSystemIdentity(uuid,name);}
    private static void refresh(Set<UUID> changed){
        // Balance reads bypass the cache on every node. Redis remains useful for
        // display invalidation but delayed messages cannot overwrite a balance.
        Runnable update=()->{
            for(UUID id:changed){try{DataCon.deletedatafromcache(id);PlayerData pd=DataLink.getPlayerData(id);if(pd!=null&&XConomyLoad.getSyncData_Enable())DataCon.SendMessTask(pd);}catch(Throwable failure){XConomy.getInstance().getLogger().warning("Committed balance cache notification failed; database receipt remains authoritative.");}}
        };
        if(Bukkit.isPrimaryThread())update.run();else Bukkit.getScheduler().runTask(XConomy.getInstance(),update);
    }
}
