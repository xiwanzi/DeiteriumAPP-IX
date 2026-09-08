package cafe.deuterium.probe;

import cafe.deuterium.core.api.PlayerDataService;
import java.lang.reflect.*;
import java.sql.*;
import java.util.concurrent.atomic.AtomicInteger;

/** Restorable test-only fault at the real database commit acknowledgement boundary. */
final class SyncCommitFault implements AutoCloseable {
    final AtomicInteger commits=new AtomicInteger();
    private final Object store,factory;
    private final Field field;
    SyncCommitFault(PlayerDataService service)throws Exception{
        Field storeField=service.getClass().getDeclaredField("store");storeField.setAccessible(true);store=storeField.get(service);
        field=store.getClass().getDeclaredField("connections");field.setAccessible(true);factory=field.get(store);
        Class<?> type=field.getType();Thread game=Thread.currentThread();
        Object faulty=Proxy.newProxyInstance(type.getClassLoader(),new Class<?>[]{type},(proxy,method,args)->{
            Object connection;try{connection=method.invoke(factory,args);}catch(InvocationTargetException e){throw e.getCause();}
            if(Thread.currentThread()!=game)return connection;
            return Proxy.newProxyInstance(Connection.class.getClassLoader(),new Class<?>[]{Connection.class},(p,m,a)->{
                Object result;try{result=m.invoke(connection,a);}catch(InvocationTargetException e){throw e.getCause();}
                if(m.getName().equals("commit")&&commits.incrementAndGet()==2)throw new SQLException("Isolated fault: commit succeeded but response was lost");return result;
            });
        });field.set(store,faulty);
    }
    @Override public void close()throws Exception{field.set(store,factory);}
}
