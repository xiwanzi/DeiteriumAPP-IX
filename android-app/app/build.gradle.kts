plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
    id("org.jetbrains.kotlin.plugin.compose")
}

android {
    namespace = "com.deuterium.app"
    compileSdk = 36
    buildToolsVersion = "35.0.0"

    defaultConfig {
        applicationId = "com.deuterium.app"
        minSdk = 26
        targetSdk = 35
        versionCode = 7
        versionName = "1.0.5"
    }

    buildTypes {
        debug {
            buildConfigField("String", "HTTP_BASE_URL", "\"http://deuterium.s.odn.cc:80/api/v1/\"")
            buildConfigField("String", "CHAT_WS_URL", "\"ws://deuterium.s.odn.cc:80/api/v1/chat/ws\"")
            buildConfigField("boolean", "CANARY_UI", "false")
            manifestPlaceholders["cleartextTraffic"] = "true"
            manifestPlaceholders["appLabel"] = "Deuterium"
        }
        create("canary") {
            initWith(getByName("debug"))
            applicationIdSuffix = ".canary"
            versionNameSuffix = "-canary.26063"
            matchingFallbacks += listOf("debug")
            buildConfigField("String", "HTTP_BASE_URL", "\"http://deuterium.s.odn.cc:80/api/v1/\"")
            buildConfigField("String", "CHAT_WS_URL", "\"ws://deuterium.s.odn.cc:80/api/v1/chat/ws\"")
            buildConfigField("boolean", "CANARY_UI", "true")
            manifestPlaceholders["cleartextTraffic"] = "true"
            manifestPlaceholders["appLabel"] = "Deuterium Canary"
        }
        release {
            buildConfigField("String", "HTTP_BASE_URL", "\"https://deuterium.s.odn.cc/api/v1/\"")
            buildConfigField("String", "CHAT_WS_URL", "\"wss://deuterium.s.odn.cc/api/v1/chat/ws\"")
            buildConfigField("boolean", "CANARY_UI", "false")
            manifestPlaceholders["cleartextTraffic"] = "false"
            manifestPlaceholders["appLabel"] = "Deuterium"
        }
    }

    buildFeatures {
        buildConfig = true
        compose = true
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlin {
        compilerOptions {
            jvmTarget.set(org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17)
        }
    }
}

androidComponents {
    onVariants(selector().withBuildType("canary")) { variant ->
        variant.outputs.forEach { output ->
            output.versionCode.set(26063)
            output.versionName.set("1.0.4-canary.26063")
        }
    }
}

dependencies {
    implementation("androidx.activity:activity-compose:1.9.3")
    implementation("androidx.compose.animation:animation:1.10.3")
    implementation("androidx.compose.foundation:foundation:1.10.3")
    implementation("androidx.compose.material:material-icons-extended-android:1.6.8")
    implementation("androidx.compose.material3:material3-android:1.4.0")
    implementation("androidx.compose.ui:ui:1.10.3")
    implementation("androidx.compose.ui:ui-graphics:1.10.3")
    implementation("androidx.compose.ui:ui-tooling-preview:1.10.3")
    implementation("com.google.code.gson:gson:2.11.0")
    implementation("com.squareup.okhttp3:okhttp:4.12.0")
    implementation("com.squareup.retrofit2:converter-gson:2.11.0")
    implementation("com.squareup.retrofit2:retrofit:2.11.0")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.8.1")

    debugImplementation("androidx.compose.ui:ui-tooling:1.10.3")
    debugImplementation("androidx.compose.ui:ui-test-manifest:1.10.3")

    testImplementation("junit:junit:4.13.2")
}
