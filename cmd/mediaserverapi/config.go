package main

import (
	"emperror.dev/errors"
	"github.com/BurntSushi/toml"
	"github.com/je4/trustutil/v2/pkg/loader"
	"github.com/je4/utils/v2/pkg/config"
	"io/fs"
	"os"
)

type MediaserverAPIConfig struct {
	LocalAddr       string            `toml:"localaddr"`
	ResolverAddr    string            `toml:"resolveraddr"`
	ResolverTimeout config.Duration   `toml:"resolvertimeout"`
	ExternalAddr    string            `toml:"externaladdr"`
	ServerTLS       *loader.TLSConfig `toml:"server"`
	ClientTLS       *loader.TLSConfig `toml:"client"`
	LogFile         string            `toml:"logfile"`
	LogLevel        string            `toml:"loglevel"`
	GRPCClient      map[string]string `toml:"grpcclient"`
}

func LoadMediaserverAPIConfig(fSys fs.FS, fp string, conf *MediaserverAPIConfig) error {
	if _, err := fs.Stat(fSys, fp); err != nil {
		path, err := os.Getwd()
		if err != nil {
			return errors.Wrap(err, "cannot get current working directory")
		}
		fSys = os.DirFS(path)
		fp = "mediaserverapi.toml"
	}
	data, err := fs.ReadFile(fSys, fp)
	if err != nil {
		return errors.Wrapf(err, "cannot read file [%v] %s", fSys, fp)
	}
	_, err = toml.Decode(string(data), conf)
	if err != nil {
		return errors.Wrapf(err, "error loading config file %v", fp)
	}
	return nil
}
