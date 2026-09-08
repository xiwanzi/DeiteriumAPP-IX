# Android Gradle Troubleshooting

## SDK Location Not Found

Signals:

- `SDK location not found`
- `ANDROID_HOME is not set`
- Missing `local.properties`

Actions:

- Check `local.properties` for `sdk.dir`.
- Check `ANDROID_HOME` and `ANDROID_SDK_ROOT`.
- Do not commit `local.properties` unless the repo already intentionally tracks a portable SDK path.

## compileSdk Not Installed

Signals:

- `Failed to find target with hash string 'android-XX'`
- `Installed Build Tools revision ... is corrupted or missing`

Actions:

- Compare module `compileSdk` with installed `platforms/android-*`.
- Install the missing platform/build tools or point to the intended SDK.
- Do not lower `compileSdk` just to make a local build pass unless the user asked for that compatibility change.

## JDK, Gradle, and AGP Mismatch

Signals:

- `Unsupported class file major version`
- `Android Gradle plugin requires Java`
- `Minimum supported Gradle version is`

Actions:

- Read Gradle wrapper version from `gradle-wrapper.properties`.
- Read AGP version from plugin declarations or version catalog.
- Use a JDK compatible with that AGP/Gradle pair.
- Prefer changing local JDK over changing project Gradle/AGP versions.

## Kotlin Metadata Mismatch

Signals:

- `Module was compiled with an incompatible version of Kotlin`
- `metadata binary version`

Actions:

- Identify Kotlin plugin version and dependency Kotlin metadata version.
- Prefer aligning dependency versions with the project's Kotlin plugin.
- Do not upgrade Kotlin broadly without checking AGP compatibility.

## Dependency Resolution Failure

Signals:

- `Could not resolve`
- `No matching variant`
- Repository timeout or authentication failure.

Actions:

- Separate network/authentication failures from version conflicts.
- Check `repositories` and private Maven credentials without exposing secrets.
- For `No matching variant`, inspect JVM target, AGP attributes, flavor dimensions, and dependency variants.

## Android Resource Linking Failed

Signals:

- `Android resource linking failed`
- Missing attributes such as `android:attr/...`

Actions:

- Check whether dependency versions require a higher `compileSdk`.
- Check generated resources and manifest merges.
- Avoid blind dependency downgrades; identify the dependency requiring the attribute.

## Signing Config Missing

Signals:

- Release task fails around keystore, key alias, or signing properties.
- Release APK exists but is unsigned or not installable as expected.

Actions:

- Build debug for test handoff if release signing is not needed.
- Ask for signing expectations before changing signing config.
- Never invent keystore paths or passwords.

## lintVitalRelease Failure

Signals:

- `lintVitalRelease`
- Release build blocked by lint.

Actions:

- Treat as a release quality failure, not an APK packaging failure.
- Prefer fixing the lint issue.
- Disable lint only with explicit user approval.

## INSTALL_FAILED_VERSION_DOWNGRADE

Signals:

- Device install fails because installed `versionCode` is higher.

Actions:

- Verify APK `versionCode`.
- Increase `versionCode` at its source of truth when the user requested update-install.
- Uninstalling the existing app loses local data and must be explicit.

## INSTALL_FAILED_UPDATE_INCOMPATIBLE

Signals:

- Device install fails due to signing certificate mismatch.

Actions:

- Verify whether the installed app came from a different signing key.
- Use the same signing config, or explicitly uninstall first if data loss is acceptable.
- Do not replace release signing with debug signing for production delivery.
