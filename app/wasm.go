package app

import (
	"cosmossdk.io/core/appmodule"
	storetypes "cosmossdk.io/store/types"
	wasmd "github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

// registerWasmModules register Wasm keeper and non dependency inject modules.
func (app *App) registerWasmModules(appOpts servertypes.AppOptions) error {
	// set up non depinject support modules store keys
	wasmStoreKey := storetypes.NewKVStoreKey(wasmtypes.StoreKey)
	if err := app.RegisterStores(
		wasmStoreKey,
	); err != nil {
		return err
	}
	kvStoreService := runtime.NewKVStoreService(wasmStoreKey)

	// register the legacy subspace
	app.ParamsKeeper.Subspace(wasmtypes.ModuleName)
	subspace := app.GetSubspace(wasmtypes.ModuleName)

	// add capability keeper and ScopeToModule for wasm module
	app.ScopedWasmKeeper = app.CapabilityKeeper.ScopeToModule(wasmtypes.ModuleName)

	availableCapabilities := []string{}
	homeDir := ""
	wasmConfig := wasmtypes.DefaultWasmConfig()
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// Create Wasm keeper
	wasmKeeper := wasmkeeper.NewKeeper(
		app.appCodec,
		kvStoreService,
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		distrkeeper.NewQuerier(app.DistrKeeper),
		app.GetIBCKeeper().ChannelKeeper,
		app.GetIBCKeeper().ChannelKeeper,
		app.GetIBCKeeper().PortKeeper,
		app.ScopedWasmKeeper,
		app.TransferKeeper,
		app.MsgServiceRouter(),
		app.GRPCQueryRouter(),
		homeDir,
		wasmConfig,
		availableCapabilities,
		authority,
	)
	app.WasmKeeper = &wasmKeeper

	// register Wasm module
	if err := app.RegisterModules(
		wasm.NewAppModule(
			app.AppCodec(),
			app.WasmKeeper,
			app.StakingKeeper,
			app.AccountKeeper,
			app.BankKeeper,
			app.MsgServiceRouter(),
			subspace,
		),
	); err != nil {
		return err
	}

	anteHandler, err := wasmd.NewAnteHandler(wasmd.HandlerOptions{
		HandlerOptions: ante.HandlerOptions{
			AccountKeeper:   app.AccountKeeper,
			BankKeeper:      app.BankKeeper,
			SignModeHandler: app.TxConfig().SignModeHandler(),
			FeegrantKeeper:  app.FeeGrantKeeper,
			SigGasConsumer:  ante.DefaultSigVerificationGasConsumer,
		},
		IBCKeeper:             app.IBCKeeper,
		WasmConfig:            &wasmConfig,
		WasmKeeper:            &wasmKeeper,
		TXCounterStoreService: kvStoreService,
		CircuitKeeper:         &app.CircuitBreakerKeeper,
	})
	if err != nil {
		return err
	}
	app.SetAnteHandler(anteHandler)

	return nil
}

// RegisterWasm Since the Wasm module doesn't support dependency injection,
// we need to manually register the module on the client side.
// This needs to be removed after Wasm supports App Wiring.
func RegisterWasm(registry cdctypes.InterfaceRegistry) map[string]appmodule.AppModule {
	modules := map[string]appmodule.AppModule{
		wasmtypes.ModuleName: wasm.AppModule{},
	}

	for name, m := range modules {
		module.CoreAppModuleBasicAdaptor(name, m).RegisterInterfaces(registry)
	}

	return modules
}
