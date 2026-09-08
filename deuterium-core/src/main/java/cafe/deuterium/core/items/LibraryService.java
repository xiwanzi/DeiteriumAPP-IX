package cafe.deuterium.core.items;

import cafe.deuterium.core.api.*;
import cafe.deuterium.core.runtime.WorkPool;
import cafe.deuterium.core.storage.CatalogStore;
import cafe.deuterium.core.util.Checks;
import java.util.List;
import java.util.concurrent.CompletionStage;

public final class LibraryService implements ItemLibrary {
    private final CatalogStore store;
    private final WorkPool workers;
    private final String namespace;
    public LibraryService(CatalogStore store, WorkPool workers, String namespace) { this.store = store; this.workers = workers; this.namespace = namespace; }
    @Override public CompletionStage<ItemVersion> find(String itemRef, long revision) {
        String ref = Checks.itemRef(itemRef, namespace);
        if (revision < 0 || revision > Integer.MAX_VALUE) throw cafe.deuterium.core.util.CoreFailure.invalid("无效版本。");
        return workers.submit(() -> store.find(ref, revision));
    }
    @Override public CompletionStage<List<ItemSummary>> search(String query, int page, boolean includeArchived) { return workers.submit(() -> store.search(query, page, includeArchived)); }
    @Override public CompletionStage<List<Long>> versions(String itemRef, int page) { return workers.submit(() -> store.versions(Checks.itemRef(itemRef, namespace), page)); }
}
