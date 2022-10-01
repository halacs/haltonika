# Overview

With this project, you can receive messages from Teltonika FMB920[^1] GPS tracer and store them in an InfluxDB[^2] database.

To visualise InfluxDB content, the easiest way is to set up a Grafana[^3] instance separately. This fits the best into a microservice architecture.

# Usage
Haltonika supports configurations from CLI arguments, environment variables as well as .yaml files.

```
Usage of ./haltonika:
      --database string      InfluxDB database name (default "haltonika")
      --debug                Set log level to debug
      --imeilist string      IMEI identifiers needs to be processed. Separated by comma. Example: 123456789012345,123456789012345,123456789012345 (default "350424063817363")
      --listenip string      Teltonika server listening IP address (IPv4 or IPv6) (default "0.0.0.0")
      --listenport int       Teltonika server listening UDP port (default 9160)
      --measurement string   Name of the Influxdb measurement (default "gps")
      --metricsip string     Metrics server listening IP address (IPv4 or IPv6) (default "0.0.0.0")
      --metricsport int      Metrics server listening port (default 9161)
      --mp string            File where metrics are written (default "haltonika.met")
      --password string      InfluxDB password (default "123")
      --url string           URL of InfluxDB server (default "http://localhost:8086")
      --username string      InfluxDB username (default "haltonika")
      --verbose              Set log level to verbose
```

# Build from source
Build requirements:
- GO 1.19.1
- make
- Ubuntu linux 22.04 LTS (or compatible Debian variant)

When above requirements are met, build itself is as simple as a ```make``` command from the project root.

# SystemD unit
Build ```haltonika``` binary from source and place it at ```/usr/bin/haltonika```, make it executable and create directory for configuration files under ```/etc```.
```
git clone git@github.com:halacs/haltonika.git
cd haltonika
make
sudo adduser haltonika
sudo cp dist/haltonika /usr/bin/haltonika
sudo chown haltonika:haltonika /usr/bin/haltonika
sudo chmod +x /usr/bin/haltonika
sudo mkdir /etc/haltonika
sudo chown haltonika:haltonika /etc/haltonika
```

With help of your favorite text editor (e.g. ```vim```), create ```/lib/systemd/system/haltonika.service``` file with the below content.
```
[Unit]
Description=Haltonika Server for Teltonika GPS trackers
Documentation=https://github.com/halacs/haltonika
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/bin/haltonika
ExecStop=/bin/kill -s STOP $MAINPID
ExecReload=/bin/kill -s HUP $MAINPID
User=haltonika
Group=haltonika
Restart=always
NoNewPrivileges=true
ProtectSystem=strict
ReadWritePaths=/etc/haltonika/
WorkingDirectory=/etc/haltonika/
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

Start your ```haltonika instance```:
```
sudo systemctl enable --now haltonika.service
```

Finally, you can check if your haltonika instance is up and running by checking its logs and metrics:
```
sudo systemctl status haltonika.service
curl localhost:9161/metrics
```

# Haltonika internal metrics
Haltonika provides an HTTP endpoint to expose its internal metrics.

By default, it is available on the http://localhost:9131/metrics URL.

Haltonika provides metrics for itself:
- received bytes and packages: received bytes/packages from all remote endpoints all together
- sent bytes and packages: sent bytes/packages to all remote endpoints all together
- malformed packages: packages could not parse, in any reason
- rejected packages: packages not on the allowed list are rejected 

Packages here means byte streams could be parsed into a valid Teltonika package

# Configure Telegraf [^4] for Haltonika internal metrics

Install Telegraf then add below content into /etc/telegraf/telegraf.d/haltonika.conf file.
```
[[inputs.http]]
  ## One or more URLs from which to read formatted metrics
  urls = [
    "http://localhost:9161/metrics"
  ]

  ## HTTP method
  method = "GET"
  
  ## Amount of time allowed to complete the HTTP request
  timeout = "1s"

  ## Data format to consume.
  data_format = "influx"
```

# Start development InfluxDB

The easiest solution to start a development InfluxDB is to use Docker.

Below command starts an InfluxDB 1.8 and make 8086 port available on the host IP.

When Docker container terminates, all of its data will be lost.

```
docker run --rm -d -p 8086:8086 --name influxdb influxdb:1.8
docker exec -it influxdb influx
    create database haltonika
    use haltonika
    create user haltonika with password '123'
    grant all on haltonika to haltonika
```

[^1]: https://teltonika-gps.com/product/fmb920/
[^2]: https://www.influxdata.com/
[^3]: https://grafana.com/
[^4]: https://github.com/influxdata/telegraf
