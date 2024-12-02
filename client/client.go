package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type CotacaoReturnDTO struct {
	Bid string `json:"bid"`
}

func main() {
	exchangeRate, err := getServerAnswer()
	if err != nil {
		log.Println(err)
		return
	}

	extendedExchange := "Dólar: " + exchangeRate.Bid + "\n"
	err = saveExchangeRatesInfoInFile(extendedExchange)
	if err != nil {
		log.Println(err)
	}

	io.Copy(os.Stdout, strings.NewReader(extendedExchange))
}

func getServerAnswer() (*CotacaoReturnDTO, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://127.0.0.1:8080/cotacao", nil)
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

	if res.StatusCode != http.StatusOK {
		log.Println("Não foi possível obter um resultado do servidor: ", res.StatusCode)
		return nil, errors.New("Não foi possível obter um resultado do servidor: " + res.Status)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var exchangeRate CotacaoReturnDTO
	err = json.Unmarshal(data, &exchangeRate)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &exchangeRate, nil
}

func saveExchangeRatesInfoInFile(exchangeRate string) error {
	filename := "cotacao.txt"

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Println(err)
		return err
	}
	defer file.Close()

	if _, err := file.WriteString(exchangeRate); err != nil {
		log.Println(err)
		return err
	}

	return nil
}
