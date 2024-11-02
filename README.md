# Borg exporter

Borg exporter exposes borg metrics to be scraped by Prometheus.

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
Environment="BORG_REPOSITORIES=ssh://sopranoborg/backups/dvd-fedora,ssh://sopranoborg/backups/dvd-fedora-home"
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

## Configuration

TODO: write about collecting for multiple metrics, will not exit but log an error, allowing to still collect for
remaining repositories

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

We increased the `scrape_interval` to `10m`, as the exporter will by default only refresh them every 12 hours. 

## User considerations

The exporter should run with a user having access to the borg repositories, typically the user executing the
borg backups.
