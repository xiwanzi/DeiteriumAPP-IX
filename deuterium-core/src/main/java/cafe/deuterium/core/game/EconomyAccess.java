package cafe.deuterium.core.game;

import cafe.deuterium.core.util.CoreFailure;
import cafe.deuterium.core.util.Json;
import com.google.gson.JsonObject;
import org.bukkit.Bukkit;
import java.lang.reflect.*;
import java.util.*;

/** Only XConomy's committed, idempotent API may acknowledge Core money.
 * Ordinary Vault success is not a database commit receipt. */
public final class EconomyAccess {
    private Class<?> api() {
        var plugin=Bukkit.getPluginManager().getPlugin("XConomy");
        if(plugin==null||!plugin.isEnabled())throw unavailable();
        try {var api=Class.forName("me.yic.xconomy.deuterium.ControlledEconomyAPI",true,plugin.getClass().getClassLoader());
            if(!Integer.valueOf(1).equals(api.getMethod("apiVersion").invoke(null)))throw unavailable();return api;
        } catch(ReflectiveOperationException|LinkageError failure){throw unavailable();}
    }
    private CoreFailure unavailable(){return new CoreFailure("ECONOMY_UNAVAILABLE","XConomy 持久化资金 API 未就绪。");}
    public boolean available(){try{api();return true;}catch(CoreFailure unavailable){return false;}}
    private JsonObject call(String method,Class<?>[] types,Object...args){
        try{return Json.tree(api().getMethod(method,types).invoke(null,args));}
        catch(InvocationTargetException exception){Throwable cause=exception.getCause();String code="RESULT_UNKNOWN";
            try{code=(String)cause.getClass().getMethod("code").invoke(cause);}catch(ReflectiveOperationException ignored){}
            throw new CoreFailure(code,code.equals("RESULT_UNKNOWN")?"资金结果尚未确认，请查询原操作。":Objects.requireNonNullElse(cause.getMessage(),"经济操作被拒绝。"),cause);
        }catch(ReflectiveOperationException failure){throw unavailable();}
    }
    public JsonObject balance(UUID player){return call("balance",new Class<?>[]{UUID.class},player);}
    public JsonObject records(JsonObject payload){
        Map<String,Object> fields=new LinkedHashMap<>();
        payload.entrySet().forEach(entry->{if(!entry.getValue().isJsonPrimitive()||!entry.getValue().getAsJsonPrimitive().isString())throw CoreFailure.invalid("无效流水查询参数。");fields.put(entry.getKey(),entry.getValue().getAsString());});
        return call("records",new Class<?>[]{Map.class},fields);
    }
    public JsonObject execute(String operation,String command,JsonObject payload){
        Map<String,Object> fields=new LinkedHashMap<>();payload.entrySet().forEach(entry->{if(entry.getValue().isJsonNull())fields.put(entry.getKey(),null);else if(entry.getValue().isJsonPrimitive())fields.put(entry.getKey(),entry.getValue().getAsString());else throw CoreFailure.invalid("资金参数必须为标量。");});
        return call("execute",new Class<?>[]{String.class,String.class,Map.class},operation,command,fields);
    }
    public JsonObject initializeSystemAccounts(){return call("initializeSystemAccounts",new Class<?>[]{});}
    public JsonObject queryOperation(String id){return call("queryOperation",new Class<?>[]{String.class},id);}
    public boolean mailCreditsAvailable(){
        try{api().getMethod("rewardMail",String.class,UUID.class,long.class);return true;}
        catch(ReflectiveOperationException|CoreFailure unavailable){return false;}
    }
    public JsonObject rewardMail(String id,UUID player,long credits){
        return call("rewardMail",new Class<?>[]{String.class,UUID.class,long.class},id,player,credits);
    }
    public boolean isSystemIdentity(UUID id,String name){
        if(name.equalsIgnoreCase("DIMA")||name.equalsIgnoreCase("DaoYu"))return true;
        try{return Boolean.TRUE.equals(api().getMethod("isSystemIdentity",UUID.class,String.class).invoke(null,id,name));}
        catch(ReflectiveOperationException e){throw new CoreFailure("SYSTEM_IDENTITY_UNAVAILABLE","系统账号保护校验暂不可用。",e);}
    }
}
