# Borg exporter

Borg exporter exposes borg metrics to be scraped by Prometheus.

## User considerations

The exporter should run with a user having access to the borg repositories, typically the user executing the
borg backups.

## TODO

- It could potentially fail if a backup is currently running