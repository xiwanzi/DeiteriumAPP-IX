package com.deuterium.app.uilab

import org.json.JSONObject
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.UUID

fun ledgerEntries(values: List<JSONObject>): List<LedgerEntry> = values
    .filter { it.optString("status").equals("success", ignoreCase = true) }
    .map { value ->
        val remoteId = value.getString("recordId")
        val at = Instant.parse(value.getString("occurredAt")).atZone(ZoneId.of("Asia/Shanghai")).toLocalDateTime()
        val amount = apiCents(value.getString("amount")) * if (value.getString("direction") == "expense") -1 else 1
        val title = value.optString("title").takeUnless { it.isBlank() || it == "null" } ?: "玩家转账"
        val player = value.optJSONObject("otherPlayer")?.optString("gameId")?.takeUnless { it.isBlank() || it == "null" }
        val name = if (value.optString("source") == "GAME") title else player ?: title
        val note = value.optString("note").takeUnless { it.isBlank() || it == "null" } ?: title
        val identity = remoteId.takeIf { it.startsWith("econ_") }?.removePrefix("econ_")?.toLongOrNull()
            ?: UUID.nameUUIDFromBytes(remoteId.toByteArray()).mostSignificantBits
        LedgerEntry(identity, name, note, amount, at.format(DateTimeFormatter.ofPattern("HH:mm")), at)
    }.distinctBy { it.id }.sortedWith(compareByDescending<LedgerEntry> { it.at }.thenByDescending { it.id })
