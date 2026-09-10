package com.deuterium.app.uilab

import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.grid.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ArrowForward
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.*
import androidx.compose.ui.graphics.drawscope.*
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.unit.Dp

data class ShopProduct(val id: String, val name: String, val category: String, val subtitle: String, val price: Long, val color: Color,
    val image: Int = 0, val darkArt: Boolean = false, val contents: List<String> = emptyList(), val brand:String="Deuterium",val gallery:List<Int> = emptyList(),val poster:Boolean=true,
    val photos:List<String> = emptyList(),val description:String="",val deliverySummary:String="游戏内邮箱",val estimatedDelivery:String="以服务器交付进度为准",val version:Long=1,val stock:Int=0,val limit:Int=9,
    val originalPrice:Long=price,val deliveryCredits:Long=0,val purchaseLimits:List<String> = emptyList(),val storeId:String="")
@Composable
fun ProductArt(kind: String, modifier: Modifier = Modifier) {
    Canvas(modifier) {
        val u = size.minDimension / 100f
        translate((size.width - 100f*u)/2, (size.height - 100f*u)/2) {
            scale(u, u, Offset.Zero) {
                drawOval(Color(0xFF265640).copy(alpha = .10f), Offset(17f,79f), Size(70f,10f))
                fun polygon(vararg xy: Float, color: Color) {
                    val path = Path().apply { moveTo(xy[0],xy[1]); for(i in 2 until xy.size step 2) lineTo(xy[i],xy[i+1]); close() }
                    drawPath(path, color)
                }
                if(kind == "tools") {
                    rotate(-36f, Offset(50f,50f)) {
                        drawRoundRect(Color(0xFF865E3B), Offset(45f,32f), Size(10f,51f), androidx.compose.ui.geometry.CornerRadius(2f))
                        drawRect(Color(0xFFC19662), Offset(45f,32f), Size(3f,51f))
                        polygon(20f,25f, 68f,25f, 80f,38f, 80f,54f, 69f,42f, 20f,42f, color=Color(0xFF267F70))
                        polygon(20f,25f, 68f,25f, 76f,33f, 20f,33f, color=Color(0xFF8AE0CC))
                        drawRect(Color(0xFFB5F2DB), Offset(25f,27f), Size(34f,3f))
                    }
                } else {
                    val top = when(kind) { "garden" -> Color(0xFFA9CBA2); "starlight" -> Color(0xFFD2C4E9); else -> Color(0xFFE2BA85) }
                    val side = when(kind) { "garden" -> Color(0xFF7EAC84); "starlight" -> Color(0xFF9A83BD); else -> Color(0xFFB68454) }
                    polygon(18f,44f, 50f,26f, 83f,44f, 50f,62f, color=top)
                    polygon(18f,44f, 50f,62f, 50f,85f, 18f,67f, color=side)
                    polygon(50f,62f, 83f,44f, 83f,67f, 50f,85f, color=side.copy(red=(side.red*.8f)))
                    if(kind == "garden") {
                        drawRect(Color(0xFF8C6E50), Offset(48f,22f), Size(6f,29f))
                        polygon(29f,20f, 51f,8f, 73f,20f, 51f,32f, color=Color(0xFFF6D8E2))
                        polygon(29f,20f, 51f,32f, 51f,45f, 29f,33f, color=Color(0xFFE8B4C6))
                        polygon(51f,32f, 73f,20f, 73f,33f, 51f,45f, color=Color(0xFFCE93AF))
                        drawCircle(Color(0xFFFFF0F4), 3f, Offset(41f,24f)); drawCircle(Color(0xFFFBE0E9), 2f, Offset(62f,29f))
                    } else {
                        polygon(44f,30f, 50f,26f, 83f,44f, 77f,48f, 44f,30f, color=Color(0xFFFFE9AE))
                        polygon(18f,44f, 24f,40f, 56f,58f, 56f,82f, 50f,85f, 50f,62f, color=Color(0xFFFAE1A2))
                        drawCircle(Color.White.copy(alpha=.8f), 4f, Offset(55f,40f))
                    }
                }
                fun sparkle(x: Float,y: Float,r: Float) {
                    val p=Path().apply{moveTo(x,y-r);lineTo(x+1.3f,y-1.3f);lineTo(x+r,y);lineTo(x+1.3f,y+1.3f);lineTo(x,y+r);lineTo(x-1.3f,y+1.3f);lineTo(x-r,y);lineTo(x-1.3f,y-1.3f);close()}
                    drawPath(p, Color(0xFF88B39E))
                }
                sparkle(84f,22f,5f); sparkle(16f,30f,3f)
            }
        }
    }
}
