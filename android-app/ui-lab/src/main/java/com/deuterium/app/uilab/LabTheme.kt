package com.deuterium.app.uilab

import androidx.compose.animation.core.*
import androidx.compose.foundation.*
import androidx.compose.foundation.interaction.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.*
import androidx.compose.ui.graphics.drawscope.ContentDrawScope
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.node.*
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.*
import androidx.compose.ui.text.font.*
import androidx.compose.ui.unit.*

val LocalMotion=staticCompositionLocalOf { true }
val LocalGlassEnabled=staticCompositionLocalOf { true }
val Emerald=Color(0xFF007AFF)
val Gold=Color(0xFFB08345)
private object QuietIndication:IndicationNodeFactory {
    override fun create(interactionSource:InteractionSource):DelegatableNode=object:Modifier.Node(),DrawModifierNode {
        override fun ContentDrawScope.draw(){drawContent()}
    }
    override fun equals(other:Any?)=other===this
    override fun hashCode()=37
}

@Composable
fun LabTheme(theme:Int,motion:Boolean,glass:Boolean=true,parameters:GlassParameters=GlassParameters(),content:@Composable ()->Unit) {
    val dark=theme==2 || (theme==0 && isSystemInDarkTheme())
    val colors=if(dark)darkColorScheme(primary=Color(0xFF5AAAFF),onPrimary=Color.White,primaryContainer=Color(0xFF16334F),onPrimaryContainer=Color(0xFFAED4FF),
        background=Color(0xFF000000),surface=Color(0xFF1C1C1E),surfaceVariant=Color(0xFF2C2C2E),onSurface=Color(0xFFF5F5F7),onSurfaceVariant=Color(0xFFA7A7AD),
        secondaryContainer=Color(0xFF29292C),onSecondaryContainer=Color(0xFFF5F5F7),outlineVariant=Color(0xFF38383A),error=Color(0xFFFF6961),errorContainer=Color(0xFF3F1D1C))
    else lightColorScheme(primary=Color(0xFF007AFF),onPrimary=Color.White,primaryContainer=Color(0xFFE2EEFF),onPrimaryContainer=Color(0xFF0056B3),
        background=Color(0xFFF2F2F7),surface=Color.White,surfaceVariant=Color(0xFFE7E7EC),onSurface=Color(0xFF151518),onSurfaceVariant=Color(0xFF6E6E73),
        secondaryContainer=Color(0xFFECECF0),onSecondaryContainer=Color(0xFF28282B),outlineVariant=Color(0xFFDCDCE0),error=Color(0xFFD8322D),errorContainer=Color(0xFFFFEBE9))
    val type=Typography(
        headlineLarge=TextStyle(fontFamily=FontFamily.SansSerif,fontWeight=FontWeight.Bold,fontSize=34.sp,lineHeight=42.sp,letterSpacing=(-.8).sp),
        headlineSmall=TextStyle(fontWeight=FontWeight.Bold,fontSize=26.sp,lineHeight=34.sp,letterSpacing=(-.5).sp),
        titleLarge=TextStyle(fontWeight=FontWeight.SemiBold,fontSize=21.sp,lineHeight=29.sp),
        titleMedium=TextStyle(fontWeight=FontWeight.SemiBold,fontSize=17.sp,lineHeight=25.sp),
        bodyLarge=TextStyle(fontSize=17.sp,lineHeight=26.sp),bodyMedium=TextStyle(fontSize=15.sp,lineHeight=23.sp),bodySmall=TextStyle(fontSize=13.sp,lineHeight=20.sp),
        labelLarge=TextStyle(fontWeight=FontWeight.SemiBold,fontSize=16.sp,lineHeight=23.sp),labelMedium=TextStyle(fontSize=13.sp,lineHeight=19.sp),labelSmall=TextStyle(fontSize=11.sp,lineHeight=15.sp))
    CompositionLocalProvider(LocalMotion provides motion,LocalGlassEnabled provides glass,LocalGlassParameters provides parameters,LocalIndication provides QuietIndication) {
        MaterialTheme(colorScheme=colors,typography=type,shapes=Shapes(small=RoundedCornerShape(12.dp),medium=RoundedCornerShape(18.dp),large=RoundedCornerShape(24.dp),extraLarge=RoundedCornerShape(30.dp)),content=content)
    }
}

@Composable
fun MotionButton(onClick:()->Unit,modifier:Modifier=Modifier,enabled:Boolean=true,shape:Shape=CircleShape,colors:ButtonColors=ButtonDefaults.buttonColors(),content:@Composable RowScope.()->Unit) {
    val source=remember{MutableInteractionSource()};val pressed by source.collectIsPressedAsState()
    val scale by animateFloatAsState(if(pressed).97f else 1f,if(LocalMotion.current)spring(.8f,650f) else snap(),label="button-press")
    val background=if(enabled)colors.containerColor else colors.disabledContainerColor
    val foreground=if(enabled)colors.contentColor else colors.disabledContentColor
    Row(modifier.heightIn(min=48.dp).graphicsLayer{scaleX=scale;scaleY=scale}.clip(shape).background(background)
        .clickable(interactionSource=source,indication=null,enabled=enabled,role=Role.Button,onClick=onClick).padding(horizontal=20.dp,vertical=10.dp),
        verticalAlignment=androidx.compose.ui.Alignment.CenterVertically,horizontalArrangement=Arrangement.Center) {
        CompositionLocalProvider(LocalContentColor provides foreground){ProvideTextStyle(MaterialTheme.typography.labelLarge){content()}}
    }
}

@Composable
fun LabCard(modifier:Modifier=Modifier,content:@Composable ColumnScope.()->Unit) {
    Surface(modifier.fillMaxWidth(),shape=RoundedCornerShape(24.dp),color=MaterialTheme.colorScheme.surface){Column(Modifier.padding(18.dp),content=content)}
}
@Composable
fun RoundIcon(icon:ImageVector,description:String,onClick:()->Unit){IconButton(onClick,Modifier.size(44.dp)){Icon(icon,description,Modifier.size(22.dp),tint=MaterialTheme.colorScheme.primary)}}
@Composable
fun Eyebrow(text:String,modifier:Modifier=Modifier){Text(text,modifier,style=MaterialTheme.typography.labelSmall,color=MaterialTheme.colorScheme.onSurfaceVariant,letterSpacing=1.sp)}
