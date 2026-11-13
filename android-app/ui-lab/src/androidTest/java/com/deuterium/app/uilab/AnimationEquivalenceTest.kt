package com.deuterium.app.uilab

import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.test.*
import androidx.compose.ui.test.junit4.createEmptyComposeRule
import androidx.compose.ui.unit.dp
import androidx.test.core.app.ActivityScenario
import org.junit.Rule
import org.junit.Test

/** Check navigation capsule coverage on a controlled frame clock. */
class AnimationEquivalenceTest {
    @get:Rule val compose=createEmptyComposeRule()

    @Test fun navigationCapsuleCoversSelectedLabel() {
        compose.mainClock.autoAdvance=false
        val selected=mutableIntStateOf(0);val theme=mutableIntStateOf(1)
        val labels = listOf("商城", "市场", "信息", "我的")
        ActivityScenario.launch(DeuteriumActivity::class.java).use { scenario->
            scenario.onActivity { host->host.setContent {
                LabTheme(theme.intValue,true) {
                    Column(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.background)) {
                        val icons=listOf(DeuteriumIcons.Shop,DeuteriumIcons.Market,DeuteriumIcons.Messages,DeuteriumIcons.Person)
                        Box(Modifier.width(400.dp).height(60.dp).testTag("navigation-bar")) {
                            MovingGlassNav(labels, icons, selected.intValue, true) { selected.intValue = it }
                        }
                    }
                }
            }}
            for(color in listOf(1,2)){
                compose.runOnIdle{theme.intValue=color;selected.intValue=0}
                compose.mainClock.advanceTimeBy(1000);compose.waitForIdle()
                for(target in listOf(3,1,0)){
                    compose.runOnIdle{selected.intValue=target}
                    for(step in listOf(16L,16L,32L,32L,64L,96L,144L,250L)){
                        compose.mainClock.advanceTimeBy(step);compose.waitForIdle()
                    }
                    val capsule = compose.onNodeWithTag("navigation-selection").getUnclippedBoundsInRoot()
                    val bar = compose.onNodeWithTag("navigation-bar").getUnclippedBoundsInRoot()
                    val gaps = listOf(capsule.top - bar.top, bar.bottom - capsule.bottom) + when (target) {
                        labels.indices.first -> listOf(capsule.left - bar.left)
                        labels.indices.last -> listOf(bar.right - capsule.right)
                        else -> emptyList()
                    }
                    check(gaps.all { it in (BottomNavigationInset - .5.dp)..(BottomNavigationInset + .5.dp) }) {
                        "Navigation capsule gaps differ from the shared inset: $gaps"
                    }
                    val label = compose.onNodeWithText(labels[target], useUnmergedTree = true).getUnclippedBoundsInRoot()
                    check(capsule.left <= label.left && capsule.right >= label.right &&
                        capsule.top <= label.top && capsule.bottom >= label.bottom) {
                        "Selected label must fit inside its capsule: $label / $capsule"
                    }
                }
            }
            androidx.test.platform.app.InstrumentationRegistry.getInstrumentation().targetContext.filesDir.resolve("animation-equivalence.json")
                .writeText("{\"navigationCoverageChecks\":6,\"clock\":\"controlled-compose-clock\"}")
        }
    }
}
