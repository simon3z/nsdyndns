package main

// cspell:ignore nsdyndns

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/simon3z/nsdyndns"
	"github.com/simon3z/nsdyndns/cmd"
)

var cmdFlags = struct {
	ConfigFilePath string
	CheckInterval  int
	ShowVersion    bool
}{}

func init() {
	flag.StringVar(&cmdFlags.ConfigFilePath, "c", "", "configuration file path")
	flag.IntVar(&cmdFlags.CheckInterval, "i", 300, "check interval in seconds")
	flag.BoolVar(&cmdFlags.ShowVersion, "version", false, "print the version and exit")
}

// CheckAndUpdate detects a change of the external ip address and updates
// the given name services accordingly. The ip and dns lookup functions
// are injectable so the orchestration is unit-testable without network
// access.
func CheckAndUpdate(ctx context.Context, config *cmd.Config, services []*cmd.ConfigNameService, ipFetcher func(context.Context) (net.IP, error), dnsLookup func(string) ([]net.IP, error)) error {
	extIP, err := ipFetcher(ctx)

	if err != nil {
		return err
	}

	dnsAddrs, err := dnsLookup(config.FullDomain())

	if err != nil {
		return fmt.Errorf("address lookup for %s failed: %w", config.FullDomain(), err)
	}

	for _, dnsAddr := range dnsAddrs {
		if dnsAddr.Equal(extIP) {
			return nil
		}
	}

	log.Printf("new external ip address detected: %s", extIP.String())

	errs := make([]error, 0, len(services))

	for _, s := range services {
		log.Printf("updating %s address from %s to %s using %s", config.FullDomain(), dnsAddrs[0].String(), extIP.String(), s.Name)

		if err := s.Service.SetAddress(ctx, config.Domain, config.Host, extIP); err != nil {
			log.Printf("address of %s update using %s failed: %s", config.FullDomain(), s.Name, err)
			errs = append(errs, fmt.Errorf("%s: %w", s.Name, err))
		} else {
			log.Printf("address of %s has been successfully updated to %s using %s", config.FullDomain(), extIP.String(), s.Name)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func main() {
	log.SetFlags(log.LstdFlags)

	flag.Parse()

	if cmdFlags.ShowVersion {
		fmt.Printf("nsdyndnsd %s\n", nsdyndns.Version)
		return
	}

	if cmdFlags.ConfigFilePath == "" {
		log.Fatalln("configuration file was not specified")
	}

	if cmdFlags.CheckInterval <= 0 {
		log.Fatalln("check interval must be a positive number of seconds")
	}

	config, err := cmd.LoadConfiguration(cmdFlags.ConfigFilePath)

	if err != nil {
		log.Fatalln(err)
	}

	services := config.GetNameServices()

	log.Printf("starting nsdyndnsd %s, monitoring external address for %s using %s", nsdyndns.Version, config.FullDomain(), nsdyndns.GetExternalIPService())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	defer stop()

	ticker := time.NewTicker(time.Duration(cmdFlags.CheckInterval) * time.Second)

	defer ticker.Stop()

	for {
		if err := CheckAndUpdate(ctx, config, services, nsdyndns.GetExternalIP, net.LookupIP); err != nil {
			log.Println(err)
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			log.Printf("shutting down: %v", ctx.Err())
			return
		}
	}
}
