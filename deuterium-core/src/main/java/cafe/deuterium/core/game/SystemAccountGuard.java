package cafe.deuterium.core.game;

import org.bukkit.event.*;
import org.bukkit.event.player.AsyncPlayerPreLoginEvent;
import org.bukkit.event.player.PlayerLoginEvent;

/** Reserved system identities have no human login path, including after rename. */
public final class SystemAccountGuard implements Listener {
    private final EconomyAccess economy;
    public SystemAccountGuard(EconomyAccess economy){this.economy=economy;}
    private boolean reserved(String name){return name.equalsIgnoreCase("DIMA")||name.equalsIgnoreCase("DaoYu");}
    @EventHandler(priority=EventPriority.HIGHEST) public void preLogin(AsyncPlayerPreLoginEvent event){
        try{
            if(reserved(event.getName())||(economy.available()&&economy.isSystemIdentity(event.getUniqueId(),event.getName())))
                event.disallow(AsyncPlayerPreLoginEvent.Result.KICK_OTHER,"该名称或身份为系统资金账号，不可登录。");
        }catch(Exception unavailable){event.disallow(AsyncPlayerPreLoginEvent.Result.KICK_OTHER,"账号保护校验暂不可用，请稍后登录。");}
    }
    @EventHandler(priority=EventPriority.HIGHEST) public void login(PlayerLoginEvent event){
        if(reserved(event.getPlayer().getName()))event.disallow(PlayerLoginEvent.Result.KICK_OTHER,"系统资金账号不可登录。");
    }
}
