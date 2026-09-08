package cafe.deuterium.core.api;

public record ItemSummary(String itemRef, long latestRevision, long catalogVersion,
                          String displayName, String itemId, boolean archived) { }
