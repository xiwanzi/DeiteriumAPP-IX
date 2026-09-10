package com.deuterium.app.uilab

import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.*
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.test.*
import androidx.compose.ui.test.junit4.createEmptyComposeRule
import androidx.compose.ui.unit.dp
import androidx.test.core.app.ActivityScenario
import kotlinx.coroutines.*
import org.junit.Rule
import org.junit.Test
import java.time.LocalDateTime

class DerivedListsTest {
    @get:Rule val compose=createEmptyComposeRule()

    @Test fun cachedListsStillObserveEditsCategoriesFiltersAndNewLedgerRecords() {
        val scope=CoroutineScope(SupervisorJob()+Dispatchers.Main.immediate)
        val page=mutableStateOf("shop");val inset=mutableIntStateOf(24)
        lateinit var state:LabState
        try {ActivityScenario.launch(DeuteriumActivity::class.java).use { scenario->
            scenario.onActivity{host->
                state=LabState(scope,userName="FixtureSelf")
                ShopCatalog.clear();ShopCatalog.add(ShopProduct("live-product","最初商品","装备","固定简介",12345,Color.Blue,stock=20))
                host.setContent{MaterialTheme{Column(Modifier.fillMaxSize()){
                    if(page.value=="shop")ShopPage(state,{},inset.intValue.dp,"") else WalletPage(state,{},{},24.dp)
                }}}
            }
            compose.onAllNodesWithText("最初商品").onFirst().assertExists()
            compose.runOnIdle{ShopCatalog[0]=ShopCatalog[0].copy(name="修改后的商品",category="收藏")}
            compose.onAllNodesWithText("修改后的商品").onFirst().assertExists()
            compose.onNode(hasText("收藏") and SemanticsMatcher.expectValue(androidx.compose.ui.semantics.SemanticsProperties.Role,androidx.compose.ui.semantics.Role.RadioButton)).performClick()
            compose.runOnIdle{inset.intValue=64;ShopCatalog[0]=ShopCatalog[0].copy(category="补给")}
            compose.onNodeWithText("暂无上架商品").assertExists()
            compose.runOnIdle{ShopCatalog.add(ShopProduct("new-product","新到的收藏","收藏","固定简介",23456,Color.Blue,stock=20))}
            compose.onAllNodesWithText("新到的收藏").onFirst().assertExists()
            compose.runOnIdle {
                state.ledger.add(LedgerEntry(1,"最初流水","固定说明",100,"12:00",LocalDateTime.of(2026,9,11,12,0)))
                page.value="wallet"
            }
            compose.onNodeWithText("最初流水").assertExists()
            compose.runOnIdle{state.ledger[0]=state.ledger[0].copy(name="修改后的流水",amount=987654321)}
            compose.onNodeWithText("修改后的流水").assertExists()
            compose.runOnIdle{state.ledger.add(LedgerEntry(2,"新到账的流水","固定说明",200,"12:01",LocalDateTime.of(2026,9,11,12,1)))}
            compose.onNodeWithText("新到账的流水").assertExists()
        }} finally {scope.cancel()}
    }
}
