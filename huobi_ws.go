package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/rcolombo/tickertock/common"
	"github.com/rcolombo/tickertock/types"
	"github.com/shopspring/decimal"
)

const (
	WS_URL                 = "wss://api.huobi.pro/ws"
	TIMEOUT_RECONNECT_SECS = 1800
	WRITE_TIMEOUT_SECS     = 20
)

type Subbed struct {
	Subbed string `json:"subbed"`
	Ts     int    `json:"ts"`
	Status string `json:"status"`
	ID     string `json:"id"`
}

type DepthUpdate struct {
	Ch   string    `json:"ch"`
	Ts   int       `json:"ts"`
	Tick DepthTick `json:"tick"`
}
type DepthTick struct {
	Bids    [][]float64 `json:"bids"`
	Asks    [][]float64 `json:"asks"`
	Ts      int         `json:"ts"`
	Version int         `json:"vesion"`
}

func (h *Huobi) GetOrderBookUpdates(subscribeToCh chan types.CurrencyPair) (chan *types.BookOrder, chan types.CurrencyPair) {
	var (
		outgoing       = make(chan *types.BookOrder)
		updateHandlers = make(map[string]chan DepthUpdate) // Maps a currency pair to its update handler channel
		subscribers    = make(map[string]types.CurrencyPair)
		subscribed     = make(map[string]types.CurrencyPair) // Maps a subscription id to the currency pair
		channelNames   = make(map[string]types.CurrencyPair) // Maps a channel name (e.g. market.ltcusdt.depth.step0) to a currency pair
		disconnectCh   = make(chan types.CurrencyPair)
		ws             common.Websocket
	)

	go func() {
		pingTicker := time.NewTicker(time.Second * 5) // ping the server every second to keep our connection alive
		var (
			connectCh = make(chan bool, 2)
			pongCh    = make(chan interface{}, 100)
		)
		connectCh <- true
		for {
			select {
			case <-connectCh:
				ws.Close()
				ws = common.Websocket{}
				err := ws.Connect(WS_URL, http.Header{})

				if err != nil {
					log.Printf("[Huobi] Error when connecting to websocket, will retry: %v\n", err)
					time.Sleep(5 * time.Second)
					connectCh <- true
				}

				// Resubscribe to any pairs we had previously subscribed to
				for _, cp := range subscribers {
					subscribeToCh <- cp
				}

			case <-disconnectCh:
				// We might need to explicitly disconnect here
				return

			case cp := <-subscribeToCh:
				// handle updates in parallel
				handler, ok := updateHandlers[cp.Key()]
				if !ok {
					handler = make(chan DepthUpdate, 100)
					go h.UpdateHandler(cp, handler, outgoing)
					updateHandlers[cp.Key()] = handler
				}

				// Keep track of which pairs are subscribed in case we need to reconnect later
				subscribers[cp.Key()] = cp

				symbol := fmt.Sprintf("%v%v", strings.ToLower(cp.Base), strings.ToLower(cp.Quote))
				id := fmt.Sprintf("id_%v_%v", symbol, time.Now().Nanosecond())
				msg := []byte(fmt.Sprintf(`{"sub": "market.%v.depth.step0", "id": "%v"}"`, symbol, id))
				err := ws.WriteTextMessage(msg, WRITE_TIMEOUT_SECS)
				if err != nil {
					log.Printf("[Huobi] Error subscribing for %v: %v\n", symbol, err)
					if len(connectCh) == 0 {
						connectCh <- true // This might be overly aggressive
					}
				}
				subscribed[id] = cp

			case <-pingTicker.C:
				msg := []byte(fmt.Sprintf(`{"ping": %v}`, common.TsInMs()))
				err := ws.WriteTextMessage(msg, WRITE_TIMEOUT_SECS)
				if err != nil {
					log.Println("[Huobi] error pinging server: ", err)
					if len(connectCh) == 0 {
						connectCh <- true
					}
				}

			case ts := <-pongCh:
				msg := map[string]interface{}{"pong": ts}
				msgBytes, _ := json.Marshal(msg)
				err := ws.WriteTextMessage(msgBytes, WRITE_TIMEOUT_SECS)
				if err != nil {
					log.Println("[Huobi] error ponging server: ", err)
					if len(connectCh) == 0 {
						connectCh <- true
					}
					continue
				}

			default:
				_, msgBytes, err := ws.ReadMessage(TIMEOUT_RECONNECT_SECS)
				if err != nil {
					log.Printf("[Huobi] Error reading websocket: %v", err)
					if len(connectCh) == 0 {
						connectCh <- true
					}
					continue
				}

				// We need to unzip Huobi's responses
				b := bytes.NewBuffer(msgBytes)
				r, err := gzip.NewReader(b)
				if err != nil {
					log.Printf("[Huobi] Error unzipping websocket response: %v", err)
					if len(connectCh) == 0 {
						connectCh <- true
					}
					continue
				}

				data, _ := ioutil.ReadAll(r)
				if err != nil {
					log.Printf("[Huobi] Error reading data from websocket response: %v", err)
				}
				r.Close()

				respMap := map[string]interface{}{}
				json.Unmarshal(data, &respMap)
				if v, ok := respMap["status"]; ok && v == "error" {
					log.Printf("[Huobi] Websocket Error. Code: %v, Msg: %v", respMap["err-code"], respMap["err-msg"])
					continue
				}

				if v, ok := respMap["ping"]; ok {
					// We need to respond to pings with a pong
					pongCh <- v
					continue
				}

				if _, ok := respMap["pong"]; ok {
					// A response to our pings
					continue
				}

				if _, ok := respMap["subbed"]; ok {
					subbed := Subbed{}
					mapstructure.Decode(respMap, &subbed)
					cp, ok := subscribed[subbed.ID]
					if !ok {
						log.Println("Unable to find matching currency pair for subscribed ID %v", subbed.ID)
					}
					channelNames[subbed.Subbed] = cp
					continue
				}

				if _, ok := respMap["ch"]; ok {
					update := DepthUpdate{}
					mapstructure.Decode(respMap, &update)
					cp, ok := channelNames[update.Ch]
					if !ok {
						log.Println("Unable to find matching currency pair for channel %v", update.Ch)
						continue
					}

					updateCh, ok := updateHandlers[cp.Key()]
					if !ok {
						log.Println("Unable to find update handler for pair %v", cp.Key())
						continue
					}

					updateCh <- update
					continue
				}

				log.Println("[Huobi] Unhandled websocket type! ", respMap)
			}
		}
	}()
	return outgoing, disconnectCh
}

func (h *Huobi) UpdateHandler(cp types.CurrencyPair, updateCh chan DepthUpdate, outgoingCh chan *types.BookOrder) {
	var lastTick DepthTick
	for update := range updateCh {
		var bidLimit, askLimit = 5, 5
		if len(update.Tick.Bids) < bidLimit {
			bidLimit = len(update.Tick.Bids)
		}
		if len(update.Tick.Asks) < askLimit {
			askLimit = len(update.Tick.Asks)
		}
		update.Tick.Bids = update.Tick.Bids[:bidLimit]
		update.Tick.Asks = update.Tick.Asks[:askLimit]

		removedBids, newBids := h.GetDepthChanges(lastTick.Bids, update.Tick.Bids)
		removedAsks, newAsks := h.GetDepthChanges(lastTick.Asks, update.Tick.Asks)

		for idx, ask := range [][][]float64{newAsks, removedAsks} {
			isBuy := false
			isDelete := idx == 1
			for _, o := range ask {
				price := decimal.NewFromFloatWithExponent(o[0], -8)
				amount := decimal.NewFromFloatWithExponent(o[1], -8)
				outgoingCh <- types.NewBookOrder(h, cp, isBuy, isDelete, price, amount)
			}
		}
		for idx, bid := range [][][]float64{newBids, removedBids} {
			isBuy := true
			isDelete := idx == 1
			for _, o := range bid {
				price := decimal.NewFromFloatWithExponent(o[0], -8)
				amount := decimal.NewFromFloatWithExponent(o[1], -8)
				outgoingCh <- types.NewBookOrder(h, cp, isBuy, isDelete, price, amount)
			}
		}
		lastTick = update.Tick
	}
}
