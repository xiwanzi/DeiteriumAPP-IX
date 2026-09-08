package cafe.deuterium.core.storage;

import java.sql.*;

public final class Sql {
    private Sql() { }
    public static PreparedStatement statement(Connection c, String sql, Object... values) throws SQLException {
        PreparedStatement s = c.prepareStatement(sql);
        try { for (int i = 0; i < values.length; i++) s.setObject(i + 1, values[i]); return s; }
        catch (SQLException e) { s.close(); throw e; }
    }
    public static int update(Connection c, String sql, Object... values) throws SQLException {
        try (PreparedStatement s = statement(c, sql, values)) { return s.executeUpdate(); }
    }
    public static long count(Connection c, String sql, Object... values) throws SQLException {
        try (PreparedStatement s = statement(c, sql, values); ResultSet r = s.executeQuery()) { return r.next() ? r.getLong(1) : 0; }
    }
    public static void audit(Connection c, String actor, String action, String resource, String result) throws SQLException {
        update(c, "INSERT INTO dc_audit(actor,action,resource,result,created_at) VALUES(?,?,?,?,?)",
                actor, action, resource, result, System.currentTimeMillis());
    }
}
