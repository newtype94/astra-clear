@echo off
REM Stop Besu Networks
echo Stopping Hyperledger Besu networks...

REM Navigate to docker directory
cd /d "%~dp0\..\docker"

REM Stop the networks using docker-compose
docker-compose -f docker-compose.besu.yml down

echo Besu networks stopped successfully!