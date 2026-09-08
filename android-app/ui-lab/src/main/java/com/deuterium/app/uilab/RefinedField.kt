package com.deuterium.app.uilab

import androidx.compose.foundation.*
import androidx.compose.foundation.interaction.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.*
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.text.input.*
import androidx.compose.ui.unit.dp

@Composable
fun RefinedField(value:String,onValueChange:(String)->Unit,modifier:Modifier=Modifier,label:(@Composable ()->Unit)?=null,
    placeholder:(@Composable ()->Unit)?=null,singleLine:Boolean=false,minLines:Int=1,maxLines:Int=Int.MAX_VALUE,
    leadingIcon:(@Composable ()->Unit)?=null,trailingIcon:(@Composable ()->Unit)?=null,suffix:(@Composable ()->Unit)?=null,
    isError:Boolean=false,shape:Shape=RoundedCornerShape(16.dp),visualTransformation:VisualTransformation=VisualTransformation.None,
    keyboardOptions:KeyboardOptions=KeyboardOptions.Default,keyboardActions:KeyboardActions=KeyboardActions.Default) {
    var internal by remember { mutableStateOf(TextFieldValue(value)) }
    val effective = if(internal.text == value) internal else internal.copy(text=value,selection=androidx.compose.ui.text.TextRange(value.length))
    RefinedField(effective,{internal=it;onValueChange(it.text)},modifier,label,placeholder,singleLine,minLines,maxLines,
        leadingIcon,trailingIcon,suffix,isError,shape,visualTransformation,keyboardOptions,keyboardActions)
}

@Composable
fun RefinedField(value:TextFieldValue,onValueChange:(TextFieldValue)->Unit,modifier:Modifier=Modifier,label:(@Composable ()->Unit)?=null,
    placeholder:(@Composable ()->Unit)?=null,singleLine:Boolean=false,minLines:Int=1,maxLines:Int=Int.MAX_VALUE,
    leadingIcon:(@Composable ()->Unit)?=null,trailingIcon:(@Composable ()->Unit)?=null,suffix:(@Composable ()->Unit)?=null,
    isError:Boolean=false,shape:Shape=RoundedCornerShape(16.dp),visualTransformation:VisualTransformation=VisualTransformation.None,
    keyboardOptions:KeyboardOptions=KeyboardOptions.Default,keyboardActions:KeyboardActions=KeyboardActions.Default) {
    val source=remember { MutableInteractionSource() }
    val focus=remember { FocusRequester() }
    val focused by source.collectIsFocusedAsState()
    val colors=MaterialTheme.colorScheme
    Column(modifier) {
        if(label!=null) CompositionLocalProvider(LocalContentColor provides colors.onSurfaceVariant) {
            ProvideTextStyle(MaterialTheme.typography.labelMedium) { Box(Modifier.clickable(indication=null,interactionSource=source){focus.requestFocus()}.padding(start=3.dp,bottom=8.dp)){label()} }
        }
        BasicTextField(value,onValueChange,Modifier.fillMaxWidth().focusRequester(focus).background(colors.surfaceVariant.copy(alpha=.58f),shape)
            .border(if(focused || isError) 1.dp else 0.dp,if(isError)colors.error else if(focused)colors.primary.copy(alpha=.45f) else Color.Transparent,shape),
            textStyle=MaterialTheme.typography.bodyLarge.copy(color=colors.onSurface),singleLine=singleLine,minLines=minLines,maxLines=maxLines,
            visualTransformation=visualTransformation,keyboardOptions=keyboardOptions,keyboardActions=keyboardActions,
            interactionSource=source,cursorBrush=SolidColor(colors.primary),decorationBox={inner->
                Row(Modifier.defaultMinSize(minHeight=52.dp).padding(horizontal=14.dp),verticalAlignment=Alignment.CenterVertically) {
                    if(leadingIcon!=null){leadingIcon();Spacer(Modifier.width(9.dp))}
                    Box(Modifier.weight(1f).padding(vertical=10.dp)){if(value.text.isEmpty()&&placeholder!=null)CompositionLocalProvider(LocalContentColor provides colors.onSurfaceVariant){placeholder()};inner()}
                    suffix?.invoke();trailingIcon?.invoke()
                }
            })
    }
}
