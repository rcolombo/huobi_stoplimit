package main

type BaseResponse struct {
	Status  string `json:"status"`
	Ch      string `json:"ch"`
	Ts      int64  `json:"ts"`
	ErrCode string `json:"err-code"`
	ErrMsg  string `json:"err-msg"`
}

// GET /acount/accounts
type Accounts struct {
	BaseResponse
	Data []Account `json:"data"`
}
type Account struct {
	ID     int64  `json:"id"`
	UserID int64  `json:"user-id"`
	Type   string `json:"type"`
	State  string `json:"state"`
}

// POST /v1/order/orders/place
type NewOrder struct {
	BaseResponse
	OrderID string `json:"data"`
}
