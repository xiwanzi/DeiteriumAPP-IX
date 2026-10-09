package com.deuterium.app.uilab

import android.content.ContentValues
import android.content.Context
import android.database.sqlite.SQLiteDatabase
import android.database.sqlite.SQLiteOpenHelper
import kotlinx.coroutines.flow.MutableSharedFlow
import org.json.JSONObject
import java.io.File
import java.security.MessageDigest

internal data class ChatCacheRecord(val channel:String,val value:JSONObject)

/** One 500MiB budget across accounts, with account/origin-scoped file identities.
 * Age controls eviction order only. Neither reads nor server expiry delete bytes. */
internal class ChatCache(context:Context, val scope:FinancialScope) {
    private val app=context.applicationContext
    private val owner=hash("${scope.origin}:${scope.owner}".toByteArray())
    private val root=File(app.filesDir,"chat-cache")
    private val media=File(root,"media/$owner")
    private val db get()=database(app).writableDatabase
    fun file(sha:String):File?=synchronized(lock){require(sha.matches(Regex("[a-f0-9]{64}")));File(media,"$sha.webp").takeIf{it.isFile}}
    fun putFile(sha:String,bytes:ByteArray,expectedEpoch:Long=epoch):File=synchronized(lock){
        check(epoch==expectedEpoch){"聊天缓存已清理，请重试"};require(hash(bytes)==sha){"图片不完整，请重试"}
        media.mkdirs();val target=File(media,"$sha.webp")
        if(!target.exists()){
            trim(app,reserve=bytes.size.toLong())
            val temp=File(media,"$sha.part")
            try{temp.outputStream().use{it.write(bytes);it.fdSyncIfPossible()};check(temp.renameTo(target)){"本机存储空间不足"}}finally{temp.delete()}
        }
        trim(app,setOf(target.canonicalPath));target
    }
    fun save(channel:String,value:JSONObject,expectedEpoch:Long=epoch)=synchronized(lock){
        if(epoch!=expectedEpoch)return@synchronized
        val id=value.optString("messageId");if(id.isBlank())return@synchronized
        val at=runCatching{java.time.Instant.parse(value.getString("sentAt")).toEpochMilli()}.getOrDefault(System.currentTimeMillis())
        val values=ContentValues().apply{put("owner",owner);put("message_id",id);put("channel",channel);put("sent_at",at);put("payload",value.toString())}
        db.insertWithOnConflict("messages",null,values,SQLiteDatabase.CONFLICT_REPLACE);trim(app)
    }
    fun read(channel:String?=null,before:Long=Long.MAX_VALUE,limit:Int=200):List<ChatCacheRecord> = synchronized(lock){
        val selection="owner=? AND sent_at<?"+(if(channel!=null)" AND channel=?" else "")
        val args=mutableListOf(owner,before.toString());channel?.let{args+=it}
        db.query("messages",arrayOf("channel","payload"),selection,args.toTypedArray(),null,null,"sent_at DESC",limit.coerceIn(1,1000).toString()).use{cursor->buildList{while(cursor.moveToNext())runCatching{ChatCacheRecord(cursor.getString(0),JSONObject(cursor.getString(1)))}.getOrNull()?.let{add(it)}}}.asReversed()
    }
    fun removePlayers(refs:Set<String>)=synchronized(lock){
        if(refs.isEmpty())return@synchronized
        val ids=mutableListOf<String>();val updates=mutableListOf<Pair<String,String>>()
        db.query("messages",arrayOf("message_id","payload"),"owner=?",arrayOf(owner),null,null,null).use{cursor->
            while(cursor.moveToNext()){
                val value=runCatching{JSONObject(cursor.getString(1))}.getOrNull() ?: continue
                if(value.optJSONObject("sender")?.optString("playerRef") in refs||value.optString("_cachePeerRef") in refs)ids+=cursor.getString(0)
                else{redactErasedAccounts(value,refs);val redacted=value.toString();if(redacted!=cursor.getString(1))updates+=cursor.getString(0) to redacted}
            }
        }
        ids.forEach{db.delete("messages","owner=? AND message_id=?",arrayOf(owner,it))}
        updates.forEach{(id,text)->db.update("messages",ContentValues().apply{put("payload",text)},"owner=? AND message_id=?",arrayOf(owner,id))}
    }
    companion object {
        const val LIMIT=500L*1024*1024
        private val lock=Any()
        @Volatile var epoch=0L;private set
        val cleared=MutableSharedFlow<Boolean>(extraBufferCapacity=8)
        private var helper:ChatRecordsDB?=null
        private fun database(context:Context)=helper ?: ChatRecordsDB(context.applicationContext).also{helper=it}
        fun hash(bytes:ByteArray)=MessageDigest.getInstance("SHA-256").digest(bytes).joinToString(""){"%02x".format(it)}
        fun bytes(context:Context):Long=synchronized(lock){StorageFiles.bytes(File(context.filesDir,"chat-cache"))}
        fun mediaBytes(context:Context):Long=synchronized(lock){StorageFiles.bytes(File(context.filesDir,"chat-cache/media"))}
        fun clear(context:Context,records:Boolean):Long=synchronized(lock){
            val before=bytes(context);epoch++
            val folder=File(context.filesDir,"chat-cache/media");folder.listFiles()?.forEach{StorageFiles.clearChildren(it);it.delete()}
            if(records){
                try{val db=database(context).writableDatabase;db.delete("messages",null,null);db.execSQL("VACUUM")}
                catch(_:android.database.sqlite.SQLiteException){
                    helper?.close();helper=null
                    // The user explicitly chose to clear local records. Recover
                    // an unreadable cache without touching pending sends/data.
                    listOf("records.db","records.db-journal","records.db-wal","records.db-shm").forEach{File(context.filesDir,"chat-cache/$it").delete()}
                }
            }
            cleared.tryEmit(records);(before-bytes(context)).coerceAtLeast(0)
        }
        private fun trim(context:Context,protected:Set<String> = emptySet(),reserve:Long=0) {
            val root=File(context.filesDir,"chat-cache");var total=StorageFiles.bytes(root);val budget=minOf(LIMIT-reserve,total-(reserve-context.filesDir.usableSpace).coerceAtLeast(0));if(total<=budget)return
            File(root,"media").walkTopDown().filter{it.isFile&&it.canonicalPath !in protected}.sortedBy{it.lastModified()}.forEach{file->if(total>budget){val n=file.length();if(file.delete())total-=n}}
            // Records are kept preferentially; only a records-only overflow can
            // evict oldest local rows. Cloud history is never changed here.
            if(total>budget){val db=database(context).writableDatabase;while(total>budget){val removed=db.delete("messages","rowid IN (SELECT rowid FROM messages ORDER BY sent_at LIMIT 1000)",null);if(removed==0)break;db.execSQL("VACUUM");total=StorageFiles.bytes(root)}}
            check(total<=budget){"聊天缓存空间不足"}
        }
    }
}

private fun java.io.OutputStream.fdSyncIfPossible(){(this as? java.io.FileOutputStream)?.fd?.sync()}
private class ChatRecordsDB(context:Context):SQLiteOpenHelper(context,File(context.filesDir,"chat-cache/records.db").apply{parentFile?.mkdirs()}.path,null,1){
    override fun onCreate(db:SQLiteDatabase){db.execSQL("CREATE TABLE messages(owner TEXT NOT NULL,message_id TEXT NOT NULL,channel TEXT NOT NULL,sent_at INTEGER NOT NULL,payload TEXT NOT NULL,PRIMARY KEY(owner,message_id))");db.execSQL("CREATE INDEX chat_records_order ON messages(owner,channel,sent_at)")}
    override fun onUpgrade(db:SQLiteDatabase,oldVersion:Int,newVersion:Int)=Unit
}
