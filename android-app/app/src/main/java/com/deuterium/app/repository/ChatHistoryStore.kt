package com.deuterium.app.repository

import android.content.ContentValues
import android.content.Context
import android.database.sqlite.SQLiteDatabase
import android.database.sqlite.SQLiteOpenHelper
import com.deuterium.app.data.ChatFeedItem
import com.deuterium.app.data.ChatHistoryItem
import java.time.Instant

class ChatHistoryStore(context: Context) : SQLiteOpenHelper(
    context.applicationContext,
    DATABASE_NAME,
    null,
    DATABASE_VERSION
) {
    override fun onCreate(db: SQLiteDatabase) {
        db.execSQL(
            """
            CREATE TABLE $TABLE_MESSAGES (
                account_id TEXT NOT NULL,
                message_id TEXT NOT NULL,
                sender TEXT NOT NULL,
                content TEXT NOT NULL,
                kind TEXT NOT NULL,
                sent_at TEXT NOT NULL,
                display_time TEXT NOT NULL,
                mine INTEGER NOT NULL,
                event INTEGER NOT NULL,
                received_at TEXT NOT NULL,
                PRIMARY KEY(account_id, message_id)
            )
            """.trimIndent()
        )
        db.execSQL("CREATE INDEX idx_chat_history_account_time ON $TABLE_MESSAGES(account_id, sent_at, received_at)")
        db.execSQL("CREATE INDEX idx_chat_history_account_sender ON $TABLE_MESSAGES(account_id, sender)")
    }

    override fun onUpgrade(db: SQLiteDatabase, oldVersion: Int, newVersion: Int) {
        db.execSQL("DROP TABLE IF EXISTS $TABLE_MESSAGES")
        onCreate(db)
    }

    fun save(accountId: String?, item: ChatFeedItem) {
        saveAll(accountId, listOf(item))
    }

    fun saveAll(accountId: String?, items: List<ChatFeedItem>) {
        val normalizedAccountId = accountId?.takeIf { it.isNotBlank() } ?: return
        if (items.isEmpty()) return

        writableDatabase.transaction {
            items.forEach { item ->
                insertWithOnConflict(
                    TABLE_MESSAGES,
                    null,
                    item.toValues(normalizedAccountId),
                    SQLiteDatabase.CONFLICT_REPLACE
                )
            }
            trimAccountHistory(normalizedAccountId)
        }
    }

    fun search(accountId: String?, query: String, limit: Int = DEFAULT_QUERY_LIMIT): List<ChatHistoryItem> {
        val normalizedAccountId = accountId?.takeIf { it.isNotBlank() } ?: return emptyList()
        val trimmedQuery = query.trim()
        val (selection, args) = if (trimmedQuery.isBlank()) {
            "account_id = ?" to arrayOf(normalizedAccountId)
        } else {
            val like = "%${trimmedQuery.escapeLike()}%"
            "account_id = ? AND (sender LIKE ? ESCAPE '\\' OR content LIKE ? ESCAPE '\\')" to
                arrayOf(normalizedAccountId, like, like)
        }

        val rows = mutableListOf<ChatHistoryItem>()
        readableDatabase.query(
            TABLE_MESSAGES,
            COLUMNS,
            selection,
            args,
            null,
            null,
            "COALESCE(NULLIF(sent_at, ''), received_at) DESC, received_at DESC",
            limit.coerceAtLeast(1).toString()
        ).use { cursor ->
            val messageIdIndex = cursor.getColumnIndexOrThrow("message_id")
            val senderIndex = cursor.getColumnIndexOrThrow("sender")
            val contentIndex = cursor.getColumnIndexOrThrow("content")
            val kindIndex = cursor.getColumnIndexOrThrow("kind")
            val sentAtIndex = cursor.getColumnIndexOrThrow("sent_at")
            val displayTimeIndex = cursor.getColumnIndexOrThrow("display_time")
            val mineIndex = cursor.getColumnIndexOrThrow("mine")
            val eventIndex = cursor.getColumnIndexOrThrow("event")
            while (cursor.moveToNext()) {
                rows.add(
                    ChatHistoryItem(
                        messageId = cursor.getString(messageIdIndex),
                        sender = cursor.getString(senderIndex),
                        content = cursor.getString(contentIndex),
                        kind = cursor.getString(kindIndex),
                        sentAt = cursor.getString(sentAtIndex),
                        displayTime = cursor.getString(displayTimeIndex),
                        mine = cursor.getInt(mineIndex) == 1,
                        event = cursor.getInt(eventIndex) == 1
                    )
                )
            }
        }
        return rows
    }

    fun deleteAll(accountId: String?) {
        val normalizedAccountId = accountId?.takeIf { it.isNotBlank() } ?: return
        writableDatabase.delete(TABLE_MESSAGES, "account_id = ?", arrayOf(normalizedAccountId))
    }

    private fun ChatFeedItem.toValues(accountId: String): ContentValues {
        return ContentValues().apply {
            put("account_id", accountId)
            put("message_id", id)
            put("sender", sender)
            put("content", content)
            put("kind", kind)
            put("sent_at", sentAt.orEmpty())
            put("display_time", time)
            put("mine", if (mine) 1 else 0)
            put("event", if (event) 1 else 0)
            put("received_at", Instant.now().toString())
        }
    }

    private fun SQLiteDatabase.trimAccountHistory(accountId: String) {
        delete(
            TABLE_MESSAGES,
            """
            account_id = ? AND rowid NOT IN (
                SELECT rowid FROM $TABLE_MESSAGES
                WHERE account_id = ?
                ORDER BY COALESCE(NULLIF(sent_at, ''), received_at) DESC, received_at DESC
                LIMIT $MAX_MESSAGES_PER_ACCOUNT
            )
            """.trimIndent(),
            arrayOf(accountId, accountId)
        )
    }

    private fun String.escapeLike(): String {
        return replace("\\", "\\\\")
            .replace("%", "\\%")
            .replace("_", "\\_")
    }

    private inline fun SQLiteDatabase.transaction(block: SQLiteDatabase.() -> Unit) {
        beginTransaction()
        try {
            block()
            setTransactionSuccessful()
        } finally {
            endTransaction()
        }
    }

    private companion object {
        const val DATABASE_NAME = "deuterium_chat_history.db"
        const val DATABASE_VERSION = 1
        const val TABLE_MESSAGES = "chat_messages"
        const val DEFAULT_QUERY_LIMIT = 200
        const val MAX_MESSAGES_PER_ACCOUNT = 5000
        val COLUMNS = arrayOf(
            "message_id",
            "sender",
            "content",
            "kind",
            "sent_at",
            "display_time",
            "mine",
            "event"
        )
    }
}
