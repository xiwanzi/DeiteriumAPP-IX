package com.deuterium.app.uilab

import org.json.JSONObject

enum class LauncherIconChoice(val id:String,val alias:String) {
    Default("default",".MainActivity"),
    Anniversary("anniversary_911",".LauncherAnniversary");

    companion object {
        fun fromId(id:String)=entries.firstOrNull{it.id==id}
    }
}

data class LauncherIconConfiguration(val icon:LauncherIconChoice,val version:Long) {
    companion object {
        fun parse(json:JSONObject,appVersion:Int):LauncherIconConfiguration {
            val rawVersion=json.get("version")
            val rawMinimum=json.get("minAppVersionCode")
            require(rawVersion is Number && rawVersion.toDouble()==rawVersion.toLong().toDouble()) { "Invalid icon version" }
            require(rawMinimum is Number && rawMinimum.toDouble()==rawMinimum.toLong().toDouble()) { "Invalid icon compatibility" }
            val version=rawVersion.toLong()
            val minimum=rawMinimum.toLong()
            require(version in 1..9_007_199_254_740_991L && minimum in 1..appVersion.toLong()) { "Unsupported icon configuration" }
            val icon=LauncherIconChoice.fromId(json.getString("iconId")) ?: error("Unknown launcher icon")
            return LauncherIconConfiguration(icon,version)
        }
    }
}
