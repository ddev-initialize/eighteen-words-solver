param(
    [string]$InstallDir = "$env:LOCALAPPDATA\Programs\eighteen-words-solver"
)

$ErrorActionPreference = "Stop"
$repo = if ($env:EIGHTEEN_WORDS_REPO) { $env:EIGHTEEN_WORDS_REPO } else { "ddev-initialize/eighteen-words-solver" }
$architecture = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
$arch = if ($architecture -eq "Arm64") { "arm64" } elseif ($architecture -eq "X64") { "amd64" } else { throw "Unsupported architecture: $architecture" }
$asset = "eighteen-words-solver-windows-$arch.zip"
$url = "https://github.com/$repo/releases/latest/download/$asset"
$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString())

try {
    New-Item -ItemType Directory -Path $tempDir | Out-Null
    $archive = Join-Path $tempDir $asset
    Invoke-WebRequest -Uri $url -OutFile $archive
    Expand-Archive -Path $archive -DestinationPath $tempDir

    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    Copy-Item (Join-Path $tempDir "eighteen-words-solver.exe") $InstallDir -Force

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $pathEntries = @($userPath -split ";" | Where-Object { $_ })
    if ($InstallDir -notin $pathEntries) {
        $newPath = (@($pathEntries) + $InstallDir) -join ";"
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        $env:Path = "$env:Path;$InstallDir"
    }

    Write-Host "Installed eighteen-words-solver to $InstallDir"
} finally {
    Remove-Item -Recurse -Force $tempDir -ErrorAction SilentlyContinue
}
