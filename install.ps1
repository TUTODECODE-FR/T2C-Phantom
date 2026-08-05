# Script d'installation de service T2C-Phantom (Windows)
# Doit être exécuté en tant qu'Administrateur dans PowerShell

$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Error "❌ Veuillez lancer cette console PowerShell en tant qu'Administrateur."
    exit
}

Write-Host "========================================="
Write-Host " Installation du Service Windows T2C-Phantom "
Write-Host "========================================="

# 1. Dossier d'installation
$installDir = "C:\Program Files\T2C-Phantom"
if (-not (Test-Path $installDir)) {
    New-Item -Path $installDir -ItemType Directory | Out-Null
}

# 2. Copie du binaire
$binPath = Join-Path $installDir "t2c.exe"
if (Test-Path "t2c-windows-v2.exe") {
    Copy-Item "t2c-windows-v2.exe" $binPath -Force
} elseif (Test-Path "t2c.exe") {
    Copy-Item "t2c.exe" $binPath -Force
} else {
    Write-Warning "⚠️ Binaire Windows non trouvé localement. Veuillez placer 't2c.exe' dans $installDir manuellement."
}

# 3. Création du service via Tâche Planifiée (Recommandé pour les démons Go standards sous Windows)
$taskName = "T2C-Phantom-Daemon"
$taskAction = New-ScheduledTaskAction -Execute $binPath -Argument "start --metrics-port 2112"
$taskTrigger = New-ScheduledTaskTrigger -AtStartup
$taskSettings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 1)

# Enregistrement de la tâche planifiée sous le compte système "LocalSystem"
Register-ScheduledTask -TaskName $taskName -Action $taskAction -Trigger $taskTrigger -Settings $taskSettings -User "SYSTEM" -Force | Out-Null

# Démarrage de la tâche
Start-ScheduledTask -TaskName $taskName

Write-Host "✅ Tâche démarrée en arrière-plan !"
Write-Host "L'application tournera en tâche de fond à chaque démarrage de Windows."
