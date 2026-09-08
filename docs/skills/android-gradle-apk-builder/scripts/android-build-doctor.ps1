param(
    [string]$ProjectDir = ".",
    [string]$Variant = "",
    [switch]$ListTasks,
    [switch]$Json
)

$ErrorActionPreference = "Stop"

function Resolve-ExistingPath {
    param([string]$Path)
    return (Resolve-Path -LiteralPath $Path).Path
}

function Find-GradleRoot {
    param([string]$StartDir)

    $current = Get-Item -LiteralPath $StartDir
    if (-not $current.PSIsContainer) {
        $current = $current.Directory
    }

    while ($null -ne $current) {
        $settingsKts = Join-Path $current.FullName "settings.gradle.kts"
        $settingsGroovy = Join-Path $current.FullName "settings.gradle"
        if ((Test-Path -LiteralPath $settingsKts) -or (Test-Path -LiteralPath $settingsGroovy)) {
            return $current.FullName
        }
        $current = $current.Parent
    }

    $start = Get-Item -LiteralPath $StartDir
    if (-not $start.PSIsContainer) {
        $start = $start.Directory
    }
    if ((Test-Path -LiteralPath (Join-Path $start.FullName "build.gradle.kts")) -or
        (Test-Path -LiteralPath (Join-Path $start.FullName "build.gradle"))) {
        return $start.FullName
    }

    throw "Could not find a Gradle root from '$StartDir'."
}

function Get-FirstMatch {
    param(
        [string]$Text,
        [string[]]$Patterns
    )

    foreach ($pattern in $Patterns) {
        $match = [regex]::Match($Text, $pattern, [Text.RegularExpressions.RegexOptions]::Multiline)
        if ($match.Success) {
            return $match.Groups[1].Value
        }
    }
    return $null
}

function Get-LocalProperty {
    param(
        [string]$Root,
        [string]$Name
    )

    $file = Join-Path $Root "local.properties"
    if (-not (Test-Path -LiteralPath $file)) {
        return $null
    }

    foreach ($line in Get-Content -LiteralPath $file) {
        if ($line -match "^\s*$([regex]::Escape($Name))\s*=\s*(.+?)\s*$") {
            return ($Matches[1] -replace "\\:", ":" -replace "\\\\", "\")
        }
    }
    return $null
}

function Get-WrapperInfo {
    param([string]$Root)

    $gradlewBat = Join-Path $Root "gradlew.bat"
    $gradlew = Join-Path $Root "gradlew"
    $wrapperProperties = Join-Path $Root "gradle\wrapper\gradle-wrapper.properties"
    $distribution = $null
    $version = $null

    if (Test-Path -LiteralPath $wrapperProperties) {
        $distribution = Get-FirstMatch -Text (Get-Content -Raw -LiteralPath $wrapperProperties) -Patterns @("distributionUrl\s*=\s*(.+)")
        if ($distribution -and $distribution -match "gradle-([0-9][^-]+)-") {
            $version = $Matches[1]
        }
    }

    return [pscustomobject]@{
        GradlewBat = if (Test-Path -LiteralPath $gradlewBat) { $gradlewBat } else { $null }
        Gradlew = if (Test-Path -LiteralPath $gradlew) { $gradlew } else { $null }
        DistributionUrl = $distribution
        GradleVersion = $version
    }
}

function Get-SdkInfo {
    param([string]$Root)

    $candidates = @()
    $localSdk = Get-LocalProperty -Root $Root -Name "sdk.dir"
    if ($localSdk) { $candidates += [pscustomobject]@{ Source = "local.properties"; Path = $localSdk } }
    if ($env:ANDROID_HOME) { $candidates += [pscustomobject]@{ Source = "ANDROID_HOME"; Path = $env:ANDROID_HOME } }
    if ($env:ANDROID_SDK_ROOT) { $candidates += [pscustomobject]@{ Source = "ANDROID_SDK_ROOT"; Path = $env:ANDROID_SDK_ROOT } }

    $selected = $candidates | Where-Object { Test-Path -LiteralPath $_.Path } | Select-Object -First 1
    $platforms = @()
    $buildTools = @()

    if ($selected) {
        $platformDir = Join-Path $selected.Path "platforms"
        $buildToolsDir = Join-Path $selected.Path "build-tools"
        if (Test-Path -LiteralPath $platformDir) {
            $platforms = @(Get-ChildItem -LiteralPath $platformDir -Directory | Select-Object -ExpandProperty Name)
        }
        if (Test-Path -LiteralPath $buildToolsDir) {
            $buildTools = @(Get-ChildItem -LiteralPath $buildToolsDir -Directory | Select-Object -ExpandProperty Name)
        }
    }

    return [pscustomobject]@{
        Selected = $selected
        Candidates = $candidates
        InstalledPlatforms = $platforms
        InstalledBuildTools = $buildTools
    }
}

function ConvertTo-ModulePath {
    param(
        [string]$Root,
        [string]$BuildFile
    )

    $moduleDir = Split-Path -Parent $BuildFile
    $rootFull = (Resolve-Path -LiteralPath $Root).Path.TrimEnd("\", "/")
    $moduleFull = (Resolve-Path -LiteralPath $moduleDir).Path.TrimEnd("\", "/")
    if ($moduleFull.Equals($rootFull, [StringComparison]::OrdinalIgnoreCase)) {
        $relative = "."
    } else {
        $rootUri = [Uri]($rootFull + [IO.Path]::DirectorySeparatorChar)
        $moduleUri = [Uri]($moduleFull + [IO.Path]::DirectorySeparatorChar)
        $relative = [Uri]::UnescapeDataString($rootUri.MakeRelativeUri($moduleUri).ToString()).TrimEnd("/")
    }
    if ($relative -eq ".") {
        return ":"
    }
    return ":" + (($relative -split "[\\/]+") -join ":")
}

function Find-BuildFiles {
    param([string]$Root)

    Get-ChildItem -LiteralPath $Root -Recurse -File -ErrorAction SilentlyContinue |
        Where-Object {
            if ($_.Name -notin @("build.gradle", "build.gradle.kts")) {
                return $false
            }
            $path = ($_.FullName -replace "/", "\")
            return ($path -notmatch "\\(\.git|\.gradle[^\\]*|build|\.idea|node_modules)\\")
        }
}

function Get-AndroidModuleInfo {
    param(
        [string]$Root,
        [System.IO.FileInfo]$BuildFile
    )

    $text = Get-Content -Raw -LiteralPath $BuildFile.FullName
    $mentionsApplication = $text -match "com\.android\.application" -or $text -match "android\.application"
    $mentionsLibrary = $text -match "com\.android\.library" -or $text -match "android\.library"
    $applicationApplyFalse = $text -match "(com\.android\.application|android\.application)[^\r\n]*apply\s+false"
    $libraryApplyFalse = $text -match "(com\.android\.library|android\.library)[^\r\n]*apply\s+false"
    $isApplication = $mentionsApplication -and -not $applicationApplyFalse
    $isLibrary = $mentionsLibrary -and -not $libraryApplyFalse
    $hasAndroidBlock = $text -match "(?m)^\s*android\s*\{"

    if (-not ($isApplication -or $isLibrary -or $hasAndroidBlock)) {
        return $null
    }

    return [pscustomobject]@{
        ModulePath = ConvertTo-ModulePath -Root $Root -BuildFile $BuildFile.FullName
        BuildFile = $BuildFile.FullName
        Dsl = if ($BuildFile.Extension -eq ".kts") { "kotlin" } else { "groovy" }
        IsApplication = [bool]$isApplication
        IsLibrary = [bool]$isLibrary
        HasAndroidBlock = [bool]$hasAndroidBlock
        HasProductFlavors = [bool]($text -match "productFlavors\s*\{")
        Namespace = Get-FirstMatch -Text $text -Patterns @(
            "^\s*namespace\s*=\s*[""']([^""']+)[""']",
            "^\s*namespace\s+[""']([^""']+)[""']"
        )
        ApplicationId = Get-FirstMatch -Text $text -Patterns @(
            "^\s*applicationId\s*=\s*[""']([^""']+)[""']",
            "^\s*applicationId\s+[""']([^""']+)[""']"
        )
        CompileSdk = Get-FirstMatch -Text $text -Patterns @(
            "^\s*compileSdk\s*=\s*([0-9]+)",
            "^\s*compileSdkVersion\s*\(?\s*([0-9]+)"
        )
        MinSdk = Get-FirstMatch -Text $text -Patterns @(
            "^\s*minSdk\s*=\s*([0-9]+)",
            "^\s*minSdkVersion\s*\(?\s*([0-9]+)"
        )
        TargetSdk = Get-FirstMatch -Text $text -Patterns @(
            "^\s*targetSdk\s*=\s*([0-9]+)",
            "^\s*targetSdkVersion\s*\(?\s*([0-9]+)"
        )
        VersionCode = Get-FirstMatch -Text $text -Patterns @(
            "^\s*versionCode\s*=\s*([0-9]+)",
            "^\s*versionCode\s+([0-9]+)"
        )
        VersionName = Get-FirstMatch -Text $text -Patterns @(
            "^\s*versionName\s*=\s*[""']([^""']+)[""']",
            "^\s*versionName\s+[""']([^""']+)[""']"
        )
    }
}

function Find-ApkCandidates {
    param(
        [string]$Root,
        [string]$Variant
    )

    $files = Get-ChildItem -LiteralPath $Root -Recurse -File -Filter "*.apk" -ErrorAction SilentlyContinue |
        Where-Object { ($_.FullName -replace "\\", "/") -match "/build/outputs/apk/" }

    $files | ForEach-Object {
        $normalized = $_.FullName -replace "\\", "/"
        $variantPath = $null
        if ($normalized -match "/build/outputs/apk/(.+)/[^/]+\.apk$") {
            $variantPath = $Matches[1]
        }
        $matchesVariant = $false
        if ([string]::IsNullOrWhiteSpace($Variant)) {
            $matchesVariant = $true
        } elseif ($variantPath -and $variantPath.Replace("/", "").ToLowerInvariant().Contains($Variant.ToLowerInvariant())) {
            $matchesVariant = $true
        } elseif ($_.Name.ToLowerInvariant().Contains($Variant.ToLowerInvariant())) {
            $matchesVariant = $true
        }

        [pscustomobject]@{
            Path = $_.FullName
            VariantPath = $variantPath
            SizeBytes = $_.Length
            LastWriteTime = $_.LastWriteTime
            MatchesRequestedVariant = $matchesVariant
        }
    } | Sort-Object -Property @{ Expression = "MatchesRequestedVariant"; Descending = $true }, @{ Expression = "LastWriteTime"; Descending = $true }
}

function Get-GradleTasks {
    param(
        [string]$Root,
        [object]$Wrapper
    )

    $gradleCommand = $null
    if ($Wrapper.GradlewBat) {
        $gradleCommand = $Wrapper.GradlewBat
    } elseif ($Wrapper.Gradlew) {
        $gradleCommand = $Wrapper.Gradlew
    } elseif (Get-Command gradle -ErrorAction SilentlyContinue) {
        $gradleCommand = "gradle"
    }

    if (-not $gradleCommand) {
        return [pscustomobject]@{
            Command = $null
            Error = "No Gradle wrapper or system gradle command found."
            AssembleTasks = @()
            BundleTasks = @()
        }
    }

    Push-Location $Root
    try {
        $output = & $gradleCommand tasks --all --console=plain 2>&1
        $assemble = @()
        $bundle = @()
        foreach ($line in $output) {
            if ($line -match "^\s*(?:(\S+):)?(assemble\S*)\s*(?:-|$)") {
                $taskName = $Matches[2]
                if ($taskName -notmatch "(AndroidTest|UnitTest)") {
                    $assemble += if ($Matches[1]) { "$($Matches[1]):$taskName" } else { $taskName }
                }
            }
            if ($line -match "^\s*(?:(\S+):)?(bundle\S*)\s*(?:-|$)") {
                $taskName = $Matches[2]
                if ($taskName -notmatch "(Classes|Resources|Manifest|Listing|Ide|Apks|Jar|Assets|Dex)") {
                    $bundle += if ($Matches[1]) { "$($Matches[1]):$taskName" } else { $taskName }
                }
            }
        }
        return [pscustomobject]@{
            Command = "$gradleCommand tasks --all --console=plain"
            Error = $null
            AssembleTasks = $assemble | Sort-Object -Unique
            BundleTasks = $bundle | Sort-Object -Unique
        }
    } catch {
        return [pscustomobject]@{
            Command = "$gradleCommand tasks --all --console=plain"
            Error = $_.Exception.Message
            AssembleTasks = @()
            BundleTasks = @()
        }
    } finally {
        Pop-Location
    }
}

$resolvedProjectDir = Resolve-ExistingPath -Path $ProjectDir
$root = Find-GradleRoot -StartDir $resolvedProjectDir
$wrapper = Get-WrapperInfo -Root $root
$sdk = Get-SdkInfo -Root $root
$modules = @(Find-BuildFiles -Root $root | ForEach-Object { Get-AndroidModuleInfo -Root $root -BuildFile $_ } | Where-Object { $null -ne $_ })
$tasks = if ($ListTasks) { Get-GradleTasks -Root $root -Wrapper $wrapper } else { $null }
$apkCandidates = @(Find-ApkCandidates -Root $root -Variant $Variant)

$result = [pscustomobject]@{
    ProjectRoot = $root
    Wrapper = $wrapper
    Sdk = $sdk
    AndroidModules = $modules
    Tasks = $tasks
    ApkCandidates = $apkCandidates
    Notes = @(
        if (-not $wrapper.GradlewBat -and -not $wrapper.Gradlew) { "No Gradle wrapper found; prefer project wrapper when available." }
        if (-not $sdk.Selected) { "No usable Android SDK path found from local.properties, ANDROID_HOME, or ANDROID_SDK_ROOT." }
        if (($modules | Where-Object { $_.IsApplication }).Count -eq 0) { "No direct com.android.application module detected; inspect convention plugins or Gradle tasks." }
        if (-not $ListTasks) { "Run again with -ListTasks to discover assemble/bundle tasks." }
    )
}

if ($Json) {
    $result | ConvertTo-Json -Depth 8
} else {
    $result | Format-List
}
