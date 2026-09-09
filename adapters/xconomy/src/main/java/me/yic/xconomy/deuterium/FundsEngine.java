package me.yic.xconomy.deuterium;

import com.google.gson.Gson;
import com.google.gson.reflect.TypeToken;
import java.math.*;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.sql.*;
import java.time.Instant;
import java.util.*;
import java.util.function.Consumer;

/** GPL-3.0-or-later. The XConomy database is the single commit boundary for
 * balances, frozen counterparties, holds, operation receipts and both ledger legs.
 * No Bukkit objects, caches or asynchronous SQL writes occur in this engine. */
public final class FundsEngine {
    @FunctionalInterface public interface Connections { Connection open() throws SQLException; }
    private static final BigDecimal ZERO=new BigDecimal("0.00"),MAX=new BigDecimal("1000000000000.00");
    private static final Gson JSON=new com.google.gson.GsonBuilder().serializeNulls().disableHtmlEscaping().create();
    private static final java.lang.reflect.Type MAP=new TypeToken<LinkedHashMap<String,Object>>(){}.getType();
    private final Connections connections;
    private final String accounts,prefix;
    private final Consumer<Set<UUID>> committed;
    private final BigDecimal balanceLimit;
    private volatile boolean initialized;
    public FundsEngine(Connections connections,String accounts,Consumer<Set<UUID>> committed){
        this(connections,accounts,committed,MAX);
    }
    public FundsEngine(Connections connections,String accounts,Consumer<Set<UUID>> committed,BigDecimal balanceLimit){
        if(!accounts.matches("[A-Za-z0-9_]{1,40}"))throw fail("INVALID_CONFIGURATION","无效经济表名。");
        this.connections=connections;this.accounts=accounts;this.prefix=accounts+"_dc_";this.committed=committed;this.balanceLimit=balanceLimit.min(MAX);
    }
    public synchronized void initialize(){
        if(initialized)return;
        try(Connection c=connections.open()){
            if(c==null)throw new SQLException("XConomy database unavailable");
            String lock="dc-xconomy-schema:"+c.getCatalog()+":"+accounts;
            try(var s=statement(c,"SELECT GET_LOCK(?,10)",lock);var r=s.executeQuery()){if(!r.next()||r.getInt(1)!=1)throw new SQLException("schema lease unavailable");}
            try {
                try(var s=statement(c,"SELECT @@innodb_flush_log_at_trx_commit");var r=s.executeQuery()){if(!r.next()||r.getInt(1)!=1)throw fail("ECONOMY_DURABILITY_UNAVAILABLE","资金事务要求 innodb_flush_log_at_trx_commit=1。");}
                try(var s=statement(c,"SELECT ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=?",accounts);var r=s.executeQuery()){
                    if(!r.next()||!"InnoDB".equalsIgnoreCase(r.getString(1)))throw fail("ECONOMY_SCHEMA_UNSUPPORTED","受控资金要求 XConomy 使用 MySQL/MariaDB InnoDB。");
                }
                exec(c,"CREATE TABLE IF NOT EXISTS "+prefix+"system_accounts (account_key VARCHAR(16) CHARACTER SET ascii PRIMARY KEY,player_uuid CHAR(36) CHARACTER SET ascii NOT NULL UNIQUE,player_name VARCHAR(32) NOT NULL UNIQUE) ENGINE=InnoDB");
                exec(c,"CREATE TABLE IF NOT EXISTS "+prefix+"initialization (id INT PRIMARY KEY) ENGINE=InnoDB");
                exec(c,"INSERT IGNORE INTO "+prefix+"initialization VALUES(1)");
                exec(c,"CREATE TABLE IF NOT EXISTS "+prefix+"operations (operation_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,fingerprint CHAR(64) CHARACTER SET ascii NOT NULL,command_type VARCHAR(64) CHARACTER SET ascii NOT NULL,state VARCHAR(16) CHARACTER SET ascii NOT NULL,result MEDIUMTEXT NULL,created_at DATETIME(6) NOT NULL,committed_at DATETIME(6) NULL) ENGINE=InnoDB");
                exec(c,"CREATE TABLE IF NOT EXISTS "+prefix+"holds (escrow_ref VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,business_ref VARCHAR(128) CHARACTER SET ascii NOT NULL UNIQUE,business_type VARCHAR(24) CHARACTER SET ascii NOT NULL,payer_uuid CHAR(36) CHARACTER SET ascii NOT NULL,payee_uuid CHAR(36) CHARACTER SET ascii NULL,reserved DECIMAL(16,2) NOT NULL,settled DECIMAL(16,2) NOT NULL DEFAULT 0,refunded DECIMAL(16,2) NOT NULL DEFAULT 0,created_at DATETIME(6) NOT NULL,updated_at DATETIME(6) NOT NULL,INDEX ix_payer(payer_uuid)) ENGINE=InnoDB");
                exec(c,"CREATE TABLE IF NOT EXISTS "+prefix+"ledger (sequence_id BIGINT AUTO_INCREMENT PRIMARY KEY,operation_id VARCHAR(128) CHARACTER SET ascii NOT NULL,player_uuid CHAR(36) CHARACTER SET ascii NOT NULL,business_ref VARCHAR(128) CHARACTER SET ascii NOT NULL,escrow_ref VARCHAR(128) CHARACTER SET ascii NULL,business_type VARCHAR(32) CHARACTER SET ascii NOT NULL,delta DECIMAL(16,2) NOT NULL,before_balance DECIMAL(16,2) NOT NULL,after_balance DECIMAL(16,2) NOT NULL,created_at DATETIME(6) NOT NULL,UNIQUE KEY uq_leg(operation_id,player_uuid),INDEX ix_player(player_uuid,sequence_id)) ENGINE=InnoDB");
                ensureLedgerIndex(c,"ix_ledger_time","created_at,sequence_id");
                ensureLedgerIndex(c,"ix_ledger_player_time","player_uuid,created_at,sequence_id");
                initialized=true;
            } finally {try{exec(c,"DO RELEASE_LOCK(?)",lock);}catch(SQLException ignored){}}
        }catch(SQLException error){throw fail("STORAGE_UNAVAILABLE","经济存储未就绪。",error);}
    }
    public Map<String,Object> initializeSystemAccounts(){
        initialize();return transaction(c->{
            try(var s=statement(c,"SELECT id FROM "+prefix+"initialization WHERE id=1 FOR UPDATE");var r=s.executeQuery()){if(!r.next())throw new SQLException("missing initialization row");}
            var result=new LinkedHashMap<String,Object>();
            for(String name:List.of("DIMA","DaoYu")){
                String id=system(c,name);
                if(id==null){
                    try(var s=statement(c,"SELECT UID FROM "+accounts+" WHERE LOWER(player)=LOWER(?) FOR UPDATE",name);var r=s.executeQuery()){
                        if(r.next())throw fail("SYSTEM_ACCOUNT_COLLISION","系统账号名称已有经济身份，禁止接管："+name);
                    }
                    id=UUID.randomUUID().toString();
                    exec(c,"INSERT INTO "+accounts+"(UID,player,balance,hidden) VALUES(?,?,0,1)",id,name);
                    exec(c,"INSERT INTO "+prefix+"system_accounts VALUES(?,?,?)",name.toLowerCase(Locale.ROOT),id,name);
                }
                Account a=account(c,id,true);
                if(!a.name.equals(name))throw fail("SYSTEM_ACCOUNT_CONFLICT","系统账号 UUID 与名称不匹配。");
                result.put(name,Map.of("playerUuid",id,"gameId",name,"balance",a.balance.toPlainString()));
            }
            return result;
        });
    }
    public Map<String,Object> balance(UUID uuid){
        initialize();return transaction(c->{
            String pool=system(c,"DaoYu");var ids=new HashSet<String>();ids.add(uuid.toString());if(pool!=null)ids.add(pool);Map<String,Account> locked=lockAccounts(c,ids);
            Account a=locked.get(uuid.toString());ordinary(c,a);if(pool!=null)assertPool(c,locked.get(pool),null);
            var breakdown=new LinkedHashMap<String,String>();breakdown.put("market","0.00");breakdown.put("commissions","0.00");breakdown.put("store","0.00");BigDecimal held=ZERO;
            try(var s=statement(c,"SELECT business_type,SUM(reserved-settled-refunded) FROM "+prefix+"holds WHERE payer_uuid=? GROUP BY business_type",uuid.toString());var r=s.executeQuery()){
                while(r.next()){BigDecimal value=money(r.getBigDecimal(2),true);held=held.add(value);String group=switch(r.getString(1)){case "MARKET_ORDER"->"market";case "COMMISSION"->"commissions";default->"store";};breakdown.put(group,value.toPlainString());}
            }
            return Map.of("currency","CREDIT","amount",a.balance.toPlainString(),"availableAmount",a.balance.toPlainString(),"heldAmount",held.toPlainString(),"heldBreakdown",breakdown,"refreshedAt",Instant.now().toString(),"today",LedgerReader.today(c,prefix,uuid));
        });
    }
    private void ensureLedgerIndex(Connection c,String name,String columns)throws SQLException{
        try(var s=statement(c,"SELECT 1 FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=? AND INDEX_NAME=? LIMIT 1",prefix+"ledger",name);var r=s.executeQuery()){
            if(r.next())return;
        }
        exec(c,"CREATE INDEX "+name+" ON "+prefix+"ledger ("+columns+")");
    }
    public Map<String,Object> records(Map<String,Object> input){initialize();return new LedgerReader(connections,accounts,prefix).records(input);}
    public Map<String,Object> execute(String operation,String command,Map<String,Object> input){
        initialize();identifier(operation);TreeMap<String,Object> fields=new TreeMap<>(input);
        String fingerprint=sha(command+":"+JSON.toJson(fields));Set<UUID> changed=new HashSet<>();
        Map<String,Object> result=transaction(c->{
            exec(c,"INSERT INTO "+prefix+"operations VALUES(?,?,?,'EXECUTING',NULL,UTC_TIMESTAMP(6),NULL) ON DUPLICATE KEY UPDATE operation_id=VALUES(operation_id)",operation,fingerprint,command);
            try(var s=statement(c,"SELECT fingerprint,state,result FROM "+prefix+"operations WHERE operation_id=? FOR UPDATE",operation);var r=s.executeQuery()){
                if(!r.next()||!r.getString(1).equals(fingerprint))throw fail("IDEMPOTENCY_CONFLICT","操作标识已对应其他资金请求。");
                if(!"EXECUTING".equals(r.getString(2)))return decode(r.getString(3));
            }
            Savepoint before=c.setSavepoint();Map<String,Object> receipt;
            try{receipt=perform(c,operation,command,fields,changed);receipt.put("status","COMPLETED");}
            catch(FundsFailure rejected){c.rollback(before);changed.clear();receipt=new LinkedHashMap<>();receipt.put("status","FAILED");receipt.put("code",rejected.code());receipt.put("message",rejected.getMessage());}
            receipt.put("operationId",operation);receipt.put("committedAt",Instant.now().toString());
            exec(c,"UPDATE "+prefix+"operations SET state=?,result=?,committed_at=UTC_TIMESTAMP(6) WHERE operation_id=?",receipt.get("status"),JSON.toJson(receipt),operation);return receipt;
        });
        if(!changed.isEmpty())try{committed.accept(Set.copyOf(changed));}catch(Throwable ignored){/* committed ledger stays authoritative; caches are never a read source */}
        if("FAILED".equals(result.get("status")))throw fail((String)result.get("code"),(String)result.get("message"));return result;
    }
    public Map<String,Object> operation(String id){initialize();identifier(id);return transaction(c->{try(var s=statement(c,"SELECT state,result FROM "+prefix+"operations WHERE operation_id=?",id);var r=s.executeQuery()){return r.next()?Map.of("state",r.getString(1),"result",decode(r.getString(2))):Map.of("state","NOT_FOUND");}});}
    public Map<String,Object> hold(String ref,String business){initialize();identifier(ref);identifier(business);return transaction(c->holdReceipt(hold(c,ref,business,false),ZERO,null,null));}
    public BigDecimal nativeChange(UUID id,BigDecimal amount,Boolean add,String type,String command){
        Map<String,Object> input=new LinkedHashMap<>();input.put("playerUuid",id.toString());input.put("amount",money(amount,true).toPlainString());input.put("mode",add==null?"SET":add?"ADD":"SUBTRACT");input.put("nativeType",bounded(type,50));input.put("nativeCommand",bounded(command,255));
        return new BigDecimal((String)execute("native_"+UUID.randomUUID(),"native.change",input).get("balance"));
    }
    public void nativeTransfer(UUID from,UUID to,BigDecimal debit,BigDecimal credit,String command){
        execute("pay_"+UUID.randomUUID(),"native.pay",Map.of("fromUuid",from.toString(),"toUuid",to.toString(),"amount",money(credit,false).toPlainString(),"debitAmount",money(debit,false).toPlainString(),"nativeCommand",bounded(command,255)));
    }
    public void nativeBulk(Collection<UUID> uuids,BigDecimal amount,Boolean add,String type,String command){
        // Administrative all-player changes are a single transaction and skip
        // protected system accounts; a failed balance check rolls everything back.
        initialize();var changed=new HashSet<UUID>();transaction(c->{
            var names=new TreeSet<String>();if(uuids!=null){uuids.forEach(id->names.add(id.toString()));}else{try(var s=statement(c,"SELECT UID FROM "+accounts+" ORDER BY UID");var r=s.executeQuery()){while(r.next())names.add(r.getString(1));}}
            Map<String,Account> locked=lockAccounts(c,names);String op="bulk_"+UUID.randomUUID();
            for(Account a:locked.values()) {if(isSystem(c,a))continue;BigDecimal after=add==null?amount:add?a.balance.add(amount):a.balance.subtract(amount);setBalance(c,a,money(after,true));ledger(c,op,a,after,op,null,"NATIVE_BULK");changed.add(UUID.fromString(a.id));}
            exec(c,"INSERT INTO "+prefix+"operations VALUES(?,?,?,'COMPLETED',?,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))",op,sha(op),"native.bulk",JSON.toJson(Map.of("operationId",op,"type",bounded(type,50),"command",bounded(command,255),"status","COMPLETED")));return null;
        });try{committed.accept(Set.copyOf(changed));}catch(Throwable ignored){}
    }
    private Map<String,Object> perform(Connection c,String op,String command,Map<String,Object> p,Set<UUID> changed)throws SQLException{
        if(command.equals("native.change")){
            fields(p,"playerUuid","amount","mode","nativeType","nativeCommand");Account a=account(c,uuid(string(p,"playerUuid")),true);ordinary(c,a);BigDecimal amount=amount(p,true);String mode=string(p,"mode");
            BigDecimal after=switch(mode){case "ADD"->a.balance.add(amount);case "SUBTRACT"->a.balance.subtract(amount);case "SET"->amount;default->throw fail("INVALID_REQUEST","无效经济操作。");};
            if(after.signum()<0)throw fail("INSUFFICIENT_BALANCE","可用余额不足。");after=money(after,true);setBalance(c,a,after);ledger(c,op,a,after,op,null,"NATIVE_"+mode);changed.add(UUID.fromString(a.id));return new LinkedHashMap<>(Map.of("amount",amount.toPlainString(),"balance",after.toPlainString(),"playerUuid",a.id,"currency","CREDIT"));
        }
        if(command.equals("wallet.transfer")||command.equals("native.pay")){
            if(command.equals("wallet.transfer"))fields(p,"fromUuid","toUuid","amount");else fields(p,"fromUuid","toUuid","amount","debitAmount","nativeCommand");
            String from=uuid(string(p,"fromUuid")),to=uuid(string(p,"toUuid"));if(from.equals(to))throw fail("INVALID_REQUEST","不能给自己转账。");
            BigDecimal amount=amount(p,false),debit=command.equals("native.pay")?money(new BigDecimal(string(p,"debitAmount")),false):amount;
            if(debit.compareTo(amount)<0)throw fail("INVALID_REQUEST","税后扣款不能低于到账金额。");
            var locked=lockAccounts(c,Set.of(from,to));ordinary(c,locked.get(from));ordinary(c,locked.get(to));move(c,op,locked.get(from),locked.get(to),debit,amount,op,null,"TRANSFER",changed);
            return new LinkedHashMap<>(Map.of("businessRef",op,"amount",amount.toPlainString(),"currency","CREDIT","payerUuid",from,"payeeUuid",to));
        }
        if(!command.startsWith("wallet.escrow."))throw fail("COMMAND_NOT_ALLOWED","不允许此资金操作。");
        String ref=identifier(string(p,"escrowRef")),business=identifier(string(p,"businessRef"));String pool=requiredSystem(c,"DaoYu");
        if(command.equals("wallet.escrow.reserve")){
            fields(p,"escrowRef","businessRef","businessType","payerUuid","payeeUuid","amount","currency");currency(p);BigDecimal amount=amount(p,false);String payer=uuid(string(p,"payerUuid")),kind=string(p,"businessType"),payee=optional(p,"payeeUuid");
            if(!Set.of("OFFICIAL_STORE","MARKET_ORDER","COMMISSION").contains(kind))throw fail("INVALID_REQUEST","无效托管业务类型。");
            if(kind.equals("OFFICIAL_STORE")){String official=requiredSystem(c,"DIMA");if(payee!=null&&!payee.equals(official))throw fail("PAYEE_CONFLICT","官方收入必须结算到 DIMA。");payee=official;}
            if(payee!=null)payee=uuid(payee);if(kind.equals("MARKET_ORDER")&&payee==null)throw fail("INVALID_REQUEST","市场订单必须冻结卖家。");if(payer.equals(payee))throw fail("INVALID_REQUEST","付款人与收款人不能相同。");
            exec(c,"INSERT INTO "+prefix+"holds VALUES(?,?,?,?,?,?,0,0,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6)) ON DUPLICATE KEY UPDATE escrow_ref=escrow_ref",ref,business,kind,payer,payee,amount);
            Hold existing=hold(c,ref,business,true);
            if(!existing.kind.equals(kind)||!existing.payer.equals(payer)||!Objects.equals(existing.payee,payee)||existing.reserved.compareTo(amount)!=0)throw fail("ESCROW_CONFLICT","该担保标识已对应其他冻结快照。");
            // A new operation may not reserve the same business a second time.
            try(var s=statement(c,"SELECT 1 FROM "+prefix+"ledger WHERE escrow_ref=? LIMIT 1",ref);var r=s.executeQuery()){if(r.next())throw fail("ESCROW_ALREADY_RESERVED","已预付，请查询原操作。");}
            Set<String> ids=new HashSet<>(List.of(payer,pool));if(payee!=null)ids.add(payee);var locked=lockAccounts(c,ids);ordinary(c,locked.get(payer));if(payee!=null&&!kind.equals("OFFICIAL_STORE"))ordinary(c,locked.get(payee));
            assertPool(c,locked.get(pool),ref);move(c,op,locked.get(payer),locked.get(pool),amount,amount,business,ref,kind+"_RESERVE",changed);assertPool(c,account(c,pool,false),null);return holdReceipt(existing,amount,payer,pool);
        }
        Hold held=hold(c,ref,business,true);
        if(command.equals("wallet.escrow.bind")){
            fields(p,"escrowRef","businessRef","payeeUuid");String payee=uuid(string(p,"payeeUuid"));if(!held.kind.equals("COMMISSION"))throw fail("PAYEE_LOCKED","此业务不允许后置绑定收款人。");if(held.payee!=null&&!held.payee.equals(payee))throw fail("PAYEE_LOCKED","收款人已冻结，不能替换。");if(held.payer.equals(payee))throw fail("INVALID_REQUEST","不能接取自己的委托。");
            if(held.remaining().signum()<=0)throw fail("ESCROW_CLOSED","担保款已经处理完毕。");ordinary(c,account(c,payee,true));exec(c,"UPDATE "+prefix+"holds SET payee_uuid=?,updated_at=UTC_TIMESTAMP(6) WHERE escrow_ref=?",payee,ref);return holdReceipt(hold(c,ref,business,false),ZERO,null,payee);
        }
        if(!Set.of("wallet.escrow.settle","wallet.escrow.refund").contains(command))throw fail("COMMAND_NOT_ALLOWED","无效担保操作。");
        fields(p,"escrowRef","businessRef","amount","currency");currency(p);BigDecimal amount=amount(p,false);if(amount.compareTo(held.remaining())>0)throw fail("ESCROW_AMOUNT_EXCEEDED","金额超过剩余担保款。");
        boolean refund=command.endsWith("refund");String payee=refund?held.payer:held.payee;if(payee==null)throw fail("PAYEE_NOT_BOUND","委托尚未锁定接取人。");
        var locked=lockAccounts(c,Set.of(pool,payee));if(refund||!held.kind.equals("OFFICIAL_STORE"))ordinary(c,locked.get(payee));
        assertPool(c,locked.get(pool),null);move(c,op,locked.get(pool),locked.get(payee),amount,amount,business,ref,refund?"REFUND":held.kind+"_SETTLE",changed);
        exec(c,"UPDATE "+prefix+"holds SET "+(refund?"refunded":"settled")+"="+(refund?"refunded":"settled")+"+?,updated_at=UTC_TIMESTAMP(6) WHERE escrow_ref=?",amount,ref);
        assertPool(c,account(c,pool,false),null);return holdReceipt(hold(c,ref,business,false),amount,pool,payee);
    }
    private record Account(String id,String name,BigDecimal balance){}
    private record Hold(String ref,String business,String kind,String payer,String payee,BigDecimal reserved,BigDecimal settled,BigDecimal refunded){BigDecimal remaining(){return reserved.subtract(settled).subtract(refunded);}}
    private Hold hold(Connection c,String ref,String business,boolean lock)throws SQLException{
        try(var s=statement(c,"SELECT * FROM "+prefix+"holds WHERE escrow_ref=?"+(lock?" FOR UPDATE":""),ref);var r=s.executeQuery()){
            if(!r.next())throw fail("ESCROW_NOT_FOUND","担保记录不存在。");if(!business.equals(r.getString("business_ref")))throw fail("ESCROW_CONFLICT","担保记录与业务不一致。");
            return new Hold(ref,business,r.getString("business_type"),r.getString("payer_uuid"),r.getString("payee_uuid"),r.getBigDecimal("reserved"),r.getBigDecimal("settled"),r.getBigDecimal("refunded"));
        }
    }
    private Map<String,Object> holdReceipt(Hold h,BigDecimal amount,String from,String to){
        var result=new LinkedHashMap<String,Object>();result.put("businessRef",h.business);result.put("businessType",h.kind);result.put("escrowRef",h.ref);result.put("payerUuid",h.payer);result.put("payeeUuid",h.payee);result.put("fromUuid",from);result.put("toUuid",to);result.put("currency","CREDIT");result.put("amount",amount.toPlainString());result.put("reservedAmount",h.reserved.toPlainString());result.put("settledAmount",h.settled.toPlainString());result.put("refundedAmount",h.refunded.toPlainString());result.put("heldAmount",h.remaining().toPlainString());result.put("escrowStatus",h.remaining().signum()==0?"CLOSED":"HELD");return result;
    }
    private Account account(Connection c,String id,boolean lock)throws SQLException{
        try(var s=statement(c,"SELECT UID,player,balance FROM "+accounts+" WHERE UID=?"+(lock?" FOR UPDATE":""),id);var r=s.executeQuery()){
            if(!r.next())throw fail("ECONOMY_ACCOUNT_MISSING","游戏经济账号尚未建立。");return new Account(r.getString(1),r.getString(2),money(r.getBigDecimal(3),true));
        }
    }
    private Map<String,Account> lockAccounts(Connection c,Collection<String> ids)throws SQLException{var map=new LinkedHashMap<String,Account>();for(String id:new TreeSet<>(ids))map.put(id,account(c,id,true));return map;}
    private void move(Connection c,String op,Account from,Account to,BigDecimal debit,BigDecimal credit,String business,String escrow,String kind,Set<UUID> changed)throws SQLException{
        if(from.balance.compareTo(debit)<0)throw fail("INSUFFICIENT_BALANCE","可用余额不足。");BigDecimal out=money(from.balance.subtract(debit),true),in=money(to.balance.add(credit),true);
        setBalance(c,from,out);setBalance(c,to,in);ledger(c,op,from,out,business,escrow,kind);ledger(c,op,to,in,business,escrow,kind);changed.add(UUID.fromString(from.id));changed.add(UUID.fromString(to.id));
    }
    private void setBalance(Connection c,Account a,BigDecimal after)throws SQLException{
        if(after.compareTo(balanceLimit)>0)throw fail("MAX_BALANCE_EXCEEDED","收款余额超过经济配置上限。");
        if(after.compareTo(a.balance)==0)return;
        if(exec(c,"UPDATE "+accounts+" SET balance=? WHERE UID=?",after,a.id)!=1)throw new SQLException("account disappeared");
        // Existing XConomy columns may be DOUBLE(20,2). Verify exact cents after
        // every write before committing instead of assuming floating conversion.
        if(account(c,a.id,false).balance.compareTo(after)!=0)throw fail("ECONOMY_PRECISION_UNSUPPORTED","经济库不能无损保存该金额，操作已回滚。");
    }
    private void ledger(Connection c,String op,Account a,BigDecimal after,String business,String escrow,String kind)throws SQLException{
        exec(c,"INSERT INTO "+prefix+"ledger(operation_id,player_uuid,business_ref,escrow_ref,business_type,delta,before_balance,after_balance,created_at) VALUES(?,?,?,?,?,?,?,?,UTC_TIMESTAMP(6))",op,a.id,business,escrow,kind,after.subtract(a.balance),a.balance,after);
    }
    private void assertPool(Connection c,Account pool,String exclude)throws SQLException{
        String filter=exclude==null?"":" WHERE escrow_ref<>?";
        try(var s=statement(c,"SELECT COALESCE(SUM(reserved-settled-refunded),0) FROM "+prefix+"holds"+filter,exclude==null?new Object[]{}:new Object[]{exclude});var r=s.executeQuery()){
            if(!r.next()||r.getBigDecimal(1).compareTo(pool.balance)!=0)throw fail("ESCROW_LEDGER_MISMATCH","DaoYu 余额与担保账本不一致，已暂停资金操作。");
        }
    }
    public boolean isSystemIdentity(UUID id,String name){if(name!=null&&(name.equalsIgnoreCase("DIMA")||name.equalsIgnoreCase("DaoYu")))return true;initialize();return transaction(c->id.toString().equals(system(c,"DIMA"))||id.toString().equals(system(c,"DaoYu")));}
    private String system(Connection c,String name)throws SQLException{try(var s=statement(c,"SELECT player_uuid FROM "+prefix+"system_accounts WHERE account_key=?",name.toLowerCase(Locale.ROOT));var r=s.executeQuery()){return r.next()?r.getString(1):null;}}
    private String requiredSystem(Connection c,String name)throws SQLException{String id=system(c,name);if(id==null)throw fail("SYSTEM_ACCOUNTS_NOT_INITIALIZED","请先由控制台初始化 DIMA、DaoYu。");return id;}
    private boolean isSystem(Connection c,Account a)throws SQLException{return a.name.equalsIgnoreCase("DIMA")||a.name.equalsIgnoreCase("DaoYu")||a.id.equals(system(c,"DIMA"))||a.id.equals(system(c,"DaoYu"));}
    private void ordinary(Connection c,Account a)throws SQLException{if(isSystem(c,a))throw fail("SYSTEM_ACCOUNT_PROTECTED","系统资金账号不能通过普通经济操作使用。");}
    public void deleteAccount(UUID id,String aliasTable){
        if(aliasTable!=null&&!aliasTable.matches("[A-Za-z0-9_]{1,40}"))throw fail("INVALID_CONFIGURATION","无效经济映射表。");
        initialize();transaction(c->{
            if(id.toString().equals(system(c,"DIMA"))||id.toString().equals(system(c,"DaoYu")))throw fail("SYSTEM_ACCOUNT_PROTECTED","系统账号不可删除。");
            try(var s=statement(c,"SELECT UID FROM "+accounts+" WHERE UID=? FOR UPDATE",id.toString());var r=s.executeQuery()){if(!r.next())return null;}
            Account a=account(c,id.toString(),true);ordinary(c,a);
            try(var s=statement(c,"SELECT 1 FROM "+prefix+"holds WHERE (payer_uuid=? OR payee_uuid=?) AND reserved>settled+refunded LIMIT 1",id.toString(),id.toString());var r=s.executeQuery()){if(r.next())throw fail("ESCROW_ACTIVE","账号仍关联未结清担保款。");}
            String op="delete_"+UUID.randomUUID();ledger(c,op,a,ZERO,op,null,"NATIVE_DELETE");exec(c,"DELETE FROM "+accounts+" WHERE UID=?",id.toString());if(aliasTable!=null)exec(c,"DELETE FROM "+aliasTable+" WHERE DUUID=?",id.toString());return null;
        });try{committed.accept(Set.of(id));}catch(Throwable ignored){}
    }
    private static BigDecimal amount(Map<String,Object> p,boolean zero){String text=string(p,"amount");if(!text.matches("[0-9]{1,13}(\\.[0-9]{1,2})?"))throw fail("INVALID_REQUEST","金额必须为最多两位小数的十进制数字。");try{return money(new BigDecimal(text),zero);}catch(NumberFormatException e){throw fail("INVALID_REQUEST","无效金额。");}}
    private static BigDecimal money(BigDecimal value,boolean zero){try{if(value==null||value.precision()>32||value.scale()<-2||value.scale()>16)throw new ArithmeticException();value=value.setScale(2,RoundingMode.UNNECESSARY);}catch(Exception e){throw fail("INVALID_REQUEST","金额最多两位小数。");}if(value.signum()<0||(!zero&&value.signum()==0)||value.compareTo(MAX)>0)throw fail("INVALID_AMOUNT","金额超出受控范围。");return value;}
    private static String string(Map<String,Object> p,String key){Object v=p.get(key);if(!(v instanceof String text)||text.isEmpty()||text.length()>512)throw fail("INVALID_REQUEST","缺少或无效字段："+key);return text;}
    private static String optional(Map<String,Object> p,String key){return p.get(key)==null?null:string(p,key);}
    private static String uuid(String value){try{String id=UUID.fromString(value).toString();if(!id.equals(value))throw new IllegalArgumentException();return id;}catch(Exception e){throw fail("INVALID_REQUEST","无效 UUID。");}}
    private static String identifier(String value){if(!value.matches("[A-Za-z0-9][A-Za-z0-9_:.\\-]{0,127}"))throw fail("INVALID_REQUEST","无效操作或业务标识。");return value;}
    private static void currency(Map<String,Object> p){if(!"CREDIT".equals(string(p,"currency")))throw fail("INVALID_REQUEST","币种必须为 CREDIT。");}
    private static void fields(Map<String,Object> p,String...allowed){Set<String> keys=Set.of(allowed);if(!keys.containsAll(p.keySet()))throw fail("INVALID_REQUEST","存在未知资金字段。");}
    private static String bounded(String text,int limit){if(text==null)return "";return text.substring(0,Math.min(text.length(),limit));}
    private static String sha(String value){try{return HexFormat.of().formatHex(MessageDigest.getInstance("SHA-256").digest(value.getBytes(StandardCharsets.UTF_8)));}catch(Exception impossible){throw new IllegalStateException(impossible);}}
    private static Map<String,Object> decode(String raw){return JSON.fromJson(raw,MAP);}
    @FunctionalInterface private interface Task<T>{T run(Connection c)throws SQLException;}
    private <T>T transaction(Task<T> work){
        for(int attempt=0;;attempt++)try(Connection c=connections.open()){
            if(c==null)throw new SQLException("no connection");c.setTransactionIsolation(Connection.TRANSACTION_READ_COMMITTED);c.setAutoCommit(false);
            try{T result=work.run(c);c.commit();return result;}
            catch(SQLException|RuntimeException error){try{c.rollback();}catch(SQLException rollback){error.addSuppressed(rollback);}throw error;}
        }catch(SQLException error){if(attempt<2&&("40001".equals(error.getSQLState())||error.getErrorCode()==1213))continue;throw fail("RESULT_UNKNOWN","经济事务结果无法确认；请查询原操作。",error);}
    }
    private static PreparedStatement statement(Connection c,String sql,Object...args)throws SQLException{PreparedStatement s=c.prepareStatement(sql);s.setQueryTimeout(8);for(int i=0;i<args.length;i++)s.setObject(i+1,args[i]);return s;}
    private static int exec(Connection c,String sql,Object...args)throws SQLException{try(var s=statement(c,sql,args)){return s.executeUpdate();}}
    private static FundsFailure fail(String code,String message){return new FundsFailure(code,message);}
    private static FundsFailure fail(String code,String message,Throwable cause){return new FundsFailure(code,message,cause);}
}
