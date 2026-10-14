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
}{}

var cmdConfig *cmd.Config
var cmdServices []*cmd.ConfigNameService

func init() {
	flag.StringVar(&cmdFlags.ConfigFilePath, "c", "", "configuration file path")
	flag.IntVar(&cmdFlags.CheckInterval, "i", 300, "check interval in seconds")
}

// CheckAndUpdate detects a change of the external ip address and updates
// the configured name services accordingly.
func CheckAndUpdate(ctx context.Context) error {
	extIP, err := nsdyndns.GetExternalIP(ctx)

	if err != nil {
		return err
	}

	dnsAddrs, err := net.LookupIP(cmdConfig.FullDomain())

	if err != nil {
		return fmt.Errorf("address lookup for %s failed: %w", cmdConfig.FullDomain(), err)
	}

	for _, dnsAddr := range dnsAddrs {
		if dnsAddr.Equal(extIP) {
			return nil
		}
	}

	log.Printf("new external ip address detected: %s", extIP.String())

	errs := make([]error, 0, len(cmdServices))

	for _, s := range cmdServices {
		log.Printf("updating %s address from %s to %s using %s", cmdConfig.FullDomain(), dnsAddrs[0].String(), extIP.String(), s.Name)

		if err := s.Service.SetAddress(ctx, cmdConfig.Domain, cmdConfig.Host, extIP); err != nil {
			log.Printf("address of %s update using %s failed: %s", cmdConfig.FullDomain(), s.Name, err)
			errs = append(errs, fmt.Errorf("%s: %w", s.Name, err))
		} else {
			log.Printf("address of %s has been successfully updated to %s using %s", cmdConfig.FullDomain(), extIP.String(), s.Name)
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

	if cmdFlags.ConfigFilePath == "" {
		log.Fatalln("configuration file was not specified")
	}

	if cmdFlags.CheckInterval <= 0 {
		log.Fatalln("check interval must be a positive number of seconds")
	}

	var err error

	cmdConfig, err = cmd.LoadConfiguration(cmdFlags.ConfigFilePath)

	if err != nil {
		log.Fatalln(err)
	}

	cmdServices = cmdConfig.GetNameServices()

	log.Printf("starting to monitor external address for %s using %s", cmdConfig.FullDomain(), nsdyndns.GetExternalIPService())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	defer stop()

	ticker := time.NewTicker(time.Duration(cmdFlags.CheckInterval) * time.Second)

	defer ticker.Stop()

	for {
		if err := CheckAndUpdate(ctx); err != nil {
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
