@echo off
REM Interbank Netting Engine - Development Environment Setup Script

echo 🌌 Interbank Netting Engine - Development Setup
echo ================================================

REM Check if Go is installed
go version >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Go is not installed. Please install Go first.
    exit /b 1
) else (
    echo ✅ Go is installed
)

REM Check if Docker is installed
docker --version >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Docker is not installed. Please install Docker first.
    exit /b 1
) else (
    echo ✅ Docker is installed
)

REM Check if Docker Compose is installed
docker-compose --version >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Docker Compose is not installed. Please install Docker Compose first.
    exit /b 1
) else (
    echo ✅ Docker Compose is installed
)

echo.
echo 🚀 Setting up Cosmos Hub...
cd cosmos

REM Install Go dependencies
echo Installing Go dependencies...
go mod tidy

REM Build the binary
echo Building interbank-nettingd...
go build -o build/interbank-nettingd.exe ./cmd/interbank-nettingd

REM Initialize the chain (if not already done)
if not exist "%USERPROFILE%\.interbank-netting" (
    echo Initializing blockchain...
    build\interbank-nettingd.exe init mynode --chain-id interbank-netting
    
    REM Create validator key
    echo Creating validator key...
    build\interbank-nettingd.exe keys add validator --keyring-backend test
    
    REM Add genesis account
    echo Adding genesis account...
    for /f "tokens=*" %%i in ('build\interbank-nettingd.exe keys show validator -a --keyring-backend test') do set VALIDATOR_ADDR=%%i
    build\interbank-nettingd.exe add-genesis-account %VALIDATOR_ADDR% 1000000000stake
    
    REM Create genesis transaction
    echo Creating genesis transaction...
    build\interbank-nettingd.exe gentx validator 1000000stake --chain-id interbank-netting --keyring-backend test
    
    REM Collect genesis transactions
    echo Collecting genesis transactions...
    build\interbank-nettingd.exe collect-gentxs
    
    echo ✅ Blockchain initialized successfully!
) else (
    echo ✅ Blockchain already initialized
)

cd ..

echo.
echo 🔗 Setting up Hyperledger Besu networks...

REM Start Besu networks
echo Starting Besu networks...
call scripts\start-besu-networks.bat

echo.
echo 🎉 Development environment setup complete!
echo.
echo 📋 Next steps:
echo 1. Start Cosmos Hub: cd cosmos ^&^& build\interbank-nettingd.exe start
echo 2. Run tests: cd cosmos ^&^& go test ./...
echo 3. Check network status:
echo    - Cosmos Hub: curl http://localhost:26657/status
echo    - Bank A: curl -X POST --data "{\"jsonrpc\":\"2.0\",\"method\":\"eth_blockNumber\",\"params\":[],\"id\":1}" http://localhost:8545
echo    - Bank B: curl -X POST --data "{\"jsonrpc\":\"2.0\",\"method\":\"eth_blockNumber\",\"params\":[],\"id\":1}" http://localhost:8547
echo.
echo 📚 For more information, see README.md