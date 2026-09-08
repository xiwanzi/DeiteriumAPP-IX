package cafe.deuterium.core.api;

import java.util.List;
import java.util.concurrent.CompletionStage;

/** Bukkit Services read API, owned and shipped exactly once by DeuteriumCore. */
public interface ItemLibrary {
    /** revision=0 selects the current non-archived version; explicit versions retain history. */
    CompletionStage<ItemVersion> find(String itemRef, long revision);
    CompletionStage<List<ItemSummary>> search(String query, int page, boolean includeArchived);
    CompletionStage<List<Long>> versions(String itemRef, int page);
}
