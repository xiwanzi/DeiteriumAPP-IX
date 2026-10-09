package com.deuterium.app.uilab

import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.*
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsPressedAsState
import androidx.compose.animation.Crossfade
import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.tween
import androidx.compose.material3.*
import androidx.compose.material.icons.outlined.SentimentSatisfied
import androidx.compose.material.icons.outlined.Keyboard
import androidx.compose.material.icons.outlined.AttachFile
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.Send
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.*
import androidx.compose.ui.graphics.*
import androidx.compose.ui.text.input.*
import androidx.compose.ui.unit.*

@Composable
fun MessageComposer(value:TextFieldValue,onValue:(TextFieldValue)->Unit,onSend:()->Unit,placeholder:String,focus:FocusRequester?=null,reply:ChatReply?=null,onCancelReply:()->Unit={},onEmoji:(()->Unit)?=null,onImage:(()->Unit)?=null,onFocus:(()->Unit)?=null,emojiOpen:Boolean=false) {
    val colors=MaterialTheme.colorScheme
    val motion=LocalMotion.current
    val inputInteraction=remember{MutableInteractionSource()}
    val pressed by inputInteraction.collectIsPressedAsState()
    LaunchedEffect(pressed){if(pressed)onFocus?.invoke()}
    val enabled=value.text.isNotBlank()
    val sendColor by animateColorAsState(if(enabled)colors.primary else colors.surfaceVariant,tween(if(motion)160 else 0),label="send available")
    Column {
    reply?.let{ReplyPreview(it,onCancelReply)}
    LiquidGlass(Modifier.fillMaxWidth().padding(horizontal=8.dp,vertical=6.dp),radius=26.dp){
      Row(Modifier.fillMaxWidth().padding(horizontal=3.dp,vertical=2.dp),verticalAlignment=Alignment.Bottom){
        onEmoji?.let{action->IconButton(action,Modifier.size(44.dp)){
            Crossfade(emojiOpen,animationSpec=tween(if(motion)140 else 0),label="expression toggle"){open->
                Icon(if(open)Icons.Outlined.Keyboard else Icons.Outlined.SentimentSatisfied,if(open)"切换键盘" else "选择表情",Modifier.size(23.dp),tint=colors.onSurfaceVariant)
            }
        }}
        BasicTextField(value,onValue,Modifier.weight(1f).heightIn(min=44.dp).padding(start=if(onEmoji==null)12.dp else 0.dp,end=4.dp,top=10.dp,bottom=10.dp).then(if(focus!=null)Modifier.focusRequester(focus) else Modifier).onFocusChanged{if(it.isFocused)onFocus?.invoke()},interactionSource=inputInteraction,textStyle=MaterialTheme.typography.bodyLarge.copy(fontSize=16.sp,lineHeight=23.sp,color=colors.onSurface),maxLines=5,cursorBrush=SolidColor(colors.primary),keyboardOptions=KeyboardOptions(imeAction=ImeAction.Send),keyboardActions=KeyboardActions(onSend={if(enabled)onSend()}),decorationBox={inner->Box{if(value.text.isEmpty())Text(placeholder,fontSize=16.sp,maxLines=1,color=colors.onSurfaceVariant);inner()}})
        onImage?.let{IconButton(it,Modifier.size(44.dp)){Icon(Icons.Outlined.AttachFile,"发送图片",Modifier.size(23.dp),tint=colors.onSurfaceVariant)}}
        IconButton(onSend,Modifier.size(44.dp),enabled=enabled){
            Box(Modifier.size(34.dp).background(sendColor,CircleShape),contentAlignment=Alignment.Center){Icon(Icons.AutoMirrored.Outlined.Send,"发送消息",Modifier.size(19.dp),tint=if(enabled)colors.onPrimary else colors.onSurfaceVariant)}
        }
      }
    }
}
}
