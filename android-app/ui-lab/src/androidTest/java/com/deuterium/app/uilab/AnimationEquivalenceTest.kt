package com.deuterium.app.uilab

import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.asAndroidBitmap
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.test.*
import androidx.compose.ui.test.junit4.createEmptyComposeRule
import androidx.compose.ui.unit.dp
import androidx.test.core.app.ActivityScenario
import org.junit.Rule
import org.junit.Test

/** Both implementations run on one controlled frame clock; no wall-clock screenshot alignment. */
class AnimationEquivalenceTest {
    @get:Rule val compose=createEmptyComposeRule()

    @Test fun originalAndOptimizedSpringsRenderIdenticalIntermediateFrames() {
        compose.mainClock.autoAdvance=false
        val selected=mutableIntStateOf(0);val checked=mutableStateOf(false);val theme=mutableIntStateOf(1)
        val legacy=mutableStateOf(true)
        ActivityScenario.launch(DeuteriumActivity::class.java).use { scenario->
            scenario.onActivity { host->host.setContent {
                LabTheme(theme.intValue,true) {
                    Column(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.background)) {
                        val labels=listOf("商城","市场","信息","我的")
                        val icons=listOf(DeuteriumIcons.Shop,DeuteriumIcons.Market,DeuteriumIcons.Messages,DeuteriumIcons.Person)
                        Box(Modifier.width(400.dp).height(70.dp).testTag("nav")){
                            Box(Modifier.matchParentSize().graphicsLayer{alpha=if(legacy.value)1f else 0f}){LegacyMovingGlassNav(labels,icons,selected.intValue,true){selected.intValue=it}}
                            Box(Modifier.matchParentSize().graphicsLayer{alpha=if(legacy.value)0f else 1f}){MovingGlassNav(labels,icons,selected.intValue,true){selected.intValue=it}}
                        }
                        Box(Modifier.size(55.dp,44.dp).testTag("switch")){
                            Box(Modifier.matchParentSize().graphicsLayer{alpha=if(legacy.value)1f else 0f}){LegacyIosSwitch(checked.value,{checked.value=it})}
                            Box(Modifier.matchParentSize().graphicsLayer{alpha=if(legacy.value)0f else 1f}){IosSwitch(checked.value,{checked.value=it})}
                        }
                    }
                }
            }}
            fun identical(tag:String,label:String) {
                val time=compose.mainClock.currentTime
                compose.runOnIdle{legacy.value=true};compose.waitForIdle()
                val left=compose.onNodeWithTag(tag).captureToImage().asAndroidBitmap()
                compose.runOnIdle{legacy.value=false};compose.waitForIdle()
                val right=compose.onNodeWithTag(tag).captureToImage().asAndroidBitmap()
                check(compose.mainClock.currentTime==time){"Capture advanced the animation clock"}
                check(left.width==right.width&&left.height==right.height){"Size differs: $label"}
                val a=IntArray(left.width*left.height);val b=IntArray(a.size)
                left.getPixels(a,0,left.width,0,0,left.width,left.height);right.getPixels(b,0,right.width,0,0,right.width,right.height)
                val difference=a.indices.count{a[it]!=b[it]}
                if(difference!=0){
                    val directory=androidx.test.platform.app.InstrumentationRegistry.getInstrumentation().targetContext.filesDir
                    directory.resolve("animation-before.png").outputStream().use{left.compress(android.graphics.Bitmap.CompressFormat.PNG,100,it)}
                    directory.resolve("animation-after.png").outputStream().use{right.compress(android.graphics.Bitmap.CompressFormat.PNG,100,it)}
                }
                check(difference==0){"$label: $difference pixels differ"}
            }
            for(color in listOf(1,2)){
                compose.runOnIdle{theme.intValue=color;selected.intValue=0;checked.value=false}
                compose.mainClock.advanceTimeBy(1000);compose.waitForIdle()
                for(target in listOf(3,1,0)){
                    compose.runOnIdle{selected.intValue=target;checked.value=!checked.value}
                    for(step in listOf(16L,16L,32L,32L,64L,96L,144L,250L)){
                        compose.mainClock.advanceTimeBy(step);compose.waitForIdle()
                        identical("nav","navigation theme=$color target=$target t=${compose.mainClock.currentTime}")
                        identical("switch","switch theme=$color target=$target t=${compose.mainClock.currentTime}")
                    }
                }
            }
            androidx.test.platform.app.InstrumentationRegistry.getInstrumentation().targetContext.filesDir.resolve("animation-equivalence.json")
                .writeText("{\"comparisons\":96,\"differentPixels\":0,\"clock\":\"controlled-compose-clock\"}")
        }
    }
}
