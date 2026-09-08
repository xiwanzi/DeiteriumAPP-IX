package com.deuterium.app.uilab

import android.hardware.*
import android.content.Context
import androidx.compose.runtime.*
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.platform.LocalContext
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner

val LocalDeviceTilt = staticCompositionLocalOf<State<Offset>> { mutableStateOf(Offset.Zero) }

@Composable
fun rememberDeviceTilt(enabled: Boolean): State<Offset> {
    val context = LocalContext.current
    val owner = LocalLifecycleOwner.current
    val output = remember { mutableStateOf(Offset.Zero) }
    DisposableEffect(enabled, owner) {
        val manager = context.getSystemService(Context.SENSOR_SERVICE) as SensorManager
        val sensor = manager.getDefaultSensor(Sensor.TYPE_GAME_ROTATION_VECTOR)
            ?: manager.getDefaultSensor(Sensor.TYPE_ROTATION_VECTOR)
            ?: manager.getDefaultSensor(Sensor.TYPE_ACCELEROMETER)
        val matrix = FloatArray(9)
        val angles = FloatArray(3)
        var baseline: Offset? = null
        val listener = object : SensorEventListener {
            override fun onAccuracyChanged(sensor: Sensor?, accuracy: Int) = Unit
            override fun onSensorChanged(event: SensorEvent) {
                val value = if(event.sensor.type == Sensor.TYPE_ACCELEROMETER) Offset(-event.values[0]/6f, event.values[1]/6f)
                else {
                    SensorManager.getRotationMatrixFromVector(matrix, event.values)
                    SensorManager.getOrientation(matrix, angles)
                    Offset(angles[2], -angles[1])
                }
                if(baseline == null) baseline = value
                val relative = value - baseline!!
                val target = Offset(relative.x.coerceIn(-1f, 1f), relative.y.coerceIn(-1f, 1f))
                output.value += (target-output.value)*.14f
            }
        }
        fun stop() { manager.unregisterListener(listener); baseline = null; output.value = Offset.Zero }
        fun start() { if(enabled && sensor != null) manager.registerListener(listener, sensor, SensorManager.SENSOR_DELAY_UI) }
        val observer = LifecycleEventObserver { _, event ->
            when(event) { Lifecycle.Event.ON_RESUME -> start(); Lifecycle.Event.ON_PAUSE -> stop(); else -> Unit }
        }
        owner.lifecycle.addObserver(observer)
        if(owner.lifecycle.currentState.isAtLeast(Lifecycle.State.RESUMED)) start()
        onDispose { owner.lifecycle.removeObserver(observer); stop() }
    }
    return output
}
