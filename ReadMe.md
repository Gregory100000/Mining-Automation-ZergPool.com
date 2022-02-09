# **ZergPool Mining Stats**

## See the primary mining automation program here first: https://github.com/GregoryUnderscore/Mining-Automation

### **Summary**
Download ZergPool shared pool statistics into a PostgreSQL database. Various SQL queries can then be utilized
to aid in mining automation or predictions.

### **Description**
ZergPool provides several useful statistics for every pool they host. This allows a miner to calculate projections
and possible profit opportunities. However, to properly calculate these projections, information about the current
Bitcoin price is mandatory. Therefore, information must be obtained from ZergPool and another site, in this case
CoinGecko, to properly make predictions.

The aim of this repository is to maneuver ZergPool mining closer to autonomy. This pulls down the statistics into a database that can be easily queried by optimization clients (clients not yet developed). The statistics can also be used to calculate predictions based on historical data which is currently unavailable from ZergPool directly.

See https://github.com/GregoryUnderscore/Mining-Automation-Miner-Stats for how to calculate miner statistics and report on
actual/estimated profitability.

In short, the zerg.go program does the following:
1. Connects to a database defined in the configuration file, ZergPoolConfig.hcl.
2. Automatically creates the required schema.
3. Obtains every coin from the CoingGecko REST.
4. Obtains the current Bitcoin price from CoinGecko.
5. Loads all the pools and their current statistics from ZergPool.

NOTE: Number 3 above has the unique benefit also of aiding in the identification of newly created coins on a day
to day basis.

### **How to Use**

1. Install PostgreSQL
2. Create a database
3. Update the ZergPoolConfig.hcl file with the appropriate database details etc.
4. Execute zerg.exe or zerg (if on Linux) or create a scheduled task in Windows Task Scheduler (or crontab on Linux) to 
execute zerg.exe or zerg every few hours. This will automatically keep the latest data in your database.

Step 6 can be automated on Linux based operating systems by using crontab. On Windows, the task scheduler can 
be used.

### **Included Reports**
In the sql folder is a SQL report that can be used to see the latest pool statistics for each pool and the potential
profit estimates/actuals in dollars. There is also a report to see any new coins added from CoinGecko in the last 24 hours. 
That may be useful in spotting opportunities with new projects. Additional reports may be added down the road.

### This solution is not affiliated with Zergpool.com.
