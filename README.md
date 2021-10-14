# Octoflux

Sync your [Octopus](https://octopus.energy) smart meter usage into an InfluxDB
instance.

![A grafana panel showing the data collected by Octoflux. Two stat panels are
shown at the top, one for electricity and one for gas. They both have
sparklines. Below that is a normal time series graph showing electricity and gas
on the same chart. The left axis shows electricity and the right axis shows
gas.](image/grafana.png)

## Requirements

* An InfluxDB v2 installation with a bucket you can write to
* An octopus gas and/or electricity smart meter account
* Somewhere to run this script

## Configuration and operation

You'll get best resolution if you configure 30 minute submissions from your
smart meter, via the Octopus website, if sending that level of detail to your
energy provider doesn't bother you.

Supply the following configuration via environment variables or commandline
flags.

| Environment variable  | Flag                   | Description                                                                         |
| --------------------- | ---------------------- | ----------------------------------------------------------------------------------- |
| `$INFLUX_HOST`        | `--influx-host`        | The hostname of the InfluxDB to connect to, with port. e.g. `http://localhost:8081` |
| `$INFLUX_BUCKET`      | `--influx-bucket`      | The bucket to put your readings in                                                  |
| `$INFLUX_ORG`         | `--influx-org`         | The organisation your bucket belongs to                                             |
| `$INFLUX_TOKEN`       | `--influx-token`       | A token which grants write access to the bucket                                     |
| `$OCTOPUS_API_KEY`    | `--octopus-api-key`    | An API key for reading from Octopus                                                 |
| `$ELECTRICITY_MPN`    | `--electricity-mpn`    | An electricity meter point number                                                   |
| `$ELECTRICITY_SERIAL` | `--electricity-serial` | An electricity serial number                                                        |
| `$GAS_MPN`            | `--gas-mpn`            | A gas meter point number                                                            |
| `$GAS_SERIAL`         | `--gas-serial`         | A gas serial number                                                                 |

All of the above are required, except you don't have to have both a gas and en
electricity account. Only one is needed.

Docker images are auto-built from this repository, and [provided via
ghcr](https://github.com/iainlane/octoflux/pkgs/container/octoflux). Semver tags
will be generated, e.g. `v1`, `v1.0` and `v1.0.0` depending on how much you like
to *roll*. You can also follow `main` or `edge` to follow the main branch or
`latest` to get the latest tagged version.

## Kubernetes deployment

An example kustomization file is provided. Edit the files in `conf/` and deploy it using

```
kubectl apply -k ./
```

It will then run as a Kubernetes cronjob every 30 minutes.

## Limitations

* The Octopus API only provides you with data that are 24 hours old. So you can't see your current usage.

## Future work

* Include Grafana dashboards here, and provision them when deploying to Kubernetes.
* Add some prometheus rules to alert if this breaks.
