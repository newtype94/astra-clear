module github.com/interbank-netting/cosmos

go 1.21

require (
	github.com/cosmos/cosmos-sdk v0.50.1
	github.com/cosmos/ibc-go/v8 v8.0.0
	github.com/cometbft/cometbft v0.38.2
	github.com/cometbft/cometbft-db v0.9.1
	github.com/spf13/cast v1.5.1
	github.com/spf13/cobra v1.8.0
	github.com/spf13/viper v1.17.0
)

require (
	cosmossdk.io/api v0.7.2
	cosmossdk.io/core v0.11.0
	cosmossdk.io/depinject v1.0.0-alpha.4
	cosmossdk.io/log v1.2.1
	cosmossdk.io/math v1.2.0
	cosmossdk.io/store v1.0.1
	cosmossdk.io/tools/rosetta v0.2.1
	cosmossdk.io/x/evidence v0.1.0
	cosmossdk.io/x/feegrant v0.1.0
	cosmossdk.io/x/nft v0.1.0
	cosmossdk.io/x/upgrade v0.1.1
	github.com/cosmos/cosmos-proto v1.0.0-beta.3
	github.com/cosmos/gogoproto v1.4.11
	github.com/golang/protobuf v1.5.3
	github.com/gorilla/mux v1.8.1
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	google.golang.org/genproto/googleapis/api v0.0.0-20231120223509-83a465c0220f
	google.golang.org/grpc v1.59.0
	google.golang.org/protobuf v1.31.0
)