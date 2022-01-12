package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/hcl/v2/hclsimple"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"ZergPoolData/models"
)

// ====================================
// Configuration File (ZergPoolData.hcl)
// ====================================
type Config struct {
	Host        string `hcl:"host"`        // The server hosting the database
	Port        string `hcl:"port"`        // The port of the database server
	Database    string `hcl:"database"`    // The database name
	User        string `hcl:"user"`        // The user to use for login to the database server
	Password    string `hcl:"password"`    // The user's password for login
	TimeZone    string `hcl:"timezone"`    // The time zone where the program is run
	ZergRegion  string `hcl:"zergregion"`  // This is the region prefix used in the pool URL, e.g. "na"
	ZergBaseURL string `hcl:"zergbaseurl"` // This is the base URL for ZergPool, e.g. mine.zergpool.com.
}

// ====================================
// REST
// ====================================

// CoinGecko coin data from REST
type CoinGeckoCoin struct {
	CoinGeckoID string `json:"id"`
	Symbol      string `json:"symbol"`
	Name        string `json:"name"`
}

// CoinGecko simple coin price for Bitcoin in USD from REST. This is not OHLC.
type BitCoin struct {
	Name  string `json:"bitcoin"`
	Price PriceInUSD
}

// Price for a coin in USD. This is not OHLC.
type PriceInUSD struct {
	Price float64 `json:"usd"`
}

// ZergPool Pool statistics from REST
// http://api.zergpool.com:8080/api/status
type ZergPoolStats struct {
	Name                  string  `json:"name"`
	Port                  int     `json:"port"`
	Coins                 int     `json:"coins"`
	Fees                  float64 `json:"fees"`
	Hashrate              int     `json:"hashrate"`
	HashrateShared        int     `json:"hashrate_shared"`
	HashrateSolo          int     `json:"hashrate_solo"`
	Workers               int     `json:"workers"`
	WorkersShared         int     `json:"workers_shared"`
	WorkersSolo           int     `json:"workers_solo"`
	EstimateCurrent       float64 `json:"estimate_current"`
	EstimateLast24H       float64 `json:"estimate_last24h"`
	ActualLast24H         float64 `json:"actual_last24h"`
	ActualLast24HShared   float64 `json:"actual_last24h_shared"`
	ActualLast24HSolo     float64 `json:"actual_last24h_solo"`
	MbtcMhFactor          float64 `json:"mbtc_mh_factor"`
	HashrateLast24H       int     `json:"hashrate_last24h"`
	HashrateLast24HShared int     `json:"hashrate_last24h_shared"`
	HashrateLast24HSolo   int     `json:"hashrate_last24h_solo"`
}

// Connect to the database and create the tables if they are not present. Afterward, connect to
// ZergPool and query for the latest pool statistics. Parse those statistics and store everything
// into the database for later review.
func main() {

	// Used for pulling down the Bitcoin price. Most of the Zergpool estimates reference
	// Bitcoin. Therefore, it is imperative to have the Bitcoin price associated to every
	// statistic pull.
	const coingGeckoURL = "https://api.coingecko.com/api/v3/"
	const zergPoolStatsURL = "http://api.zergpool.com:8080/api/status"
	const configFileName = "ZergPoolData.hcl"
	var config Config

	// Grab the configuration details for the database connection. These are stored in ZergPoolData.hcl.
	err := hclsimple.DecodeFile(configFileName, nil, &config)
	if err != nil {
		log.Fatalf("Failed to load config file "+configFileName+".\n", err)
	}

	log.Println("Using the following configuration:")
	log.Println("Database Server: " + config.Host + ":" + config.Port)
	log.Println("Database: " + config.Database)
	log.Println("User: " + config.User + "\n")

	dsn := "host=" + config.Host + " "
	dsn += "port=" + config.Port + " "
	dsn += "dbname=" + config.Database + " "
	dsn += "user=" + config.User + " "
	dsn += "password=" + config.Password + " "
	dsn += "TimeZone=" + config.TimeZone + " "
	dsn += "sslmode=disable"
	log.Println("Connecting to " + config.Host + "...")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		log.Fatalf("Failed to connect to the database server.\n", err)
	}

	log.Println("Connected to " + config.Host + ".")
	log.Println("Verifying/updating schema")
	// Create the schema if it does not exist.
	db.AutoMigrate(&models.Provider{}, &models.Algorithm{}, &models.Pool{}, &models.PoolStats{},
		&models.Coin{}, &models.CoinPrice{})
	log.Println("Schema verified.")

	// Open the new database transaction and get all the coins from CoinGecko along with the BTC price.
	tx := db.Begin()
	bitcoinPriceID := getCoinsAndBTCPrice(coingGeckoURL, tx)

	// Get all the pool statistics from ZergPool.
	stats := getPoolStats(zergPoolStatsURL)

	// Cycle over the stats and add them to the database.
	log.Println("Storing statistics...")
	defer func() { // Ensure transaction rollback on panic
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Check if the ZergPool provider record exists, and if not create it.
	var provider models.Provider
	result := tx.Where("name = ?", "ZergPool").First(&provider)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		provider.Name = "ZergPool"
		provider.Website = "https://zergpool.com/"
		provider.Fee = 0.5
		result = tx.Create(&provider)
		if result.Error != nil {
			log.Fatalf("Issue creating provider.\n", result.Error)
		}
	} else if result.Error != nil {
		log.Fatalf("Unknown issue storing provider.\n", result.Error)
	}

	// Cycle over the statistics for all the pools.
	// Create the pool and algo records if they do not exist.
	// Store the statistics.
	for _, stat := range stats {

		// ==> Algorithm
		var algo models.Algorithm
		result := tx.Where("name = ?", stat.Name).First(&algo)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			algo.Name = stat.Name
			result = tx.Create(&algo)
			if result.Error != nil {
				log.Fatalf("Issue creating algorithm.\n", result.Error)
			}
		} else if result.Error != nil {
			log.Fatalf("Unknown issue storing algorithm: "+stat.Name+"\n", result.Error)
		}

		// ==> Pool
		var pool models.Pool
		result = tx.Where("provider_id = ? AND algorithm_id=?", provider.ID, algo.ID).First(&pool)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Generate the URL
			url := stat.Name + "." + config.ZergRegion + "." + config.ZergBaseURL
			pool.ProviderID = provider.ID
			pool.AlgorithmID = algo.ID
			pool.Name = stat.Name // Just use the algo name for ZergPool
			pool.URL = url
			pool.Port = uint32(stat.Port)
			pool.MhFactor = stat.MbtcMhFactor
			result = tx.Create(&pool)
			if result.Error != nil {
				log.Fatalf("Issue creating pool.\n", result.Error)
			}
		} else if result.Error != nil {
			log.Fatalf("Unknown issue storing algorithm: "+stat.Name+"\n", result.Error)
		}

		// ==> Stats
		poolStat := models.PoolStats{
			PoolID:              pool.ID,
			Instant:             time.Now(),
			CurrentHashrate:     uint64(stat.HashrateShared),
			Workers:             uint32(stat.WorkersShared),
			ProfitEstimate:      stat.EstimateCurrent,
			ProfitActual24Hours: stat.ActualLast24HShared,
			CoinPriceID:         bitcoinPriceID,
		}
		result = tx.Create(&poolStat)
		if result.Error != nil {
			// Sometimes Zergpool returns strangely high hash rates that are outside the bounds
			// of a 64 bit integer. When that happens, skip the statistics.
			if strings.Contains(fmt.Sprint(result.Error), "is greater than maximum value") {
				log.Println("Skipping pool statistics for " + pool.Name + " due to bad data.")
				continue
			}
			log.Fatalf("Issue creating stats.\n", result.Error)
		}
	}
	err = tx.Commit().Error // Finalize data storage
	if err != nil {
		log.Fatalf("Issue committing changes.\n", result.Error)
	}
	log.Println("Statistics stored.\nOperations complete.\n")
}

// Get pool statistics from ZergPool's REST API.
func getPoolStats(url string) []ZergPoolStats {
	var toMap interface{}              // Used to convert JSON response to map
	zergPoolStats := []ZergPoolStats{} // All the stats are returned in this array

	log.Println("Connecting to pool for statistic pull...")

	// Make the call and get the raw response in bytes.
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Pool connection error.\n", err)
	}
	defer resp.Body.Close() // Ensures the response is eliminated on exit.
	bodyBytes, _ := ioutil.ReadAll(resp.Body)

	// Convert to map
	json.Unmarshal(bodyBytes, &toMap)
	data := toMap.(map[string]interface{})

	// Process data add to array.
	for _, value := range data {
		// Must get the map within the map
		mappedValue := value.(map[string]interface{})
		parseStringsToFloats(&mappedValue) // ZergPool has some floats as strings

		// Map to JSON
		jsonBody, err := json.Marshal(mappedValue)
		if err != nil {
			log.Fatalf("Pool stat processing issue.\n", err)
		}

		// JSON to Struct
		zergPoolStat := ZergPoolStats{}
		if err := json.Unmarshal(jsonBody, &zergPoolStat); err != nil {
			log.Fatalf("Pool stat processing issue.\n", err)
		}

		// Add to the array to be returned.
		zergPoolStats = append(zergPoolStats, zergPoolStat)
	}

	log.Println("Statistics retrieved...")

	return zergPoolStats
}

// Calls out to CoinGecko to get all the coins in their database. Those are stored
// in the local database, and afterward, the BTC price is obtained.
// @param url - The base URL for the CoinGecko REST API
// @param tx - The active database transaction
// @returns - The ID of the latest Bitcoin price in the coin_price table.
func getCoinsAndBTCPrice(url string, tx *gorm.DB) uint64 {
	var toMap interface{}          // Used to convert JSON response to map
	coinURL := url + "/coins/list" // URL for pulling all the coins
	// URL for pulling the current bitcoin price. This is not OHLC.
	priceURL := url + "/simple/price/?ids=bitcoin&vs_currencies=usd"

	log.Println("Connecting to CoinGecko for coins and BTC price...")

	// ====> COINS
	// Make the call and get the raw response in bytes.
	resp, err := http.Get(coinURL)
	if err != nil {
		log.Fatalf("CoinGecko connection error.\n", err)
	}
	defer resp.Body.Close() // Ensures the response is eliminated on exit.
	bodyBytes, _ := ioutil.ReadAll(resp.Body)

	// Convert JSON bytes to slice
	var coins []CoinGeckoCoin
	if err := json.Unmarshal(bodyBytes, &coins); err != nil {
		log.Fatalf("Coin processing issue.\n", err)
	}

	// Cycle over the coins from CoinGecko and store anything not in the database.
	for _, coin := range coins {
		var coinToStore models.Coin
		result := tx.Where("coin_gecko_id = ?", coin.CoinGeckoID).First(&coinToStore)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			coinToStore.CoinGeckoID = coin.CoinGeckoID
			coinToStore.Name = coin.Name
			coinToStore.Symbol = coin.Symbol
			coinToStore.Added = time.Now()
			result = tx.Create(&coinToStore)
			if result.Error != nil {
				log.Fatalf("Issue creating coin.\n", result.Error)
			}
		} else if result.Error != nil {
			log.Fatalf("Unknown issue storing coin: "+coin.Name+"\n", result.Error)
		}
	}

	// ===> Bitcoin price retrieval
	resp.Body.Close() // Close out prior
	// Make the call and get the raw response in bytes.
	resp, err = http.Get(priceURL)
	if err != nil {
		log.Fatalf("CoinGecko connection error.\n", err)
	}
	defer resp.Body.Close() // Ensures the response is eliminated on exit.
	bodyBytes, _ = ioutil.ReadAll(resp.Body)

	// Convert to map
	json.Unmarshal(bodyBytes, &toMap)
	data := toMap.(map[string]interface{})
	// Must get the map within the map
	rawPrice := data["bitcoin"].(map[string]interface{})
	// Pull out the price
	price := rawPrice["usd"].(float64)

	log.Println("Bitcoin Price (USD): " + strconv.FormatFloat(price, 'f', 2, 64))

	// Get Bitcoin's ID in the database.
	var bitcoin models.Coin
	result := tx.Where("coin_gecko_id = ?", "bitcoin").First(&bitcoin)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		log.Fatalf("Issue locating Bitcoin in the database.\n", result.Error)
	}
	// Store the price for Bitcoin.
	var priceToStore models.CoinPrice
	priceToStore.Price = price
	priceToStore.CoinID = bitcoin.ID
	priceToStore.Instant = time.Now()
	result = tx.Create(&priceToStore)
	if result.Error != nil {
		log.Fatalf("Issue creating coin.\n", result.Error)
	}

	log.Println("Coins/Bitcoin price retrieved/stored...")

	return priceToStore.ID
}

// The JSON returned from ZergPool's REST can contain strings as numbers.
// Convert those strings to numbers.
func parseStringsToFloats(mappedValue *map[string]interface{}) {
	// Cycle over the map and handle any oddities with numbers passed as strings.
	for keyInValue, val := range *mappedValue {
		switch val.(type) {
		case string:
			// Some values are passed as strings, but they are actually numbers.
			// Convert these to numbers in the map.
			if keyInValue != "name" {
				var err error
				valString := val.(string)
				(*mappedValue)[keyInValue], err = strconv.ParseFloat(valString, 64)
				if err != nil {
					log.Fatalf("Error converting data, " + valString + " on " +
						keyInValue)
				}
			}
		default:
			continue
		}
	}
}
