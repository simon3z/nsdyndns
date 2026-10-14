# nsdyndnsd

nsdyndnsd is a dynamic DNS update daemon written in Go. It periodically
determines the external IP address of the host and updates the A record of
the configured domain with the GoDaddy and/or DigitalOcean API, whenever the
address has changed.

The external IP address is detected via api.ipify.org, falling back to
api.ip.sb and ifconfig.me.

# Installing

Installing the RPM (built on tag release and attached to the GitHub
release):

    # dnf install nsdyndns-*.rpm

This installs the daemon, an example configuration at
`/etc/nsdyndnsd.conf`, the systemd unit, the `nsdyndns` system user and
`/var/lib/nsdyndns`.

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

Install the nsdyndnsd binary and configuration:

    # cp -vi nsdyndnsd /usr/bin/nsdyndnsd
    # (umask 077 && cp -vi init/nsdyndnsd.conf /etc/nsdyndnsd.conf)

Create the system user and its home directory:

    # useradd -r -s /usr/sbin/nologin -d /var/lib/nsdyndns nsdyndns
    # install -d /var/lib/nsdyndns

Edit the configuration file:

    # vim /etc/nsdyndnsd.conf

Install the nsdyndnsd systemd service:

    # cp -vi init/nsdyndnsd.service /etc/systemd/system/nsdyndnsd.service

Reload systemd services:

    # systemctl daemon-reload

Enable the nsdyndnsd systemd service:

    # systemctl start nsdyndnsd.service
    # systemctl enable nsdyndnsd.service

Check the nsdyndnsd service status:

    # systemctl status nsdyndnsd.service
