$ErrorActionPreference = "Stop"

$repo = "tanaka-mambinge/redan-plugin"
$releaseBase = "https://github.com/$repo/releases/latest/download"

$architecture = $env:PROCESSOR_ARCHITECTURE.ToUpperInvariant()
switch ($architecture) {
    "AMD64" { $asset = "redan_windows_amd64.exe" }
    "ARM64" { $asset = "redan_windows_arm64.exe" }
    default { throw "Unsupported Redan CLI architecture: $architecture." }
}

$cacheDir = Join-Path $HOME ".redan\bin"
$null = New-Item -ItemType Directory -Force -Path $cacheDir
$null = & attrib +h $cacheDir 2>$null
$target = Join-Path $cacheDir "redan.exe"
$versionFile = Join-Path $cacheDir "redan.version"
$cachedVersion = $null
if ((Test-Path -LiteralPath $target -PathType Leaf) -and (Test-Path -LiteralPath $versionFile -PathType Leaf)) {
    $cachedVersion = (Get-Content -LiteralPath $versionFile | Select-Object -First 1).Trim()
}

function Get-LatestReleaseTag {
    $location = $null
    try {
        $head = Invoke-WebRequest -Uri "$releaseBase/$asset" -Method Head -MaximumRedirection 0 -ErrorAction Stop
        $location = $head.Headers.Location
    } catch {
        if ($null -ne $_.Exception.Response) { $location = $_.Exception.Response.Headers["Location"] }
        if ([string]::IsNullOrWhiteSpace($location)) { throw "Unable to check the latest Redan CLI release: $($_.Exception.Message)" }
    }
    $segments = $location.TrimEnd("/") -split "/"
    $downloadIndex = [Array]::IndexOf($segments, "download")
    if (($downloadIndex -lt 0) -or (($downloadIndex + 1) -ge $segments.Length)) { throw "Unable to determine the latest Redan CLI release from the redirect." }
    return $segments[$downloadIndex + 1]
}

$latestTag = Get-LatestReleaseTag
if ((Test-Path -LiteralPath $target -PathType Leaf) -and ($cachedVersion -eq $latestTag)) { Write-Output $target; exit 0 }

$guid = [Guid]::NewGuid().ToString("N")
$binaryTemp = Join-Path $cacheDir ".redan-cli-$guid.tmp"
$checksumsTemp = Join-Path $cacheDir ".redan-checksums-$guid.tmp"
$versionTemp = Join-Path $cacheDir ".redan-version-$guid.tmp"
try {
    Invoke-WebRequest -Uri "$releaseBase/$asset" -OutFile $binaryTemp
    Invoke-WebRequest -Uri "$releaseBase/checksums.txt" -OutFile $checksumsTemp
    $checksumLine = Get-Content -LiteralPath $checksumsTemp | Where-Object { $_ -match "(^|\s)$([regex]::Escape($asset))$" } | Select-Object -First 1
    if ([string]::IsNullOrWhiteSpace($checksumLine)) { throw "No checksum found for $asset." }
    $expected = ($checksumLine -split "\s+")[0].ToLowerInvariant()
    $actual = (Get-FileHash -Algorithm SHA256 -LiteralPath $binaryTemp).Hash.ToLowerInvariant()
    if ($actual -ne $expected) { throw "Checksum verification failed for $asset." }
    Move-Item -Force -LiteralPath $binaryTemp -Destination $target
    Set-Content -LiteralPath $versionTemp -Value $latestTag -NoNewline
    Move-Item -Force -LiteralPath $versionTemp -Destination $versionFile
    Write-Output $target
} finally {
    Remove-Item -Force -ErrorAction SilentlyContinue -LiteralPath $binaryTemp, $checksumsTemp, $versionTemp
}
