[CmdletBinding()]
param(
    [string]$InstallDir = (Join-Path $env:USERPROFILE ".gopanel"),
    [ValidateRange(1, 65535)]
    [int]$Port = 5470,
    [string]$User = "admin",
    [string]$Password = "",
    [string]$SafeEnter = "",
    [string]$ApiBaseUrl = "https://gopanel.cn",
    [string]$PackagePath = "",
    [switch]$NoStart
)

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

function Write-Info([string]$Message) {
    Write-Host "[INFO] $Message" -ForegroundColor Cyan
}

function New-RandomToken([int]$Bytes = 8) {
    $buffer = New-Object byte[] $Bytes
    [Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($buffer)
    return ([BitConverter]::ToString($buffer)).Replace("-", "").ToLowerInvariant()
}

function Quote-Yaml([string]$Value) {
    return '"' + $Value.Replace('\', '\\').Replace('"', '\"') + '"'
}

function Stop-GoPanel([string]$ExecutablePath) {
    Get-CimInstance Win32_Process -Filter "Name = 'gopanel.exe'" -ErrorAction SilentlyContinue |
        Where-Object { $_.ExecutablePath -eq $ExecutablePath } |
        ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }
}

function Resolve-Package([string]$TargetDir) {
    if ($PackagePath) {
        $resolved = (Resolve-Path -LiteralPath $PackagePath).Path
        return @{ Path = $resolved; Temporary = $false; Version = "local" }
    }
    $upgradeUrl = $ApiBaseUrl.TrimEnd('/') + "/api/panel/upgrade"
    $query = "versionCode=0&version=0.0.0&os=windows&arch=amd64&appBrand=GoPanel&source=github"
    Write-Info "正在获取 Windows AMD64 最新版本"
    $release = Invoke-RestMethod -Uri "$upgradeUrl`?$query" -Method Get
    if (-not $release.downloadUrl) {
        throw "版本接口未返回 Windows 下载地址"
    }
    $archive = Join-Path $TargetDir ([IO.Path]::GetFileName([string]$release.downloadUrl))
    Invoke-WebRequest -Uri ([string]$release.downloadUrl) -OutFile $archive
    return @{ Path = $archive; Temporary = $true; Version = [string]$release.latestVersionName }
}

if (-not [Environment]::Is64BitOperatingSystem) {
    throw "GoPanel 当前仅支持 64 位 Windows"
}
$osVersion = [Environment]::OSVersion.Version
if ($osVersion.Major -lt 10 -or ($osVersion.Major -eq 10 -and $osVersion.Build -lt 17763)) {
    throw "AI 原生终端需要 Windows 10 1809（Build 17763）或更高版本"
}

$InstallDir = [IO.Path]::GetFullPath($InstallDir)
$executable = Join-Path $InstallDir "gopanel.exe"
$configFile = Join-Path $InstallDir "conf.yaml"
$isUpgrade = Test-Path -LiteralPath $configFile
$backupExists = $false
$workDir = Join-Path ([IO.Path]::GetTempPath()) ("gopanel-install-" + [Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $workDir -Force | Out-Null

try {
    $package = Resolve-Package $workDir
    $extractDir = Join-Path $workDir "package"
    New-Item -ItemType Directory -Path $extractDir -Force | Out-Null
    Write-Info "正在解压 $($package.Path)"
    if ($package.Path.EndsWith(".zip", [StringComparison]::OrdinalIgnoreCase)) {
        Expand-Archive -LiteralPath $package.Path -DestinationPath $extractDir -Force
    } else {
        tar.exe -xzf $package.Path -C $extractDir
        if ($LASTEXITCODE -ne 0) {
            throw "安装包解压失败"
        }
    }
    $sourceExecutable = Get-ChildItem -Path $extractDir -Filter "gopanel.exe" -Recurse -File | Select-Object -First 1
    if (-not $sourceExecutable) {
        throw "安装包中未找到 gopanel.exe"
    }

    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Stop-GoPanel $executable
    if (Test-Path -LiteralPath $executable) {
        Copy-Item -LiteralPath $executable -Destination "$executable.bak" -Force
        $backupExists = $true
    }
    Copy-Item -LiteralPath $sourceExecutable.FullName -Destination $executable -Force

    if (-not $isUpgrade) {
        if (-not $Password) { $Password = New-RandomToken }
        if (-not $SafeEnter) { $SafeEnter = New-RandomToken }
        $initYaml = @(
            "base_dir: $(Quote-Yaml $InstallDir)"
            "port: $Port"
            "user: $(Quote-Yaml $User)"
            "password: $(Quote-Yaml $Password)"
            "safe_enter: $(Quote-Yaml $SafeEnter)"
        ) -join [Environment]::NewLine
        [IO.File]::WriteAllText((Join-Path $InstallDir "init.yaml"), $initYaml, [Text.UTF8Encoding]::new($false))
    }

    $startupDir = Join-Path $env:APPDATA "Microsoft\Windows\Start Menu\Programs\Startup"
    $startupScript = Join-Path $startupDir "GoPanel.cmd"
    $startupContent = "@echo off`r`ncd /d `"$InstallDir`"`r`nstart `"GoPanel`" /min `"$executable`" --config `"$configFile`"`r`n"
    [IO.File]::WriteAllText($startupScript, $startupContent, [Text.ASCIIEncoding]::new())

    if (-not $NoStart) {
        Start-Process -FilePath $executable -ArgumentList @("--config", "`"$configFile`"") -WorkingDirectory $InstallDir -WindowStyle Hidden
    }

    Write-Host ""
    Write-Host "GoPanel Windows 本地开发环境安装完成" -ForegroundColor Green
    Write-Host "版本: $($package.Version)"
    Write-Host "安装目录: $InstallDir"
    if (-not $isUpgrade) {
        Write-Host "用户名: $User"
        Write-Host "密码: $Password"
        Write-Host "访问地址: http://127.0.0.1:$Port/$SafeEnter"
    } else {
        Write-Host "原有配置已保留，访问地址与账号不变"
    }
    Write-Host "说明: Windows 当前支持本地开发与 AI 终端；GPC、gp-agent 宿主管理仍仅支持 Linux/macOS。"
}
catch {
    Stop-GoPanel $executable
    if ($backupExists -and (Test-Path -LiteralPath "$executable.bak")) {
        Copy-Item -LiteralPath "$executable.bak" -Destination $executable -Force
    } elseif (Test-Path -LiteralPath $executable) {
        Remove-Item -LiteralPath $executable -Force -ErrorAction SilentlyContinue
    }
    throw
}
finally {
    Remove-Item -LiteralPath $workDir -Recurse -Force -ErrorAction SilentlyContinue
}
