db.createCollection(
    "weather",
    {
       timeseries: {
          timeField: "timestamp",
          metaField: "metadata",
          granularity: "hours"
       }
    }
)

startTime = new Date();

millisInHour = 1000 * 60 * 60;

hoursBeforeStart = function(hours) { return new Date(startTime.getTime() - hours * millisInHour ); };

db.weather.insertMany( [
   {
      "metadata": { "sensorId": 5578, "type": "temperature" },
      "timestamp": hoursBeforeStart(40),
      "temp": 12
   },
   {
      "metadata": { "sensorId": 5578, "type": "temperature" },
      "timestamp": hoursBeforeStart(36),
      "temp": 11
   },
   {
      "metadata": { "sensorId": 5578, "type": "temperature" },
      "timestamp": hoursBeforeStart(32),
      "temp": 11
   },
   {
      "metadata": { "sensorId": 5578, "type": "temperature" },
      "timestamp": hoursBeforeStart(28),
      "temp": 12
   },
   {
      "metadata": { "sensorId": 5578, "type": "temperature" },
      "timestamp": hoursBeforeStart(24),
      "temp": 16
   },
   {
      "metadata": { "sensorId": 5578, "type": "temperature" },
      "timestamp": hoursBeforeStart(24),
      "temp": 15
   }, {
      "metadata": { "sensorId": 5578, "type": "temperature" },
      "timestamp": hoursBeforeStart(20),
      "temp": 13
   },
   {
      "metadata": { "sensorId": 5578, "type": "temperature" },
      "timestamp": hoursBeforeStart(16),
      "temp": 12
   },
   {
      "metadata": { "sensorId": 5578, "type": "temperature" },
      "timestamp": hoursBeforeStart(12),
      "temp": 11
   },
   {
      "metadata": { "sensorId": 5578, "type": "temperature" },
      "timestamp": hoursBeforeStart(8),
      "temp": 12
   },
   {
      "metadata": { "sensorId": 5578, "type": "temperature" },
      "timestamp": hoursBeforeStart(4),
      "temp": 17
   },
   {
      "metadata": { "sensorId": 5578, "type": "temperature" },
      "timestamp": hoursBeforeStart(0),
      "temp": 12
   }
] )
