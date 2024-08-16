package main

import (
	"emperror.dev/errors"
	"github.com/BurntSushi/toml"
	"github.com/je4/certloader/v2/pkg/loader"
	"github.com/je4/utils/v2/pkg/config"
	"github.com/je4/utils/v2/pkg/stashconfig"
	"io/fs"
	"os"
)

type MediaserverAPIConfig struct {
	LocalAddr               string             `toml:"localaddr"`
	ExternalAddr            string             `toml:"externaladdr"`
	Domains                 []string           `toml:"domains"`
	Bearer                  map[string]string  `toml:"bearer"`
	ResolverAddr            string             `toml:"resolveraddr"`
	ResolverTimeout         config.Duration    `toml:"resolvertimeout"`
	ResolverNotFoundTimeout config.Duration    `toml:"resolvernotfoundtimeout"`
	RESTTLS                 loader.Config      `toml:"resttls"`
	ServerTLS               *loader.Config     `toml:"server"`
	ClientTLS               *loader.Config     `toml:"client"`
	LogFile                 string             `toml:"logfile"`
	LogLevel                string             `toml:"loglevel"`
	GRPCClient              map[string]string  `toml:"grpcclient"`
	Log                     stashconfig.Config `toml:"log"`
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
