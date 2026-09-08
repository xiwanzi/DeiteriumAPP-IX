package com.deuterium.app.uilab

import android.net.Uri
import android.util.TypedValue
import android.widget.TextView
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.LocalUriHandler
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import io.noties.markwon.AbstractMarkwonPlugin
import io.noties.markwon.Markwon
import io.noties.markwon.MarkwonConfiguration
import io.noties.markwon.core.MarkwonTheme
import io.noties.markwon.ext.strikethrough.StrikethroughPlugin
import io.noties.markwon.ext.tables.TablePlugin

/** Native text spans; no WebView, JavaScript or arbitrary intent links. */
@Composable
fun MarkdownBody(text: String, modifier: Modifier = Modifier, onLongClick: (() -> Unit)? = null) {
    val context = LocalContext.current
    val uriHandler = LocalUriHandler.current
    val color = MaterialTheme.colorScheme.onSurface.toArgb()
    val link = MaterialTheme.colorScheme.primary.toArgb()
    val codeBackground = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = .55f).toArgb()
    val density = LocalDensity.current
    val lineExtra = with(density) { 5.dp.toPx() }
    val longClick by rememberUpdatedState(onLongClick)
    val renderer = remember(context, color, link, codeBackground) {
        Markwon.builder(context)
            .usePlugin(StrikethroughPlugin.create())
            .usePlugin(TablePlugin.create(context))
            .usePlugin(object : AbstractMarkwonPlugin() {
                override fun configureTheme(builder: MarkwonTheme.Builder) {
                    builder.linkColor(link).codeBackgroundColor(codeBackground)
                }
                override fun configureConfiguration(builder: MarkwonConfiguration.Builder) {
                    builder.linkResolver { _, destination ->
                        val uri = Uri.parse(destination)
                        if (uri.scheme?.lowercase() in setOf("https", "http") && !uri.host.isNullOrBlank()) {
                            runCatching { uriHandler.openUri(destination) }
                        }
                    }
                }
            }).build()
    }
    val rendered = remember(text, renderer) { renderer.toMarkdown(text) }
    AndroidView(modifier = modifier.fillMaxWidth(), factory = { viewContext ->
        TextView(viewContext).apply {
            includeFontPadding = false
            setPadding(0, 0, 0, 0)
        }
    }, update = { view ->
        view.setTextSize(TypedValue.COMPLEX_UNIT_SP, 16f)
        view.setLineSpacing(lineExtra, 1f)
        view.setTextColor(color)
        view.setLinkTextColor(link)
        view.setTextIsSelectable(onLongClick == null)
        view.setOnLongClickListener { longClick?.let { it(); true } ?: false }
        if (view.tag !== rendered) {
            renderer.setParsedMarkdown(view, rendered)
            view.tag = rendered
        }
    })
}

@Composable
fun UnreadDot(modifier: Modifier = Modifier) {
    Box(modifier.size(7.dp).background(Color(0xFFFF453A), CircleShape)
        .semantics { contentDescription = "有未读消息" })
}
