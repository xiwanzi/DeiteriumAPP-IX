package me.yic.xconomy.deuterium;

import java.math.BigDecimal;
import java.sql.*;
import java.time.*;
import java.util.*;

/** Read-only views of committed economic events, including pre-upgrade history. */
final class LedgerReader {
    private final FundsEngine.Connections connections;
    private final String accounts, prefix;
    LedgerReader(FundsEngine.Connections connections, String accounts, String prefix) {
        this.connections = connections; this.accounts = accounts; this.prefix = prefix;
    }

    Map<String, Object> records(Map<String, Object> input) {
        try {
            if (!input.keySet().equals(Set.of("playerUuid", "from", "to", "direction", "businessType", "beforeTime", "beforeSequence", "snapshot", "limit", "recordId"))) throw invalid();
            String player = string(input, "playerUuid"), direction = string(input, "direction"), business = string(input, "businessType");
            if (!player.isEmpty() && !UUID.fromString(player).toString().equals(player)) throw invalid();
            if (!Set.of("", "income", "expense").contains(direction) || !Set.of("", "TRANSFER", "GAME", "OFFICIAL_STORE", "MARKET_ORDER", "COMMISSION").contains(business)) throw invalid();
            Instant from = Instant.parse(string(input, "from")), to = Instant.parse(string(input, "to"));
            long before = Long.parseLong(string(input, "beforeSequence")), snapshot = Long.parseLong(string(input, "snapshot"));
            String record = string(input, "recordId"), beforeTime = string(input, "beforeTime");
            int limit = Integer.parseInt(string(input, "limit"));
            if (!from.isBefore(to) || before < 0 || snapshot < 0 || limit < 1 || limit > 100 || (!record.isEmpty() && !record.matches("[1-9][0-9]{0,18}"))) throw invalid();
            if (before > 0 && (snapshot < before || beforeTime.isEmpty())) throw invalid();
            try (Connection c = connections.open()) {
                c.setReadOnly(true); c.setTransactionIsolation(Connection.TRANSACTION_REPEATABLE_READ); c.setAutoCommit(false);
                try {
                    if (before == 0) try (var s = c.prepareStatement("SELECT COALESCE(MAX(sequence_id),0) FROM " + prefix + "ledger"); var r = s.executeQuery()) { r.next(); snapshot = r.getLong(1); }
                    String nativeEvent = "(COALESCE(o.command_type,'') LIKE 'native.%' OR l.business_type LIKE 'NATIVE\\_%')";
                    String query = "SELECT l.*,a.player AS player_name,COALESCE(o.command_type,'') AS command_type,"
                        + "(SELECT p.player_uuid FROM " + prefix + "ledger p WHERE p.operation_id=l.operation_id AND p.player_uuid<>l.player_uuid AND COALESCE(o.command_type,'')<>'native.bulk' LIMIT 1) AS other_uuid "
                        + "FROM " + prefix + "ledger l LEFT JOIN " + accounts + " a ON a.UID=l.player_uuid LEFT JOIN " + prefix + "operations o ON o.operation_id=l.operation_id "
                        + "WHERE l.sequence_id<=? AND l.created_at>=? AND l.created_at<? AND l.delta<>0";
                    List<Object> args = new ArrayList<>(List.of(snapshot, timestamp(from), timestamp(to)));
                    if (!player.isEmpty()) { query += " AND l.player_uuid=?"; args.add(player); }
                    else query += " AND NOT EXISTS(SELECT 1 FROM " + prefix + "system_accounts sa WHERE sa.player_uuid=l.player_uuid)";
                    if (!direction.isEmpty()) query += direction.equals("income") ? " AND l.delta>0" : " AND l.delta<0";
                    if (business.equals("GAME")) query += " AND (" + nativeEvent + ")";
                    else if (business.equals("TRANSFER")) query += " AND l.business_type='TRANSFER' AND NOT(" + nativeEvent + ")";
                    else if (!business.isEmpty()) { query += " AND l.business_type LIKE ?"; args.add(business + "\\_%"); }
                    if (before > 0) {
                        Instant at = Instant.parse(beforeTime);
                        if (at.isBefore(from) || !at.isBefore(to)) throw invalid();
                        query += " AND (l.created_at<? OR (l.created_at=? AND l.sequence_id<?))";
                        args.add(timestamp(at)); args.add(timestamp(at)); args.add(before);
                    }
                    if (!record.isEmpty()) { query += " AND l.sequence_id=?"; args.add(Long.parseLong(record)); }
                    query += " ORDER BY l.created_at DESC,l.sequence_id DESC LIMIT ?"; args.add(limit + 1);
                    List<Map<String, Object>> rows = new ArrayList<>();
                    try (var s = c.prepareStatement(query)) {
                        for (int i = 0; i < args.size(); i++) s.setObject(i + 1, args.get(i));
                        try (var r = s.executeQuery()) { while (r.next()) {
                            Map<String, Object> row = new LinkedHashMap<>();
                            BigDecimal delta = r.getBigDecimal("delta");
                            row.put("sequence", Long.toString(r.getLong("sequence_id")));
                            row.put("playerUuid", r.getString("player_uuid")); row.put("gameId", Objects.requireNonNullElse(r.getString("player_name"), r.getString("player_uuid")));
                            row.put("otherUuid", r.getString("other_uuid")); row.put("operationId", r.getString("operation_id"));
                            row.put("businessRef", r.getString("business_ref")); row.put("businessType", r.getString("business_type"));
                            row.put("source", r.getString("command_type").startsWith("native.") || r.getString("business_type").startsWith("NATIVE_") ? "GAME" : "APP");
                            row.put("direction", delta.signum() > 0 ? "income" : "expense"); row.put("amount", delta.abs().toPlainString());
                            row.put("beforeBalance", r.getBigDecimal("before_balance").toPlainString()); row.put("afterBalance", r.getBigDecimal("after_balance").toPlainString());
                            row.put("occurredAt", r.getObject("created_at", LocalDateTime.class).toInstant(ZoneOffset.UTC).toString());
                            rows.add(row);
                        } }
                    }
                    c.commit(); return Map.of("records", rows, "snapshot", Long.toString(snapshot));
                } catch (Throwable error) { c.rollback(); throw error; }
            }
        } catch (SQLException error) { throw new FundsFailure("STORAGE_UNAVAILABLE", "经济流水暂时无法读取。", error); }
        catch (IllegalArgumentException | java.time.DateTimeException error) { throw invalid(); }
    }

    static Map<String, String> today(Connection c, String prefix, UUID player) throws SQLException {
        ZoneOffset zone = ZoneOffset.ofHours(8);
        Instant from = LocalDate.now(zone).atStartOfDay().toInstant(zone), to = from.plusSeconds(86400);
        try (var s = c.prepareStatement("SELECT COALESCE(SUM(CASE WHEN delta>0 THEN delta ELSE 0 END),0),COALESCE(SUM(CASE WHEN delta<0 THEN -delta ELSE 0 END),0) FROM " + prefix + "ledger WHERE player_uuid=? AND created_at>=? AND created_at<?")) {
            s.setString(1, player.toString()); s.setTimestamp(2, timestamp(from)); s.setTimestamp(3, timestamp(to));
            try (var r = s.executeQuery()) { r.next(); return Map.of("income", r.getBigDecimal(1).setScale(2).toPlainString(), "expense", r.getBigDecimal(2).setScale(2).toPlainString(), "date", LocalDate.ofInstant(from, zone).toString(), "timeZone", "+08:00"); }
        }
    }
    private static Timestamp timestamp(Instant at) { return Timestamp.valueOf(LocalDateTime.ofInstant(at, ZoneOffset.UTC)); }
    private static String string(Map<String, Object> input, String key) { if (!(input.get(key) instanceof String value)) throw invalid(); return value; }
    private static FundsFailure invalid() { return new FundsFailure("INVALID_REQUEST", "无效流水查询。"); }
}
