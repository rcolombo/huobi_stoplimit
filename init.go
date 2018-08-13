package main

import (
	"flag"
	"log"
	"runtime"
	"strings"

	"github.com/rcolombo/tickertock/types"
	"github.com/shopspring/decimal"
)

var (
	apiKey     = flag.String("api-key", "", "Your Huobi API Key")
	apiSecret  = flag.String("api-secret", "", "Your Huobi API Secret")
	market     = flag.String("market", "", "The name of the market in the form of base_quote (e.g. zrx_btc, xrp_btc")
	stopPrice  = flag.String("stop-price", "", "Issue order when the price hits this amount")
	limitPrice = flag.String("limit-price", "", "The price on your order")
	amount     = flag.String("amount", "", "Amount to order")
	orderSide  = flag.String("order-side", "", "buy or sell")

	cp            types.CurrencyPair
	stopPriceDec  decimal.Decimal
	limitPriceDec decimal.Decimal
	amountDec     decimal.Decimal
	isBuy         bool
)

func init() {
	var err error
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()

	if *apiKey == "" {
		log.Fatal("Please enter an API Key")
	}

	if *apiSecret == "" {
		log.Fatal("Please enter an API Secret Key")
	}

	if *stopPrice == "" {
		log.Fatal("Please enter a stop price")
	}

	if *limitPrice == "" {
		log.Fatal("Please enter a limit price")
	}

	if *amount == "" {
		log.Fatal("Please enter an amount")
	}

	if *market == "" {
		log.Fatal("Please enter a valid market (e.g zrx_btc)")
	}
	baseQuote := strings.Split(*market, "_")
	if len(baseQuote) != 2 {
		log.Fatal("Please enter a valid market (e.g zrx_btc)")
	}
	cp = types.CurrencyPair{Base: baseQuote[0], Quote: baseQuote[1]}

	orderSideLower := strings.ToLower(*orderSide)
	if !(orderSideLower == "buy" || orderSideLower == "sell") {
		log.Fatal("Invalid order side. Please pick one of 'buy', 'sell'")
	}
	isBuy = orderSideLower == "buy"

	stopPriceDec, err = decimal.NewFromString(*stopPrice)
	if err != nil {
		log.Fatal("Error parsing stop price: ", err)
	}

	limitPriceDec, err = decimal.NewFromString(*limitPrice)
	if err != nil {
		log.Fatal("Error parsing limit price: ", err)
	}

	amountDec, err = decimal.NewFromString(*amount)
	if err != nil {
		log.Fatal("Error parsing amount: ", err)
	}
}
