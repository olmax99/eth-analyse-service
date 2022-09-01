package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgtype"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/georgysavva/scany/pgxscan"

	oapi "github.com/swaggo/echo-swagger"
)

const apiVersion = "v1"

// @title eth-analyse-service API
// @version 1.0
// @description Api for eth-anaylse
// @termsOfService http://swagger.io/terms/

// @contact.name Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host web-api.local
// @BasePath /v2
func New() *echo.Echo {
	e := echo.New()

	AddMiddlewareLog(e)

	public := e.Group("/api/" + apiVersion)

	RegisterRoutesPublic(public)

	if err := InitDB(); err != nil {
		fmt.Printf("Abort. No Ping.. %s", err)
		os.Exit(1)
	}

	e.GET("/swagger/*", oapi.WrapHandler)

	return e
}

// ------------- Middleware----------------

// ------------- Logger--------------------
func AddMiddlewareLog(e *echo.Echo) {
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "time=${time_rfc3339}, method=${method}, uri=${uri}, status=${status}\n",
	}))
}

// ------------- Database------------------

var pgxConf pgx.ConnConfig

// ------------- PGSQL---------------------
func InitDB() error {
	pgxConf = pgx.ConnConfig{
		Port:     uint16(5432),
		Host:     "psql.local",
		Database: "eth",
		User:     "test",
		Password: "test",
	}
	conn, err := pgx.Connect(pgxConf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()
	if err := conn.Ping(context.Background()); err != nil {
		return err
	}
	return nil
}

// ------------- Routes-------------------
func RegisterRoutesPublic(g *echo.Group) {
	g.GET("/gashourly/:day", GetHourlyGasByDay)
}

// ------------- Services-----------------
type ethAnalyseServiceServer struct {
	conn string
	pool *pgxpool.Pool
	mu   sync.Mutex

	transactions []*resultTransactions
	// txEOATransfer []*resultTxEOATransfer
	hourly []*resultGetHourlyGasByDay
}

func NewEthAnalyseService(c pgx.ConnConfig) *ethAnalyseServiceServer {
	pgxConf = c
	srv := &ethAnalyseServiceServer{
		conn: fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
			pgxConf.User,
			pgxConf.Password,
			pgxConf.Host,
			strconv.Itoa(int(pgxConf.Port)),
			pgxConf.Database,
		)}
	return srv
}

// ------------- Handlers-----------------

const shortForm = "2006-Jan-02"

type resultTransactions struct {
	Txid        string         `json:"txid"`
	BlockHeight int            `json:"block_height"`
	BlockHash   string         `json:"block_hash"`
	BlockTime   time.Time      `json:"block_time"`
	From        string         `json:"from"`
	To          string         `json:"to"`
	Value       pgtype.Numeric `json:"value"`
	GasProvided pgtype.Numeric `json:"gas_provided"`
	GasUsed     pgtype.Numeric `json:"gas_used"`
	GasPrice    pgtype.Numeric `json:"gas_price"`
	Status      string         `json:"status"`
}

type resultGetHourlyGasByDay struct {
	T int     `json:"t"`
	V float64 `json:"v"`
}

// GetHourlyGasByDay
// @Description hourly amount of fees spent by transactions between externaly owned accounts
// @DAY get-hourly-gas-by-day
// @Accept  json
// @Produce  json
// @Param   2021-Jan-01     path    string     true        "2021-Jan-01"
// @Param   offset     query    int     true        "Offset"
// @Param   limit      query    int     true        "Offset"
// @Success 200 {string} string	"ok"
// @Failure 400 {object} HTTPError "Please ensure that the date format is like 2021-Jan-01."
// @Failure 404 {object} HTTPError "Can not find data for gashourly."
// @Router /api/v1/gashourly/{2021-Jan-01} [get]
func GetHourlyGasByDay(c echo.Context) error {
	day := c.Param("day")
	_, err := time.Parse(shortForm, day)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"Message": "input time not valid"})
	}

	// ------------- filter on transactions between eoa only-----------------
	if pgxConf.Host == "" {
		return c.JSON(http.StatusNotFound, map[string]string{"Message": "pgx not connected"})
	}
	pgxConf = pgx.ConnConfig{
		Port:     uint16(5432),
		Host:     "psql.local",
		Database: "eth",
		User:     "test",
		Password: "test",
	}
	s := NewEthAnalyseService(pgxConf)

	pgxCtx := context.Background()

	sqlView := fmt.Sprint(`create or replace view public."20200907_tx_eao_transfer" as
select tx.*
from (SELECT *
      FROM   public.transactions
      WHERE  "from" NOT IN (SELECT DISTINCT address FROM public.contracts)
      AND "to" NOT IN (SELECT DISTINCT address FROM public.contracts)
      AND "from" != '0x0000000000000000000000000000000000000000'
      AND "to" != '0x0000000000000000000000000000000000000000'
      AND status = 'true'
      ) as tx`)

	if s.pool, err = pgxpool.Connect(pgxCtx, s.conn); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"Message": "pgx create pool failed"})
	}
	defer s.pool.Close()
	if _, err := s.pool.Exec(pgxCtx, sqlView); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"Message": err.Error()})
	}

	sqlSelect := fmt.Sprint(`SELECT txid, block_height, block_hash, block_time, "from", "to", value, gas_provided, gas_used, gas_price, status
FROM public."20200907_tx_eao_transfer"`)
	s.mu.Lock()
	if err := pgxscan.Select(pgxCtx, s.pool, &s.transactions, sqlSelect); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"Message": err.Error()})
	}
	s.mu.Unlock()

	// -------------- calculate/aggregate hourly--------------------------------
	if len(s.transactions) == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"Message": "no transactions found."})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"result": s.transactions[:5],
	})
}

// ----------------- errors openapi-------------------
func NewError(c echo.Context, status int, err error) {
	er := HTTPError{
		Code:    status,
		Message: err.Error(),
	}
	c.JSON(status, er)
}

type HTTPError struct {
	Code    int    `json:"code" example:"400"`
	Message string `json:"message" example:"status bad request"`
}
