package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

const (
	baseURL = "http://localhost:8080/balance/add"
)

type BalanceRequest struct {
	Accounts map[string]float64 `json:"accounts"`
}

func main() {
	accounts := make(map[string]float64)
	for i := 1001; i <= 1100; i++ {
		accounts[fmt.Sprintf("%d", i)] = 10000000
	}

	data := BalanceRequest{Accounts: accounts}
	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}

	var wg sync.WaitGroup
	runRequest := func() {
		defer wg.Done()
		resp, err := http.Post(baseURL, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Println("Error making request:", err)
			return
		}
		defer resp.Body.Close()
		fmt.Println("Response Status:", resp.Status)
	}

	wg.Add(1)
	go runRequest()
	wg.Wait()
}
