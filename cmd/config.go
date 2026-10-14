package cmd

// cspell:ignore godaddy nsdyndns

import (
	"errors"
	"os"

	"github.com/simon3z/nsdyndns"
	"gopkg.in/yaml.v2"
)

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
