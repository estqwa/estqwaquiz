# PowerShell script for Trivia API cluster
# This script runs multiple instances of the application on different ports
# to demonstrate WebSocket clustering using Redis.

# Check if docker containers are running
Write-Host "Checking dependencies (PostgreSQL, Redis)..."
if (Test-Path .\docker-up.ps1) {
    # Run our custom Docker script
    .\docker-up.ps1
} else {
    Write-Host "Warning: docker-up.ps1 not found. Containers may not be running!" -ForegroundColor Yellow
}

# Variables
$BASE_PORT = 8080
$NUM_INSTANCES = 3

# Create temporary configuration for each instance
for ($i = 0; $i -lt $NUM_INSTANCES; $i++) {
    $PORT = $BASE_PORT + $i
    $INSTANCE_ID = "instance_$i"
    $CONFIG_DIR = "config_$i"
    
    # Create configuration directory if it doesn't exist
    if (-not (Test-Path $CONFIG_DIR)) {
        New-Item -ItemType Directory -Force -Path $CONFIG_DIR | Out-Null
    }
    
    # Copy main config
    Copy-Item "config/config.yaml" -Destination "$CONFIG_DIR/config.yaml"
    
    # Modify port and instance ID
    (Get-Content "$CONFIG_DIR/config.yaml") -replace "port: `"8080`"", "port: `"$PORT`"" | 
    Set-Content "$CONFIG_DIR/config.yaml"
    
    (Get-Content "$CONFIG_DIR/config.yaml") -replace "instanceID: `"`"", "instanceID: `"$INSTANCE_ID`"" | 
    Set-Content "$CONFIG_DIR/config.yaml"
    
    Write-Host "Created configuration for instance $i (port: $PORT, ID: $INSTANCE_ID)"
}

# Start instances in different PowerShell windows
for ($i = 0; $i -lt $NUM_INSTANCES; $i++) {
    $CONFIG_DIR = "config_$i"
    $PORT = $BASE_PORT + $i
    
    # Create command
    $cmd = "Write-Host 'Starting instance $i on port $PORT...'; "
    $cmd += "`$env:CONFIG_PATH='$CONFIG_DIR/config.yaml'; "
    $cmd += "go run cmd/api/main.go"
    
    # Start new PowerShell window for each instance
    Start-Process powershell -ArgumentList "-NoExit", "-Command", $cmd
    
    Write-Host "Started instance $i on port $PORT"
    # Small delay between startups
    Start-Sleep -Seconds 2
}

Write-Host "Cluster started! $NUM_INSTANCES instances running."
Write-Host "To stop the cluster, close all PowerShell windows and run: docker stop trivia-postgres trivia-redis" 