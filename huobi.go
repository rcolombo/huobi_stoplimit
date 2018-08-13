package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/rcolombo/tickertock/types"
	"github.com/shopspring/decimal"
)

type Huobi struct {
	types.BaseExchange
	HTTPClient     http.Client
	APIKey         string
	APISecret      string
	TradingAccount Account
}

func New(apiKey, apiSecret string) (*Huobi, error) {
	h := Huobi{
		HTTPClient: http.Client{Timeout: time.Second * 10},
		APIKey:     apiKey,
		APISecret:  apiSecret,
	}
	accounts, err := h.GetAccounts()
	if err != nil {
		return nil, fmt.Errorf("Error retrieving accounts: %v", err)
	}
	if len(accounts.Data) != 1 {
		return nil, fmt.Errorf("Expected exactly 1 account, but got something else: %+v", accounts.Data)
	}
	h.TradingAccount = accounts.Data[0]
	return &h, nil

}

func (h *Huobi) Symbolize(cp types.CurrencyPair) string {
	base := strings.ToLower(cp.Base)
	quote := strings.ToLower(cp.Quote)
	if base == "bcc" {
		base = "bch"
	} else if quote == "bcc" {
		quote = "bch"
	}
	return base + quote
}

func (h *Huobi) SubscribeForPair(cp types.CurrencyPair) (chan *types.BookOrder, chan bool) {
	var (
		subscribeTo  = make(chan types.CurrencyPair, 1000)
		disconnectCh = make(chan bool)
		outgoing     = make(chan *types.BookOrder, 1000000)
	)
	newBookOrderCh, disconnectForPairCh := h.GetOrderBookUpdates(subscribeTo)
	subscribeTo <- cp

	go func() {
		for {
			select {
			case <-disconnectCh:
				disconnectForPairCh <- cp
				return
			case nbo := <-newBookOrderCh:
				outgoing <- nbo
			}
		}
	}()
	return outgoing, disconnectCh
}

func (h *Huobi) SubscribePair(cp types.CurrencyPair) (*types.Book, chan bool) {
	var (
		book         *types.Book
		subscribeTo  = make(chan types.CurrencyPair, 1000)
		disconnectCh = make(chan bool)
		readyCh      = make(chan bool)
	)
	newBookOrderCh, disconnectForPairCh := h.GetOrderBookUpdates(subscribeTo)
	subscribeTo <- cp
	book = types.NewBook(10)
	h.SetBook(cp, book)
	go func() {
		var bidsLoaded, asksLoaded, sent bool
		for {
			select {
			case <-disconnectCh:
				disconnectForPairCh <- cp
				return
			case nbo := <-newBookOrderCh:
				newHigh, newLow := book.ProcessBookOrder(*nbo)
				bidsLoaded = bidsLoaded || newHigh
				asksLoaded = asksLoaded || newLow
				if bidsLoaded && asksLoaded && !sent {
					readyCh <- true
					sent = true
				}
			}
		}
	}()
	select {
	case <-readyCh:
	}
	return book, disconnectCh
}

func (h *Huobi) Name() string {
	return "Huobi"
}

func (h *Huobi) GetFees(cp types.CurrencyPair, isBuy bool) (decimal.Decimal, decimal.Decimal) {
	log.Fatal("[Huobi] `GetFees' not implemented!")
	return decimal.RequireFromString("0"), decimal.RequireFromString("0")
}

func (h *Huobi) PriceTickSize(cp types.CurrencyPair) *decimal.Decimal {
	log.Fatal("[Huobi] `PriceTickSize' not implemented!")
	return nil
}

func (h *Huobi) LotSize(cp types.CurrencyPair) *decimal.Decimal {
	log.Fatal("[Huobi] `LotSize' not implemented!")
	return nil
}

func (h *Huobi) MinBet(cp types.CurrencyPair) (*decimal.Decimal, bool, error) {
	log.Fatal("[Huobi] `MinBet' not implemented!")
	return nil, false, nil
}

func (h *Huobi) MaxBet(cp types.CurrencyPair) (*decimal.Decimal, bool, error) {
	log.Fatal("[Huobi] `MaxBet' not implemented!")
	return nil, false, nil
}

func (h *Huobi) GetAllBalances() (map[string]decimal.Decimal, error) {
	log.Fatal("[Huobi] `GetAllBalances' not implemented!")
	return map[string]decimal.Decimal{}, nil
}

func (h *Huobi) UpdateBalancesFromTrade(t types.Trade) {
	log.Fatal("[Huobi] `UpdateBalancesFromTrade' not implemented!")
}

func (h *Huobi) Tradeable(cp types.CurrencyPair) bool {
	log.Fatal("[Huobi] `Tradeable' not implemented!")
	return false
}

func (h *Huobi) GetTicker(cp types.CurrencyPair) (*types.Ticker, error) {
	log.Fatal("[Huobi] `GetTicker' not implemented!")
	return nil, nil
}

func (h *Huobi) Subscribe() chan *types.BookOrder {
	log.Fatal("[Huobi] `Subscribe' not implemented!")
	outgoing := make(chan *types.BookOrder)
	return outgoing
}

func (h *Huobi) LimitOrder(cp types.CurrencyPair, isBuy bool, amount, price decimal.Decimal, timeout time.Duration) (types.Order, error) {
	log.Fatal("[Huobi] `LimitOrder' not implemented!")
	return types.Order{}, nil
}
