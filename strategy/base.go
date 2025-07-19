package strategy

import (
    "high-frequency-matching-engine/engine"
)

type Strategy interface {
    OnMarketData(data *engine.MarketData) []*engine.Order
    OnTrade(trade *engine.Trade) []*engine.Order
    OnOrderUpdate(order *engine.Order) []*engine.Order
    GetName() string
}

type BaseStrategy struct {
    Name string
}

func (bs *BaseStrategy) GetName() string {
    return bs.Name
}

func (bs *BaseStrategy) OnMarketData(data *engine.MarketData) []*engine.Order {
    return nil
}

func (bs *BaseStrategy) OnTrade(trade *engine.Trade) []*engine.Order {
    return nil
}

func (bs *BaseStrategy) OnOrderUpdate(order *engine.Order) []*engine.Order {
    return nil
}