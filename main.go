package main

import (
	"log"

	"github.com/rcolombo/tickertock/types"
)

func main() {
	h, err := New(*apiKey, *apiSecret)
	if err != nil {
		log.Fatal("Error creating Huobi client: ", err)
	}
	bookOrderCh, _ := h.SubscribeForPair(cp)
	book := types.NewBook(10)
	for o := range bookOrderCh {
		newHigh, newLow := book.ProcessBookOrder(*o)
		if isBuy && newLow {
			la := book.GetLowAsk()
			if la.Price.LessThanOrEqual(stopPriceDec) {
				newOrder, err := h.PlaceLimitOrder(h.Symbolize(cp), isBuy, amountDec, limitPriceDec)
				if err != nil {
					log.Fatal("Error placing order: ", err)
				}
				log.Println("Order Placed: %+v", newOrder)
				return
			}
		}
		if !isBuy && newHigh {
			hb := book.GetHighBid()
			if hb.Price.LessThanOrEqual(stopPriceDec) {
				newOrder, err := h.PlaceLimitOrder(h.Symbolize(cp), isBuy, amountDec, limitPriceDec)
				if err != nil {
					log.Fatal("Error placing order: ", err)
				}
				log.Println("Order Placed: %+v", newOrder)
				return
			}
		}
	}
}
