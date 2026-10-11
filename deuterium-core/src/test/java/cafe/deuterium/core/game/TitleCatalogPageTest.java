package cafe.deuterium.core.game;
import com.google.gson.*;
import org.junit.jupiter.api.Test;
import java.util.Set;
import static org.junit.jupiter.api.Assertions.*;

class TitleCatalogPageTest {
    private JsonObject title(String id,boolean visible){var t=new JsonObject();t.addProperty("id",id);t.addProperty("displayName",id);t.addProperty("public",visible);t.addProperty("description","介绍及获取方式。".repeat(100));return t;}
    @Test void privateTitlesAreOwnedOnlyAndNeverLeakThroughSearchOrCounts(){
        var catalog=new JsonObject();catalog.addProperty("revision",1);var items=new JsonArray();items.add(title("public",true));items.add(title("custom",false));catalog.add("items",items);
        var request=new JsonObject();request.addProperty("ownedOnly",false);request.addProperty("requestId","test");
        var page=TitleCatalogPage.build(catalog,Set.of("custom"),request);
        assertEquals(1,page.get("total").getAsInt());assertEquals("public",page.getAsJsonArray("items").get(0).getAsJsonObject().get("id").getAsString());
        request.addProperty("query","custom");assertEquals(0,TitleCatalogPage.build(catalog,Set.of("custom"),request).get("total").getAsInt());
        request.addProperty("ownedOnly",true);assertEquals("custom",TitleCatalogPage.build(catalog,Set.of("custom"),request).getAsJsonArray("items").get(0).getAsJsonObject().get("id").getAsString());
    }
    @Test void pagesBoundRealUtf8BytesAndContinueWithoutSkipping(){
        var catalog=new JsonObject();catalog.addProperty("revision",1);var items=new JsonArray();for(int i=0;i<30;i++)items.add(title("title"+i,true));catalog.add("items",items);
        var request=new JsonObject();request.addProperty("ownedOnly",false);request.addProperty("requestId","test");int count=0,offset=0;
        do{request.addProperty("offset",offset);var page=TitleCatalogPage.build(catalog,Set.of(),request);assertTrue(page.toString().getBytes(java.nio.charset.StandardCharsets.UTF_8).length<24000);count+=page.getAsJsonArray("items").size();offset=page.get("nextOffset").getAsInt();}while(offset>=0);
        assertEquals(30,count);
    }
}
