package engine

import (
    "sync"
    "time"
)

type MatchingEngine struct {
    orderBooks map[string]*OrderBook
    mutex      sync.RWMutex
    tradesChan chan *Trade
    ordersChan chan *Order
}

func NewMatchingEngine() *MatchingEngine {
    return &MatchingEngine{
        orderBooks: make(map[string]*OrderBook),
        tradesChan: make(chan *Trade, 10000),
        ordersChan: make(chan *Order, 10000),
    }
}

func (me *MatchingEngine) GetOrCreateOrderBook(symbol string) *OrderBook {
    me.mutex.RLock()
    ob, exists := me.orderBooks[symbol]
    me.mutex.RUnlock()
    
    if !exists {
        me.mutex.Lock()
        // Double-check after acquiring write lock
        if ob, exists = me.orderBooks[symbol]; !exists {
            ob = NewOrderBook(symbol)
            me.orderBooks[symbol] = ob
        }
        me.mutex.Unlock()
    }
    
    return ob
}

func (me *MatchingEngine) ProcessOrder(order *Order) []*Trade {
    order.Timestamp = time.Now()
    
    ob := me.GetOrCreateOrderBook(order.Symbol)
    trades := ob.AddOrder(order)
    
    // Send order update
    select {
    case me.ordersChan <- order:
    default:
        // Channel full, handle appropriately
    }
    
    // Send trades
    for _, trade := range trades {
        select {
        case me.tradesChan <- trade:
        default:
            // Channel full, handle appropriately
        }
    }
    
    return trades
}

func (me *MatchingEngine) CancelOrder(symbol, orderID string) bool {
    me.mutex.RLock()
    ob, exists := me.orderBooks[symbol]
    me.mutex.RUnlock()
    
    if !exists {
        return false
    }
    
    return ob.CancelOrder(orderID)
}

func (me *MatchingEngine) GetOrderBookSnapshot(symbol string) *OrderBookSnapshot {
    me.mutex.RLock()
    ob, exists := me.orderBooks[symbol]
    me.mutex.RUnlock()
    
    if !exists {
        return nil
    }
    
    return ob.GetSnapshot()
}

func (me *MatchingEngine) GetTradesChannel() <-chan *Trade {
    return me.tradesChan
}

func (me *MatchingEngine) GetOrdersChannel() <-chan *Order {
    return me.ordersChan
}