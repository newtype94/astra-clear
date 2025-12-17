package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	grpc1 "github.com/gogo/protobuf/grpc"
)

// RegisterCodec registers the necessary x/netting interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgIssueCreditToken{}, "netting/MsgIssueCreditToken", nil)
	cdc.RegisterConcrete(&MsgBurnCreditToken{}, "netting/MsgBurnCreditToken", nil)
	cdc.RegisterConcrete(&MsgTriggerNetting{}, "netting/MsgTriggerNetting", nil)
}

// RegisterInterfaces registers the x/netting interfaces types with the interface registry
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgIssueCreditToken{},
		&MsgBurnCreditToken{},
		&MsgTriggerNetting{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

// RegisterMsgServer registers the msg server
func RegisterMsgServer(s grpc1.Server, srv MsgServer) {
	// TODO: Implement when protobuf is available
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

func init() {
	RegisterCodec(Amino)
	Amino.Seal()
}