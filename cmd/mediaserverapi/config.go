package main

import (
	"emperror.dev/errors"
	"github.com/BurntSushi/toml"
	loaderConfig "github.com/je4/trustutil/v2/pkg/config"
	"github.com/je4/utils/v2/pkg/config"
	"github.com/je4/utils/v2/pkg/zLogger"
	"io/fs"
	"os"
)

type MediaserverAPIConfig struct {
	LocalAddr               string                  `toml:"localaddr"`
	ExternalAddr            string                  `toml:"externaladdr"`
	Bearer                  string                  `toml:"bearer"`
	ResolverAddr            string                  `toml:"resolveraddr"`
	ResolverTimeout         config.Duration         `toml:"resolvertimeout"`
	ResolverNotFoundTimeout config.Duration         `toml:"resolvernotfoundtimeout"`
	RESTTLS                 loaderConfig.TLSConfig  `toml:"resttls"`
	ServerTLS               *loaderConfig.TLSConfig `toml:"server"`
	ClientTLS               *loaderConfig.TLSConfig `toml:"client"`
	LogFile                 string                  `toml:"logfile"`
	LogLevel                string                  `toml:"loglevel"`
	GRPCClient              map[string]string       `toml:"grpcclient"`
	Log                     zLogger.Config          `toml:"log"`
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
