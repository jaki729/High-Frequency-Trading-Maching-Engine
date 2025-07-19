package strategy

import (
    "fmt"
    "high-frequency-matching-engine/engine"
    "time"
)

type MarketMakerStrategy struct {
    BaseStrategy
    symbol     string
    spread     float64
    quantity   float64
    lastPrice  float64
    activeOrders map[string]*engine.Order
}

func NewMarketMakerStrategy(symbol string, spread, quantity float64) *MarketMakerStrategy {
    return &MarketMakerStrategy{
        BaseStrategy: BaseStrategy{Name: "MarketMaker"},
        symbol:      symbol,
        spread:      spread,
        quantity:    quantity,
        activeOrders: make(map[string]*engine.Order),
    }
}

func (mms *MarketMakerStrategy) OnMarketData(data *engine.MarketData) []*engine.Order {
    if data.Symbol != mms.symbol {
        return nil
    }
    
    mms.lastPrice = data.Price
    return mms.generateOrders()
}

func (mms *MarketMakerStrategy) OnTrade(trade *engine.Trade) []*engine.Order {
    if trade.Symbol != mms.symbol {
        return nil
    }
    
    mms.lastPrice = trade.Price
    return mms.generateOrders()
}

func (mms *MarketMakerStrategy) OnOrderUpdate(order *engine.Order) []*engine.Order {
    if order.Status == engine.FILLED || order.Status == engine.CANCELLED {
        delete(mms.activeOrders, order.ID)
    }
    return nil
}

func (mms *MarketMakerStrategy) generateOrders() []*engine.Order {
    if mms.lastPrice <= 0 {
        return nil
    }
    
    var orders []*engine.Order
    
    // Cancel existing orders (simplified)
    for orderID := range mms.activeOrders {
        // In a real implementation, you'd send cancel requests to the engine
        delete(mms.activeOrders, orderID)
    }
    
    // Generate new bid and ask orders
    bidPrice := mms.lastPrice * (1 - mms.spread/2)
    askPrice := mms.lastPrice * (1 + mms.spread/2)
    
    bidOrder := &engine.Order{
        ID:       fmt.Sprintf("MM_BID_%d", time.Now().UnixNano()),
        Symbol:   mms.symbol,
        Side:     engine.BUY,
        Type:     engine.LIMIT,
        Quantity: mms.quantity,
        Price:    bidPrice,
        Status:   engine.PENDING,
        ClientID: "MarketMaker",
    }
    
    askOrder := &engine.Order{
        ID:       fmt.Sprintf("MM_ASK_%d", time.Now().UnixNano()),
        Symbol:   mms.symbol,
        Side:     engine.SELL,
        Type:     engine.LIMIT,
        Quantity: mms.quantity,
        Price:    askPrice,
        Status:   engine.PENDING,
        ClientID: "MarketMaker",
    }
    
    orders = append(orders, bidOrder, askOrder)
    mms.activeOrders[bidOrder.ID] = bidOrder
    mms.activeOrders[askOrder.ID] = askOrder
    
    return orders
}