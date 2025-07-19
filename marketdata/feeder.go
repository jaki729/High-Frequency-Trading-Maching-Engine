package marketdata

import (
    "encoding/json"
    "github.com/gorilla/websocket"
    "go.uber.org/zap"
    "high-frequency-matching-engine/engine"
    "time"
)
type MarketDataFeeder struct {
    conn     *websocket.Conn
    logger   *zap.Logger
    dataChan chan *engine.MarketData
    wsUrl    string
    symbols  []string
}

func NewMarketDataFeeder(wsUrl string, symbols []string, logger *zap.Logger) *MarketDataFeeder {
    return &MarketDataFeeder{
        logger:   logger,
        dataChan: make(chan *engine.MarketData, 10000),
        wsUrl:    wsUrl,
        symbols:  symbols,
    }
}

func (mdf *MarketDataFeeder) Connect() error {
    var err error
    mdf.conn, _, err = websocket.DefaultDialer.Dial(mdf.wsUrl, nil)
    if err != nil {
        return err
    }
    
    // Subscribe to symbols
    for _, symbol := range mdf.symbols {
        subscribeMsg := map[string]interface{}{
            "method": "SUBSCRIBE",
            "params": []string{symbol + "@ticker"},
            "id":     1,
        }
        
        if err := mdf.conn.WriteJSON(subscribeMsg); err != nil {
            mdf.logger.Error("Failed to subscribe", zap.String("symbol", symbol), zap.Error(err))
        }
    }
    
    return nil
}

func (mdf *MarketDataFeeder) StartFeed() {
    go func() {
        defer mdf.conn.Close()
        
        for {
            var rawMessage json.RawMessage
            err := mdf.conn.ReadJSON(&rawMessage)
            if err != nil {
                mdf.logger.Error("WebSocket read error", zap.Error(err))
                // Implement reconnection logic here
                time.Sleep(5 * time.Second)
                if err := mdf.Connect(); err != nil {
                    mdf.logger.Error("Reconnection failed", zap.Error(err))
                    continue
                }
            }
            
            // Parse market data (this is a simplified example)
            var tickerData struct {
                Symbol string  `json:"s"`
                Price  string  `json:"c"`
                Volume string  `json:"v"`
            }
            
            if err := json.Unmarshal(rawMessage, &tickerData); err != nil {
                continue // Skip malformed messages
            }
            
            // Convert to internal format and send to channel
            // This is a simplified conversion - real implementation would be more robust
            if tickerData.Symbol != "" {
                marketData := &engine.MarketData{
                    Symbol:    tickerData.Symbol,
                    Timestamp: time.Now(),
                }
                
                select {
                case mdf.dataChan <- marketData:
                default:
                    mdf.logger.Warn("Market data channel full, dropping message")
                }
            }
        }
    }()
}

func (mdf *MarketDataFeeder) GetDataChannel() <-chan *engine.MarketData {
    return mdf.dataChan
}

func (mdf *MarketDataFeeder) Close() error {
    if mdf.conn != nil {
        return mdf.conn.Close()
    }
    return nil
}
