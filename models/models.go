package models

import (
	"time"
)

// ====================================
// Database Tables
// ====================================

// A mining algorithm such as scrypt.
// Making this distinct will allow mapping between various pools and mining software.
type Algorithm struct {
	ID   uint64 `gorm:"primaryKey"`
	Name string
}

// A crypto coin.
type Coin struct {
	ID          uint64 `gorm:"primaryKey"`
	CoinGeckoID string
	Name        string
	Symbol      string
	Added       time.Time // The date/time added. Can be used to track new coins.
}

// A price for a coin in USD. This is not OHLC over a range.
type CoinPrice struct {
	ID      uint64 `gorm:"primaryKey"`
	CoinID  uint64
	Instant time.Time // The instant of the price
	Price   float64
}

// A mining pool.
type Pool struct {
	ID          uint64 `gorm:"primaryKey"`
	ProviderID  uint64
	AlgorithmID uint64 // The name will not necessarily match the algorithm precisely
	Name        string // In some cases, the name of the pool could be different than the algo.
	URL         string // This is generated to the full address for easier automation
	Port        uint32
	// Used for mining profit calculations/estimates
	MhFactor float64 // 1 = Mh/s, 0.001 = Kh/s, 1000 = Gh/s
}

// Statistics at a certain point in time for a pool.
// Used to optimize mining operations by examining profit estimates/actuals.
type PoolStats struct {
	ID                  uint64 `gorm:"primaryKey"`
	PoolID              uint64
	Instant             time.Time // The date/time of the statistics
	CurrentHashrate     uint64    // The current shared hashrate for the pool
	Workers             uint32    // The current number of workers sharing the pool
	ProfitEstimate      float64   // An forward look at potential profit/day
	ProfitActual24Hours float64   // The actual profit/day for those sharing the pool
	CoinPriceID         uint64    // The line to the relevant coin price, if any. Bitcoin is usually used.
}

// A pool provider such as ZergPool.
type Provider struct {
	ID      uint64 `gorm:"primaryKey"`
	Name    string
	Website string
	Fee     float32
}
