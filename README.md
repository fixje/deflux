# deflux

deflux connects to deCONZ rest api, listens for sensor updates and write these to InfluxDB.

deCONZ supports a variaty of Zigbee sensors but have no historical data about their values - with deflux you'll be able to store all these measurements in influxdb where they can be queried from the command line or graphical tools such as grafana. 

This software was forked from the original [deflux](https://github.com/fasmide/deflux) and added support for InfluxDB
version 2.
Influx did major changes moving from version 1 to version 2, most notably the
introduction of a new query language called
[Flux](https://docs.influxdata.com/influxdb/cloud/query-data/get-started/).
Note that writing to InfluxDB v1 is still possible. See the section about 
[InfluxDB v1 compatibility](#influxdb-version-1-compatibility).

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

deflux requires a configuration file named `deflux.yml` in the current working directory or in `/etc/deflux.yml`. The
current directory is preferred over `/etc`.

Use `deflux -config-gen` to create such a file. Deflux tries to discover existing gateways in your network and print
the config to `stdout`.

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
```

Edit the file according to your needs. If you want to write to InfluxDB version 1, see the section about
[InfluxDB v1 configuration](#influx1compat).

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

Sensor measurements are added as InfluxDB field values and tagged with sensor type, id and name. In addition to that,
a tag `source` indicates if the value has been obtained live via the websocket or the REST API (_pull-once-mode_).
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

Get temperature grouped by sensor name:

```
$ influx query --org YOUR_ORG << EOF
from(bucket: "YOUR_BUCKET")
  |> range(start: -3h)
  |> filter(fn: (r) =>
    r._measurement == "deflux_ZHATemperature"
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
time                battery id name  pressure source    type
----                ------- -- ----  -------- ------    ----
1641727554442270164 95      4  th-sz 993      rest      ZHAPressure
1641728526808217267 95      4  th-sz 993      rest      ZHAPressure
1641729979208970180 95      4  th-sz 994      websocket ZHAPressure
1641730180633580793 95      4  th-sz 993      websocket ZHAPressure
...
```


## Development

The software can be built with standard Go tooling (`go build`).

You can cross-compile for Raspberry Pi 4 by setting `GOARCH` and `GOARM`:

```bash
GOOS=linux GOARCH=arm GOARM=7 go build
```


## Resources

- [deCONZ sensor state attributes](https://dresden-elektronik.github.io/deconz-rest-doc/endpoints/sensors/#supported-state-attributes_1)
- [deCONZ websocket API docs](https://dresden-elektronik.github.io/deconz-rest-doc/endpoints/websocket/#message-fields)
