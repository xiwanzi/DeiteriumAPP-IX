package cafe.deuterium.probe;

import cafe.deuterium.core.util.Checks;
import cafe.deuterium.core.util.Json;
import org.bukkit.Material;
import org.bukkit.entity.Player;
import org.bukkit.inventory.ItemStack;
import java.nio.charset.StandardCharsets;
import java.util.*;

/** Uses the real public Core adapter and the real player /dmail command in an isolated database. */
final class MailProbe {
    private final CoreProbe probe;
    MailProbe(CoreProbe probe){this.probe=probe;}
    void claim(boolean recovery)throws Exception {
        var runtime=probe.core().runtime();Player player=probe.player("CoreProbeUser");
        CoreProbe.check(runtime.config().storage().database().startsWith("dc_core_game_"),"isolated database required");
        CoreProbe.check(runtime.mailbox().capabilities().get("commerceReady").getAsBoolean(),"Mailbox and real Sync provider are not ready");
        ItemStack item=new ItemStack(Material.EMERALD);
        var version=runtime.catalog.save(runtime.codec.capture("deuterium:mail_probe_emerald",item,"Mail integration emerald",runtime.config()),"integration-probe");
        String order="probe_order_"+UUID.randomUUID(),delivery="probe_delivery_"+UUID.randomUUID(),operation="probe_create_"+UUID.randomUUID();
        String snapshot=Json.GSON.toJson(Map.of("schemaVersion",1,"orderId",order,"recipientUuid",player.getUniqueId().toString(),"inventoryDomain",runtime.config().inventoryDomain(),"allowedServerIds",List.of("amiya"),"attachments",List.of(Map.of("itemRef",version.itemRef(),"revision",version.revision(),"quantity",3,"payloadSha256",version.payloadSha256()))));
        Map<String,Object> payload=new LinkedHashMap<>();
        payload.put("source","deuterium-commerce");payload.put("orderId",order);payload.put("deliveryId",delivery);payload.put("recipientUuid",player.getUniqueId().toString());payload.put("inventoryDomain",runtime.config().inventoryDomain());payload.put("allowedServerIds",List.of("amiya"));payload.put("snapshotJson",snapshot);payload.put("snapshotSha256",Checks.sha(snapshot.getBytes(StandardCharsets.UTF_8)));payload.put("title","Isolated Core/Sync/Mail integration");payload.put("body","Synthetic test mail; no production player or order.");payload.put("sender","CoreProbe");
        var created=runtime.mailbox().execute(operation,"mailbox.create",Json.tree(payload));
        CoreProbe.check(created.get("code").getAsString().equals("OK"),"Mailbox create failed: "+created);
        long mail=created.getAsJsonObject("value").get("mailId").getAsLong();
        var replay=runtime.mailbox().execute(operation,"mailbox.create",Json.tree(payload));CoreProbe.check(replay.getAsJsonObject("value").get("mailId").getAsLong()==mail,"Create replay changed mail ID");
        int before=count(player,item);
        if(recovery){try(var fault=new SyncCommitFault(runtime.inventoryAccess())){player.performCommand("dmail claim "+mail);CoreProbe.check(fault.commits.get()>=2,"Commit acknowledgement fault not reached");}}
        else player.performCommand("dmail claim "+mail);
        var query=runtime.mailbox().execute("probe_query_"+UUID.randomUUID(),"mailbox.query",Json.tree(Map.of("source","deuterium-commerce","deliveryId",delivery)));
        CoreProbe.check(query.get("code").getAsString().equals("OK")&&query.getAsJsonObject("value").get("status").getAsString().equals("CLAIMED"),"Real mail claim was not confirmed: "+query);
        CoreProbe.check(count(player,item)==before+3,"Mail item count mismatch");player.performCommand("dmail claim "+mail);CoreProbe.check(count(player,item)==before+3,"Repeat mail claim granted twice");
        var revoke=runtime.mailbox().execute("probe_revoke_"+UUID.randomUUID(),"mailbox.revoke",Json.tree(Map.of("source","deuterium-commerce","orderId",order,"deliveryId",delivery,"expectedSnapshotSha256",payload.get("snapshotSha256"),"reasonCode","integration-check")));
        CoreProbe.check(revoke.get("code").getAsString().equals("ALREADY_CLAIMED"),"Claimed mail was revocable");
        runtime.mailbox().pumpEvents();
        if(recovery){
            var proof=runtime.database.read(c->{try(var p=c.prepareStatement("SELECT phase,operation_id,epoch FROM dc_sync_sessions WHERE uuid=?")){p.setString(1,player.getUniqueId().toString());try(var r=p.executeQuery()){CoreProbe.check(r.next()&&r.getString(1).equals("FAILED"),"Sync unknown session was not quarantined");return runtime.inventoryAccess().lookupSaveProof(player.getUniqueId(),runtime.config().inventoryDomain(),UUID.fromString(r.getString(2)),r.getString(3));}}});
            CoreProbe.check(proof!=null,"Mail recovery lacks original committed Sync proof");
            probe.report("mail-recovery",Map.of("passed",true,"deliveryId",delivery,"mailId",mail,"orderId",order,"claimedOnce",true,"additionalItems",3,"claimReceipt",query.getAsJsonObject("value"),"recoveredFromCommittedProof",proof,"syncSessionQuarantined",true));
        }else probe.report("mail",Map.of("passed",true,"deliveryId",delivery,"mailId",mail,"orderId",order,"createdOnce",true,"claimedOnce",true,"additionalItems",3,"claimReceipt",query.getAsJsonObject("value"),"postClaimRevokeCode",revoke.get("code").getAsString()));
    }
    static int count(Player player,ItemStack expected){int n=0;for(ItemStack item:player.getInventory().getStorageContents())if(item!=null&&item.isSimilar(expected))n+=item.getAmount();return n;}
}
