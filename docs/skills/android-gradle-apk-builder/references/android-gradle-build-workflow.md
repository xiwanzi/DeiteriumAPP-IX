# Android Gradle Build Workflow

## Purpose

Use this reference when a project is not a simple single-module `app` project, when versions are indirect, when product flavors exist, or when artifact selection is unclear.

## Discovery Order

1. Find the Gradle root:
   - Prefer the nearest ancestor containing `settings.gradle` or `settings.gradle.kts`.
   - If no settings file exists, accept a directory with an Android `build.gradle(.kts)` as a single-module root.
2. Prefer the wrapper:
   - Windows: `gradlew.bat`.
   - Unix-like shells: `./gradlew`.
   - Read `gradle/wrapper/gradle-wrapper.properties` to identify the Gradle distribution.
3. Find Android modules:
   - Application modules usually apply `com.android.application`.
   - Library modules usually apply `com.android.library` and should not be delivered as APKs.
   - Version catalog aliases or convention plugins may hide the plugin id; if direct detection fails, inspect Gradle tasks.
4. Read environment:
   - `local.properties` `sdk.dir`.
   - `ANDROID_HOME`.
   - `ANDROID_SDK_ROOT`.
   - Installed `platforms/android-*` and `build-tools/*`.

## Task Selection

Use `gradlew tasks --all --console=plain` when variants are unclear.

Common outputs:

- `assembleDebug`: normal debug APK.
- `assembleRelease`: release APK; may require signing.
- `assemble<Flavor>Debug`: flavored debug APK.
- `assemble<Flavor>Release`: flavored release APK; may require signing.
- `bundleRelease` or `bundle<Flavor>Release`: Android App Bundle.

Default choices:

- For direct install/testing: debug APK.
- For Play Console or distribution requiring AAB: bundle task.
- For production APK: release APK only after signing expectations are known.

## Version Source

Before changing versions, locate the real source:

- `versionCode = 5` or `versionCode 5`: direct module value.
- `versionName = "1.2.0"` or `versionName "1.2.0"`: direct module value.
- `versionCode = project.property(...)`: likely `gradle.properties`.
- `versionCode = libs.versions...`: likely `gradle/libs.versions.toml`.
- `versionCode = computeVersionCode()` or Git/CI-derived values: dynamic; do not guess.

If the source is indirect, edit the source file, not the generated or consuming expression. If multiple application modules exist, update only the module being delivered.

## Artifact Discovery

Do not assume output filenames. Scan these directories from the Gradle root:

- APK: `**/build/outputs/apk/**/*.apk`
- AAB: `**/build/outputs/bundle/**/*.aab`

Rank candidates by:

1. Requested module.
2. Requested variant/build type/flavor.
3. Newest modification time.

Verify APK metadata with `aapt dump badging` before handoff. For AAB, verify with bundletool if available, or at minimum report that APK metadata verification does not apply to AAB.

## Delivery Report

A complete handoff includes:

- Project root.
- Module and task.
- Build command.
- Artifact path.
- Package name.
- `versionCode`.
- `versionName`.
- Any install caveat, such as downgrade risk, signature mismatch, or unsigned release.
