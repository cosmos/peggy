package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/gravity-bridge/module/x/gravity/types"
)

// InitGenesis starts a chain from a genesis state
func InitGenesis(ctx sdk.Context, k Keeper, data types.GenesisState) {
	k.SetParams(ctx, *data.Params)
	// reset valsets in state
	for _, vs := range data.Valsets {
		// TODO: block height?
		k.StoreValsetUnsafe(ctx, vs)
	}

	// reset valset confirmations in state
	for _, conf := range data.SignerSetTxSignatures {
		k.SetSignerSetTxSignature(ctx, *conf)
	}

	// reset batches in state
	for _, batch := range data.Batches {
		// TODO: block height?
		k.StoreBatchUnsafe(ctx, batch)
	}

	// reset batch confirmations in state
	for _, conf := range data.BatchConfirms {
		conf := conf
		k.SetBatchConfirm(ctx, &conf)
	}

	// reset logic calls in state
	for _, call := range data.LogicCalls {
		k.SetContractCallTx(ctx, call)
	}

	// reset batch confirmations in state
	for _, conf := range data.LogicCallConfirms {
		conf := conf
		k.SetLogicCallConfirm(ctx, &conf)
	}

	// reset pool transactions in state
	for _, tx := range data.UnbatchedTransfers {
		if err := k.setPoolEntry(ctx, tx); err != nil {
			panic(err)
		}
	}

	// reset ethereumEventVoteRecords in state
	for _, att := range data.EthereumEventVoteRecords {
		att := att
		claim, err := k.UnpackEthereumEventVoteRecordClaim(&att)
		if err != nil {
			panic("couldn't cast to claim")
		}

		// TODO: block height?
		k.SetEthereumEventVoteRecord(ctx, claim.GetEventNonce(), claim.ClaimHash(), &att)
	}
	k.setLastObservedEventNonce(ctx, data.LastObservedNonce)

	// reset ethereumEventVoteRecord state of specific validators
	// this must be done after the above to be correct
	for _, att := range data.EthereumEventVoteRecords {
		att := att
		claim, err := k.UnpackEthereumEventVoteRecordClaim(&att)
		if err != nil {
			panic("couldn't cast to claim")
		}
		// reconstruct the latest event nonce for every validator
		// if somehow this genesis state is saved when all ethereumEventVoteRecords
		// have been cleaned up GetLastEventNonceByValidator handles that case
		//
		// if we where to save and load the last event nonce for every validator
		// then we would need to carry that state forever across all chain restarts
		// but since we've already had to handle the edge case of new validators joining
		// while all ethereumEventVoteRecords have already been cleaned up we can do this instead and
		// not carry around every validators event nonce counter forever.
		for _, vote := range att.Votes {
			val, err := sdk.ValAddressFromBech32(vote)
			if err != nil {
				panic(err)
			}
			last := k.GetLastEventNonceByValidator(ctx, val)
			if claim.GetEventNonce() > last {
				k.setLastEventNonceByValidator(ctx, val, claim.GetEventNonce())
			}
		}
	}

	// reset delegate keys in state
	for _, keys := range data.DelegateKeys {
		err := keys.ValidateBasic()
		if err != nil {
			panic("Invalid delegate key in Genesis!")
		}
		val, err := sdk.ValAddressFromBech32(keys.Validator)
		if err != nil {
			panic(err)
		}

		orch, err := sdk.AccAddressFromBech32(keys.Orchestrator)
		if err != nil {
			panic(err)
		}

		// set the orchestrator address
		k.SetOrchestratorValidator(ctx, val, orch)
		// set the ethereum address
		k.SetEthAddressForValidator(ctx, val, keys.EthAddress)
	}

	// populate state with cosmos originated denom-erc20 mapping
	for _, item := range data.Erc20ToDenoms {
		k.setCosmosOriginatedDenomToERC20(ctx, item.Denom, item.Erc20)
	}
}

// ExportGenesis exports all the state needed to restart the chain
// from the current state of the chain
func ExportGenesis(ctx sdk.Context, k Keeper) types.GenesisState {
	var (
		p                        = k.GetParams(ctx)
		calls                    = k.GetContractCallTxs(ctx)
		batches                  = k.GetBatchTxs(ctx)
		valsets                  = k.GetValsets(ctx)
		attmap                   = k.GetEthereumEventVoteRecordMapping(ctx)
		vsconfs                  = []*types.MsgSignerSetTxSignature{}
		batchconfs               = []types.MsgConfirmBatch{}
		callconfs                = []types.MsgConfirmLogicCall{}
		ethereumEventVoteRecords = []types.EthereumEventVoteRecord{}
		delegates                = k.GetDelegateKeys(ctx)
		lastobserved             = k.GetLastObservedEventNonce(ctx)
		erc20ToDenoms            = []*types.ERC20ToDenom{}
		unbatchedTransfers       = k.GetPoolTransactions(ctx)
	)

	// export valset confirmations from state
	for _, vs := range valsets {
		// TODO: set height = 0?
		vsconfs = append(vsconfs, k.GetSignerSetTxSignatures(ctx, vs.Nonce)...)
	}

	// export batch confirmations from state
	for _, batch := range batches {
		// TODO: set height = 0?
		batchconfs = append(batchconfs,
			k.GetBatchConfirmByNonceAndTokenContract(ctx, batch.BatchNonce, batch.TokenContract)...)
	}

	// export logic call confirmations from state
	for _, call := range calls {
		// TODO: set height = 0?
		callconfs = append(callconfs,
			k.GetLogicConfirmByInvalidationIDAndNonce(ctx, call.InvalidationId, call.InvalidationNonce)...)
	}

	// export ethereumEventVoteRecords from state
	for _, atts := range attmap {
		// TODO: set height = 0?
		ethereumEventVoteRecords = append(ethereumEventVoteRecords, atts...)
	}

	// export erc20 to denom relations
	k.IterateERC20ToDenom(ctx, func(key []byte, erc20ToDenom *types.ERC20ToDenom) bool {
		erc20ToDenoms = append(erc20ToDenoms, erc20ToDenom)
		return false
	})

	return types.GenesisState{
		Params:                   &p,
		LastObservedNonce:        lastobserved,
		Valsets:                  valsets,
		SignerSetTxSignatures:    vsconfs,
		Batches:                  batches,
		BatchConfirms:            batchconfs,
		LogicCalls:               calls,
		LogicCallConfirms:        callconfs,
		EthereumEventVoteRecords: ethereumEventVoteRecords,
		DelegateKeys:             delegates,
		Erc20ToDenoms:            erc20ToDenoms,
		UnbatchedTransfers:       unbatchedTransfers,
	}
}
