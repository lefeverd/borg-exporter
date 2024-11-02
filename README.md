# Borg exporter

Borg exporter exposes borg metrics to be scraped by Prometheus.

## Metrics

The following metrics are exposed :

| Name                                     | Description                                      | Type    |
|------------------------------------------|--------------------------------------------------|---------|
| borg_last_backup_duration_seconds        | Duration of the last backup in seconds           | Gauge   |
| borg_last_backup_compressed_size_bytes   | Compressed size of the last backup in bytes      | Gauge   |
| borg_last_backup_deduplicated_size_bytes | Deduplicated size of the last backup in bytes    | Gauge   |
| borg_last_backup_files                   | Number of files in the last backup               | Gauge   |
| borg_last_backup_original_size_bytes     | Original size of the last backup in bytes        | Gauge   |
| borg_last_backup_timestamp               | Timestamp of the last backup (unix epoch*)       | Gauge   |
| borg_total_chunks                        | Repository total chunks                          | Gauge   |
| borg_total_compressed_size_bytes         | Repository total compressed size                 | Gauge   |
| borg_total_size_bytes                    | Repository total size                            | Gauge   |
| borg_total_unique_chunks                 | Repository total unique chunks                   | Gauge   |
| borg_deduplicated_compressed_size_bytes  | Repository deduplicated compressed size          | Gauge   |
| borg_deduplicated_size_bytes             | Repository deduplicated size                     | Gauge   |
| borg_collect_errors                      | Number of errors encountered by borg exporter    | Counter |
| borg_last_collect_error                  | 1 if the last collection failed, 0 if successful | Gauge   |
| borg_last_collect_duration_seconds       | Duration of the last metrics collection          | Gauge   |
| borg_last_collect_timestamp              | Timestamp of the last metrics collection         | Gauge   |
| borg_last_archive_info                   | Information about the last backup archive        | Gauge   |
| borg_repository_info                     | Information about the backup repository          | Gauge   |
| borg_system_info                         | Information about the borg backup system         | Gauge   |

\* number of seconds that have elapsed since January 1, 1970

Each of these metrics are in reality "labeled" metrics, such as `GaugeVec` and `CounterVec`, grouped (or labeled) by
`repository`.  
When using multiple repositories, each of these will be exposed for each repository.

## Configuration

The following environment variables can be used to configure the exporter :

| Name                     | Description                                                                          | Required | Default    |
|--------------------------|--------------------------------------------------------------------------------------|----------|------------|
| LISTEN_ADDRESS           | Address on which the server is to listen for connections                             |          | `:9099`    |
| METRICS_PATH             | Path on which the server exposes the metrics                                         |          | `/metrics` |
| METRICS_REFRESH_INTERVAL | Defines the frequency (interval of time) at which the exporter refreshes the metrics |          | `12h`      |
| COMMAND_TIMEOUT          | Timeout for borg commands                                                            |          | `120s`     |
| BORG_REPOSITORIES        | Comma-separated list of borg repositories to expose metrics for                      | `yes`    | ``         |
| LOG_LEVEL                | Logging level (debug, info, warn, error)                                             |          | `info`     |

The `METRICS_REFRESH_INTERVAL` is by default set to a value of `12h`, but you can tweak it depending on your
requirement,
for instance depending on the frequency of your backups.  
This value is optimized for daily backups, for which metrics won't change frequently.

When using multiple repositories in `BORG_REPOSITORIES`, the exporter will not crash if it cannot retrieve metrics for
one of them, but instead an error will be logged.  
This is to allow collecting metrics for the other repositories.

## Installation

You can install it by downloading the latest version and placing it in `/usr/local/bin/borg-exporter`.  
Depending on your OS, you can then create a service to run it.  
For instance, using systemd :

```
[Unit]
Description=Borg Prometheus Exporter
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/borg-exporter
Environment="BORG_REPOSITORIES=ssh://my-repository/backups/my-machine,ssh://my-other-repository/backups/my-machine"
Restart=always
RestartSec=10

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=read-only
PrivateTmp=true
PrivateDevices=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictAddressFamilies=AF_INET AF_INET6
RestrictNamespaces=true

[Install]
WantedBy=multi-user.target
```

### User considerations

The exporter should run with a user having access to the borg repositories, typically the user executing the
borg backups.


### Prometheus

To scrape the metrics from Prometheus, you can add a scrape configuration, for instance :

```
- job_name: 'borg'
  scrape_interval: 10m
  scrape_timeout: 30s
  static_configs:
    - targets:
      - '<hostname>:9099'
```

We set the `scrape_interval` to `10m`, as the exporter will by default only refresh them every 12 hours,
but you can tweak this value depending on your requirements.
