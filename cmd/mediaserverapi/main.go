package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/je4/certloader/v2/pkg/loader"
	"github.com/je4/mediaserverapi/v2/config"
	"github.com/je4/mediaserverapi/v2/pkg/rest"
	mediaserverproto "github.com/je4/mediaserverproto/v2/pkg/mediaserver/proto"
	"github.com/je4/miniresolver/v2/pkg/resolver"
	configutil "github.com/je4/utils/v2/pkg/config"
	"github.com/je4/utils/v2/pkg/zLogger"
	ublogger "gitlab.switch.ch/ub-unibas/go-ublogger"
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
		ServerTLS: &loader.Config{
			Type: "DEV",
		},
		ClientTLS: &loader.Config{
			Type: "DEV",
		},
	}
	if err := LoadMediaserverAPIConfig(cfgFS, cfgFile, conf); err != nil {
		log.Fatalf("cannot load toml from [%v] %s: %v", cfgFS, cfgFile, err)
	}
	// create logger instance
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("cannot get hostname: %v", err)
	}

	var loggerTLSConfig *tls.Config
	var loggerLoader io.Closer
	if conf.Log.Stash.TLS != nil {
		loggerTLSConfig, loggerLoader, err = loader.CreateClientLoader(conf.Log.Stash.TLS, nil)
		if err != nil {
			log.Fatalf("cannot create client loader: %v", err)
		}
		defer loggerLoader.Close()
	}

	_logger, _logstash, _logfile := ublogger.CreateUbMultiLoggerTLS(conf.Log.Level, conf.Log.File,
		ublogger.SetDataset(conf.Log.Stash.Dataset),
		ublogger.SetLogStash(conf.Log.Stash.LogstashHost, conf.Log.Stash.LogstashPort, conf.Log.Stash.Namespace, conf.Log.Stash.LogstashTraceLevel),
		ublogger.SetTLS(conf.Log.Stash.TLS != nil),
		ublogger.SetTLSConfig(loggerTLSConfig),
	)
	if _logstash != nil {
		defer _logstash.Close()
	}
	if _logfile != nil {
		defer _logfile.Close()
	}

	l2 := _logger.With().Timestamp().Str("host", hostname).Str("addr", conf.LocalAddr).Logger() //.Output(output)
	var logger zLogger.ZLogger = &l2

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

	dbClients, err := resolver.NewClients[mediaserverproto.DatabaseClient](miniResolverClient, mediaserverproto.NewDatabaseClient, mediaserverproto.Database_ServiceDesc.ServiceName, conf.Domains)
	if err != nil {
		logger.Panic().Msgf("cannot create mediaserverdatabase grpc client: %v", err)
	}
	for _, dbClient := range dbClients {
		resolver.DoPing(dbClient, logger)
	}

	deleterClients, err := resolver.NewClients[mediaserverproto.DeleterClient](miniResolverClient, mediaserverproto.NewDeleterClient, mediaserverproto.Deleter_ServiceDesc.ServiceName, conf.Domains)
	if err != nil {
		logger.Panic().Msgf("cannot create mediaserverdeleter grpc client: %v", err)
	}
	for _, deleterClient := range deleterClients {
		resolver.DoPing(deleterClient, logger)
	}

	actionControllerClients, err := resolver.NewClients[mediaserverproto.ActionClient](miniResolverClient, mediaserverproto.NewActionClient, mediaserverproto.Action_ServiceDesc.ServiceName, conf.Domains)
	if err != nil {
		logger.Panic().Msgf("cannot create mediaserveractionController grpc client: %v", err)
	}
	for _, actionControllerClient := range actionControllerClients {
		resolver.DoPing(actionControllerClient, logger)
	}

	actionDispatcherClients, err := resolver.NewClients[mediaserverproto.ActionDispatcherClient](miniResolverClient, mediaserverproto.NewActionDispatcherClient, mediaserverproto.ActionDispatcher_ServiceDesc.ServiceName, conf.Domains)
	if err != nil {
		logger.Panic().Msgf("cannot create mediaserveractionDispatcher grpc client: %v", err)
	}
	for _, actionDispatcherClient := range actionDispatcherClients {
		resolver.DoPing(actionDispatcherClient, logger)
	}

	ctrl, err := rest.NewController(conf.LocalAddr, conf.ExternalAddr, restTLSConfig, conf.Bearer, dbClients, actionControllerClients, deleterClients, actionDispatcherClients, logger)
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
