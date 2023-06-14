package config

import (
	"errors"
	"net/url"
	"os"

	"gopkg.in/yaml.v2"
)

// Config maps to the on-disk yaml format
type Config struct {
	ListenAddr string     `yaml:"listenAddr"`
	ChainNodes ChainNodes `yaml:"chainNodes"`
}

func (c *Config) Nodes() (out []string) {
	for _, n := range c.ChainNodes {
		out = append(out, n.PrivValAddr)
	}
	return out
}

func (c *Config) MustMarshalYaml() []byte {
	out, err := yaml.Marshal(c)
	if err != nil {
		panic(err)
	}
	return out
}

type RuntimeConfig struct {
	Config     Config
	ConfigFile string
	HomeDir    string
}

func (c RuntimeConfig) WriteConfigFile() error {
	return os.WriteFile(c.ConfigFile, c.Config.MustMarshalYaml(), 0600)
}

type ChainNode struct {
	PrivValAddr string `json:"privValAddr" yaml:"privValAddr"`
}

type ChainNodes []ChainNode

func (cn ChainNode) Validate() error {
	_, err := url.Parse(cn.PrivValAddr)
	return err
}

func (cns ChainNodes) Validate() error {
	var errs []error
	for _, cn := range cns {
		if err := cn.Validate(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func ChainNodesFromFlag(nodes []string) (ChainNodes, error) {
	out := make(ChainNodes, len(nodes))
	for i, n := range nodes {
		cn := ChainNode{PrivValAddr: n}
		out[i] = cn
	}
	if err := out.Validate(); err != nil {
		return nil, err
	}
	return out, nil
}
