# The isolated instrumentation fixtures install local sessions through this existing private method.
# Production authentication continues through the regular login and registration methods.
-keepclassmembers class com.deuterium.app.uilab.BackendApi {
    private void rememberSession(org.json.JSONObject);
    private okhttp3.OkHttpClient http;
}

# Keep the app's callable surface for on-device equivalence fixtures. Method bodies are optimized;
# unused classes and library code can still be removed. Test APK references use AGP's mapping.
-keepclassmembers,allowoptimization class com.deuterium.app.uilab.** {
    public *;
}
# Instrumentation and the target share these runtime APIs. Preserve their callable ABI while
# optimizing bodies; icon libraries remain eligible for shrinking (the largest unused dependency).
-keep,allowoptimization class kotlin.** { public *; }
# Coroutine test dispatchers subclass runtime jobs; do not finalize their overridable methods.
-keep class kotlinx.coroutines.** { public *; protected *; }
-keep,allowoptimization class androidx.compose.runtime.** { public *; }
-keep,allowoptimization class androidx.compose.ui.** { public *; }
-keep,allowoptimization class androidx.compose.foundation.** { public *; }
-keep,allowoptimization class androidx.compose.material3.** { public *; }
-keep,allowoptimization class androidx.compose.animation.** { public *; }
-keep,allowoptimization class androidx.activity.compose.** { public *; }
-keepclassmembers,allowoptimization class androidx.activity.ComponentActivity { public *; }
-keep,allowoptimization class okhttp3.** { public *; }
-keep,allowoptimization class coil3.** { public *; }
-keep,allowoptimization class androidx.tracing.** { public *; }
-keep,allowoptimization class androidx.collection.** { public *; }
-keep,allowoptimization class com.google.common.util.concurrent.ListenableFuture { public *; }
-keep,allowoptimization class androidx.concurrent.futures.** { public *; }
-keep,allowoptimization class androidx.lifecycle.** { public *; }
-keep,allowoptimization class androidx.savedstate.** { public *; }
