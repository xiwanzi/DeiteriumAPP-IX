package com.deuterium.app.uilab

import java.net.URLEncoder
import java.time.LocalDate
import java.time.ZoneId

/** The API supplies the current cutoff; a device clock must not make it a future query. */
internal fun billHistoryPath(start:Long,end:Long,type:String,today:LocalDate,cursor:String?=null):String {
    fun encoded(value:String)=URLEncoder.encode(value,"UTF-8")
    if(cursor!=null)return "/wallet/records?limit=25&cursor=${encoded(cursor)}"
    require(end>=start && end-start<366){"单次查询最多一年，请缩小日期范围"}
    require(type in listOf("all","income","expense")){"请选择正确的账单类型"}
    val zone=ZoneId.of("Asia/Shanghai")
    val from=LocalDate.ofEpochDay(start).atStartOfDay(zone).toInstant()
    val to=if(end<today.toEpochDay())LocalDate.ofEpochDay(end+1).atStartOfDay(zone).toInstant() else null
    return "/wallet/records?limit=25&from=${encoded(from.toString())}"+
        (to?.let{"&to=${encoded(it.toString())}"} ?: "")+
        (if(type=="all")"" else "&direction=$type")
}
