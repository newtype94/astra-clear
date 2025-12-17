@echo off
REM Start Besu Networks for Bank A and Bank B
echo Starting Hyperledger Besu networks...

REM Navigate to docker directory
cd /d "%~dp0\..\docker"

REM Start the networks using docker-compose
docker-compose -f docker-compose.besu.yml up -d

echo Besu networks started successfully!
echo Bank A RPC: http://localhost:8545
echo Bank A WebSocket: ws://localhost:8546
echo Bank B RPC: http://localhost:8547
echo Bank B WebSocket: ws://localhost:8548

REM Wait for networks to be ready
echo Waiting for networks to be ready...
timeout /t 10 /nobreak

echo Besu networks are ready!