package srvgrpc

import (
	"strings"

	"github.com/spf13/viper"
)

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
