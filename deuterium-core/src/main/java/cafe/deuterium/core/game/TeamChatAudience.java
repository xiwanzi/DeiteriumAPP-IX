package cafe.deuterium.core.game;

import java.lang.reflect.Method;
import java.util.*;

/** Read-only adapter to the deployed Deuterium Teams committed FTB snapshot. */
public final class TeamChatAudience {
    public record Team(String id,long revision,Set<UUID> members) {}
    private Method runtime;
    public Team resolve(UUID player) {
        try {
            if(runtime==null)runtime=Class.forName("com.deuteriumix.teams.Gateway").getMethod("runtime");
            Object service=runtime.invoke(null);
            if(service==null||!(boolean)service.getClass().getMethod("ready").invoke(service))throw new IllegalStateException("团队服务尚未同步完成");
            Object snapshot=service.getClass().getMethod("snapshot").invoke(service);
            Object team=snapshot.getClass().getMethod("membership",UUID.class).invoke(snapshot,player);
            if(team==null)throw new IllegalStateException("你还没有加入团队");
            Object id=team.getClass().getMethod("id").invoke(team);
            Map<?,?> ranks=(Map<?,?>)team.getClass().getMethod("ranks").invoke(team);
            Set<UUID> members=new HashSet<>();
            for(var member:ranks.entrySet())if((boolean)member.getValue().getClass().getMethod("member").invoke(member.getValue()))members.add((UUID)member.getKey());
            if(!(boolean)service.getClass().getMethod("ready").invoke(service)||service.getClass().getMethod("snapshot").invoke(service)!=snapshot)
                throw new IllegalStateException("团队成员正在更新，请稍后重试");
            return new Team(id.toString(),(long)snapshot.getClass().getMethod("revision").invoke(snapshot),Set.copyOf(members));
        }catch(IllegalStateException e){throw e;}catch(ReflectiveOperationException|LinkageError e){throw new IllegalStateException("现有团队服务暂不可用");}
    }
}
