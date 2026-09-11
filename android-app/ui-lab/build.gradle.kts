plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
    id("org.jetbrains.kotlin.plugin.compose")
}

android {
    namespace = "com.deuterium.app.uilab"
    compileSdk = 36
    buildToolsVersion = "35.0.0"
    defaultConfig {
        applicationId = "com.deuterium.app.uilab"
        minSdk = 26
        targetSdk = 35
        versionCode = 21100
        versionName = "2.0.11"
        testInstrumentationRunner = providers.gradleProperty("deuteriumTestRunner").orElse("com.deuterium.app.uilab.LiveBackendInstrumentation").get()
        val apiBase = providers.gradleProperty("deuteriumApiBase").orElse("https://47.103.99.34").get()
        require(apiBase.matches(Regex("https?://[A-Za-z0-9.:-]+"))) { "deuteriumApiBase must be an HTTP(S) origin" }
        buildConfigField("String", "API_BASE_URL", "\"$apiBase\"")
    }
    buildFeatures { compose = true; buildConfig = true }
    buildTypes {
        create("performance") {
            initWith(getByName("release"))
            // Keep the installed application's existing certificate for local comparisons.
            signingConfig=signingConfigs.getByName("debug")
            isDebuggable=false
            isMinifyEnabled=true
            isShrinkResources=true
            proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"),"proguard-rules.pro")
            testProguardFiles+=file("proguard-test-rules.pro")
            matchingFallbacks+=listOf("release")
        }
        create("profileCapture") {
            initWith(getByName("performance"))
            isMinifyEnabled=false
            isShrinkResources=false
        }
    }
    testBuildType=providers.gradleProperty("deuteriumTestBuildType").orElse("debug").get()
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
    kotlin { compilerOptions { jvmTarget.set(org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17) } }
}

dependencies {
    testImplementation("junit:junit:4.13.2")
    testImplementation("org.json:json:20240303")
    testImplementation("com.squareup.okhttp3:mockwebserver:4.12.0")
    androidTestImplementation("com.squareup.okhttp3:mockwebserver:4.12.0")
    androidTestImplementation("androidx.compose.ui:ui-test-junit4:1.10.3")
    implementation("com.squareup.okhttp3:okhttp:4.12.0")
    implementation("io.coil-kt.coil3:coil-compose:3.3.0")
    implementation("io.noties.markwon:core:4.6.2")
    implementation("io.noties.markwon:ext-strikethrough:4.6.2")
    implementation("io.noties.markwon:ext-tables:4.6.2")
    implementation("androidx.activity:activity-compose:1.9.3")
    implementation("androidx.compose.animation:animation:1.10.3")
    implementation("androidx.compose.foundation:foundation:1.10.3")
    implementation("androidx.compose.material:material-icons-extended-android:1.6.8")
    implementation("androidx.compose.material3:material3-android:1.4.0")
    implementation("androidx.compose.ui:ui:1.10.3")
    implementation("androidx.compose.ui:ui-graphics:1.10.3")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.8.1")
}
