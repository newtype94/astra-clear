package cmd

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/cometbft/cometbft/libs/log"
	dbm "github.com/cometbft/cometbft-db"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/interbank-netting/cosmos/app"
)

// NewRootCmd creates a new root command for interbank-nettingd. It is called once in the
// main function.
func NewRootCmd() (*cobra.Command, interface{}) {
	// Set config for prefixes
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(app.AccountAddressPrefix, app.AccountAddressPrefix+"pub")
	config.SetBech32PrefixForValidator(app.AccountAddressPrefix+"valoper", app.AccountAddressPrefix+"valoperpub")
	config.SetBech32PrefixForConsensusNode(app.AccountAddressPrefix+"valcons", app.AccountAddressPrefix+"valconspub")
	config.Seal()

	rootCmd := &cobra.Command{
		Use:   "interbank-nettingd",
		Short: "Interbank Netting Engine daemon",
		Long: `Interbank Netting Engine is a blockchain application built using Cosmos SDK.
It provides credit tokenization and periodic netting for interbank transfers.`,
	}

	server.AddCommands(rootCmd, app.DefaultNodeHome, newApp, nil, addModuleInitFlags)

	return rootCmd, nil
}

func addModuleInitFlags(startCmd *cobra.Command) {
	// Module initialization flags will be added here
}

// newApp creates the application
func newApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	appOpts servertypes.AppOptions,
) servertypes.Application {
	return app.New(logger, db, traceStore, true, appOpts)
}