package me.yic.xconomy.deuterium;

import org.junit.jupiter.api.*;
import java.lang.reflect.*;
import java.math.BigDecimal;
import java.sql.*;
import java.util.*;
import java.util.concurrent.*;
import java.util.concurrent.atomic.*;
import static org.junit.jupiter.api.Assertions.*;

final class FundsEngineTest {
    String url,password,schema;FundsEngine engine;String payer=UUID.randomUUID().toString(),payee=UUID.randomUUID().toString(),third=UUID.randomUUID().toString();
    @BeforeEach void setup()throws Exception{
        url=System.getenv("DEUTERIUM_FUNDS_TEST_URL");password=System.getenv("DEUTERIUM_FUNDS_TEST_PASSWORD");
        Assumptions.assumeTrue(url!=null,"explicit loopback database required for funds integration");
        assertTrue(url.matches("jdbc:mariadb://127\\.0\\.0\\.1:[0-9]+/"),"tests must use loopback and no existing schema");
        schema="dc_funds_test_"+UUID.randomUUID().toString().replace("-","");
        try(Connection c=DriverManager.getConnection(url,"root",password)){c.createStatement().executeUpdate("CREATE DATABASE "+schema);}
        url+=schema;
        try(Connection c=connection()){
            c.createStatement().executeUpdate("CREATE TABLE xconomy (UID VARCHAR(50) PRIMARY KEY,player VARCHAR(50),balance DOUBLE(20,2),hidden INT) ENGINE=InnoDB");
            insert(c,payer,"Payer","1000.00");insert(c,payee,"Payee","0.00");insert(c,third,"Third","0.00");
        }
        engine=new FundsEngine(this::connection,"xconomy",ignored->{});engine.initializeSystemAccounts();
    }
    @AfterEach void cleanup()throws Exception{if(schema!=null){assertTrue(schema.startsWith("dc_funds_test_"));try(Connection c=connection()){c.createStatement().executeUpdate("DROP DATABASE "+schema);}}}
    Connection connection()throws SQLException{return DriverManager.getConnection(url,"root",password);}
    static void insert(Connection c,String id,String name,String balance)throws Exception{try(var s=c.prepareStatement("INSERT INTO xconomy VALUES(?,?,?,0)")){s.setString(1,id);s.setString(2,name);s.setBigDecimal(3,new BigDecimal(balance));s.executeUpdate();}}
    BigDecimal value(String id)throws Exception{try(Connection c=connection();var s=c.prepareStatement("SELECT balance FROM xconomy WHERE UID=?")){s.setString(1,id);try(var r=s.executeQuery()){assertTrue(r.next());return r.getBigDecimal(1).setScale(2);}}}
    int count(String table)throws Exception{try(Connection c=connection();var r=c.createStatement().executeQuery("SELECT COUNT(*) FROM "+table)){r.next();return r.getInt(1);}}
    Map<String,Object> transfer(String from,String to,String amount){return Map.of("fromUuid",from,"toUuid",to,"amount",amount);}
    Map<String,Object> ledgerQuery(String player) {
        return new LinkedHashMap<>(Map.of("playerUuid",player,"from","2020-01-01T00:00:00Z","to","2099-01-01T00:00:00Z","direction","","businessType","","beforeTime","","beforeSequence","0","snapshot","0","limit","25","recordId",""));
    }
    @SuppressWarnings("unchecked") List<Map<String,Object>> ledgerRows(Map<String,Object> query){return (List<Map<String,Object>>)engine.records(query).get("records");}
    @Test void ledgerIncludesNativeAndAppMoneyWithoutDuplicateOrFailedPayments() throws Exception {
        engine.nativeChange(UUID.fromString(payer),new BigDecimal("5.00"),false,"VAULT","purchase");
        engine.nativeTransfer(UUID.fromString(payer),UUID.fromString(payee),new BigDecimal("11.00"),new BigDecimal("10.00"),"pay");
        var request=transfer(payer,payee,"3.00");engine.execute("app_transfer","wallet.transfer",request);engine.execute("app_transfer","wallet.transfer",request);
        assertThrows(FundsFailure.class,()->engine.execute("failed_transfer","wallet.transfer",transfer(payee,payer,"100.00")));
        var rows=ledgerRows(ledgerQuery(payer));assertEquals(3,rows.size());
        assertEquals("APP",rows.get(0).get("source"));assertEquals("3.00",rows.get(0).get("amount"));
        assertEquals("GAME",rows.get(1).get("source"));assertEquals("11.00",rows.get(1).get("amount"));assertEquals(payee,rows.get(1).get("otherUuid"));
        assertEquals("GAME",rows.get(2).get("source"));assertEquals("5.00",rows.get(2).get("amount"));
        var summary=(Map<?,?>)engine.balance(UUID.fromString(payer)).get("today");assertEquals("19.00",summary.get("expense"));assertEquals("0.00",summary.get("income"));
        var other=ledgerRows(ledgerQuery(payee));assertEquals(2,other.size());assertEquals("income",other.get(0).get("direction"));
        var game=ledgerQuery(payer);game.put("businessType","GAME");assertEquals(2,ledgerRows(game).size());game.put("businessType","TRANSFER");assertEquals(1,ledgerRows(game).size());
        var all=ledgerRows(ledgerQuery(""));assertEquals(5,all.size());
    }
    @Test void ledgerPagesUseTimeAndSequenceAndExcludeNewWrites() throws Exception {
        for(int n=0;n<4;n++)engine.nativeChange(UUID.fromString(payer),new BigDecimal("1.00"),true,"TEST","deposit");
        try(var c=connection()){c.createStatement().executeUpdate("UPDATE xconomy_dc_ledger SET created_at='2026-09-08 12:00:00.000000'");}
        var query=ledgerQuery(payer);query.put("limit","2");var first=engine.records(query);
        @SuppressWarnings("unchecked") var rows=(List<Map<String,Object>>)first.get("records");assertEquals(3,rows.size());assertEquals("4",rows.get(0).get("sequence"));
        query.put("snapshot",first.get("snapshot"));query.put("beforeTime",rows.get(1).get("occurredAt"));query.put("beforeSequence",rows.get(1).get("sequence"));
        engine.nativeChange(UUID.fromString(payer),new BigDecimal("1.00"),true,"TEST","new deposit");
        var second=ledgerRows(query);assertEquals(List.of("2","1"),second.stream().map(row->row.get("sequence")).toList());
        var detail=ledgerQuery(payee);detail.put("recordId","1");assertTrue(ledgerRows(detail).isEmpty());
        var invalid=ledgerQuery(payer);invalid.put("direction","DROP");assertEquals("INVALID_REQUEST",assertThrows(FundsFailure.class,()->engine.records(invalid)).code());
    }
    Map<String,Object> reserve(String hold,String kind,String amount,String recipient){var m=new HashMap<String,Object>();m.put("escrowRef",hold);m.put("businessRef","business_"+hold);m.put("businessType",kind);m.put("amount",amount);m.put("currency","CREDIT");m.put("payerUuid",payer);if(recipient!=null)m.put("payeeUuid",recipient);return m;}
    Map<String,Object> release(String hold,String amount){return Map.of("escrowRef",hold,"businessRef","business_"+hold,"amount",amount,"currency","CREDIT");}
    String system(String name){return (String)((Map<?,?>)engine.initializeSystemAccounts().get(name)).get("playerUuid");}
    @Test void transferIsAtomicIdempotentAndRejectsChangedOrFailedReplay()throws Exception{
        var payload=transfer(payer,payee,"12.30");var first=engine.execute("operation_1","wallet.transfer",payload);assertEquals(first,engine.execute("operation_1","wallet.transfer",payload));
        assertEquals(new BigDecimal("987.70"),value(payer));assertEquals(new BigDecimal("12.30"),value(payee));assertEquals(2,count("xconomy_dc_ledger"));
        assertEquals("IDEMPOTENCY_CONFLICT",assertThrows(FundsFailure.class,()->engine.execute("operation_1","wallet.transfer",transfer(payer,payee,"12.31"))).code());
        assertEquals("INSUFFICIENT_BALANCE",assertThrows(FundsFailure.class,()->engine.execute("failure_1","wallet.transfer",transfer(payee,payer,"99.00"))).code());
        engine.nativeChange(UUID.fromString(payee),new BigDecimal("500.00"),true,"TEST","fund");
        assertEquals("INSUFFICIENT_BALANCE",assertThrows(FundsFailure.class,()->engine.execute("failure_1","wallet.transfer",transfer(payee,payer,"99.00"))).code());
        assertEquals(new BigDecimal("512.30"),value(payee));
    }
    @Test void officialReserveSettlementAndRefundConserveRealServiceBalances()throws Exception{
        String pool=system("DaoYu"),official=system("DIMA");assertEquals(ZERO,value(pool));assertEquals(ZERO,value(official));
        var held=engine.execute("reserve_official","wallet.escrow.reserve",reserve("hold_official","OFFICIAL_STORE","100.00",null));assertEquals(official,held.get("payeeUuid"));assertEquals(new BigDecimal("900.00"),value(payer));assertEquals(new BigDecimal("100.00"),value(pool));
        engine.execute("settle_partial","wallet.escrow.settle",release("hold_official","70.00"));engine.execute("refund_partial","wallet.escrow.refund",release("hold_official","30.00"));
        assertEquals(ZERO,value(pool));assertEquals(new BigDecimal("70.00"),value(official));assertEquals(new BigDecimal("930.00"),value(payer));
        assertEquals("ESCROW_AMOUNT_EXCEEDED",assertThrows(FundsFailure.class,()->engine.execute("over_refund","wallet.escrow.refund",release("hold_official","1.00"))).code());
        for(String id:List.of(pool,official))assertEquals("SYSTEM_ACCOUNT_PROTECTED",assertThrows(FundsFailure.class,()->engine.nativeChange(UUID.fromString(id),new BigDecimal("1"),true,"ADMIN","give")).code());
    }
    @Test void competingSettlementRefundAndPayeeBindingAreBounded()throws Exception{
        engine.execute("reserve_commission","wallet.escrow.reserve",reserve("hold_job","COMMISSION","100.00",null));
        engine.execute("bind_1","wallet.escrow.bind",Map.of("escrowRef","hold_job","businessRef","business_hold_job","payeeUuid",payee));
        assertEquals("PAYEE_LOCKED",assertThrows(FundsFailure.class,()->engine.execute("bind_2","wallet.escrow.bind",Map.of("escrowRef","hold_job","businessRef","business_hold_job","payeeUuid",third))).code());
        var start=new CountDownLatch(1);var successes=new AtomicInteger();
        try(var pool=Executors.newFixedThreadPool(2)){var futures=new ArrayList<Future<?>>();for(String action:List.of("settle","refund")){futures.add(pool.submit(()->{start.await();try{engine.execute("race_"+action,"wallet.escrow."+action,release("hold_job","80.00"));successes.incrementAndGet();}catch(FundsFailure e){assertEquals("ESCROW_AMOUNT_EXCEEDED",e.code());}return null;}));}start.countDown();for(var f:futures)f.get(15,TimeUnit.SECONDS);}
        assertEquals(1,successes.get());assertEquals("20.00",engine.hold("hold_job","business_hold_job").get("heldAmount"));assertEquals(new BigDecimal("20.00"),value(system("DaoYu")));
    }
    @Test void twoNodeNativeVaultPayAndCoreWritesDoNotUseStaleCacheOrOverdraw()throws Exception{
        FundsEngine otherNode=new FundsEngine(this::connection,"xconomy",ignored->{});otherNode.initialize();var start=new CountDownLatch(1);var ok=new AtomicInteger();
        try(var pool=Executors.newFixedThreadPool(12)){var futures=new ArrayList<Future<?>>();for(int i=0;i<60;i++){int n=i;futures.add(pool.submit(()->{start.await();FundsEngine node=(n%2==0?engine:otherNode);try{
            if(n%3==0)node.nativeChange(UUID.fromString(payer),new BigDecimal("30.00"),false,"PLUGIN","withdraw");
            else if(n%3==1)node.nativeTransfer(UUID.fromString(payer),UUID.fromString(payee),new BigDecimal("30.00"),new BigDecimal("30.00"),"pay");
            else node.execute("parallel_"+n,"wallet.transfer",transfer(payer,third,"30.00"));ok.incrementAndGet();
        }catch(FundsFailure rejected){assertEquals("INSUFFICIENT_BALANCE",rejected.code());}return null;}));}start.countDown();for(var f:futures)f.get(30,TimeUnit.SECONDS);}
        assertEquals(33,ok.get());assertEquals(new BigDecimal("10.00"),value(payer));
        // Legacy stale pd.balance is deliberately absent from all write inputs.
        engine.nativeChange(UUID.fromString(payer),new BigDecimal("5.00"),true,"PLUGIN","deposit");otherNode.nativeTransfer(UUID.fromString(payee),UUID.fromString(payer),new BigDecimal("1.00"),new BigDecimal("1.00"),"pay");assertEquals(new BigDecimal("16.00"),value(payer));
    }
    @Test void interruptedCommitIsRecoveredByOriginalOperationWithoutSecondDebit()throws Exception{
        AtomicBoolean lose=new AtomicBoolean(true);
        FundsEngine faulty=new FundsEngine(()->{Connection delegate=connection();return (Connection)Proxy.newProxyInstance(getClass().getClassLoader(),new Class<?>[]{Connection.class},(proxy,method,args)->{try{Object result=method.invoke(delegate,args);if(method.getName().equals("commit")&&lose.compareAndSet(true,false))throw new SQLException("synthetic lost commit acknowledgement","08006");return result;}catch(InvocationTargetException e){throw e.getCause();}});},"xconomy",ignored->{});
        assertEquals("RESULT_UNKNOWN",assertThrows(FundsFailure.class,()->faulty.execute("commit_lost","wallet.transfer",transfer(payer,payee,"17.00"))).code());
        assertEquals("COMPLETED",engine.operation("commit_lost").get("state"));faulty.execute("commit_lost","wallet.transfer",transfer(payer,payee,"17.00"));assertEquals(new BigDecimal("983.00"),value(payer));assertEquals(new BigDecimal("17.00"),value(payee));assertEquals(2,count("xconomy_dc_ledger"));
    }
    @Test void secondLegSqlFailureRollsBackDebitAndReceipt()throws Exception{
        try(Connection c=connection()){c.createStatement().execute("CREATE TRIGGER reject_payee BEFORE UPDATE ON xconomy FOR EACH ROW BEGIN IF NEW.UID='"+payee+"' THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='synthetic second leg failure'; END IF; END");}
        assertThrows(FundsFailure.class,()->engine.execute("sql_fail","wallet.transfer",transfer(payer,payee,"10.00")));assertEquals(new BigDecimal("1000.00"),value(payer));assertEquals(ZERO,value(payee));assertEquals(0,count("xconomy_dc_ledger"));assertEquals("NOT_FOUND",engine.operation("sql_fail").get("state"));
        try(Connection c=connection()){c.createStatement().execute("DROP TRIGGER reject_payee");}engine.execute("sql_fail","wallet.transfer",transfer(payer,payee,"10.00"));assertEquals(new BigDecimal("990.00"),value(payer));
    }
    @Test void protectedIdentityCannotBeDeletedAndPoolMismatchStopsRelease()throws Exception{
        String pool=system("DaoYu");assertThrows(FundsFailure.class,()->engine.deleteAccount(UUID.fromString(pool),null));
        engine.execute("reserve_mismatch","wallet.escrow.reserve",reserve("hold_mismatch","MARKET_ORDER","100.00",payee));assertThrows(FundsFailure.class,()->engine.deleteAccount(UUID.fromString(payee),null));
        try(Connection c=connection();var s=c.prepareStatement("UPDATE xconomy SET balance=90 WHERE UID=?")){s.setString(1,pool);s.executeUpdate();}
        assertEquals("ESCROW_LEDGER_MISMATCH",assertThrows(FundsFailure.class,()->engine.execute("settle_mismatch","wallet.escrow.settle",release("hold_mismatch","10.00"))).code());assertEquals(ZERO,value(payee));
    }
    @Test void externalAmountsCannotUseExponentsOrChangeAProtectedAccount()throws Exception{
        for(String amount:List.of("1e100000000","NaN","-1","0.001","1000000000001.00"))assertThrows(FundsFailure.class,()->engine.execute("bad_"+UUID.randomUUID(),"wallet.transfer",transfer(payer,payee,amount)));
        assertEquals(new BigDecimal("1000.00"),value(payer));assertEquals(0,count("xconomy_dc_ledger"));
        assertTrue(ControlledEconomyAPI.protectedName("dImA"));assertTrue(ControlledEconomyAPI.protectedName("DAoYu"));
        assertFalse(ControlledEconomyAPI.protectedName("DimaPlayer"));
    }
    private static final BigDecimal ZERO=new BigDecimal("0.00");
}
