package eth_test

import (
	"context"
	"fmt"
	"testing"

	eth "eth-analyse-service/pkg/service"

	"github.com/jackc/pgx"
)

func TestTransactions(t *testing.T) {
	tests := []struct {
		name     string
		expected int
	}{
		{
			"tx1",
			1,
		},
		{
			"tx5",
			5,
		},
	}
	for _, test := range tests {
		test := test // required for parralel due to pointer conflict
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			pgxConf := pgx.ConnConfig{
				Port:     uint16(6545),
				Host:     "localhost",
				Database: "eth",
				User:     "test",
				Password: "test",
			}

			s := eth.NewEthAnalyseServiceServer(eth.WithPool(ctx, pgxConf))

			sql := fmt.Sprintf(`(SELECT *
FROM public.transactions
LIMIT %d)`, test.expected)
			if err := s.PgxPoolScanSelectTransactions(ctx, sql); err != nil {
				t.Error(err)
			}
			actual := len(s.ResultsTransactions())
			if actual < test.expected {
				t.Errorf("Expected actual < %d, got %d.", test.expected, actual)
			}
		})
	}
}
