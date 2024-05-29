package main

import (
	"flag"
	"fmt"
	"github.com/je4/mediaserverapi/v2/config"
	"github.com/je4/mediaserverapi/v2/pkg/rest"
	mediaserverproto "github.com/je4/mediaserverproto/v2/pkg/mediaserver/proto"
	"github.com/je4/miniresolver/v2/pkg/resolver"
	"github.com/je4/trustutil/v2/pkg/loader"
	configutil "github.com/je4/utils/v2/pkg/config"
	"github.com/je4/utils/v2/pkg/zLogger"
	"github.com/rs/zerolog"
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
		LocalAddr: "localhost:8443",
		//ResolverTimeout: config.Duration(10 * time.Minute),
		ExternalAddr:            "https://localhost:8443",
		LogLevel:                "DEBUG",
		ResolverTimeout:         configutil.Duration(10 * time.Minute),
		ResolverNotFoundTimeout: configutil.Duration(10 * time.Second),
		ServerTLS: &loader.TLSConfig{
			Type: "DEV",
		},
		ClientTLS: &loader.TLSConfig{
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

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("cannot get hostname: %v", err)
	}

	output := zerolog.ConsoleWriter{Out: out, TimeFormat: time.RFC3339}
	_logger := zerolog.New(output).With().Timestamp().Str("service", "mediaserverapi"). /*.Array("addrs", zLogger.StringArray(addrStr))*/ Str("host", hostname).Str("addr", conf.LocalAddr).Logger()
	_logger.Level(zLogger.LogLevel(conf.LogLevel))
	var logger zLogger.ZLogger = &_logger

	restTLSConfig, restLoader, err := loader.CreateServerLoader(false, &conf.RESTTLS, nil, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("cannot create server loader")
	}
	defer restLoader.Close()

	clientCert, clientLoader, err := loader.CreateClientLoader(conf.ClientTLS, logger)
	if err != nil {
		logger.Panic().Msgf("cannot create client loader: %v", err)
	}
	defer clientLoader.Close()

	logger.Info().Msgf("resolver address is %s", conf.ResolverAddr)
	miniResolverClient, err := resolver.NewMiniresolverClient(conf.ResolverAddr, conf.GRPCClient, clientCert, nil, time.Duration(conf.ResolverTimeout), time.Duration(conf.ResolverNotFoundTimeout), logger)
	if err != nil {
		logger.Fatal().Msgf("cannot create resolver client: %v", err)
	}
	defer miniResolverClient.Close()

	dbClient, err := resolver.NewClient[mediaserverproto.DatabaseClient](miniResolverClient, mediaserverproto.NewDatabaseClient, mediaserverproto.Database_ServiceDesc.ServiceName)
	if err != nil {
		logger.Panic().Msgf("cannot create mediaserverdatabase grpc client: %v", err)
	}
	resolver.DoPing(dbClient, logger)

	deleterClient, err := resolver.NewClient[mediaserverproto.DeleterClient](miniResolverClient, mediaserverproto.NewDeleterClient, mediaserverproto.Database_ServiceDesc.ServiceName)
	if err != nil {
		logger.Panic().Msgf("cannot create mediaserverdeleter grpc client: %v", err)
	}
	resolver.DoPing(dbClient, logger)

	actionControllerClient, err := resolver.NewClient[mediaserverproto.ActionClient](miniResolverClient, mediaserverproto.NewActionClient, mediaserverproto.Database_ServiceDesc.ServiceName)
	if err != nil {
		logger.Panic().Msgf("cannot create mediaserveractionController grpc client: %v", err)
	}
	resolver.DoPing(dbClient, logger)

	ctrl, err := rest.NewController(conf.LocalAddr, conf.ExternalAddr, restTLSConfig, conf.Bearer, dbClient, actionControllerClient, deleterClient, logger)
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
