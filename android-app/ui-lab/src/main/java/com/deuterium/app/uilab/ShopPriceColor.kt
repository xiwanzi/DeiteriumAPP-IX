package com.deuterium.app.uilab

import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.runtime.Composable
import androidx.compose.material3.MaterialTheme
import androidx.compose.ui.graphics.Color

/** Optional demo override; normal builds retain their original theme colors. */
val LocalShopPriceColor = staticCompositionLocalOf<Color?> { null }

val ShopDiscountPink = Color(0xFFED6697)

@Composable
fun shopPriceColor(discounted:Boolean=false):Color = LocalShopPriceColor.current
    ?: if(discounted)ShopDiscountPink else MaterialTheme.colorScheme.onSurface
