package engine

import (
    "container/heap"
    "sort"
    "sync"
    "time"
	"fmt"
	"sync/atomic"
)

// Priority queue for buy orders (max heap)
type BuyOrderQueue []*Order

func (pq BuyOrderQueue) Len() int { return len(pq) }

func (pq BuyOrderQueue) Less(i, j int) bool {
    if pq[i].Price == pq[j].Price {
        return pq[i].Timestamp.Before(pq[j].Timestamp)
    }
    return pq[i].Price > pq[j].Price // Max heap for buy orders
}

func (pq BuyOrderQueue) Swap(i, j int) {
    pq[i], pq[j] = pq[j], pq[i]
}

func (pq *BuyOrderQueue) Push(x interface{}) {
    *pq = append(*pq, x.(*Order))
}

func (pq *BuyOrderQueue) Pop() interface{} {
    old := *pq
    n := len(old)
    item := old[n-1]
    *pq = old[0 : n-1]
    return item
}

// Priority queue for sell orders (min heap)
type SellOrderQueue []*Order

func (pq SellOrderQueue) Len() int { return len(pq) }

func (pq SellOrderQueue) Less(i, j int) bool {
    if pq[i].Price == pq[j].Price {
        return pq[i].Timestamp.Before(pq[j].Timestamp)
    }
    return pq[i].Price < pq[j].Price // Min heap for sell orders
}

func (pq SellOrderQueue) Swap(i, j int) {
    pq[i], pq[j] = pq[j], pq[i]
}

func (pq *SellOrderQueue) Push(x interface{}) {
    *pq = append(*pq, x.(*Order))
}

func (pq *SellOrderQueue) Pop() interface{} {
    old := *pq
    n := len(old)
    item := old[n-1]
    *pq = old[0 : n-1]
    return item
}

type OrderBook struct {
    Symbol     string
    BuyOrders  *BuyOrderQueue
    SellOrders *SellOrderQueue
    Orders     map[string]*Order
    LastPrice  float64
    mutex      sync.RWMutex
    tradeSeq   int64
}

func NewOrderBook(symbol string) *OrderBook {
    buyQueue := &BuyOrderQueue{}
    sellQueue := &SellOrderQueue{}
    heap.Init(buyQueue)
    heap.Init(sellQueue)
    
    return &OrderBook{
        Symbol:     symbol,
        BuyOrders:  buyQueue,
        SellOrders: sellQueue,
        Orders:     make(map[string]*Order),
        tradeSeq:   0,
    }
}

func (ob *OrderBook) AddOrder(order *Order) []*Trade {
    ob.mutex.Lock()
    defer ob.mutex.Unlock()
    
    if order.Type == MARKET {
        return ob.processMarketOrder(order)
    }
    return ob.processLimitOrder(order)
}

func (ob *OrderBook) processMarketOrder(order *Order) []*Trade {
    var trades []*Trade
    remaining := order.Quantity
    
    if order.Side == BUY {
        // Match against sell orders
        for ob.SellOrders.Len() > 0 && remaining > 0 {
            bestSell := (*ob.SellOrders)[0]
            matchQty := min(remaining, bestSell.Quantity-bestSell.Filled)
            
            trade := ob.createTrade(order, bestSell, bestSell.Price, matchQty)
            trades = append(trades, trade)
            
            remaining -= matchQty
            order.Filled += matchQty
            bestSell.Filled += matchQty
            
            if bestSell.Filled >= bestSell.Quantity {
                bestSell.Status = FILLED
                heap.Pop(ob.SellOrders)
                delete(ob.Orders, bestSell.ID)
            }
        }
    } else {
        // Match against buy orders
        for ob.BuyOrders.Len() > 0 && remaining > 0 {
            bestBuy := (*ob.BuyOrders)[0]
            matchQty := min(remaining, bestBuy.Quantity-bestBuy.Filled)
            
            trade := ob.createTrade(bestBuy, order, bestBuy.Price, matchQty)
            trades = append(trades, trade)
            
            remaining -= matchQty
            order.Filled += matchQty
            bestBuy.Filled += matchQty
            
            if bestBuy.Filled >= bestBuy.Quantity {
                bestBuy.Status = FILLED
                heap.Pop(ob.BuyOrders)
                delete(ob.Orders, bestBuy.ID)
            }
        }
    }
    
    if order.Filled >= order.Quantity {
        order.Status = FILLED
    } else if order.Filled > 0 {
        order.Status = PARTIAL
    }
    
    return trades
}

func (ob *OrderBook) processLimitOrder(order *Order) []*Trade {
    var trades []*Trade
    remaining := order.Quantity
    
    if order.Side == BUY {
        // Try to match against sell orders
        for ob.SellOrders.Len() > 0 && remaining > 0 {
            bestSell := (*ob.SellOrders)[0]
            if order.Price < bestSell.Price {
                break // No more matches possible
            }
            
            matchQty := min(remaining, bestSell.Quantity-bestSell.Filled)
            trade := ob.createTrade(order, bestSell, bestSell.Price, matchQty)
            trades = append(trades, trade)
            
            remaining -= matchQty
            order.Filled += matchQty
            bestSell.Filled += matchQty
            
            if bestSell.Filled >= bestSell.Quantity {
                bestSell.Status = FILLED
                heap.Pop(ob.SellOrders)
                delete(ob.Orders, bestSell.ID)
            }
        }
        
        // Add remaining quantity to order book
        if remaining > 0 {
            order.Quantity = remaining
            heap.Push(ob.BuyOrders, order)
            ob.Orders[order.ID] = order
        }
    } else {
        // Try to match against buy orders
        for ob.BuyOrders.Len() > 0 && remaining > 0 {
            bestBuy := (*ob.BuyOrders)[0]
            if order.Price > bestBuy.Price {
                break // No more matches possible
            }
            
            matchQty := min(remaining, bestBuy.Quantity-bestBuy.Filled)
            trade := ob.createTrade(bestBuy, order, bestBuy.Price, matchQty)
            trades = append(trades, trade)
            
            remaining -= matchQty
            order.Filled += matchQty
            bestBuy.Filled += matchQty
            
            if bestBuy.Filled >= bestBuy.Quantity {
                bestBuy.Status = FILLED
                heap.Pop(ob.BuyOrders)
                delete(ob.Orders, bestBuy.ID)
            }
        }
        
        // Add remaining quantity to order book
        if remaining > 0 {
            order.Quantity = remaining
            heap.Push(ob.SellOrders, order)
            ob.Orders[order.ID] = order
        }
    }
    
    if order.Filled >= order.Quantity {
        order.Status = FILLED
    } else if order.Filled > 0 {
        order.Status = PARTIAL
    }
    
    return trades
}

func (ob *OrderBook) createTrade(buyOrder, sellOrder *Order, price, quantity float64) *Trade {
    tradeID := atomic.AddInt64(&ob.tradeSeq, 1)
    ob.LastPrice = price
    
    return &Trade{
        ID:          fmt.Sprintf("T%d", tradeID),
        Symbol:      ob.Symbol,
        BuyOrderID:  buyOrder.ID,
        SellOrderID: sellOrder.ID,
        Price:       price,
        Quantity:    quantity,
        Timestamp:   time.Now(),
    }
}

func (ob *OrderBook) CancelOrder(orderID string) bool {
    ob.mutex.Lock()
    defer ob.mutex.Unlock()
    
    order, exists := ob.Orders[orderID]
    if !exists {
        return false
    }
    
    order.Status = CANCELLED
    delete(ob.Orders, orderID)
    
    // Remove from appropriate queue (this is simplified - in production you'd need more efficient removal)
    ob.rebuildQueues()
    return true
}

func (ob *OrderBook) rebuildQueues() {
    // Rebuild queues without cancelled orders
    buyQueue := &BuyOrderQueue{}
    sellQueue := &SellOrderQueue{}
    
    for _, order := range ob.Orders {
        if order.Status != CANCELLED {
            if order.Side == BUY {
                heap.Push(buyQueue, order)
            } else {
                heap.Push(sellQueue, order)
            }
        }
    }
    
    ob.BuyOrders = buyQueue
    ob.SellOrders = sellQueue
}

func (ob *OrderBook) GetSnapshot() *OrderBookSnapshot {
    ob.mutex.RLock()
    defer ob.mutex.RUnlock()
    
    var bids, asks []OrderBookLevel
    
    // Aggregate buy orders by price level
    buyLevels := make(map[float64]float64)
    for _, order := range *ob.BuyOrders {
        buyLevels[order.Price] += order.Quantity - order.Filled
    }
    
    for price, qty := range buyLevels {
        bids = append(bids, OrderBookLevel{Price: price, Quantity: qty, Orders: 1})
    }
    
    // Aggregate sell orders by price level
    sellLevels := make(map[float64]float64)
    for _, order := range *ob.SellOrders {
        sellLevels[order.Price] += order.Quantity - order.Filled
    }
    
    for price, qty := range sellLevels {
        asks = append(asks, OrderBookLevel{Price: price, Quantity: qty, Orders: 1})
    }
    
    // Sort bids (highest first) and asks (lowest first)
    sort.Slice(bids, func(i, j int) bool { return bids[i].Price > bids[j].Price })
    sort.Slice(asks, func(i, j int) bool { return asks[i].Price < asks[j].Price })
    
    return &OrderBookSnapshot{
        Symbol:    ob.Symbol,
        Bids:      bids,
        Asks:      asks,
        Timestamp: time.Now(),
    }
}

func min(a, b float64) float64 {
    if a < b {
        return a
    }
    return b
}