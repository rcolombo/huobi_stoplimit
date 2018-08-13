package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
)

const (
	ROOT_URL    = "api.huobi.pro"
	MARKET_URL  = "/market"
	PUBLIC_URL  = "/v1/common"
	TRADING_URL = "/v1"
)

func encodeSortQueryString(req *http.Request, params map[string]string) string {
	// add the query params, sort by byte order
	var keys []string
	q := req.URL.Query()
	for k, _ := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		q.Set(k, params[k])
	}
	return q.Encode()
}

// https://github.com/huobiapi/API_Docs_en/wiki
func (h *Huobi) Do(method, path string, params map[string]string, auth bool) (*bytes.Buffer, error) {
	endpoint := "https://" + ROOT_URL + path

	var (
		req *http.Request
		err error
	)
	switch method {
	case "GET":
		req, err = http.NewRequest(method, endpoint, nil)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	case "POST":
		b, _ := json.Marshal(params)
		req, err = http.NewRequest(method, endpoint, bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
	default:
		return nil, fmt.Errorf("Unhandled HTTP Method: %v", method)
	}
	if err != nil {
		return nil, err
	}

	req.URL.RawQuery = encodeSortQueryString(req, params)

	if auth {
		params["AccessKeyId"] = h.APIKey
		params["SignatureMethod"] = "HmacSHA256"
		params["SignatureVersion"] = "2"
		params["Timestamp"] = time.Now().UTC().Format("2006-01-02T15:04:05")

		req.URL.RawQuery = encodeSortQueryString(req, params)

		signStr := method + "\n" + ROOT_URL + "\n" + path + "\n" + req.URL.RawQuery

		hashed := hmac.New(sha256.New, []byte(h.APISecret))
		_, err = hashed.Write([]byte(signStr))
		if err != nil {
			return nil, err
		}
		signature := base64.StdEncoding.EncodeToString(hashed.Sum(nil))

		req.URL.RawQuery = req.URL.RawQuery + "&Signature=" + url.QueryEscape(signature)
	}

	resp, err := h.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(respBytes), err
}

// GET /v1/account/accounts
func (h *Huobi) GetAccounts() (*Accounts, error) {
	url := TRADING_URL + "/account/accounts"
	params := map[string]string{}
	resp, err := h.Do("GET", url, params, true)
	if err != nil {
		return nil, err
	}
	accounts := Accounts{}
	json.NewDecoder(resp).Decode(&accounts)
	if accounts.ErrMsg != "" {
		return &accounts, errors.New(accounts.ErrMsg)
	}
	return &accounts, nil
}

// POST /v1/order/orders/place Make an order in huobi.pro
func (h *Huobi) PlaceLimitOrder(symbol string, isBuy bool, amount, price decimal.Decimal) (*NewOrder, error) {
	url := TRADING_URL + "/order/orders/place"
	orderTypes := map[bool]string{
		false: "sell-limit",
		true:  "buy-limit",
	}
	params := map[string]string{
		"account-id": strconv.Itoa(int(h.TradingAccount.ID)),
		"amount":     amount.String(),
		"price":      price.String(),
		"source":     "api", // 'api' for spot trade and 'margin-api' for margin trade
		"symbol":     symbol,
		"type":       orderTypes[isBuy],
	}
	resp, err := h.Do("POST", url, params, true)
	if err != nil {
		return nil, err
	}
	no := NewOrder{}
	json.NewDecoder(resp).Decode(&no)
	if no.ErrMsg != "" {
		return &no, errors.New(no.ErrMsg)
	}
	return &no, nil
}
