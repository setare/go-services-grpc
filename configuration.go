package srvgrpc

import (
	"strings"

	"github.com/spf13/viper"
)

// MemoryConfiguration stores the configuration in memory.
type MemoryConfiguration struct {
	bindAddr string
}

// NewConfiguration creates a `MemoryConfiguration`.
func NewConfiguration() *MemoryConfiguration {
	return &MemoryConfiguration{}
}

// BindAddr returns the bind address set in memory.
func (c *MemoryConfiguration) BindAddr() (string, error) {
	return c.bindAddr, nil
}

// SetBindAddr sets the bind address returned for this configuration instance.
//
// It returns "this" for sugar.
func (c *MemoryConfiguration) SetBindAddr(value string) *MemoryConfiguration {
	c.bindAddr = value
	return c
}

// ViperConfiguration is the `GRPCConfiguration` implementation for `viper`.
type ViperConfiguration struct {
	Prefix string
	Viper  *viper.Viper
}

func (c *ViperConfiguration) BindAddr() (string, error) {
	v := c.Viper
	if v == nil {
		v = viper.GetViper()
	}
	key := "bind_addr"
	if c.Prefix != "" {
		if strings.HasSuffix(c.Prefix, ".") {
			key = c.Prefix + key
		} else {
			key = c.Prefix + "." + key
		}
	}
	return v.GetString(key), nil
}
