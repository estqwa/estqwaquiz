# Скрипт для получения JWT токена для тестирования

# Параметры
param (
    [string]$url = "http://localhost:8080",
    [string]$username,
    [string]$password
)

# Запрашиваем учетные данные, если не указаны
if (-not $username) {
    $username = Read-Host "Введите имя пользователя (email)"
}

if (-not $password) {
    $securePassword = Read-Host "Введите пароль" -AsSecureString
    $BSTR = [System.Runtime.InteropServices.Marshal]::SecureStringToBSTR($securePassword)
    $password = [System.Runtime.InteropServices.Marshal]::PtrToStringAuto($BSTR)
}

# Формируем JSON для запроса
$body = @{
    email = $username
    password = $password
} | ConvertTo-Json

# Отправляем запрос на авторизацию
try {
    Write-Host "Отправка запроса авторизации на $url/api/auth/login" -ForegroundColor Cyan
    $response = Invoke-RestMethod -Method Post -Uri "$url/api/auth/login" -Body $body -ContentType "application/json" -ErrorAction Stop
    
    # Извлекаем токен
    $token = $response.token
    
    if ($token) {
        Write-Host "✅ Токен успешно получен:" -ForegroundColor Green
        Write-Host $token
        
        # Копируем токен в буфер обмена (если поддерживается)
        if (Get-Command "Set-Clipboard" -ErrorAction SilentlyContinue) {
            $token | Set-Clipboard
            Write-Host "📋 Токен скопирован в буфер обмена" -ForegroundColor Green
        }
        
        # Выводим пример команды запуска ботов
        Write-Host ""
        Write-Host "🚀 Пример запуска ботов с полученным токеном:" -ForegroundColor Yellow
        Write-Host ".\bin\bottest.exe run --token=$token --quiz=1 --bots=3" -ForegroundColor Yellow
    } else {
        Write-Host "❌ Токен не получен в ответе сервера" -ForegroundColor Red
    }
} catch {
    Write-Host "❌ Ошибка при авторизации: $_" -ForegroundColor Red
    Write-Host "Ответ сервера: $($_.Exception.Response)" -ForegroundColor Red
} 