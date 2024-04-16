package sequencer

import (
	"context"
	"math/big"
	"time"

	"github.com/0xPolygon/cdk-validium-node/ethtxmanager"
	"github.com/0xPolygon/cdk-validium-node/pool"
	"github.com/0xPolygon/cdk-validium-node/state"
	"github.com/0xPolygon/cdk-validium-node/state/metrics"
	"github.com/0xPolygon/cdk-validium-node/state/runtime/executor"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/jackc/pgx/v4"
)

// Consumer interfaces required by the package.

// txPool contains the methods required to interact with the tx pool.
type txPool interface {
	DeleteTransactionsByHashes(ctx context.Context, hashes []common.Hash) error
	DeleteTransactionByHash(ctx context.Context, hash common.Hash) error
	MarkWIPTxsAsPending(ctx context.Context) error
	GetNonWIPPendingTxs(ctx context.Context) ([]pool.Transaction, error)
	UpdateTxStatus(ctx context.Context, hash common.Hash, newStatus pool.TxStatus, isWIP bool, failedReason *string) error
	GetTxZkCountersByHash(ctx context.Context, hash common.Hash) (*state.ZKCounters, error)
	UpdateTxWIPStatus(ctx context.Context, hash common.Hash, isWIP bool) error
	GetGasPrices(ctx context.Context) (pool.GasPrices, error)
	GetDefaultMinGasPriceAllowed() uint64
	GetL1GasPrice() uint64
}

// etherman contains the methods required to interact with ethereum.
type etherman interface {
	GetSendSequenceFee(numBatches uint64) (*big.Int, error)
	TrustedSequencer() (common.Address, error)
	GetLatestBatchNumber() (uint64, error)
	GetLastBatchTimestamp() (uint64, error)
	GetLatestBlockTimestamp(ctx context.Context) (uint64, error)
	GetLatestBlockNumber(ctx context.Context) (uint64, error)
}

// stateInterface gathers the methods required to interact with the state.
type stateInterface interface {
	GetTimeForLatestBatchVirtualization(ctx context.Context, dbTx pgx.Tx) (time.Time, error)
	GetTxsOlderThanNL1Blocks(ctx context.Context, nL1Blocks uint64, dbTx pgx.Tx) ([]common.Hash, error)
	GetBatchByNumber(ctx context.Context, batchNumber uint64, dbTx pgx.Tx) (*state.Batch, error)
	GetTransactionsByBatchNumber(ctx context.Context, batchNumber uint64, dbTx pgx.Tx) (txs []types.Transaction, effectivePercentages []uint8, err error)
	BeginStateTransaction(ctx context.Context) (pgx.Tx, error)
	GetLastVirtualBatchNum(ctx context.Context, dbTx pgx.Tx) (uint64, error)
	IsBatchClosed(ctx context.Context, batchNum uint64, dbTx pgx.Tx) (bool, error)
	Begin(ctx context.Context) (pgx.Tx, error)
	GetBalanceByStateRoot(ctx context.Context, address common.Address, root common.Hash) (*big.Int, error)
	GetNonceByStateRoot(ctx context.Context, address common.Address, root common.Hash) (*big.Int, error)
	GetLastStateRoot(ctx context.Context, dbTx pgx.Tx) (common.Hash, error)
	ProcessBatch(ctx context.Context, request state.ProcessRequest, updateMerkleTree bool) (*state.ProcessBatchResponse, error)
	CloseBatch(ctx context.Context, receipt state.ProcessingReceipt, dbTx pgx.Tx) error
	ExecuteBatch(ctx context.Context, batch state.Batch, updateMerkleTree bool, dbTx pgx.Tx) (*executor.ProcessBatchResponse, error)
	GetForcedBatch(ctx context.Context, forcedBatchNumber uint64, dbTx pgx.Tx) (*state.ForcedBatch, error)
	GetLastBatch(ctx context.Context, dbTx pgx.Tx) (*state.Batch, error)
	GetLastBatchNumber(ctx context.Context, dbTx pgx.Tx) (uint64, error)
	OpenBatch(ctx context.Context, processingContext state.ProcessingContext, dbTx pgx.Tx) error
	GetLastNBatches(ctx context.Context, numBatches uint, dbTx pgx.Tx) ([]*state.Batch, error)
	StoreTransaction(ctx context.Context, batchNumber uint64, processedTx *state.ProcessTransactionResponse, coinbase common.Address, timestamp uint64, dbTx pgx.Tx) error
	GetLastClosedBatch(ctx context.Context, dbTx pgx.Tx) (*state.Batch, error)
	GetLastL2Block(ctx context.Context, dbTx pgx.Tx) (*types.Block, error)
	GetLastBlock(ctx context.Context, dbTx pgx.Tx) (*state.Block, error)
	GetLatestGlobalExitRoot(ctx context.Context, maxBlockNumber uint64, dbTx pgx.Tx) (state.GlobalExitRoot, time.Time, error)
	GetLastL2BlockHeader(ctx context.Context, dbTx pgx.Tx) (*types.Header, error)
	UpdateBatchL2Data(ctx context.Context, batchNumber uint64, batchL2Data []byte, dbTx pgx.Tx) error
	ProcessSequencerBatch(ctx context.Context, batchNumber uint64, batchL2Data []byte, caller metrics.CallerLabel, dbTx pgx.Tx) (*state.ProcessBatchResponse, error)
	GetForcedBatchesSince(ctx context.Context, forcedBatchNumber, maxBlockNumber uint64, dbTx pgx.Tx) ([]*state.ForcedBatch, error)
	GetLastTrustedForcedBatchNumber(ctx context.Context, dbTx pgx.Tx) (uint64, error)
	GetLatestVirtualBatchTimestamp(ctx context.Context, dbTx pgx.Tx) (time.Time, error)
	CountReorgs(ctx context.Context, dbTx pgx.Tx) (uint64, error)
	GetLatestGer(ctx context.Context, maxBlockNumber uint64) (state.GlobalExitRoot, time.Time, error)
	FlushMerkleTree(ctx context.Context) error
	GetStoredFlushID(ctx context.Context) (uint64, string, error)
	GetForkIDByBatchNumber(batchNumber uint64) uint64
}

type workerInterface interface {
	GetBestFittingTx(resources state.BatchResources) *TxTracker
	UpdateAfterSingleSuccessfulTxExecution(from common.Address, touchedAddresses map[common.Address]*state.InfoReadWrite) []*TxTracker
	UpdateTxZKCounters(txHash common.Hash, from common.Address, ZKCounters state.ZKCounters)
	AddTxTracker(ctx context.Context, txTracker *TxTracker) (replacedTx *TxTracker, dropReason error)
	MoveTxToNotReady(txHash common.Hash, from common.Address, actualNonce *uint64, actualBalance *big.Int) []*TxTracker
	DeleteTx(txHash common.Hash, from common.Address)
	AddPendingTxToStore(txHash common.Hash, addr common.Address)
	DeletePendingTxToStore(txHash common.Hash, addr common.Address)
	HandleL2Reorg(txHashes []common.Hash)
	NewTxTracker(tx types.Transaction, counters state.ZKCounters, ip string) (*TxTracker, error)
	AddForcedTx(txHash common.Hash, addr common.Address)
	DeleteForcedTx(txHash common.Hash, addr common.Address)
}

// The dbManager will need to handle the errors inside the functions which don't return error as they will be used async in the other abstractions.
// Also if dbTx is missing this needs also to be handled in the dbManager
type dbManagerInterface interface {
	OpenBatch(ctx context.Context, processingContext state.ProcessingContext, dbTx pgx.Tx) error
	BeginStateTransaction(ctx context.Context) (pgx.Tx, error)
	CreateFirstBatch(ctx context.Context, sequencerAddress common.Address) state.ProcessingContext
	GetLastBatchNumber(ctx context.Context) (uint64, error)
	DeleteTransactionFromPool(ctx context.Context, txHash common.Hash) error
	CloseBatch(ctx context.Context, params ClosingBatchParameters) error
	GetWIPBatch(ctx context.Context) (*WipBatch, error)
	GetTransactionsByBatchNumber(ctx context.Context, batchNumber uint64) (txs []types.Transaction, effectivePercentages []uint8, err error)
	GetLastBatch(ctx context.Context) (*state.Batch, error)
	GetLastNBatches(ctx context.Context, numBatches uint) ([]*state.Batch, error)
	GetLastClosedBatch(ctx context.Context) (*state.Batch, error)
	GetBatchByNumber(ctx context.Context, batchNumber uint64, dbTx pgx.Tx) (*state.Batch, error)
	IsBatchClosed(ctx context.Context, batchNum uint64) (bool, error)
	GetLatestGer(ctx context.Context, maxBlockNumber uint64) (state.GlobalExitRoot, time.Time, error)
	ProcessForcedBatch(ForcedBatchNumber uint64, request state.ProcessRequest) (*state.ProcessBatchResponse, error)
	GetForcedBatchesSince(ctx context.Context, forcedBatchNumber, maxBlockNumber uint64, dbTx pgx.Tx) ([]*state.ForcedBatch, error)
	GetLastL2BlockHeader(ctx context.Context, dbTx pgx.Tx) (*types.Header, error)
	GetLastBlock(ctx context.Context, dbTx pgx.Tx) (*state.Block, error)
	GetLastTrustedForcedBatchNumber(ctx context.Context, dbTx pgx.Tx) (uint64, error)
	GetBalanceByStateRoot(ctx context.Context, address common.Address, root common.Hash) (*big.Int, error)
	UpdateTxStatus(ctx context.Context, hash common.Hash, newStatus pool.TxStatus, isWIP bool, reason *string) error
	GetLatestVirtualBatchTimestamp(ctx context.Context, dbTx pgx.Tx) (time.Time, error)
	CountReorgs(ctx context.Context, dbTx pgx.Tx) (uint64, error)
	FlushMerkleTree(ctx context.Context) error
	GetGasPrices(ctx context.Context) (pool.GasPrices, error)
	GetDefaultMinGasPriceAllowed() uint64
	GetL1GasPrice() uint64
	GetStoredFlushID(ctx context.Context) (uint64, string, error)
	StoreProcessedTxAndDeleteFromPool(ctx context.Context, tx transactionToStore) error
	GetForcedBatch(ctx context.Context, forcedBatchNumber uint64, dbTx pgx.Tx) (*state.ForcedBatch, error)
	GetForkIDByBatchNumber(batchNumber uint64) uint64
}

type ethTxManager interface {
	Add(ctx context.Context, owner, id string, from common.Address, to *common.Address, value *big.Int, data []byte, dbTx pgx.Tx) error
	Result(ctx context.Context, owner, id string, dbTx pgx.Tx) (ethtxmanager.MonitoredTxResult, error)
	ResultsByStatus(ctx context.Context, owner string, statuses []ethtxmanager.MonitoredTxStatus, dbTx pgx.Tx) ([]ethtxmanager.MonitoredTxResult, error)
	ProcessPendingMonitoredTxs(ctx context.Context, owner string, failedResultHandler ethtxmanager.ResultHandler, dbTx pgx.Tx)
}