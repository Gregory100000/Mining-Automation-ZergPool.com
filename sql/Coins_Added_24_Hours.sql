-- This report shows all the coins added in the last 24 hours.
-- It can be used to see new coins recently added to CoinGecko if the database is updated on a regular basis (i.e. every 4 hours).
-- Researching newer coins may provide interesting opportunities.

SELECT * 
FROM coins
WHERE added > NOW() - INTERVAL '24 HOURS'
ORDER BY name;