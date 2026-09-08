/*
 *  This file (DataCon.java) is a part of project XConomy
 *  Copyright (C) YiC and contributors
 *
 *  This program is free software: you can redistribute it and/or modify it
 *  under the terms of the GNU General Public License as published by the
 *  Free Software Foundation, either version 3 of the License, or
 *  (at your option) any later version.
 *
 *  This program is distributed in the hope that it will be useful, but
 *  WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
 *  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License
 *  for more details.
 *
 *  You should have received a copy of the GNU General Public License along
 *  with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */
package me.yic.xconomy.data;

import me.yic.xconomy.AdapterManager;
import me.yic.xconomy.XConomy;
import me.yic.xconomy.XConomyLoad;
import me.yic.xconomy.adapter.comp.CPlayer;
import me.yic.xconomy.adapter.comp.CallAPI;
import me.yic.xconomy.data.caches.Cache;
import me.yic.xconomy.data.caches.CacheNonPlayer;
import me.yic.xconomy.data.redis.RedisPublisher;
import me.yic.xconomy.data.syncdata.PlayerData;
import me.yic.xconomy.data.syncdata.SyncBalanceAll;
import me.yic.xconomy.data.syncdata.SyncData;
import me.yic.xconomy.data.syncdata.SyncDelData;
import me.yic.xconomy.info.MessageConfig;
import me.yic.xconomy.info.RecordInfo;
import me.yic.xconomy.info.SyncChannalType;
import me.yic.xconomy.utils.SendPluginMessage;

import java.math.BigDecimal;
import java.util.UUID;

public class DataCon {

    public static PlayerData getPlayerData(UUID uuid) {
        return getPlayerDatai(uuid);
    }

    public static PlayerData getPlayerData(String username) {
        return getPlayerDatai(username);
    }

    public static BigDecimal getAccountBalance(String account) {
        BigDecimal bal = null;
        if (XConomyLoad.Config.DISABLE_CACHE){
            return DataLink.getBalNonPlayer(account);
        }
        if (CacheNonPlayer.CacheContainsKey(account)) {
            bal = CacheNonPlayer.getBalanceFromCacheOrDB(account);
        }
        if (bal == null){
            bal =  DataLink.getBalNonPlayer(account);
        }
        return bal;
    }

    private static <T> PlayerData getPlayerDatai(T u) {
        // Deuterium: every read uses committed SQL; delayed Redis/cache payloads
        // may update display caches but can never become a balance authority.
        return DataLink.getPlayerData(u);
    }

    public static int getPlayerHiddenState(UUID uid) {
        if (Cache.phids.containsKey(uid)){
            return Cache.phids.get(uid);
        }
        if (DataLink.getPlayerData(uid) != null) {
            return Cache.phids.get(uid);
        }
        return 1;
    }
    public static void removePlayerHiddenState(UUID uid) {
        Cache.phids.remove(uid);
    }

    public static void deletePlayerData(PlayerData pd) {
        DataLink.deletePlayerData(pd.getUniqueId());
        Cache.removefromCache(pd.getUniqueId());

        if (!(pd instanceof SyncDelData) && XConomyLoad.getSyncData_Enable()) {
            SendMessTask(new SyncDelData(pd));
        }

        CPlayer cp = AdapterManager.PLUGIN.getplayer(pd);
        if(cp.isOnline()){
            cp.kickPlayer("[XConomy] " + AdapterManager.translateColorCodes(MessageConfig.DELETE_DATA));
        }
    }

    public static boolean hasaccountdatacache(String name) {
        return CacheNonPlayer.CacheContainsKey(name);
    }

    public static void deletedatafromcache(UUID u) {
        Cache.deleteDataFromCache(u);
    }

    public static boolean containinfieldslist(String name) {
        if (XConomyLoad.Config.NON_PLAYER_ACCOUNT_SUBSTRING != null) {
            for (String field : XConomyLoad.Config.NON_PLAYER_ACCOUNT_SUBSTRING) {
                if (name.contains(field)) {
                    return true;
                }
            }
        }
        return false;
    }

    public static BigDecimal changeplayerdata(final String type, final UUID uid, final BigDecimal amount, final Boolean isAdd, final String command, final Object comment) {
        BigDecimal after = me.yic.xconomy.deuterium.ControlledEconomyAPI.nativeChange(uid, amount, isAdd, type, command);
        // Account events are notifications, not cancellable transactions.
        try {
            PlayerData current = getPlayerData(uid);
            BigDecimal before = isAdd == null ? after : (isAdd ? after.subtract(amount) : after.add(amount));
            CallAPI.CallPlayerAccountEvent(uid, current.getName(), before, amount, isAdd, new RecordInfo(type, command, comment));
        } catch (Throwable notificationFailure) {
            XConomy.getInstance().getLogger().warning("Committed account notification failed; ledger remains authoritative.");
        }
        return after;
    }


    @SuppressWarnings("ConstantConditions")
    public static void changeaccountdata(final String type, final String u, final BigDecimal amount, final Boolean isAdd, final String command) {
        if (me.yic.xconomy.deuterium.ControlledEconomyAPI.protectedName(u)) throw new me.yic.xconomy.deuterium.FundsFailure("SYSTEM_ACCOUNT_PROTECTED", "系统账号不能作为普通非玩家账户使用。");
        BigDecimal newvalue = amount;
        BigDecimal balance = getAccountBalance(u);

        RecordInfo ri = new RecordInfo(type, command, null);

        CallAPI.CallNonPlayerAccountEvent(u, balance, amount, isAdd, type);
        if (isAdd != null) {
            if (isAdd) {
                newvalue = balance.add(amount);
            } else {
                newvalue = balance.subtract(amount);
            }
        }
        CacheNonPlayer.insertIntoCache(u, newvalue);

        if (XConomyLoad.DConfig.canasync && AdapterManager.checkisMainThread()) {
            final BigDecimal fnewvalue = newvalue;
            AdapterManager.runTaskAsynchronously(() -> DataLink.saveNonPlayer(u, amount, fnewvalue, isAdd, ri));
        } else {
            DataLink.saveNonPlayer(u, amount, newvalue, isAdd, ri);
        }
    }

    public static void changeallplayerdata(String targettype, String type, BigDecimal amount, Boolean isAdd, String command, StringBuilder comment) {
        java.util.Collection<UUID> players = targettype.equalsIgnoreCase("all") ? null : AdapterManager.PLUGIN.getOnlinePlayersUUIDs();
        me.yic.xconomy.deuterium.ControlledEconomyAPI.nativeBulk(players, amount, isAdd, type, command);
    }

    public static void baltop() {
        Cache.baltop.clear();
        Cache.baltop_papi.clear();
        sumbal();
        DataLink.getTopBal();
    }


    public static void sumbal() {
        Cache.sumbalance = DataFormat.formatString(DataLink.getBalSum());
    }


    public static void SendMessTask(SyncData pd) {
        if (XConomyLoad.Config.SYNCDATA_TYPE.equals(SyncChannalType.REDIS)) {
            RedisPublisher.publishmessage(pd.toByteArray(XConomy.syncversion).toByteArray());
        }else if (XConomyLoad.Config.SYNCDATA_TYPE.equals(SyncChannalType.BUNGEECORD)) {
            SendPluginMessage.SendMessTask("xconomy:acb", pd);
        }
    }

}
