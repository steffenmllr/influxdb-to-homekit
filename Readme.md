# Influxdb-to-homekit
> Display query results as Homekit Accessories


## Automated Build

[![steffenmllr/influxdb-to-homekit](http://dockeri.co/image/steffenmllr/influxdb-to-homekit)](https://registry.hub.docker.com/u/steffenmllr/influxdb-to-homekit/)


## Usage

```
docker run --restart=always -d --net=host \
-v path/to/local/config.toml:/app/config.toml \
steffenmllr/influxdb-to-homekit
```