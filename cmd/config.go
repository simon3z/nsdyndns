package cmd

// cspell:ignore godaddy nsdyndns

import (
	"errors"
	"os"

	"github.com/simon3z/nsdyndns"
	"gopkg.in/yaml.v2"
)

// Config holds the nsdyndnsd configuration: the domain to manage, the host
// (subdomain, or apex when empty), and credentials for the name services.
type Config struct {
	Domain  string `yaml:"domain"`
	Host    string `yaml:"host"`
	GoDaddy struct {
		Key    string `yaml:"key"`
		Secret string `yaml:"secret"`
	}
	DigitalOcean struct {
		Token string `yaml:"token"`
	}
}

// LoadConfiguration reads and validates the YAML configuration at path.
func LoadConfiguration(path string) (*Config, error) {
	f, err := os.Open(path)

	if err != nil {
		return nil, err
	}

	defer f.Close()

	config := new(Config)

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(config)

	if err != nil {
		return nil, err
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Config) Validate() error {
	if c.Domain == "" {
		return errors.New("configuration: domain must not be empty")
	}

	if len(c.GetNameServices()) == 0 {
		return errors.New("configuration: at least one name service (godaddy or digitalocean) must be configured")
	}

	// Require both credentials, since a half-configured godaddy pair would
	// only fail at runtime with an opaque API error.
	hasKey := c.GoDaddy.Key != ""
	hasSecret := c.GoDaddy.Secret != ""

	if hasKey != hasSecret {
		return errors.New("configuration: godaddy requires both a key and a secret")
	}

	return nil
}

// FullDomain returns the fully qualified domain name for the configured
// host. An empty host refers to the apex domain.
func (c *Config) FullDomain() string {
	if c.Host == "" {
		return c.Domain
	}

	return c.Host + "." + c.Domain
}

// ConfigNameService pairs a name service with its display name.
type ConfigNameService struct {
	Name    string
	Service nsdyndns.NameService
}

func (c *Config) GetNameServices() []*ConfigNameService {
	l := []*ConfigNameService{}

	if c.GoDaddy.Key != "" {
		l = append(l, &ConfigNameService{"godaddy", nsdyndns.NewGoDaddyService(c.GoDaddy.Key, c.GoDaddy.Secret)})
	}

	if c.DigitalOcean.Token != "" {
		l = append(l, &ConfigNameService{"digitalocean", nsdyndns.NewDigitalOceanService(c.DigitalOcean.Token)})
	}

	return l
}
