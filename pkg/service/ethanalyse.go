package eth

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/v4/pgxpool"
)

type EthAnalyser interface {
	PgxPoolExec(context.Context, string) error
	PgxPoolScanSelectTransactions(context.Context, string) error
	ResultsTransactions() []*ResultTransactions
	ResultGetHourlyGasByDay() []*ResultGetHourlyGasByDay
}

type ResultTransactions struct {
	Txid        string         `json:"txid"`
	BlockHeight pgtype.Numeric `json:"block_height"`
	BlockHash   string         `json:"block_hash"`
	BlockTime   time.Time      `json:"block_time"`
	From        string         `json:"from"`
	To          string         `json:"to"`
	Value       pgtype.Numeric `json:"value"`
	GasProvided pgtype.Numeric `json:"gas_provided"`
	GasUsed     pgtype.Numeric `json:"gas_used"`
	GasPrice    pgtype.Numeric `json:"gas_price"`
	Status      string         `json:"status"`
	IntervStart time.Time      `json:"interv_start"`
	IntervEnd   time.Time      `json:"interv_end"`
	Avg         float64        `json:"avg"`
}

type ByDateTime []ResultGetHourlyGasByDay

func (a ByDateTime) Len() int           { return len(a) }
func (a ByDateTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByDateTime) Less(i, j int) bool { return a[i].T < a[j].T }

type ResultGetHourlyGasByDay struct {
	T int64   `json:"t"`
	V float64 `json:"v"`
}

// ethAnalyseServiceServer main struct
type ethAnalyseServiceServer struct {
	conn string
	pool *pgxpool.Pool

	Transactions []ResultTransactions
	// txEOATransfer []*resultTxEOATransfer
	Hourly []ResultGetHourlyGasByDay
}

// ethAnalyseServiceServerConf config closure for ethAnalyseServiceServer
type ethAnalyseServiceServerConf func(eas *ethAnalyseServiceServer)

// NewEthAnalyseServiceServer initializes a new ethAnalyseServiceServer
func NewEthAnalyseServiceServer(opts ...ethAnalyseServiceServerConf) *ethAnalyseServiceServer {
	eas := &ethAnalyseServiceServer{}
	for _, c := range opts {
		c(eas)
	}
	return eas
}

func WithPool(ctx context.Context, pgxConf pgx.ConnConfig) ethAnalyseServiceServerConf {
	pool, err := pgxpool.Connect(ctx, fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		pgxConf.User,
		pgxConf.Password,
		pgxConf.Host,
		strconv.Itoa(int(pgxConf.Port)),
		pgxConf.Database,
	))
	if err != nil {
		return func(eas *ethAnalyseServiceServer) {
			eas.pool = &pgxpool.Pool{}
		}
	}
	return func(eas *ethAnalyseServiceServer) {
		eas.pool = pool
	}
}

func (eas *ethAnalyseServiceServer) PgxPoolExec(ctx context.Context, sql string) error {
	if _, err := eas.pool.Exec(ctx, sql); err != nil {
		return err
	}
	return nil
}

func (eas *ethAnalyseServiceServer) PgxPoolScanSelectTransactions(ctx context.Context, sql string) error {
	if err := pgxscan.Select(ctx, eas.pool, &eas.Transactions, sql); err != nil {
		return err
	}
	return nil
}

func (eas *ethAnalyseServiceServer) ResultsTransactions() []ResultTransactions {
	return eas.Transactions
}

func (eas *ethAnalyseServiceServer) ResultsGetHourlyGasByDay() []ResultGetHourlyGasByDay {
	eas.Hourly = make([]ResultGetHourlyGasByDay, len(eas.Transactions))
	for i, v := range eas.Transactions {
		eas.Hourly[i].T = v.IntervStart.Unix()
		eas.Hourly[i].V = v.Avg
	}
	return eas.Hourly
}

func (eas *ethAnalyseServiceServer) SortByBlockTime(tx []ResultGetHourlyGasByDay) {
	sort.Sort(ByDateTime(tx))
}
