package utils

import (
    "time"
    "go.uber.org/zap"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)
var (
    OrdersProcessed = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "orders_processed_total",
            Help: "Total number of orders processed",
        },
        []string{"symbol", "side", "type"},
    )
    
    TradesExecuted = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "trades_executed_total",
            Help: "Total number of trades executed",
        },
        []string{"symbol"},
    )
    
    OrderProcessingLatency = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "order_processing_latency_seconds",
            Help:    "Order processing latency in seconds",
            Buckets: prometheus.ExponentialBuckets(0.000001, 2, 20), // Start at 1μs
        },
        []string{"symbol"},
    )
    
    OrderBookDepth = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "orderbook_depth",
            Help: "Current order book depth",
        },
        []string{"symbol", "side"},
    )
)

type LatencyTracker struct {
    logger *zap.Logger
}

func NewLatencyTracker(logger *zap.Logger) *LatencyTracker {
    return &LatencyTracker{logger: logger}
}

func (lt *LatencyTracker) TrackOrderLatency(symbol string, startTime time.Time) {
    duration := time.Since(startTime)
    OrderProcessingLatency.WithLabelValues(symbol).Observe(duration.Seconds())
    
    if duration > time.Millisecond {
        lt.logger.Warn("High order processing latency",
            zap.String("symbol", symbol),
            zap.Duration("latency", duration),
        )
    }
}

func (lt *LatencyTracker) LogTrade(symbol string, price, quantity float64) {
    TradesExecuted.WithLabelValues(symbol).Inc()
    lt.logger.Info("Trade executed",
        zap.String("symbol", symbol),
        zap.Float64("price", price),
        zap.Float64("quantity", quantity),
    )
}

