package engine

import (
    "time"
)

type OrderSide int

const (
    BUY OrderSide = iota
    SELL 
)

type OrderType int

const (
    MARKET OrderType = iota
    LIMIT
)

type OrderStatus int

const (
    PENDING OrderStatus = iota
    PARTIAL
    FILLED
    CANCELLED
)

type Order struct {
    ID          string      `json:"id"`
    Symbol      string      `json:"symbol"`
    Side        OrderSide   `json:"side"`
    Type        OrderType   `json:"type"`
    Quantity    float64     `json:"quantity"`
    Price       float64     `json:"price"`
    Filled      float64     `json:"filled"`
    Status      OrderStatus `json:"status"`
    Timestamp   time.Time   `json:"timestamp"`
    ClientID    string      `json:"client_id"`
}

type Trade struct {
    ID           string    `json:"id"`
    Symbol       string    `json:"symbol"`
    BuyOrderID   string    `json:"buy_order_id"`
    SellOrderID  string    `json:"sell_order_id"`
    Price        float64   `json:"price"`
    Quantity     float64   `json:"quantity"`
    Timestamp    time.Time `json:"timestamp"`
}

type MarketData struct {
    Symbol    string    `json:"symbol"`
    Price     float64   `json:"price"`
    Quantity  float64   `json:"quantity"`
    Side      OrderSide `json:"side"`
    Timestamp time.Time `json:"timestamp"`
}

type OrderBookLevel struct {
    Price    float64 `json:"price"`
    Quantity float64 `json:"quantity"`
    Orders   int     `json:"orders"`
}

type OrderBookSnapshot struct {
    Symbol    string           `json:"symbol"`
    Bids      []OrderBookLevel `json:"bids"`
    Asks      []OrderBookLevel `json:"asks"`
    Timestamp time.Time        `json:"timestamp"`
}