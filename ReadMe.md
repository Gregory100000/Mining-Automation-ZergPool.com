# **ZergPool Mining Stats**

### **Summary**
Download ZergPool shared pool statistics into a PostgreSQL database. Various SQL queries can then be utilized
to aid in mining automation or predictions.

### **Description**
ZergPool provides several useful statistics for every pool they host. This allows a miner to calculate projections
and possible profit opportunities. However, to properly calculate these projections, information about the current
Bitcoin price is mandatory. Therefore, information must be obtained from ZergPool and another site, in this case
CoinGecko, to properly make predictions.

The aim of this repository is to create several utilities to maneuver ZergPool mining closer to autonomy. The
first phase is to pull down the statistics into a database that can be easily queried by optimization clients
(not yet developed). The statistics can also be used to calculate predictions based on historical data which is
currently unavailable from ZergPool directly.

In short, the zerg.go program will do the following:
1. Connect to a database defined in the configuration file, ZergPoolConfig.hcl.
2. Automatically create the required schema.
3. Obtain every coin from the CoingGecko REST.
4. Obtain the current Bitcoin price from CoinGecko.
5. Load all the pools and their current statistics from ZergPool.

NOTE: Number 3 above has the unique benefit also of aiding in the identification of newly created coins on a day
to day basis.

### **How to Use**

1. Install Go
2. Clone the repository
3. Install PostgreSQL
4. Create a database
5. Update the ZergPoolConfig.hcl file with the appropriate details.
6. go run zerg.go

Step 6 can be automated on Linux based operating systems by using crontab. On Windows, the task scheduler can 
be used.

NOTE: If you see a bunch of "record not found" errors onscreen, please ignore those. GORM outputs those without 
any obvious way to suppress them, and this program verifies that certain records (i.e. especially coins) do not
exist before adding them to the database. The errors require no action.

### **Included Reports**
In the sql folder is a SQL report that can be used to see the latest pool statistics for each pool and the potential
profit estimates/actuals in dollars. There is also a report to see any new coins added from CoinGecko in the last 24 hours. 
That may be useful in spotting opportunities with new projects. Additional reports may be added down the road.
