package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/je4/mediaserverapi/v2/config"
	"github.com/je4/mediaserverapi/v2/pkg/rest"
	mediaserverdbClient "github.com/je4/mediaserverdb/v2/pkg/client"
	"github.com/je4/mediaserverdb/v2/pkg/mediaserverdbproto"
	"github.com/je4/trustutil/v2/pkg/loader"
	"github.com/je4/utils/v2/pkg/zLogger"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/types/known/emptypb"
	"io"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

var configfile = flag.String("config", "", "location of toml configuration file")

func main() {
	flag.Parse()

	var cfgFS fs.FS
	var cfgFile string
	if *configfile != "" {
		cfgFS = os.DirFS(filepath.Dir(*configfile))
		cfgFile = filepath.Base(*configfile)
	} else {
		cfgFS = config.ConfigFS
		cfgFile = "mediaserverapi.toml"
	}

	conf := &MediaserverAPIConfig{
		LocalAddr:    "localhost:8443",
		ExternalAddr: "https://localhost:8443",
		LogLevel:     "DEBUG",
		Server: &loader.TLSConfig{
			Type: "DEV",
		},
		Client: &loader.TLSConfig{
			Type: "DEV",
		},
	}
	if err := LoadMediaserverAPIConfig(cfgFS, cfgFile, conf); err != nil {
		log.Fatalf("cannot load toml from [%v] %s: %v", cfgFS, cfgFile, err)
	}
	// create logger instance
	var out io.Writer = os.Stdout
	if conf.LogFile != "" {
		fp, err := os.OpenFile(conf.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("cannot open logfile %s: %v", conf.LogFile, err)
		}
		defer fp.Close()
		out = fp
	}

	output := zerolog.ConsoleWriter{Out: out, TimeFormat: time.RFC3339}
	_logger := zerolog.New(output).With().Timestamp().Logger()
	_logger.Level(zLogger.LogLevel(conf.LogLevel))
	var logger zLogger.ZLogger = &_logger

	serverCert, serverLoader, err := loader.CreateServerLoader(false, conf.Server, nil, logger)
	if err != nil {
		logger.Panic().Msgf("cannot create server loader: %v", err)
	}
	defer serverLoader.Close()

	clientCert, clientLoader, err := loader.CreateClientLoader(conf.Client, logger)
	if err != nil {
		logger.Panic().Msgf("cannot create client loader: %v", err)
	}
	defer clientLoader.Close()

	if _, ok := conf.GRPCClient["mediaserverdb"]; !ok {
		logger.Fatal().Msg("no mediaserverdb grpc client defined")
	}
	dbClient, dbClientConn, err := mediaserverdbClient.CreateClient(conf.GRPCClient["mediaserverdb"], clientCert)
	if err != nil {
		logger.Panic().Msgf("cannot create mediaserverdb grpc client: %v", err)
	}
	defer dbClientConn.Close()
	if resp, err := dbClient.Ping(context.Background(), &emptypb.Empty{}); err != nil {
		logger.Error().Msgf("cannot ping mediaserverdb: %v", err)
	} else {
		if resp.GetStatus() != mediaserverdbproto.ResultStatus_OK {
			logger.Error().Msgf("cannot ping mediaserverdb: %v", resp.GetStatus())
		} else {
			logger.Info().Msgf("mediaserverdb ping response: %s", resp.GetMessage())
		}
	}
	ctrl, err := rest.NewController(conf.LocalAddr, conf.ExternalAddr, serverCert, dbClient, logger)
	if err != nil {
		logger.Fatal().Msgf("cannot create controller: %v", err)
	}
	var wg = &sync.WaitGroup{}
	ctrl.Start(wg)

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	fmt.Println("press ctrl+c to stop server")
	s := <-done
	fmt.Println("got signal:", s)

	ctrl.GracefulStop()
	wg.Wait()

}
