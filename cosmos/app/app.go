package app

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/cometbft/cometbft/libs/log"
	dbm "github.com/cometbft/cometbft-db"

	"github.com/interbank-netting/cosmos/x/oracle"
	oraclekeeper "github.com/interbank-netting/cosmos/x/oracle/keeper"
	oracletypes "github.com/interbank-netting/cosmos/x/oracle/types"
	"github.com/interbank-netting/cosmos/x/netting"
	nettingkeeper "github.com/interbank-netting/cosmos/x/netting/keeper"
	nettingtypes "github.com/interbank-netting/cosmos/x/netting/types"
	"github.com/interbank-netting/cosmos/x/multisig"
	multisigkeeper "github.com/interbank-netting/cosmos/x/multisig/keeper"
	multisigtypes "github.com/interbank-netting/cosmos/x/multisig/types"
)

const (
	AccountAddressPrefix = "cosmos"
	Name                 = "interbank-netting"
)

var (
	// DefaultNodeHome default home directories for the application daemon
	DefaultNodeHome string

	// ModuleBasics defines the module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration
	// and genesis verification.
	ModuleBasics = module.NewBasicManager(
		auth.AppModuleBasic{},
		genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
		bank.AppModuleBasic{},
		staking.AppModuleBasic{},
		oracle.AppModuleBasic{},
		netting.AppModuleBasic{},
		multisig.AppModuleBasic{},
		// Custom modules will be added here
	)
)

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	DefaultNodeHome = filepath.Join(userHomeDir, "."+Name)
}

// App extends an ABCI application, but with most of its parameters exported.
// They are exported for convenience in creating helper functions, as object
// capabilities aren't needed for testing.
type App struct {
	*baseapp.BaseApp

	cdc               *codec.LegacyAmino
	appCodec          codec.Codec
	interfaceRegistry types.InterfaceRegistry
	txConfig          client.TxConfig

	invCheckPeriod uint

	// keys to access the substores
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// keepers
	AccountKeeper authkeeper.AccountKeeper
	BankKeeper    bankkeeper.Keeper
	StakingKeeper *stakingkeeper.Keeper
	OracleKeeper  oraclekeeper.Keeper
	NettingKeeper nettingkeeper.Keeper
	MultisigKeeper multisigkeeper.Keeper

	// Custom module keepers will be added here

	// the module manager
	mm *module.Manager

	// simulation manager
	sm *module.SimulationManager

	// module configurator
	configurator module.Configurator
}

// New returns a reference to an initialized blockchain app
func New(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	appOpts types.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *App {
	interfaceRegistry := types.NewInterfaceRegistry()
	appCodec := codec.NewProtoCodec(interfaceRegistry)
	legacyAmino := codec.NewLegacyAmino()
	txConfig := authtx.NewTxConfig(appCodec, authtx.DefaultSignModes)

	bApp := baseapp.NewBaseApp(Name, logger, db, txConfig.TxDecoder(), baseAppOptions...)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetVersion(version.Version)
	bApp.SetInterfaceRegistry(interfaceRegistry)
	bApp.SetTxEncoder(txConfig.TxEncoder())

	keys := storetypes.NewKVStoreKeys(
		authtypes.StoreKey, 
		banktypes.StoreKey, 
		stakingtypes.StoreKey,
		oracletypes.StoreKey,
		nettingtypes.StoreKey,
		multisigtypes.StoreKey,
		// Custom module store keys will be added here
	)

	tkeys := storetypes.NewTransientStoreKeys()
	memKeys := storetypes.NewMemoryStoreKeys(oracletypes.MemStoreKey, nettingtypes.MemStoreKey, multisigtypes.MemStoreKey)

	app := &App{
		BaseApp:           bApp,
		cdc:               legacyAmino,
		appCodec:          appCodec,
		interfaceRegistry: interfaceRegistry,
		txConfig:          txConfig,
		keys:              keys,
		tkeys:             tkeys,
		memKeys:           memKeys,
	}

	// Initialize keepers and modules here
	// Initialize Oracle Keeper (simplified without params for now)
	app.OracleKeeper = *oraclekeeper.NewKeeper(
		appCodec,
		keys[oracletypes.StoreKey],
		memKeys[oracletypes.MemStoreKey],
		paramtypes.Subspace{}, // Empty subspace for now
		app.BankKeeper,
		app.StakingKeeper,
	)

	// Initialize Netting Keeper
	app.NettingKeeper = *nettingkeeper.NewKeeper(
		appCodec,
		keys[nettingtypes.StoreKey],
		memKeys[nettingtypes.MemStoreKey],
		paramtypes.Subspace{}, // Empty subspace for now
		app.BankKeeper,
		app.AccountKeeper,
	)

	// Initialize Multisig Keeper
	app.MultisigKeeper = *multisigkeeper.NewKeeper(
		appCodec,
		keys[multisigtypes.StoreKey],
		memKeys[multisigtypes.MemStoreKey],
		paramtypes.Subspace{}, // Empty subspace for now
		app.BankKeeper,
		app.StakingKeeper,
	)

	// Set cross-module dependencies
	app.OracleKeeper.SetNettingKeeper(&app.NettingKeeper)

	// This is a basic structure - full implementation will be added in subsequent tasks

	return app
}

// Name returns the name of the App
func (app *App) Name() string { return app.BaseApp.Name() }

// LegacyAmino returns SimApp's amino codec.
func (app *App) LegacyAmino() *codec.LegacyAmino {
	return app.cdc
}

// AppCodec returns an app codec.
func (app *App) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns an InterfaceRegistry
func (app *App) InterfaceRegistry() types.InterfaceRegistry {
	return app.interfaceRegistry
}

// TxConfig returns SimApp's TxConfig
func (app *App) TxConfig() client.TxConfig {
	return app.txConfig
}

// DefaultGenesis returns a default genesis from the registered AppModuleBasic's.
func (app *App) DefaultGenesis() map[string]json.RawMessage {
	return ModuleBasics.DefaultGenesis(app.appCodec)
}

// GetKey returns the KVStoreKey for the provided store key.
func (app *App) GetKey(storeKey string) *storetypes.KVStoreKey {
	return app.keys[storeKey]
}

// GetTKey returns the TransientStoreKey for the provided store key.
func (app *App) GetTKey(storeKey string) *storetypes.TransientStoreKey {
	return app.tkeys[storeKey]
}

// GetMemKey returns the MemStoreKey for the provided mem key.
func (app *App) GetMemKey(storeKey string) *storetypes.MemoryStoreKey {
	return app.memKeys[storeKey]
}