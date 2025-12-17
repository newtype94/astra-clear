package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	grpc1 "github.com/gogo/protobuf/grpc"
)

// RegisterCodec registers the necessary x/multisig interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgGenerateMintCommand{}, "multisig/MsgGenerateMintCommand", nil)
	cdc.RegisterConcrete(&MsgSignCommand{}, "multisig/MsgSignCommand", nil)
	cdc.RegisterConcrete(&MsgUpdateValidatorSet{}, "multisig/MsgUpdateValidatorSet", nil)
	cdc.RegisterConcrete(&MsgAddValidator{}, "multisig/MsgAddValidator", nil)
	cdc.RegisterConcrete(&MsgRemoveValidator{}, "multisig/MsgRemoveValidator", nil)
}

// RegisterInterfaces registers the x/multisig interfaces types with the interface registry
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgGenerateMintCommand{},
		&MsgSignCommand{},
		&MsgUpdateValidatorSet{},
		&MsgAddValidator{},
		&MsgRemoveValidator{},
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