# Руководство по развертыванию Trivia API

## Содержание

1. [Требования](#требования)
2. [Локальное развертывание](#локальное-развертывание)
3. [Развертывание на сервере](#развертывание-на-сервере)
4. [Docker-развертывание](#docker-развертывание)
5. [Kubernetes-развертывание](#kubernetes-развертывание)
6. [Конфигурация](#конфигурация)
7. [Проверка работоспособности](#проверка-работоспособности)
8. [Мониторинг](#мониторинг)
9. [Масштабирование](#масштабирование)
10. [Резервное копирование](#резервное-копирование)
11. [Частые проблемы и их решение](#частые-проблемы-и-их-решение)

## Требования

### Минимальные системные требования

* **Процессор**: 2+ ядра
* **ОЗУ**: 4+ ГБ
* **Хранилище**: 20+ ГБ SSD
* **Сеть**: 100+ Мбит/с

### Программные зависимости

* Go 1.16+
* PostgreSQL 12+
* Redis 6+
* Node.js 14+ (для фронтенд-части)
* Docker 20+ (опционально)
* Kubernetes 1.19+ (опционально)

### Необходимые инструменты

* Git
* curl/wget
* migrate (golang-migrate)
* make (опционально)

## Локальное развертывание

### Шаг 1: Клонирование репозитория

```bash
git clone https://github.com/yourusername/trivia-api.git
cd trivia-api
```

### Шаг 2: Настройка окружения

Создайте файл `.env` в корне проекта:

```
# Основные настройки
APP_ENV=development
APP_PORT=8080
APP_SECRET=your-secret-key-at-least-32-chars
APP_URL=http://localhost:8080

# База данных PostgreSQL
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=trivia_api
DB_SSLMODE=disable
DB_MAX_CONNECTIONS=20
DB_MAX_IDLE_CONNECTIONS=5

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# JWT
JWT_SECRET=your-jwt-secret-at-least-32-chars
JWT_ACCESS_EXPIRATION=15m
JWT_REFRESH_EXPIRATION=168h
JWT_ISSUER=trivia-api

# Другие настройки
LOG_LEVEL=debug
CORS_ALLOWED_ORIGINS=http://localhost:3000
```

### Шаг 3: Запуск базы данных

В PowerShell:

```powershell
# Запуск PostgreSQL через Docker
docker run -d --name trivia-postgres -p 5432:5432 -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=trivia_api postgres:13

# Запуск Redis через Docker
docker run -d --name trivia-redis -p 6379:6379 redis:6
```

### Шаг 4: Применение миграций

```powershell
# Установка инструмента миграции (если не установлен)
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Применение миграций
$env:PATH += ";$env:GOPATH\bin"
migrate -path migrations -database "postgresql://postgres:postgres@localhost:5432/trivia_api?sslmode=disable" up
```

### Шаг 5: Сборка и запуск

```powershell
# Сборка
go build -o trivia-api.exe cmd/api/main.go

# Запуск
$env:APP_ENV = "development"
.\trivia-api.exe
```

### Шаг 6: Проверка работоспособности

```powershell
curl http://localhost:8080/api/health
```

Должен вернуться статус 200 OK с JSON-ответом, содержащим информацию о работоспособности сервиса.

## Развертывание на сервере

### Шаг 1: Подготовка сервера

```bash
# Обновление системы
sudo apt update && sudo apt upgrade -y

# Установка необходимых пакетов
sudo apt install -y git make curl build-essential

# Установка Go
wget https://golang.org/dl/go1.18.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.18.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Установка PostgreSQL
sudo apt install -y postgresql postgresql-contrib
sudo systemctl enable postgresql
sudo systemctl start postgresql

# Установка Redis
sudo apt install -y redis-server
sudo systemctl enable redis-server
sudo systemctl start redis-server
```

### Шаг 2: Настройка PostgreSQL

```bash
# Вход в PostgreSQL
sudo -u postgres psql

# В консоли PostgreSQL:
CREATE DATABASE trivia_api;
CREATE USER trivia_user WITH ENCRYPTED PASSWORD 'secure_password';
GRANT ALL PRIVILEGES ON DATABASE trivia_api TO trivia_user;
\q

# Применение миграций
cd /path/to/trivia-api
migrate -path migrations -database "postgresql://trivia_user:secure_password@localhost:5432/trivia_api?sslmode=disable" up
```

### Шаг 3: Развертывание приложения

```bash
# Клонирование репозитория
git clone https://github.com/yourusername/trivia-api.git
cd trivia-api

# Настройка переменных окружения
cp .env.example .env
# Отредактируйте .env с помощью вашего любимого редактора

# Сборка
go build -o trivia-api cmd/api/main.go

# Запуск
./trivia-api
```

### Шаг 4: Настройка systemd-сервиса

Создайте файл `/etc/systemd/system/trivia-api.service`:

```
[Unit]
Description=Trivia API
After=network.target postgresql.service redis-server.service

[Service]
User=ubuntu
Group=ubuntu
WorkingDirectory=/path/to/trivia-api
ExecStart=/path/to/trivia-api/trivia-api
Restart=always
RestartSec=5
StandardOutput=syslog
StandardError=syslog
SyslogIdentifier=trivia-api
Environment=APP_ENV=production
EnvironmentFile=/path/to/trivia-api/.env

[Install]
WantedBy=multi-user.target
```

Активация и запуск сервиса:

```bash
sudo systemctl daemon-reload
sudo systemctl enable trivia-api
sudo systemctl start trivia-api
sudo systemctl status trivia-api
```

## Docker-развертывание

### Шаг 1: Создание Dockerfile

Создайте файл `Dockerfile` в корне проекта:

```dockerfile
# Сборка приложения
FROM golang:1.18-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o trivia-api cmd/api/main.go

# Финальный образ
FROM alpine:3.16
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/trivia-api .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/.env.example ./.env

# Экспозиция порта
EXPOSE 8080

# Запуск приложения
CMD ["./trivia-api"]
```

### Шаг 2: Создание docker-compose.yml

Создайте файл `docker-compose.yml` в корне проекта:

```yaml
version: '3.8'

services:
  api:
    build: .
    container_name: trivia-api
    restart: always
    ports:
      - "8080:8080"
    depends_on:
      - postgres
      - redis
    environment:
      - APP_ENV=production
      - APP_PORT=8080
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_USER=postgres
      - DB_PASSWORD=postgres
      - DB_NAME=trivia_api
      - REDIS_HOST=redis
      - REDIS_PORT=6379
    volumes:
      - ./logs:/app/logs
    networks:
      - trivia-network

  postgres:
    image: postgres:13
    container_name: trivia-postgres
    restart: always
    environment:
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=postgres
      - POSTGRES_DB=trivia_api
    volumes:
      - postgres-data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    networks:
      - trivia-network

  redis:
    image: redis:6
    container_name: trivia-redis
    restart: always
    volumes:
      - redis-data:/data
    ports:
      - "6379:6379"
    networks:
      - trivia-network
  
  # Для запуска миграций при первом развертывании
  migrations:
    image: migrate/migrate
    networks:
      - trivia-network
    volumes:
      - ./migrations:/migrations
    command: ["-path", "/migrations", "-database", "postgresql://postgres:postgres@postgres:5432/trivia_api?sslmode=disable", "up"]
    depends_on:
      - postgres

networks:
  trivia-network:
    driver: bridge

volumes:
  postgres-data:
  redis-data:
```

### Шаг 3: Запуск с Docker Compose

В PowerShell:

```powershell
# Сборка и запуск
docker-compose up -d

# Проверка состояния
docker-compose ps

# Просмотр логов
docker-compose logs -f api
```

## Kubernetes-развертывание

### Шаг 1: Подготовка конфигурационных файлов

#### namespace.yaml

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: trivia-api
```

#### configmap.yaml

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: trivia-api-config
  namespace: trivia-api
data:
  APP_ENV: "production"
  APP_PORT: "8080"
  DB_HOST: "postgres-service"
  DB_PORT: "5432"
  DB_NAME: "trivia_api"
  DB_SSLMODE: "disable"
  REDIS_HOST: "redis-service"
  REDIS_PORT: "6379"
  LOG_LEVEL: "info"
  CORS_ALLOWED_ORIGINS: "*"
```

#### secret.yaml

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: trivia-api-secrets
  namespace: trivia-api
type: Opaque
data:
  APP_SECRET: base64_encoded_app_secret
  DB_USER: base64_encoded_db_user
  DB_PASSWORD: base64_encoded_db_password
  REDIS_PASSWORD: base64_encoded_redis_password
  JWT_SECRET: base64_encoded_jwt_secret
```

#### postgres-deployment.yaml

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgres
  namespace: trivia-api
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:13
        ports:
        - containerPort: 5432
        env:
        - name: POSTGRES_USER
          valueFrom:
            secretKeyRef:
              name: trivia-api-secrets
              key: DB_USER
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: trivia-api-secrets
              key: DB_PASSWORD
        - name: POSTGRES_DB
          value: trivia_api
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
      volumes:
      - name: postgres-storage
        persistentVolumeClaim:
          claimName: postgres-pvc
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: postgres-pvc
  namespace: trivia-api
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
---
apiVersion: v1
kind: Service
metadata:
  name: postgres-service
  namespace: trivia-api
spec:
  selector:
    app: postgres
  ports:
  - port: 5432
    targetPort: 5432
  type: ClusterIP
```

#### redis-deployment.yaml

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
  namespace: trivia-api
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
      - name: redis
        image: redis:6
        ports:
        - containerPort: 6379
        volumeMounts:
        - name: redis-storage
          mountPath: /data
      volumes:
      - name: redis-storage
        persistentVolumeClaim:
          claimName: redis-pvc
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: redis-pvc
  namespace: trivia-api
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 5Gi
---
apiVersion: v1
kind: Service
metadata:
  name: redis-service
  namespace: trivia-api
spec:
  selector:
    app: redis
  ports:
  - port: 6379
    targetPort: 6379
  type: ClusterIP
```

#### api-deployment.yaml

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: trivia-api
  namespace: trivia-api
spec:
  replicas: 2
  selector:
    matchLabels:
      app: trivia-api
  template:
    metadata:
      labels:
        app: trivia-api
    spec:
      containers:
      - name: trivia-api
        image: your-registry/trivia-api:latest
        ports:
        - containerPort: 8080
        envFrom:
        - configMapRef:
            name: trivia-api-config
        - secretRef:
            name: trivia-api-secrets
        resources:
          limits:
            cpu: "1"
            memory: "1Gi"
          requests:
            cpu: "500m"
            memory: "512Mi"
        livenessProbe:
          httpGet:
            path: /api/health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /api/health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: trivia-api-service
  namespace: trivia-api
spec:
  selector:
    app: trivia-api
  ports:
  - port: 80
    targetPort: 8080
  type: ClusterIP
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: trivia-api-ingress
  namespace: trivia-api
  annotations:
    kubernetes.io/ingress.class: nginx
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  rules:
  - host: api.trivia-app.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: trivia-api-service
            port:
              number: 80
```

### Шаг 2: Применение конфигурации Kubernetes

```bash
# Применение конфигурации
kubectl apply -f namespace.yaml
kubectl apply -f configmap.yaml
kubectl apply -f secret.yaml
kubectl apply -f postgres-deployment.yaml
kubectl apply -f redis-deployment.yaml
kubectl apply -f api-deployment.yaml

# Проверка состояния подов
kubectl get pods -n trivia-api

# Применение миграций
kubectl run migrations --image=migrate/migrate --namespace=trivia-api --restart=Never -- -path=/migrations -database "postgresql://[username]:[password]@postgres-service:5432/trivia_api?sslmode=disable" up
```

## Конфигурация

### Переменные окружения

| Переменная | Описание | Значение по умолчанию |
|------------|----------|----------------------|
| APP_ENV | Окружение (development, testing, production) | development |
| APP_PORT | Порт HTTP-сервера | 8080 |
| APP_SECRET | Секретный ключ приложения | - |
| APP_URL | Базовый URL приложения | http://localhost:8080 |
| DB_HOST | Хост PostgreSQL | localhost |
| DB_PORT | Порт PostgreSQL | 5432 |
| DB_USER | Пользователь PostgreSQL | postgres |
| DB_PASSWORD | Пароль PostgreSQL | postgres |
| DB_NAME | Имя базы данных | trivia_api |
| DB_SSLMODE | Режим SSL для PostgreSQL | disable |
| DB_MAX_CONNECTIONS | Максимальное количество соединений | 20 |
| DB_MAX_IDLE_CONNECTIONS | Максимальное количество простаивающих соединений | 5 |
| REDIS_HOST | Хост Redis | localhost |
| REDIS_PORT | Порт Redis | 6379 |
| REDIS_PASSWORD | Пароль Redis | - |
| REDIS_DB | Номер базы данных Redis | 0 |
| JWT_SECRET | Секрет для подписи JWT-токенов | - |
| JWT_ACCESS_EXPIRATION | Время жизни access-токена | 15m |
| JWT_REFRESH_EXPIRATION | Время жизни refresh-токена | 168h |
| JWT_ISSUER | Издатель JWT-токенов | trivia-api |
| LOG_LEVEL | Уровень логирования | info |
| CORS_ALLOWED_ORIGINS | Разрешенные источники для CORS | * |

### Конфигурация через файл конфигурации

Помимо переменных окружения, приложение также может быть настроено через JSON-файл `config.json`:

```json
{
  "app": {
    "env": "production",
    "port": 8080,
    "secret": "your-secret-key",
    "url": "http://api.example.com"
  },
  "database": {
    "host": "postgres",
    "port": 5432,
    "user": "postgres",
    "password": "postgres",
    "name": "trivia_api",
    "sslmode": "disable",
    "max_connections": 20,
    "max_idle_connections": 5
  },
  "redis": {
    "host": "redis",
    "port": 6379,
    "password": "",
    "db": 0
  },
  "jwt": {
    "secret": "your-jwt-secret",
    "access_expiration": "15m",
    "refresh_expiration": "168h",
    "issuer": "trivia-api"
  },
  "log": {
    "level": "info",
    "format": "json"
  },
  "cors": {
    "allowed_origins": ["https://app.example.com"]
  },
  "websocket": {
    "shard_count": 4,
    "max_clients_per_shard": 2000,
    "enable_metrics": true
  }
}
```

## Проверка работоспособности

### Endpoint мониторинга здоровья

Приложение предоставляет endpoint для проверки работоспособности:

```
GET /api/health
```

Ответ должен содержать:

```json
{
  "status": "ok",
  "version": "1.0.0",
  "timestamp": "2023-03-26T12:34:56Z",
  "uptime": "3h 45m 12s",
  "services": {
    "database": {
      "status": "ok",
      "latency_ms": 5
    },
    "redis": {
      "status": "ok",
      "latency_ms": 2
    }
  }
}
```

### Комплексная проверка здоровья

```bash
# Проверка HTTP endpoint
curl http://localhost:8080/api/health

# Проверка WebSocket соединения
wscat -c ws://localhost:8080/ws
```

## Мониторинг

### Prometheus и Grafana

Приложение экспортирует метрики в формате Prometheus по адресу:

```
GET /metrics
```

#### Экспортируемые метрики

* `http_requests_total{method="GET|POST|PUT|DELETE", path="/api/...", status="200|404|500"}` - Общее количество HTTP-запросов
* `http_request_duration_seconds{method="GET|POST|PUT|DELETE", path="/api/..."}` - Время обработки HTTP-запросов
* `active_websocket_connections` - Количество активных WebSocket-соединений
* `websocket_messages_sent_total` - Количество отправленных WebSocket-сообщений
* `websocket_messages_received_total` - Количество полученных WebSocket-сообщений
* `db_queries_total{status="success|error"}` - Общее количество запросов к базе данных
* `db_query_duration_seconds` - Время выполнения запросов к базе данных

### Настройка Grafana Dashboard

Вы можете найти готовый JSON-конфигурацию для Grafana Dashboard в директории `/docs/grafana/trivia-api-dashboard.json`.

## Масштабирование

### Горизонтальное масштабирование

Приложение поддерживает горизонтальное масштабирование. Для обеспечения синхронизации между экземплярами используется Redis Pub/Sub.

Для масштабирования:

1. В Kubernetes увеличьте количество реплик:
   ```bash
   kubectl scale deployment trivia-api -n trivia-api --replicas=4
   ```

2. Для Docker Compose:
   ```bash
   docker-compose up -d --scale api=4
   ```

### Рекомендации по масштабированию PostgreSQL

* Для продакшена рекомендуется использовать репликацию PostgreSQL
* Настройте read-replicas для операций чтения
* Используйте connection pooling с pgBouncer

### Рекомендации по масштабированию Redis

* Для продакшена рекомендуется использовать Redis Sentinel или Redis Cluster
* Настройте Redis для сохранения данных на диск
* Мониторьте использование памяти

## Резервное копирование

### Резервное копирование PostgreSQL

```bash
# Ежедневное резервное копирование
pg_dump -h localhost -U postgres -d trivia_api -F c -f /backup/trivia_api_$(date +%Y%m%d).dump

# Восстановление из резервной копии
pg_restore -h localhost -U postgres -d trivia_api -c /backup/trivia_api_20230326.dump
```

### Резервное копирование Redis

```bash
# Включение RDB снэпшотов в конфигурации Redis
# В файле redis.conf:
save 900 1
save 300 10
save 60 10000

# Копирование файла dump.rdb
cp /var/lib/redis/dump.rdb /backup/redis_$(date +%Y%m%d).rdb
```

### Автоматизация резервного копирования

Вы можете настроить автоматическое резервное копирование с помощью cron:

```bash
# Ежедневное резервное копирование PostgreSQL в 2:00
0 2 * * * pg_dump -h localhost -U postgres -d trivia_api -F c -f /backup/trivia_api_$(date +%Y%m%d).dump

# Ежедневное резервное копирование Redis в 3:00
0 3 * * * cp /var/lib/redis/dump.rdb /backup/redis_$(date +%Y%m%d).rdb && gzip /backup/redis_$(date +%Y%m%d).rdb
```

## Частые проблемы и их решение

### Проблемы с подключением к базе данных

**Проблема**: Приложение не может подключиться к PostgreSQL.

**Решение**:
1. Проверьте, что PostgreSQL запущен:
   ```bash
   systemctl status postgresql
   ```
2. Проверьте настройки подключения в .env:
   ```
   DB_HOST=localhost
   DB_PORT=5432
   DB_USER=postgres
   DB_PASSWORD=postgres
   DB_NAME=trivia_api
   ```
3. Проверьте, что база данных существует:
   ```bash
   psql -U postgres -c "SELECT 1 FROM pg_database WHERE datname='trivia_api'"
   ```

### Проблемы с миграциями

**Проблема**: Ошибки при выполнении миграций.

**Решение**:
1. Проверьте версию migrate:
   ```bash
   migrate -version
   ```
2. Проверьте состояние миграций:
   ```bash
   migrate -path migrations -database "postgresql://postgres:postgres@localhost:5432/trivia_api?sslmode=disable" version
   ```
3. При необходимости выполните принудительную установку версии:
   ```bash
   migrate -path migrations -database "postgresql://postgres:postgres@localhost:5432/trivia_api?sslmode=disable" force VERSION
   ```

### Проблемы с WebSocket

**Проблема**: WebSocket соединения не устанавливаются.

**Решение**:
1. Проверьте, что порт WebSocket открыт:
   ```bash
   netstat -tulpn | grep 8080
   ```
2. Проверьте настройки CORS:
   ```
   CORS_ALLOWED_ORIGINS=*
   ```
3. Проверьте логи на наличие ошибок аутентификации.

### Проблемы с производительностью

**Проблема**: Приложение работает медленно.

**Решение**:
1. Проверьте использование ресурсов:
   ```bash
   top
   htop
   ```
2. Проверьте количество подключений к PostgreSQL:
   ```sql
   SELECT count(*) FROM pg_stat_activity WHERE datname = 'trivia_api';
   ```
3. Проверьте логи на наличие медленных запросов:
   ```bash
   grep "slow query" /path/to/logs/trivia-api.log
   ```
4. Увеличьте количество воркеров в конфигурации. 