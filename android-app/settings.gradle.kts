pluginManagement {
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
