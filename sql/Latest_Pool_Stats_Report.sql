-- This report provides the latest details for the pools.
-- This provides profit estimates in mBTC a day per hash unit of the pool along with actual profit over the last 24 hours.
-- The Bitcoin price at the time of the statistics is stored to determine the actual dollar amount of these as well.

SELECT pools.name,pool_stats.instant,current_hashrate,workers, profit_estimate*1000 AS profit_estimate, profit_actual24_hours, 
	CASE 
		WHEN mh_factor = 1000000 THEN 'mBTC/Th/day'
		WHEN mh_factor = 1000 THEN 'mBTC/Gh/day'
		WHEN mh_factor = 1 THEN 'mBTC/Mh/day'
		WHEN mh_factor = 0.001 THEN 'mBTC/Kh/day'
	END AS hash_unit,
	price*0.001*profit_estimate AS profit_dollar_estimate,  -- The $ estimate per hash unit / day
	price*0.001*profit_actual24_hours AS profit__dollar_actual24_hours -- The $ actual per hash unit / last day
FROM pool_stats
LEFT JOIN pools ON
	pool_id = pools.id 
LEFT JOIN coin_prices ON
	coin_price_id = coin_prices.id 
INNER JOIN (
	SELECT MAX(id) AS id
	FROM pool_stats
	GROUP BY pool_id
) latest_stats ON
	latest_stats.id = pool_stats.id
ORDER BY pools.name