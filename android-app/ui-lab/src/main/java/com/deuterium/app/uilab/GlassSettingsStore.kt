package com.deuterium.app.uilab

import android.content.SharedPreferences

/** One-time migration keeps the old header, while starting the two independent material groups. */
class GlassSettingsStore(private val preferences:SharedPreferences) {
    private fun number(key:String,fallback:Float,range:ClosedFloatingPointRange<Float>):Float =
        preferences.getFloat(key,fallback).takeIf { it.isFinite() }?.coerceIn(range) ?: fallback

    fun load():GlassMaterials {
        val defaults=GlassMaterials()
        if(preferences.getInt("materialVersion",0)<2) {
            val migrated=defaults.copy(header=HeaderGlassParameters(number("blur",18f,0f..36f),number("headerFade",32f,16f..96f)))
            save(migrated)
            return migrated
        }
        fun group(prefix:String,fallback:GlassParameters)=GlassParameters(
            number("$prefix.blur",fallback.blur,0f..36f),number("$prefix.opacity",fallback.opacity,.08f..1f),
            number("$prefix.refraction",fallback.refraction,0f..24f),number("$prefix.highlight",fallback.highlight,0f..1f))
        return GlassMaterials(group("bottomBar",defaults.bottomBar),group("overlay",defaults.overlay),
            HeaderGlassParameters(number("header.blur",18f,0f..36f),number("header.fade",32f,16f..96f)))
    }

    fun save(value:GlassMaterials) {
        preferences.edit().apply {
            fun group(prefix:String,parameters:GlassParameters) {
                putFloat("$prefix.blur",parameters.blur);putFloat("$prefix.opacity",parameters.opacity)
                putFloat("$prefix.refraction",parameters.refraction);putFloat("$prefix.highlight",parameters.highlight)
            }
            group("bottomBar",value.bottomBar);group("overlay",value.overlay)
            putFloat("header.blur",value.header.blur);putFloat("header.fade",value.header.fade)
            putInt("materialVersion",2)
        }.apply()
    }
}
