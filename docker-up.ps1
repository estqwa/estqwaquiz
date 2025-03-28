# Start Redis and PostgreSQL containers for Trivia API
Write-Host "Starting PostgreSQL and Redis containers..."

# Check if Docker is running
try {
    docker --version
    Write-Host "Docker is running"
} catch {
    Write-Host "ERROR: Docker does not appear to be running. Please start Docker Desktop first." -ForegroundColor Red
    exit 1
}

# Run docker commands individually instead of using docker-compose
# Start PostgreSQL
Write-Host "Starting PostgreSQL container..."
docker run --name trivia-postgres -p 5432:5432 -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=123456 -e POSTGRES_DB=trivia_db -d postgres:13-alpine

# Start Redis
Write-Host "Starting Redis container..."
docker run --name trivia-redis -p 6379:6379 -d redis:alpine

Write-Host "Containers are now running!" -ForegroundColor Green
Write-Host "To stop them use: docker stop trivia-postgres trivia-redis" 