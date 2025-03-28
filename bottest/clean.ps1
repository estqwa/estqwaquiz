# Скрипт для очистки проекта и сброса к начальному состоянию

# Переходим в корневую директорию проекта
$rootDir = $PSScriptRoot
Set-Location $rootDir

Write-Host "🧹 Начинаю очистку проекта..." -ForegroundColor Cyan

# Удаляем скомпилированные файлы
if (Test-Path "bin") {
    Write-Host "🗑️ Удаление скомпилированных файлов..." -ForegroundColor Yellow
    Remove-Item -Path "bin\*" -Force -Recurse -ErrorAction SilentlyContinue
}

# Удаляем временные файлы Go
Write-Host "🗑️ Удаление временных файлов Go..." -ForegroundColor Yellow
if (Test-Path "pkg") {
    Remove-Item -Path "pkg\*" -Force -Recurse -ErrorAction SilentlyContinue
}

# Инициализируем go.mod заново
Write-Host "🔄 Реинициализация go.mod..." -ForegroundColor Yellow
go mod tidy -v

# Проверяем наличие ошибок
if ($LASTEXITCODE -ne 0) {
    Write-Host "⚠️ Предупреждение при выполнении go mod tidy. Проверьте зависимости." -ForegroundColor Yellow
}

Write-Host "✅ Очистка завершена" -ForegroundColor Green
Write-Host "📝 Теперь можно выполнить .\build.ps1 для сборки проекта" -ForegroundColor Green 