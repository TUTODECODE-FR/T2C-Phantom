@echo off
echo =========================================
echo Installation de T2C Phantom...
echo =========================================

set LATEST_URL=https://github.com/TUTODECODE-FR/T2C-Phantom/releases/latest/download/t2c-windows-amd64.exe
set EXECUTABLE=t2c-windows-amd64.exe

echo Telechargement de l'application depuis GitHub...
curl -L -o %EXECUTABLE% %LATEST_URL%

if exist %EXECUTABLE% (
    echo Telechargement reussi ! Lancement...
    start "" %EXECUTABLE% dashboard
    echo L'interface web s'ouvre sur http://127.0.0.1:9090
    timeout /t 3 >nul
    start http://127.0.0.1:9090
) else (
    echo Erreur lors du telechargement.
    pause
)
