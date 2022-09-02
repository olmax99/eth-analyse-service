package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	eth "eth-analyse-service/pkg/service"

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
		Host:     "pgsql.local",
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

// ------------- Handlers-----------------

const shortForm = "2006-Jan-02"

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

	ctx := context.Background()
	pgxConf := pgx.ConnConfig{
		Port:     uint16(5432),
		Host:     "pgsql.local",
		Database: "eth",
		User:     "test",
		Password: "test",
	}

	s := eth.NewEthAnalyseServiceServer(eth.WithPool(ctx, pgxConf))

	sqlView := fmt.Sprint(`create or replace view public."20200907_tx_eao_transfer" as
select tx.*
from (SELECT *, (gas_used * gas_price * 0.000000000000000001) as wei_total
      FROM   public.transactions
      WHERE  "from" NOT IN (SELECT DISTINCT address FROM public.contracts)
      AND "to" NOT IN (SELECT DISTINCT address FROM public.contracts)
      AND "from" != '0x0000000000000000000000000000000000000000'
      AND "to" != '0x0000000000000000000000000000000000000000'
      AND status = 'true'
      AND value != 0
) as tx`)

	if err := s.PgxPoolExec(ctx, sqlView); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"Message": err.Error()})
	}

	sqlSelect := fmt.Sprint(`select
date_trunc('hour', block_time - interval '1 minute') as interv_start,
date_trunc('hour', block_time - interval '1 minute')  + interval '1 hours' as interv_end,
avg(wei_total)
from public."20200907_tx_eao_transfer"
    group by date_trunc('hour', block_time - interval '1 minute')
order by interv_start`)

	if err := s.PgxPoolScanSelectTransactions(ctx, sqlSelect); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"Message": err.Error()})
	}

	tx := s.ResultsGetHourlyGasByDay()
	s.SortByBlockTime(tx)

	return c.JSON(http.StatusOK, echo.Map{
		"result": tx,
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
