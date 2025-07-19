package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"high-frequency-matching-engine/config"
	"high-frequency-matching-engine/engine"
	"high-frequency-matching-engine/marketdata"
	"high-frequency-matching-engine/strategy"
	"high-frequency-matching-engine/utils"
)

func main() {
	// Initialize logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Load configuration
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	// Initialize matching engine
	matchingEngine := engine.NewMatchingEngine()

	// Initialize latency tracker
	latencyTracker := utils.NewLatencyTracker(logger)

	// Initialize strategies
	strategies := []strategy.Strategy{
		strategy.NewMarketMakerStrategy("BTCUSDT", 0.001, 0.01),
	}

	// Start market data feeders
	var feeders []*marketdata.MarketDataFeeder
	for _, exchCfg := range cfg.Exchanges {
		feeder := marketdata.NewMarketDataFeeder(exchCfg.WSUrl, exchCfg.Symbols, logger)
		if err := feeder.Connect(); err != nil {
			logger.Error("Failed to connect to market data feed",
				zap.String("exchange", exchCfg.Name),
				zap.Error(err))
			continue
		}
		feeder.StartFeed()
		feeders = append(feeders, feeder)
	}

	// Start metrics server if enabled
	if cfg.Metrics.Enabled {
		go func() {
			http.Handle("/metrics", promhttp.Handler())
			logger.Info("Starting metrics server", zap.Int("port", cfg.Metrics.Port))
			log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Metrics.Port), nil))
		}()
	}

	// Main event loop
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return

			case trade := <-matchingEngine.GetTradesChannel():
				latencyTracker.LogTrade(trade.Symbol, trade.Price, trade.Quantity)

				// Notify strategies of trade
				for _, strat := range strategies {
					orders := strat.OnTrade(trade)
					for _, order := range orders {
						startTime := time.Now()
						matchingEngine.ProcessOrder(order)
						latencyTracker.TrackOrderLatency(order.Symbol, startTime)
					}
				}

			case order := <-matchingEngine.GetOrdersChannel():
				utils.OrdersProcessed.WithLabelValues(
					order.Symbol,
					fmt.Sprintf("%d", order.Side),
					fmt.Sprintf("%d", order.Type),
				).Inc()

				// Notify strategies of order update
				for _, strat := range strategies {
					orders := strat.OnOrderUpdate(order)
					for _, newOrder := range orders {
						startTime := time.Now()
						matchingEngine.ProcessOrder(newOrder)
						latencyTracker.TrackOrderLatency(newOrder.Symbol, startTime)
					}
				}
			}
		}
	}()

	// Handle market data from feeders
	for _, feeder := range feeders {
		go func(f *marketdata.MarketDataFeeder) {
			for {
				select {
				case <-ctx.Done():
					return
				case data := <-f.GetDataChannel():
					// Notify strategies of market data
					for _, strat := range strategies {
						orders := strat.OnMarketData(data)
						for _, order := range orders {
							startTime := time.Now()
							matchingEngine.ProcessOrder(order)
							latencyTracker.TrackOrderLatency(order.Symbol, startTime)
						}
					}
				}
			}
		}(feeder)
	}

	// Start HTTP API server
	go startAPIServer(cfg.Server.Port, matchingEngine, logger)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down...")
	cancel()

	// Close market data feeders
	for _, feeder := range feeders {
		feeder.Close()
	}

	logger.Info("Shutdown complete")
}

func startAPIServer(port int, matchingEngine *engine.MatchingEngine, logger *zap.Logger) {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// Order placement endpoint
	mux.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var order *engine.Order
		if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Generate order ID if not provided
		if order.ID == "" {
			order.ID = fmt.Sprintf("API_%d", time.Now().UnixNano())
		}

		startTime := time.Now()
		trades := matchingEngine.ProcessOrder(order)

		response := map[string]interface{}{
			"order":      order,
			"trades":     trades,
			"latency_us": time.Since(startTime).Microseconds(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Order book snapshot endpoint
	mux.HandleFunc("/orderbook", func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Query().Get("symbol")
		if symbol == "" {
			http.Error(w, "Symbol parameter required", http.StatusBadRequest)
			return
		}

		snapshot := matchingEngine.GetOrderBookSnapshot(symbol) // Changed from engine.GetOrderBookSnapshot
		if snapshot == nil {
			http.Error(w, "Symbol not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(snapshot)
	})

	// Cancel order endpoint
	mux.HandleFunc("/orders/cancel", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		symbol := r.URL.Query().Get("symbol")
		orderID := r.URL.Query().Get("order_id")

		if symbol == "" || orderID == "" {
			http.Error(w, "Symbol and order_id parameters required", http.StatusBadRequest)
			return
		}

		success := matchingEngine.CancelOrder(symbol, orderID) // Changed from engine.CancelOrder

		response := map[string]bool{"cancelled": success}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	logger.Info("Starting API server", zap.Int("port", port))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), mux))
}
