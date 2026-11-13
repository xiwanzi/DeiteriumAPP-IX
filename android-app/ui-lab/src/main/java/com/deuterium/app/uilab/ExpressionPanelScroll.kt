package com.deuterium.app.uilab

import androidx.compose.animation.core.animate
import androidx.compose.animation.core.tween
import androidx.compose.runtime.Stable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.input.nestedscroll.NestedScrollConnection
import androidx.compose.ui.input.nestedscroll.NestedScrollSource
import androidx.compose.ui.unit.Velocity
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.launch

/** Only the panel's chrome consumes scroll. The emoji grid keeps its normal fling physics. */
@Stable
internal class ExpressionPanelScroll(
    private val scope: CoroutineScope,
    private val headerHeight: Float,
    private val footerHeight: Float,
    private val motion: Boolean
) : NestedScrollConnection {
    var headerHidden by mutableFloatStateOf(0f)
        private set
    var footerHidden by mutableFloatStateOf(0f)
        private set
    private var settling: Job? = null

    fun reveal() {
        settling?.cancel()
        headerHidden = 0f
        footerHidden = 0f
    }

    override fun onPreScroll(available: Offset, source: NestedScrollSource): Offset {
        if (source != NestedScrollSource.UserInput || available.y == 0f) return Offset.Zero
        settling?.cancel()
        val old = headerHidden
        headerHidden = (old - available.y).coerceIn(0f, headerHeight)
        footerHidden = (footerHidden - available.y).coerceIn(0f, footerHeight)
        return Offset(0f, old - headerHidden)
    }

    override suspend fun onPostFling(consumed: Velocity, available: Velocity): Velocity {
        settling?.cancel()
        settling = scope.launch {
            val fromHeader = headerHidden
            val fromFooter = footerHidden
            val toHeader = if (fromHeader > headerHeight / 2f) headerHeight else 0f
            val toFooter = if (fromFooter > footerHeight / 2f) footerHeight else 0f
            if (!motion) {
                headerHidden = toHeader
                footerHidden = toFooter
            } else animate(0f, 1f, animationSpec = tween(200)) { progress, _ ->
                headerHidden = fromHeader + (toHeader - fromHeader) * progress
                footerHidden = fromFooter + (toFooter - fromFooter) * progress
            }
        }
        return Velocity.Zero
    }
}
