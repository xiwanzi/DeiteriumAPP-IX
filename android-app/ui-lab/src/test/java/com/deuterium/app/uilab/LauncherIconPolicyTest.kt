package com.deuterium.app.uilab

import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test

class LauncherIconPolicyTest {
    private fun config(icon:String="default",version:Any=1,minimum:Any=20800)=JSONObject().put("iconId",icon).put("version",version).put("minAppVersionCode",minimum)

    @Test fun onlyBuiltInIconsCanBeSelected() {
        assertEquals(LauncherIconChoice.Default,LauncherIconConfiguration.parse(config(),20800).icon)
        assertEquals(LauncherIconChoice.Anniversary,LauncherIconConfiguration.parse(config("anniversary_911",2),20800).icon)
        for(id in listOf("", "DEFAULT", "https://example.com/icon.png", ".DeuteriumActivity", "future_icon")) {
            assertTrue(runCatching{LauncherIconConfiguration.parse(config(id),20800)}.isFailure)
        }
    }

    @Test fun malformedOrIncompatibleRevisionsDoNotBecomeCommands() {
        for(version in listOf<Any>(0,-1,1.5,"3",true,Long.MAX_VALUE)) {
            assertTrue("Rejected version $version",runCatching{LauncherIconConfiguration.parse(config(version=version),20800)}.isFailure)
        }
        for(minimum in listOf<Any>(0,-1,20801,20800.5,"20800",true)) {
            assertTrue(runCatching{LauncherIconConfiguration.parse(config(minimum=minimum),20800)}.isFailure)
        }
        assertEquals(42L,LauncherIconConfiguration.parse(config(version=42L),20800).version)
    }
}
