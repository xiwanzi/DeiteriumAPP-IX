param(
    [Parameter(Mandatory = $true)]
    [string]$ApkPath,
    [string]$AndroidSdk = "",
    [switch]$Json
)

$ErrorActionPreference = "Stop"

function Get-LocalPropertyFromAncestors {
    param(
        [string]$StartDir,
        [string]$Name
    )

    $current = Get-Item -LiteralPath $StartDir
    if (-not $current.PSIsContainer) {
        $current = $current.Directory
    }

    while ($null -ne $current) {
        $file = Join-Path $current.FullName "local.properties"
        if (Test-Path -LiteralPath $file) {
            foreach ($line in Get-Content -LiteralPath $file) {
                if ($line -match "^\s*$([regex]::Escape($Name))\s*=\s*(.+?)\s*$") {
                    return ($Matches[1] -replace "\\:", ":" -replace "\\\\", "\")
                }
            }
        }
        $current = $current.Parent
    }
    return $null
}

function Find-Aapt {
    param(
        [string]$SdkPath,
        [string]$StartDir
    )

    $candidates = @()
    if ($SdkPath) { $candidates += $SdkPath }
    $localSdk = Get-LocalPropertyFromAncestors -StartDir $StartDir -Name "sdk.dir"
    if ($localSdk) { $candidates += $localSdk }
    if ($env:ANDROID_HOME) { $candidates += $env:ANDROID_HOME }
    if ($env:ANDROID_SDK_ROOT) { $candidates += $env:ANDROID_SDK_ROOT }

    foreach ($candidate in ($candidates | Where-Object { $_ } | Select-Object -Unique)) {
        $buildToolsDir = Join-Path $candidate "build-tools"
        if (-not (Test-Path -LiteralPath $buildToolsDir)) {
            continue
        }

        $aapt = Get-ChildItem -LiteralPath $buildToolsDir -Directory |
            Sort-Object -Property Name -Descending |
            ForEach-Object {
                $path = Join-Path $_.FullName "aapt.exe"
                if (Test-Path -LiteralPath $path) { Get-Item -LiteralPath $path }
            } |
            Select-Object -First 1

        if ($aapt) {
            return [pscustomobject]@{
                Path = $aapt.FullName
                Sdk = $candidate
            }
        }
    }

    return $null
}

function Get-RegexGroup {
    param(
        [string]$Text,
        [string]$Pattern,
        [string]$Group
    )
    $match = [regex]::Match($Text, $Pattern)
    if ($match.Success) {
        return $match.Groups[$Group].Value
    }
    return $null
}

$apk = Get-Item -LiteralPath $ApkPath
if (-not $apk.FullName.EndsWith(".apk", [StringComparison]::OrdinalIgnoreCase)) {
    throw "Expected an .apk file: $($apk.FullName)"
}

$aapt = Find-Aapt -SdkPath $AndroidSdk -StartDir $apk.Directory.FullName
if (-not $aapt) {
    throw "Could not find aapt.exe. Provide -AndroidSdk or set sdk.dir, ANDROID_HOME, or ANDROID_SDK_ROOT."
}

$badging = & $aapt.Path dump badging $apk.FullName 2>&1
if ($LASTEXITCODE -ne 0) {
    throw "aapt failed: $($badging -join "`n")"
}

$badgingText = $badging -join "`n"
$packageLine = ($badging | Where-Object { $_ -match "^package:" } | Select-Object -First 1)
$normalized = $apk.FullName -replace "\\", "/"
$variantPath = $null
if ($normalized -match "/build/outputs/apk/(.+)/[^/]+\.apk$") {
    $variantPath = $Matches[1]
}

$result = [pscustomobject]@{
    ApkPath = $apk.FullName
    AaptPath = $aapt.Path
    SdkPath = $aapt.Sdk
    SizeBytes = $apk.Length
    LastWriteTime = $apk.LastWriteTime
    VariantPath = $variantPath
    PackageName = Get-RegexGroup -Text $packageLine -Pattern "name='(?<value>[^']+)'" -Group "value"
    VersionCode = Get-RegexGroup -Text $packageLine -Pattern "versionCode='(?<value>[^']+)'" -Group "value"
    VersionName = Get-RegexGroup -Text $packageLine -Pattern "versionName='(?<value>[^']*)'" -Group "value"
    SdkVersion = Get-RegexGroup -Text $badgingText -Pattern "sdkVersion:'(?<value>[^']+)'" -Group "value"
    TargetSdkVersion = Get-RegexGroup -Text $badgingText -Pattern "targetSdkVersion:'(?<value>[^']+)'" -Group "value"
    ApplicationLabel = Get-RegexGroup -Text $badgingText -Pattern "application-label:'(?<value>[^']*)'" -Group "value"
    Notes = @(
        if ($variantPath -and $variantPath -match "release") { "Release APK detected; confirm signing expectations before production handoff." }
        if ($variantPath -and $variantPath -match "debug") { "Debug APK detected; suitable for test/update install when signatures match." }
    )
}

if ($Json) {
    $result | ConvertTo-Json -Depth 5
} else {
    $result | Format-List
}
