# deflux

deflux connects to the deCONZ REST API and web socket, listens for sensor updates and writes them to InfluxDB.

deCONZ supports a variety of ZigBee sensors, but doesn't keep a history of measurements.
Deflux archives all these values in InfluxDB, where they can be queried from the command line or graphical tools
such as Grafana. 

This project is a fork of the original [deflux](https://github.com/fasmide/deflux). Thanks to the original authors, who
did all the heavy lifting.
We have added several features here, to name a few:
  - Support for InfluxDB version 2. Influx did major changes moving from version 1 to version 2, most notably the
    introduction of a new query language called [Flux](https://docs.influxdata.com/influxdb/cloud/query-data/get-started/).
    Note that writing to InfluxDB v1 is still possible. See the section about [InfluxDB v1 compatibility](#influxdb-version-1-compatibility).
  - Additional battery state in sensor measurements (where available)
  - A pull-once-mode to write only the most recent measurement
  - Implementation of more sensor types

The project is maintained. Feel free to report issues or open pull requests.


## Table of Contents

- [Supported Sensors](#supported-sensors)
- [Usage](#usage)
    - [Pull Once Mode](#pull-once-mode)
- [InfluxDB](#influxdb)
    - [Version 2](#influxdb-version-2)
    - [Version 1](#influxdb-version-1-compatibility)
      - [Configuration](#configuration)
- [Development](#development)
- [Resources](#resources)

---


## Supported Sensors

The application fully supports the following types of [sensors](https://dresden-elektronik.github.io/deconz-rest-doc/endpoints/sensors/#supported-state-attributes_1):

- CLIPPresence
- Daylight
- ZHAAirQuality
- ZHABattery
- ZHAConsumption
- ZHAFire
- ZHAHumidity
- ZHALightLevel
- ZHAOpenClose
- ZHAPower
- ZHAPressure
- ZHASwitch
- ZHATemperature
- ZHAWater

The following sensors are mostly or partially implemented according to the
[spec](https://dresden-elektronik.github.io/deconz-rest-doc/endpoints/sensors/#supported-state-attributes_1),
but lack proper tests:

- ZHAAlarm
- ZHACarbonMonoxide
- ZHAPresence
- ZHAVibration

If you own such a sensor, it would be nice if you could provide some JSON test data as in
[this test](deconz/event_test.go). You can retrieve that data either with `debug` logging enabled in deflux, or,
using the `/sensors` endpoint of the REST API.


## Usage

Use `go install` to install the application.

```bash
go install github.com/fixje/deflux
```

Use `deflux --config-gen` to create the mandatory configuration file.
Deflux tries to discover existing gateways in your network and prints the config to `stdout`.

```bash
deflux --config-gen > deflux.yml
```

If you have temporarily unlocked the deCONZ gateway (Menu -> Settings -> Gateway -> Advanced -> "Authenticate app"),
deflux should be able to fill in the API key automatically. The full configuration looks as follows:

```yaml
deconz:
  addr: http://127.0.0.1/api
  apikey: "123A4B5C67"
influxdb:
  url: http://localhost:8086
  token: SECRET
  org: organization
  bucket: default
fillvalues:
  enabled: false
  initialfill: true
  fillinterval: 30m0s
  lastseentimeout: 2h0m0s
```

Edit the file according to your needs. If you want to write to InfluxDB version 1, see the section about
[InfluxDB v1 configuration](#influx1compat).

When the `fillvalues` functionality is enabled, deflux will write the last reported value of the REST API, if a sensor
has not reported any new measurement after `fillinterval`. We assume that the sensor is working as long as deCONZ
reports a `lastseen` time stamp not older than the configured `lastseentimeout`. The config values of `fillinterval` and
`lastseentimeout` should be set to anything parse-able by Go's [`time.ParseDuration` function](https://pkg.go.dev/time#ParseDuration).
With `initialfill` set to true, the application writes measurements from the REST API to the database when it starts.

By default, deflux tries to load the config from `deflux.yml` in the current working directory. If the file is not
present, it tries `/etc/deflux.yml`. You can provide a custom location with the `--config` command line flag.

The default log level of the application is `warning`. You can set the
`-loglevel=` flag to make it a more verbose:

```
$ ./deflux --loglevel debug
INFO[2021-12-26T11:29:15+01:00] Using configuration /home/fixje/hacks/deflux/deflux.yml
INFO[2021-12-26T11:29:15+01:00] Connected to deCONZ at http://172.26.0.2:80/api 
INFO[2021-12-26T11:29:15+01:00] Deconz websocket connected
```

See `deflux -h` for more information on command line flags.

### Pull Once Mode

If you run `deflux -1`, it will fetch the most recent sensor state from the REST API, persist it in InfluxDB and exit.
It will take the current system time as timestamp for the database.

The mode is intended to persist states for sensors which rarely provide new data points. Note that sensors could also
lack recent data, because of connectivity issues or an empty battery. The pull-once-mode does not take this
into account, so be aware! We are planning to find a solution for this problem in the near future.


## InfluxDB

Sensor measurements are added as InfluxDB field values. Every measurement has the following tags:
  - _type_: the sensor type, e.g. ZHAPressure
  - _id_: a unique numeric sensor identifier of the deCONZ API, starting at 1
  - _name_: the sensor name as defined by the user in the Phoscon App
  - _source_: indicates if the value has been obtained via the websocket or the REST API. Values of the REST API
              are added either in the `pull-once-mode` mode or when `fillvalues` is enabled.

Different event types are stored in different measurements, meaning you will end up with one InfluxDB measurement per
sensor type.

For some sensors, deCONZ provides battery status in the `config` object of the REST API's `sensors` endpoint.
The information is not pushed via the websocket. However, deflux inserts the last battery state retrieved from
the REST API as an additional field along with sensor measurements. For sensors where the information is not available,
the battery status is set to `0`.


### InfluxDB Version 2

Use the Flux language to get data from InfluxDB version 2. Below are some examples.

InfluxDB 2 has a nice query builder that will help you creating Flux queries.
Visit InfluxDB's web interface, log in, and click "Explore" in the navigation
bar.

#### Schema Exploration

You can use Flux queries to explore the schema.

```
$ influx query --org YOUR_ORG << EOF
import "influxdata/influxdb/schema"
schema.measurements(bucket: "YOUR_BUCKET")
EOF

Result: _result
Table: keys: []
         _value:string
----------------------
       deflux_Daylight
    deflux_ZHAHumidity
    deflux_ZHAPressure
 deflux_ZHATemperature
```

```
$ influx query --org YOUR_ORG << EOF
import "influxdata/influxdb/schema"

schema.measurementTagKeys(
  bucket: "YOUR_BUCKET",
  measurement: "deflux_ZHATemperature"
)

EOF
Result: _result
Table: keys: []
         _value:string
----------------------
                _start
                 _stop
                _field
          _measurement
                    id
                  name
                  type
                source
```

#### Example Queries

The following query retrieves temperature grouped by sensor name. This might be useful for a Grafana dashboard.

```
$ influx query --org YOUR_ORG << EOF
from(bucket: "YOUR_BUCKET")
  |> range(start: -3h)
  |> filter(fn: (r) =>
    r._measurement == "deflux_ZHATemperature" and r._field == "temperature"
    )
  |> keep(columns: ["_time", "name", "_value"])
|> group(columns: ["name"])
EOF

Result: _result
Table: keys: [name]
   name:string                      _time:time          _value:float
--------------  ------------------------------  --------------------
         th-sz  2021-12-27T06:37:07.741970950Z                 19.12
         th-sz  2021-12-27T06:59:22.576400599Z                 18.98
         th-sz  2021-12-27T07:01:23.235873787Z                 18.46
         th-sz  2021-12-27T07:04:04.025135987Z                 17.94
...
```

Here is an example for querying all fields of a measurement type.
The `map()` is required to convert all values to the same data type (battery and age_secs are of type `int`, temperature
of type `float`).

```
$ influx query --org YOUR_ORG << EOF
from(bucket: "YOUR_BUCKET")
  |> range(start: -3h)
  |> filter(fn: (r) =>
    r._measurement == "deflux_ZHATemperature"
    )
  |> map(fn: (r) => ({r with _value: float(v: r._value)}))
|> group(columns: ["name"])
EOF
Result: _result
Table: keys: [name]
           name:string           _field:string                     _start:time                      _stop:time                      _time:time                  _value:float               id:string     _measurement:string           source:string             type:string
----------------------  ----------------------  ------------------------------  ------------------------------  ------------------------------  ----------------------------  ----------------------  ----------------------  ----------------------  ----------------------
                 th-sz                age_secs  2022-01-16T17:32:37.346588227Z  2022-01-16T20:32:37.346588227Z  2022-01-16T20:20:29.572558449Z                           995                       2   deflux_ZHATemperature                    rest          ZHATemperature
                 th-sz                age_secs  2022-01-16T17:32:37.346588227Z  2022-01-16T20:32:37.346588227Z  2022-01-16T20:31:29.583605370Z                          1655                       2   deflux_ZHATemperature                    rest          ZHATemperature
                 th-sz                age_secs  2022-01-16T17:32:37.346588227Z  2022-01-16T20:32:37.346588227Z  2022-01-16T20:32:11.106819623Z                             0                       2   deflux_ZHATemperature               websocket          ZHATemperature
...
                 th-sz                 battery  2022-01-16T17:32:37.346588227Z  2022-01-16T20:32:37.346588227Z  2022-01-16T19:50:11.497602051Z                            95                       2   deflux_ZHATemperature               websocket          ZHATemperature
                 th-sz                 battery  2022-01-16T17:32:37.346588227Z  2022-01-16T20:32:37.346588227Z  2022-01-16T20:03:54.585479987Z                            95                       2   deflux_ZHATemperature               websocket          ZHATemperature
                 th-sz                 battery  2022-01-16T17:32:37.346588227Z  2022-01-16T20:32:37.346588227Z  2022-01-16T20:32:11.106819623Z                            95                       2   deflux_ZHATemperature               websocket          ZHATemperature
...
                 th-sz             temperature  2022-01-16T17:32:37.346588227Z  2022-01-16T20:32:37.346588227Z  2022-01-16T19:50:11.497602051Z                         19.08                       2   deflux_ZHATemperature               websocket          ZHATemperature
                 th-sz             temperature  2022-01-16T17:32:37.346588227Z  2022-01-16T20:32:37.346588227Z  2022-01-16T20:03:54.585479987Z                         18.97                       2   deflux_ZHATemperature               websocket          ZHATemperature
                 th-sz             temperature  2022-01-16T17:32:37.346588227Z  2022-01-16T20:32:37.346588227Z  2022-01-16T20:32:11.106819623Z                          18.9                       2   deflux_ZHATemperature               websocket          ZHATemperature
```


### InfluxDB Version 1 Compatibility

The application still supports InfluxDB version 1.
The [minimum required version](https://github.com/influxdata/influxdb-client-go/#influxdb-18-api-compatibility) is `1.8`.


#### Configuration

To write to InfluxDB v1 instances, provide your username and password separated by colon (`:`) in the `token` field.
You need to leave the `org` field empty. The name of the database is provided as `bucket`. Here is an example
`deflux.yml`:

```yml
deconz:
  addr: ...
  apikey: ...
influxdb:
  url: http://localhost:8086
  token: "USERNAME:PASSWORD"
  org: ""
  bucket: "DATABASE"
```

#### Data Exploration

You can inspect the data and its schema using the interactive `influx` shell:

```
> use sensors;
Using database sensors

> show measurements
name: measurements
name
----
deflux_ZHAHumidity
deflux_ZHAPressure
deflux_ZHATemperature
```

Here is an example how to retrieve pressure values:

```
> select * from deflux_ZHAPressure;
time                age_secs battery id name  pressure source    type
----                -------- ------- -- ----  -------- ------    ----
1641727554442270164 123      95      4  th-sz 993      rest      ZHAPressure
1641728526808217267 1095     95      4  th-sz 993      rest      ZHAPressure
1641729979208970180 0        95      4  th-sz 994      websocket ZHAPressure
1641730180633580793 0        95      4  th-sz 993      websocket ZHAPressure
...
```


## Development

The software can be built with standard Go tooling (`go build`).

You can cross-compile for Raspberry Pi 4 by setting `GOARCH` and `GOARM`:

```bash
GOOS=linux GOARCH=arm GOARM=7 go build
```

### Integration Tests

The project includes integration tests (marked by [build tags](https://pkg.go.dev/go/build)) which are not run automatically. They require an instance of
deCONZ and InfluxDB running on `localhost`. Exceute all tests, including integration tests with

```
go test --tags=integration_test ./...
```


## Resources

- [deCONZ sensor state attributes](https://dresden-elektronik.github.io/deconz-rest-doc/endpoints/sensors/#supported-state-attributes_1)
- [deCONZ websocket API docs](https://dresden-elektronik.github.io/deconz-rest-doc/endpoints/websocket/#message-fields)
- [deCONZ lastseen and reachable flag discussion](https://github.com/dresden-elektronik/deconz-rest-plugin/issues/2590)
