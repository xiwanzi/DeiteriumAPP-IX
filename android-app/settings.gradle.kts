pluginManagement {
    buildscript {
        repositories {
            maven { url=uri("https://maven.aliyun.com/repository/google") }
            google()
            mavenCentral()
        }
        // Kotlin 2.3 metadata requires R8 >= 8.13.19. Keep AGP and runtime UI libraries unchanged.
        dependencies { classpath("com.android.tools:r8:8.13.19") }
    }
    repositories {
        maven {
            url = uri("https://maven.aliyun.com/repository/google")
            name = "AliyunGoogleMirror"
        }
        maven {
            url = uri("https://maven.google.com")
            name = "GoogleMavenDirect"
        }
        maven {
            url = uri("https://redirector.gvt1.com/edgedl/android/maven2/")
            name = "GoogleRedirector"
        }
        google()
        mavenCentral()
        gradlePluginPortal()
        maven {
            url = uri("local-maven")
            name = "ProjectLocalMaven"
        }
    }
}

dependencyResolutionManagement {
    repositoriesMode.set(RepositoriesMode.FAIL_ON_PROJECT_REPOS)
    repositories {
        maven {
            url = uri("https://maven.aliyun.com/repository/google")
            name = "AliyunGoogleMirror"
        }
        maven {
            url = uri("https://maven.google.com")
            name = "GoogleMavenDirect"
        }
        maven {
            url = uri("https://redirector.gvt1.com/edgedl/android/maven2/")
            name = "GoogleRedirector"
        }
        google()
        mavenCentral()
        maven {
            url = uri("local-maven")
            name = "ProjectLocalMaven"
        }
    }
}

rootProject.name = "DeuteriumAPP"
include(":app")
include(":ui-lab")
