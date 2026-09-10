package com.deuterium.app.uilab

import android.Manifest
import android.app.*
import android.content.*
import android.content.pm.PackageManager
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat

class LabNotifications(private val context: Context) {
    init {
        val manager=context.getSystemService(NotificationManager::class.java)
        manager.createNotificationChannel(NotificationChannel("messages","消息提醒",NotificationManager.IMPORTANCE_HIGH).apply { description="提及和特别关心" })
        manager.createNotificationChannel(NotificationChannel("orders","交易与退款",NotificationManager.IMPORTANCE_DEFAULT).apply { description="订单、退款和结算提醒" })
        manager.createNotificationChannel(NotificationChannel("wallet","钱包通知",NotificationManager.IMPORTANCE_HIGH).apply { description="转账到账提醒" })
    }
    fun allowed():Boolean = (Build.VERSION.SDK_INT<33 || context.checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS)==PackageManager.PERMISSION_GRANTED) && NotificationManagerCompat.from(context).areNotificationsEnabled()
    fun post(event:DemoNotice) {
        if(!allowed())return
        val userName=BackendApi.get(context).userName
        if(userName.isBlank())return
        val preferences=NotificationPreferences(context,userName)
        if(!preferences.permits(event.topic))return
        val preview=preferences.get("showPreviews")
        val intent=Intent(context,DeuteriumActivity::class.java).putExtra("route",event.route).addFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP or Intent.FLAG_ACTIVITY_CLEAR_TOP)
        val pending=PendingIntent.getActivity(context,event.id.toInt(),intent,PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE)
        val notification=NotificationCompat.Builder(context,if(event.route=="wallet")"wallet" else if(event.route.startsWith("order:")||event.route.startsWith("refund:")||event.route.startsWith("commission:"))"orders" else "messages")
            .setSmallIcon(R.drawable.ic_notification).setContentTitle(if(preview)event.title else "Deuterium 通知").setContentText(if(preview)event.body else "你有一条新的提醒")
            .setStyle(NotificationCompat.BigTextStyle().bigText(if(preview)event.body else "打开 App 查看详情")).setContentIntent(pending).setAutoCancel(true)
            .setCategory(if(event.route=="wallet")NotificationCompat.CATEGORY_STATUS else NotificationCompat.CATEGORY_MESSAGE)
            .setVisibility(NotificationCompat.VISIBILITY_PRIVATE).setPriority(NotificationCompat.PRIORITY_HIGH).build()
        try { NotificationManagerCompat.from(context).notify(event.id.toInt(),notification) } catch(_:SecurityException) { }
    }
}
