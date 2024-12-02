package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := maybeCreateSQLLiteDatabase()
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/cotacao", Conexao{dbConn: db}.ServeHTTP) // Corrigido aqui

	log.Println("Server initiated at port 8080")
	http.ListenAndServe(":8080", mux)
}

type Cotacao struct {
	Data CotacaoData `json:"USDBRL"`
}

type CotacaoData struct {
	Code       string `json:"code"`
	Codein     string `json:"codein"`
	Name       string `json:"name"`
	High       string `json:"high"`
	Low        string `json:"low"`
	VarBid     string `json:"varBid"`
	PctChange  string `json:"pctChange"`
	Bid        string `json:"bid"`
	Ask        string `json:"ask"`
	Timestamp  string `json:"timestamp"`
	CreateDate string `json:"create_date"`
}

type CotacaoDTO struct {
	Bid string `json:"bid"`
}

type Conexao struct {
	dbConn *sql.DB
}

func (conn Conexao) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("/cotacao request received")
	defer log.Println("/cotacao request completed")
	if conn.dbConn == nil {
		log.Println("Not possible to connect to database")
		panic("Not possible to connect to database")
	}

	ctx := r.Context()
	exchangeRate, err := getExchangeRatesInfo(ctx)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = saveExchangeRatesInfoInDatabase(ctx, conn, exchangeRate)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	retval := CotacaoDTO{Bid: exchangeRate.Data.Bid}
	retvalJSON, err := json.Marshal(retval)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(retvalJSON)
}

func maybeCreateSQLLiteDatabase() (*sql.DB, error) {
	dbAlreadyExists := existsDB("exchange.db")
	db, err := sql.Open("sqlite3", "exchange.db")
	if err != nil {
		return nil, err
	}

	if !dbAlreadyExists {
		const create string = `CREATE TABLE IF NOT EXISTS rates (id INTEGER NOT NULL PRIMARY KEY, codein TEXT NOT NULL,
							  name TEXT NOT NULL, high TEXT NOT NULL, low TEXT NOT NULL, var_bid TEXT NOT NULL,
							  pct_change  TEXT NOT NULL, bid TEXT NOT NULL, ask TEXT NOT NULL, timestamp TEXT NOT NULL,
							  create_date TEXT NOT NULL);`
		if _, err := db.Exec(create); err != nil {
			return nil, err
		}
	}
	log.Println("connected to database exchange.db")
	return db, nil
}

func getExchangeRatesInfo(ctx context.Context) (*Cotacao, error) {
	ctxInternal, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctxInternal, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var exchangeRate Cotacao
	err = json.Unmarshal(data, &exchangeRate)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &exchangeRate, nil
}

func saveExchangeRatesInfoInDatabase(ctx context.Context, conn Conexao, exchangeRate *Cotacao) error {
	ctxInternal, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
	defer cancel()
	const insertSTMT string = `INSERT INTO rates(codein, name, high, low, var_bid, pct_change, bid, ask,
                           timestamp, create_date) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);`
	stmt, err := conn.dbConn.Prepare(insertSTMT)
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.ExecContext(ctxInternal, exchangeRate.Data.Codein, exchangeRate.Data.Name, exchangeRate.Data.High,
		exchangeRate.Data.Low, exchangeRate.Data.VarBid, exchangeRate.Data.PctChange, exchangeRate.Data.Bid,
		exchangeRate.Data.Ask, exchangeRate.Data.Timestamp, exchangeRate.Data.CreateDate)
	if err != nil {
		return err
	}
	return nil
}

func existsDB(fileName string) bool {
	_, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		return false
	}
	return true
}
