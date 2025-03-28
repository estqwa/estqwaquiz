# Скрипт для сборки и запуска тестового клиента

# Проверка, что Go установлен
if (-not (Get-Command "go" -ErrorAction SilentlyContinue)) {
    Write-Host "❌ Ошибка: Go не установлен. Пожалуйста, установите Go." -ForegroundColor Red
    exit 1
}

# Переходим в корневую директорию проекта
$rootDir = $PSScriptRoot
Set-Location $rootDir

# Скачиваем зависимости
Write-Host "📥 Скачивание зависимостей..." -ForegroundColor Cyan
go mod download

# Проверяем наличие ошибок
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Ошибка при скачивании зависимостей." -ForegroundColor Red
    exit 1
}

# Сборка приложения
Write-Host "🔨 Сборка приложения..." -ForegroundColor Cyan
go build -o bin/bottest.exe ./cmd

# Проверяем наличие ошибок
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Ошибка при сборке приложения." -ForegroundColor Red
    exit 1
}

Write-Host "✅ Приложение успешно собрано: bin/bottest.exe" -ForegroundColor Green
Write-Host ""
Write-Host "🚀 Пример запуска:" -ForegroundColor Yellow
Write-Host "  .\bin\bottest.exe run --token=YOUR_TOKEN --quiz=1 --bots=3" -ForegroundColor Yellow
Write-Host ""
Write-Host "🔍 Для получения справки по командам:" -ForegroundColor Yellow
Write-Host "  .\bin\bottest.exe --help" -ForegroundColor Yellow
Write-Host "  .\bin\bottest.exe run --help" -ForegroundColor Yellow 