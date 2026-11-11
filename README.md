# nsdyndnsd

[![github][badge-github]][link-github]
[![license][badge-license]][link-license]
[![version][badge-version]][link-version]
[![go][badge-go]][link-go]
[![dependencies][badge-deps]][link-deps]
[![tests][badge-tests]][link-ci]
[![coverage][badge-coverage]][link-coverage]

[badge-github]: https://img.shields.io/badge/github-simon3z/nsdyndns-6f57b0.svg?logo=github
[badge-license]: https://img.shields.io/badge/license-Apache-2.0-blue.svg
[badge-version]: https://img.shields.io/badge/version-0.1.1-ff8000.svg
[badge-go]: https://img.shields.io/badge/go-1.21-00ADD8.svg?logo=go
[badge-deps]: https://img.shields.io/badge/dependencies-1-green.svg
[badge-tests]: https://img.shields.io/badge/tests-passing-brightgreen.svg
[badge-coverage]: https://img.shields.io/badge/coverage-go_cover-brightgreen.svg
[link-github]: https://github.com/simon3z/nsdyndns
[link-license]: LICENSE
[link-version]: version.go
[link-go]: https://go.dev
[link-deps]: go.sum
[link-ci]: https://github.com/simon3z/nsdyndns/actions/workflows/ci.yml
[link-coverage]: #code-coverage

nsdyndnsd is a dynamic DNS update daemon written in Go. It periodically
determines the external IP address of the host and updates the A record of
the configured domain with the GoDaddy and/or DigitalOcean API, whenever the
address has changed.

The external IP address is detected via api.ipify.org, falling back to
api.ip.sb and ifconfig.me.

For internal design details, see [ARCHITECTURE.md](ARCHITECTURE.md).

# Installing

Installing the RPM (built on tag release and attached to the GitHub
release):

    # dnf install nsdyndns-*.rpm

This installs the daemon, an example configuration at
`/etc/nsdyndnsd/nsdyndnsd.conf` (readable only by the service user, since
it holds API credentials), the systemd unit, the `nsdyndns` system user
and `/var/lib/nsdyndns`.

Installing from the module:

    $ go install github.com/simon3z/nsdyndns/cmd/nsdyndnsd@latest

Cross-compiling for a Raspberry Pi:

    $ GOOS=linux GOARCH=arm GOARM=7 go install github.com/simon3z/nsdyndns/cmd/nsdyndnsd@latest

# Usage

    $ nsdyndnsd -h
    Usage of nsdyndnsd:
      -c string
            configuration file path
      -i int
            check interval in seconds (default 300)
      -version
            print the version and exit

Configuration example:

```yaml
domain: example.com
host: mysubdomain
godaddy:
  key: <godaddy-api-key>
  secret: <godaddy-api-secret>
digitalocean:
  token: <digitalocean-personal-token>
```

An empty `host` updates the A record of the apex domain.

# Installing as Systemd Service

Checkout the repository and build locally:

    # go build -o nsdyndnsd ./cmd/nsdyndnsd

Create the system user and its home directory:

    # useradd -r -s /usr/sbin/nologin -d /var/lib/nsdyndns nsdyndns
    # install -d /var/lib/nsdyndns

Install the nsdyndnsd binary and configuration:

    # cp -vi nsdyndnsd /usr/bin/nsdyndnsd
    # (umask 077 && mkdir -m 700 -vi /etc/nsdyndnsd)
    # (umask 077 && cp -vi config/nsdyndnsd.conf /etc/nsdyndnsd/nsdyndnsd.conf)
    # chown -R nsdyndns:nsdyndns /etc/nsdyndnsd

Edit the configuration file:

    # vim /etc/nsdyndnsd/nsdyndnsd.conf

Install the nsdyndnsd systemd service:

    # cp -vi config/nsdyndnsd.service /etc/systemd/system/nsdyndnsd.service

Reload systemd services:

    # systemctl daemon-reload

Enable the nsdyndnsd systemd service:

    # systemctl start nsdyndnsd.service
    # systemctl enable nsdyndnsd.service

Check the nsdyndnsd service status:

    # systemctl status nsdyndnsd.service

## Code Coverage

No extra tooling needed — the standard Go toolchain is used:

```bash
./scripts/coverage.sh
```

Prints per-function coverage and writes an interactive HTML report to
`coverage.html`.

## Logging

Logs are written to stderr via the standard `log` package. When running
under systemd, these are captured automatically by journald with
timestamps and the service name.
