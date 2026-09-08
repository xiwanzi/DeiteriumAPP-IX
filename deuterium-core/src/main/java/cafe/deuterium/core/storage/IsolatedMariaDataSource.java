package cafe.deuterium.core.storage;

import org.mariadb.jdbc.Driver;
import javax.sql.DataSource;
import java.io.PrintWriter;
import java.sql.*;
import java.util.Properties;
import java.util.logging.Logger;

/** Uses Core's relocated driver directly, including authentication SPI resources.
 * Server-provided drivers accepting the same JDBC URL never participate. */
public final class IsolatedMariaDataSource implements DataSource {
    private final Driver driver=new Driver();
    private final String url,user,password;
    public IsolatedMariaDataSource(String url,String user,String password){this.url=url;this.user=user;this.password=password;}
    public static String driverName(){return Driver.class.getName();}
    @Override public Connection getConnection()throws SQLException{return getConnection(user,password);}
    @Override public Connection getConnection(String username,String credential)throws SQLException{
        Properties properties=new Properties();properties.setProperty("user",username);properties.setProperty("password",credential);
        Thread thread=Thread.currentThread();ClassLoader previous=thread.getContextClassLoader();
        try{thread.setContextClassLoader(Driver.class.getClassLoader());Connection c=driver.connect(url,properties);if(c==null)throw new SQLException("Core driver rejected JDBC URL");return c;}
        finally{thread.setContextClassLoader(previous);}
    }
    @Override public PrintWriter getLogWriter(){return null;}
    @Override public void setLogWriter(PrintWriter out){ }
    @Override public int getLoginTimeout(){return 3;}
    @Override public void setLoginTimeout(int seconds){ }
    @Override public Logger getParentLogger(){return Logger.getLogger("DeuteriumCore.Database");}
    @Override public boolean isWrapperFor(Class<?> type){return type.isInstance(this);}
    @Override public <T>T unwrap(Class<T> type)throws SQLException{if(type.isInstance(this))return type.cast(this);throw new SQLException("Not a wrapper for requested class");}
}
