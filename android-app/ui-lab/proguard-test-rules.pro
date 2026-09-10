# AndroidX test tracing references this compile-time annotation without bundling it.
# It is not executable code, and this exception applies only to the instrumentation APK.
-dontwarn com.google.errorprone.annotations.MustBeClosed
