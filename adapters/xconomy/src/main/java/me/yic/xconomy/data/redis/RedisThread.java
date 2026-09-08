/* GPL-3.0-or-later. Deuterium lifecycle replacement for XConomy 2.26.3. */
package me.yic.xconomy.data.redis;

import me.yic.xconomy.XConomy;
import me.yic.xconomy.utils.RedisConnection;
import me.yic.xc_libs.redis.jedis.Jedis;

/** Stop the active socket and join before Bukkit closes the plugin class loader. */
public final class RedisThread extends Thread {
    private static volatile RedisThread active;
    private volatile boolean stopping;
    private volatile Jedis connection;
    public RedisThread(){super("XConomyRedisSub");setDaemon(true);active=this;}
    @Override public void run(){
        int failures=0;
        try{
            while(!stopping&&!isInterrupted()){
                try(Jedis resource=RedisConnection.getResource()){
                    connection=resource;if(stopping)break;
                    resource.subscribe(RedisConnection.subscriber,RedisConnection.channelname);
                    failures=0;
                }catch(Throwable failed){if(stopping)break;XConomy.getInstance().getLogger().warning("Redis subscription interrupted; reconnecting with bounded backoff.");}
                finally{connection=null;}
                if(!stopping)try{Thread.sleep(Math.min(10000,500L<<Math.min(failures++,5)));}catch(InterruptedException stop){interrupt();break;}
            }
        }finally{if(active==this)active=null;}
    }
    public static void shutdown(){
        RedisThread thread=active;if(thread==null)return;thread.stopping=true;
        try{RedisConnection.subscriber.unsubscribe();}catch(Throwable ignored){}
        Jedis connection=thread.connection;if(connection!=null)try{connection.disconnect();}catch(Throwable ignored){}
        thread.interrupt();
        if(thread!=Thread.currentThread())try{thread.join(3000);}catch(InterruptedException interrupted){Thread.currentThread().interrupt();}
        if(thread.isAlive())XConomy.getInstance().getLogger().warning("Redis subscription did not exit within shutdown deadline; connection was closed.");
    }
}
