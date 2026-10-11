package cafe.deuterium.core.game;

import cafe.deuterium.core.util.Json;
import com.google.gson.*;
import java.nio.charset.StandardCharsets;
import java.util.*;

/** Player-visible catalog projection. Private definitions never enter the public tab. */
final class TitleCatalogPage {
    static JsonObject build(JsonObject catalog,Set<String> owned,JsonObject request){
        boolean ownedOnly=!request.has("ownedOnly")||request.get("ownedOnly").getAsBoolean();
        int offset=request.has("offset")?request.get("offset").getAsInt():0;
        if(offset<0||offset>100000)throw new IllegalArgumentException("列表位置不正确");
        String query=request.has("query")?request.get("query").getAsString().strip().toLowerCase(Locale.ROOT):"";
        if(query.length()>64)throw new IllegalArgumentException("搜索文字过长");
        var items=new JsonArray();int index=0,ownedCount=0,bytes=0,next=-1;
        for(var raw:catalog.getAsJsonArray("items")){
            var title=raw.getAsJsonObject();boolean permitted=owned.contains(Json.string(title,"id"));
            if(permitted)ownedCount++;
            if(ownedOnly?!permitted:!title.has("public")||!title.get("public").getAsBoolean())continue;
            if(!Json.string(title,"displayName").replaceAll("#[0-9a-fA-F]{6}","").toLowerCase(Locale.ROOT).contains(query))continue;
            if(index++<offset)continue;
            var item=title.deepCopy();item.addProperty("owned",permitted);
            int size=Json.GSON.toJson(item).getBytes(StandardCharsets.UTF_8).length;
            if(next<0&&items.size()<24&&bytes+size<=20000){items.add(item);bytes+=size;}else if(next<0)next=index-1;
        }
        var result=new JsonObject();result.addProperty("type","titles-list");result.add("requestId",request.get("requestId"));
        result.add("revision",catalog.get("revision"));result.addProperty("offset",offset);result.addProperty("nextOffset",next);
        result.addProperty("ownedOnly",ownedOnly);result.addProperty("total",index);result.addProperty("ownedCount",ownedCount);result.add("items",items);
        return result;
    }
}
