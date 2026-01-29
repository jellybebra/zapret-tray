Write-Host "- Creating a task in Task Scheduler..."

$taskXml = @"
<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Date>$(Get-Date -Format "yyyy-MM-ddTHH:mm:ss")</Date>
    <Author>$env:USERNAME</Author>
  </RegistrationInfo>
  <Triggers>
    <LogonTrigger>
      <Enabled>true</Enabled>
      <UserId>$env:USERDOMAIN\$env:USERNAME</UserId>
    </LogonTrigger>
  </Triggers>
  <Principals>
    <Principal id="Author">
      <UserId>$env:USERDOMAIN\$env:USERNAME</UserId>
      <LogonType>InteractiveToken</LogonType>
      <RunLevel>LeastPrivilege</RunLevel>
    </Principal>
  </Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>true</StopIfGoingOnBatteries>
    <AllowHardTerminate>true</AllowHardTerminate>
    <StartWhenAvailable>false</StartWhenAvailable>
    <RunOnlyIfNetworkAvailable>true</RunOnlyIfNetworkAvailable>
    <IdleSettings>
      <StopOnIdleEnd>true</StopOnIdleEnd>
      <RestartOnIdle>false</RestartOnIdle>
    </IdleSettings>
    <AllowStartOnDemand>true</AllowStartOnDemand>
    <Enabled>true</Enabled>
    <Hidden>false</Hidden>
    <RunOnlyIfIdle>false</RunOnlyIfIdle>
    <WakeToRun>false</WakeToRun>
    <ExecutionTimeLimit>PT72H</ExecutionTimeLimit>
    <Priority>7</Priority>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>powershell.exe</Command>
      <Arguments>-WindowStyle Hidden -Command "irm https://raw.githubusercontent.com/jellybebra/auto-zapret/refs/heads/main/auto_zapret.ps1 | iex"</Arguments>
    </Exec>
  </Actions>
</Task>
"@

$tempXmlPath = [System.IO.Path]::GetTempFileName() + ".xml"
$taskXml | Out-File -FilePath $tempXmlPath -Encoding Unicode

schtasks /Create /XML $tempXmlPath /TN "AutoZapret" /F

# Clean up the temp file
Remove-Item -Path $tempXmlPath -Force

# Create folder if doesn't exist
New-Item -Path "$env:LOCALAPPDATA\AutoZapret" -ItemType Directory -Force | Out-Null

# ----------------------------------

Write-Host "- Searching the latest installed version"

$targetPath = "$env:LOCALAPPDATA\AutoZapret"

$latestVersion = $null

# Получаем список папок, начинающихся с "zapret-discord-youtube-"
$folders = Get-ChildItem -Path $targetPath -Directory -Filter "zapret-discord-youtube-*"

# Обрабатываем список для правильной сортировки
$sortedVersions = $folders | ForEach-Object {
    # Убираем префикс, оставляем только версию (например "1.9.0b")
    $verStr = $_.Name -replace '^zapret-discord-youtube-', ''

    # Пытаемся разбить на цифровую часть и суффикс для умной сортировки
    # Группа 1: Цифры (1.9.0), Группа 3: Хвост (b)
    if ($verStr -match '^(\d+(\.\d+)*)(.*)$') {
        [PSCustomObject]@{
            Original = $verStr
            Main     = [System.Version]$matches[1] # Преобразуем в реальную версию для сравнения чисел
            Suffix   = $matches[3]                 # Буквы (beta, b и т.д.)
        }
    }
} | Sort-Object -Property Main, Suffix

# Берем последнюю (самую свежую) версию
if ($sortedVersions) {
    $latestVersion = ($sortedVersions | Select-Object -Last 1).Original
}

if ($latestVersion) {
    Write-Output "Latest installed version: $latestVersion"
} else {
    Write-Output "Latest installed version not found"
}

# -------------------------------

Write-Host "- Looking up the latest version on Github"

# Получаем последней версию с GitHub
$repo = "Flowseal/zapret-discord-youtube"
try {
    $releaseInfo = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest" -ErrorAction Stop
    $githubVersion = $releaseInfo.tag_name
    Write-Host "The latest version is: $githubVersion"
}
catch {
    Write-Warning "Coudln't check updates on GitHub."
    exit
}

# Если локальная версия не установлена ($null) ИЛИ локальная версия отличается от последней версии на GitHub
if ($null -eq $latestVersion -or $latestVersion -ne $githubVersion) {

    # Подгружаем сборку для красивых окон Windows
    Add-Type -AssemblyName System.Windows.Forms

    # Формируем текст сообщения
    $msgText = "A new version $githubVersion for zapret is available. Would you like to update?"
    $msgTitle = "AutoZapret"

    # Настраиваем кнопки и иконку
    $buttons = [System.Windows.Forms.MessageBoxButtons]::YesNo
    $icon = [System.Windows.Forms.MessageBoxIcon]::Information

    # Показываем окно и записываем результат нажатия
    $result = [System.Windows.Forms.MessageBox]::Show($msgText, $msgTitle, $buttons, $icon)

    # Обработка выбора пользователя
    if ($result -eq [System.Windows.Forms.DialogResult]::No) {
        Exit
    }
} else {
    # значит последняя версия уже установлена
    exit
}

# --------------------

Write-Host "- Downloading the latest release..."

# Find the ZIP asset URL
$zipAsset = $releaseInfo.assets | Where-Object { $_.name -like "*.zip" } | Select-Object -First 1
if (-not $zipAsset) {
    Write-Error "No ZIP archive found in the latest release."
    exit 1
}
$zipUrl = $zipAsset.browser_download_url

# Define paths
$tempZipPath = Join-Path $env:TEMP "zapret-discord-youtube.zip"
$extractPath = "$env:LOCALAPPDATA\AutoZapret\zapret-discord-youtube-$githubVersion"

# Download the ZIP
Invoke-WebRequest -Uri $zipUrl -OutFile $tempZipPath

# Extract the ZIP
Expand-Archive -Path $tempZipPath -DestinationPath $extractPath -Force

# Clean up temp file
Remove-Item $tempZipPath -Force

# Run service.bat
$batPath = Join-Path $extractPath "service.bat"
if (Test-Path $batPath) {
    Start-Process -FilePath $batPath -NoNewWindow -Wait
} else {
    Write-Error "service.bat not found in $extractPath"
}