package com.deuterium.app.uilab

import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.focus.*
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.*
import androidx.compose.ui.semantics.*
import androidx.compose.ui.text.input.*
import androidx.compose.ui.unit.*

@Composable
fun MessageComposer(value:TextFieldValue,onValue:(TextFieldValue)->Unit,onSend:()->Unit,placeholder:String,focus:FocusRequester?=null,reply:ChatReply?=null,onCancelReply:()->Unit={}) {
    val colors=MaterialTheme.colorScheme
    Column {
    reply?.let{ReplyPreview(it,onCancelReply)}
    Row(Modifier.fillMaxWidth().padding(horizontal=14.dp,vertical=8.dp).clip(RoundedCornerShape(24.dp)).background(colors.surface).border(.8.dp,colors.outlineVariant,RoundedCornerShape(24.dp)).padding(start=15.dp,end=4.dp),verticalAlignment=Alignment.Bottom){
        BasicTextField(value,onValue,Modifier.weight(1f).padding(vertical=10.dp).then(if(focus!=null)Modifier.focusRequester(focus) else Modifier),textStyle=MaterialTheme.typography.bodyLarge.copy(fontSize=16.sp,lineHeight=23.sp,color=colors.onSurface),maxLines=5,cursorBrush=SolidColor(colors.primary),keyboardOptions=KeyboardOptions(imeAction=ImeAction.Send),keyboardActions=KeyboardActions(onSend={onSend()}),decorationBox={inner->Box{if(value.text.isEmpty())Text(placeholder,fontSize=16.sp,color=colors.onSurfaceVariant);inner()}})
        val enabled=value.text.isNotBlank()
        Box(Modifier.size(42.dp).semantics{contentDescription="发送消息"}.clickable(enabled=enabled,role=Role.Button,onClick=onSend),contentAlignment=Alignment.Center){Canvas(Modifier.size(30.dp).background(if(enabled)colors.primary else colors.surfaceVariant,CircleShape)){val ink=if(enabled)Color.White else colors.onSurfaceVariant;val w=2.dp.toPx();drawLine(ink,Offset(center.x,size.height*.74f),Offset(center.x,size.height*.28f),w,StrokeCap.Round);drawLine(ink,Offset(size.width*.32f,size.height*.45f),Offset(center.x,size.height*.27f),w,StrokeCap.Round);drawLine(ink,Offset(center.x,size.height*.27f),Offset(size.width*.68f,size.height*.45f),w,StrokeCap.Round)}}
    }
}
}
