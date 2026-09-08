package com.deuterium.app.uilab

import androidx.compose.ui.graphics.*
import androidx.compose.ui.graphics.vector.*
import androidx.compose.ui.unit.dp

/** Shared optical weight, rounded corners and a quiet secondary fill at 24 units. */
object DeuteriumIcons {
    private val ink=SolidColor(Color.Black)
    private fun icon(name:String,fill:String,outline:String)=ImageVector.Builder(name,24.dp,24.dp,24f,24f).apply{
        addPath(PathParser().parsePathString(fill).toNodes(),fill=ink,fillAlpha=.17f)
        addPath(PathParser().parsePathString(outline).toNodes(),stroke=ink,strokeLineWidth=1.65f,strokeLineCap=StrokeCap.Round,strokeLineJoin=StrokeJoin.Round)
    }.build()
    val Shop=icon("DeuteriumShop","M4,9 L20,9 L20,19 Q20,21 18,21 L6,21 Q4,21 4,19 Z","M4,10 L4,19 Q4,21 6,21 L18,21 Q20,21 20,19 L20,10 M3,8 L5,3.5 L19,3.5 L21,8 Q21,11 18,11 Q15,11 15,8 Q15,11 12,11 Q9,11 9,8 Q9,11 6,11 Q3,11 3,8 Z M9,8 L9.6,3.5 M15,8 L14.4,3.5 M9,21 L9,15 L15,15 L15,21")
    val Market=icon("DeuteriumMarket","M4,3 L9,3 Q10,3 10,4 L10,9 Q10,10 9,10 L4,10 Q3,10 3,9 L3,4 Q3,3 4,3 Z M15,14 L20,14 Q21,14 21,15 L21,20 Q21,21 20,21 L15,21 Q14,21 14,20 L14,15 Q14,14 15,14 Z","M4,3 L9,3 Q10,3 10,4 L10,9 Q10,10 9,10 L4,10 Q3,10 3,9 L3,4 Q3,3 4,3 Z M15,3 L20,3 Q21,3 21,4 L21,9 Q21,10 20,10 L15,10 Q14,10 14,9 L14,4 Q14,3 15,3 Z M4,14 L9,14 Q10,14 10,15 L10,20 Q10,21 9,21 L4,21 Q3,21 3,20 L3,15 Q3,14 4,14 Z M15,14 L20,14 Q21,14 21,15 L21,20 Q21,21 20,21 L15,21 Q14,21 14,20 L14,15 Q14,14 15,14 Z")
    val Messages=icon("DeuteriumMessages","M3,7 Q3,3 7,3 L17,3 Q21,3 21,7 L21,14 Q21,18 17,18 L8,18 L3,21 Z","M3,7 Q3,3 7,3 L17,3 Q21,3 21,7 L21,14 Q21,18 17,18 L8,18 L3,21 Z M7.5,8 L16.5,8 M7.5,12 L13.5,12")
    val Person=icon("DeuteriumPerson","M12,2.8 C17.5,2.8 17.5,11.2 12,11.2 C6.5,11.2 6.5,2.8 12,2.8 Z M3.5,21 C3.5,12 20.5,12 20.5,21 Z","M12,2.8 C17.5,2.8 17.5,11.2 12,11.2 C6.5,11.2 6.5,2.8 12,2.8 Z M3.5,21 C3.5,12 20.5,12 20.5,21 Z")
    val PublicChat=icon("DeuteriumPublic","M3,6 Q3,3 6,3 L14,3 Q17,3 17,6 L17,11 Q17,14 14,14 L7,14 L3,17 Z","M3,6 Q3,3 6,3 L14,3 Q17,3 17,6 L17,11 Q17,14 14,14 L7,14 L3,17 Z M20,8 Q22,8 22,11 L22,17 Q22,19 20,19 L19,19 L21,22 L15,19 L10,19 M7,7 L13,7 M7,10 L11,10")
    val Announcement=icon("DeuteriumAnnouncement","M4,9 L10,9 L20,4 L20,18 L10,14 L4,14 Z","M4,9 L10,9 L20,4 L20,18 L10,14 L4,14 Q2.5,14 2.5,12 L2.5,11 Q2.5,9 4,9 Z M10,9 L10,14 M6,14 L7.5,21 L11,21 L9,14")
    val Commission=icon("DeuteriumCommission","M5,4 L19,4 Q21,4 21,6 L21,19 Q21,21 19,21 L5,21 Q3,21 3,19 L3,6 Q3,4 5,4 Z","M8,4 L5,4 Q3,4 3,6 L3,19 Q3,21 5,21 L19,21 Q21,21 21,19 L21,6 Q21,4 19,4 L16,4 M8,3 L16,3 L16,7 L8,7 Z M7,12 L8.5,13.5 L11,11 M14,12 L17,12 M7,17 L8.5,18.5 L11,16 M14,17 L17,17")
}
