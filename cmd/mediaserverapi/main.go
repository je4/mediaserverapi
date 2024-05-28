package main

import (
	"context"
	"flag"
	"fmt"
	genericproto "github.com/je4/genericproto/v2/pkg/generic/proto"
	"github.com/je4/mediaserverapi/v2/config"
	"github.com/je4/mediaserverapi/v2/pkg/rest"
	mediaserverclient "github.com/je4/mediaserverproto/v2/pkg/mediaserver/client"
	mediaserverproto "github.com/je4/mediaserverproto/v2/pkg/mediaserver/proto"
	miniresolverClient "github.com/je4/miniresolver/v2/pkg/client"
	"github.com/je4/miniresolver/v2/pkg/grpchelper"
	"github.com/je4/trustutil/v2/pkg/loader"
	configutil "github.com/je4/utils/v2/pkg/config"
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

	output := zerolog.ConsoleWriter{Out: out, TimeFormat: time.RFC3339}
	_logger := zerolog.New(output).With().Timestamp().Logger()
	_logger.Level(zLogger.LogLevel(conf.LogLevel))
	var logger zLogger.ZLogger = &_logger

	restTLSConfig, restLoader, err := loader.CreateServerLoader(false, &conf.RESTTLS, nil, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("cannot create server loader")
	}
	defer restLoader.Close()

	/*
		serverCert, serverLoader, err := loader.CreateServerLoader(false, conf.ServerTLS, nil, logger)
		if err != nil {
			logger.Panic().Msgf("cannot create server loader: %v", err)
		}
		defer serverLoader.Close()
	*/

	var dbClientAddr, deleterClientAddr, actionControllerAddr string
	if conf.ResolverAddr != "" {
		dbClientAddr = grpchelper.GetAddress(mediaserverproto.Database_Ping_FullMethodName)
		deleterClientAddr = grpchelper.GetAddress(mediaserverproto.Deleter_Ping_FullMethodName)
		actionControllerAddr = grpchelper.GetAddress(mediaserverproto.Action_Ping_FullMethodName)
	} else {
		if _, ok := conf.GRPCClient["mediaserverdb"]; !ok {
			logger.Fatal().Msg("no mediaserverdb grpc client defined")
		}
		dbClientAddr = conf.GRPCClient["mediaserverdeleter"]
		if _, ok := conf.GRPCClient["mediaserverdeleter"]; !ok {
			logger.Fatal().Msg("no mediaserverdeleter grpc client defined")
		}
		deleterClientAddr = conf.GRPCClient["mediaserverdeleter"]
		if _, ok := conf.GRPCClient["mediaserveraction"]; !ok {
			logger.Fatal().Msg("no mediaserveraction grpc client defined")
		}
		actionControllerAddr = conf.GRPCClient["mediaserveraction"]
	}

	clientCert, clientLoader, err := loader.CreateClientLoader(conf.ClientTLS, logger)
	if err != nil {
		logger.Panic().Msgf("cannot create client loader: %v", err)
	}
	defer clientLoader.Close()

	if conf.ResolverAddr != "" {
		logger.Info().Msgf("resolver address is %s", conf.ResolverAddr)
		miniResolverClient, miniResolverCloser, err := miniresolverClient.CreateClient(conf.ResolverAddr, clientCert)
		if err != nil {
			logger.Fatal().Msgf("cannot create resolver client: %v", err)
		}
		defer miniResolverCloser.Close()
		grpchelper.RegisterResolver(miniResolverClient, time.Duration(conf.ResolverTimeout), time.Duration(conf.ResolverNotFoundTimeout), logger)
	}

	dbClient, dbClientCloser, err := mediaserverclient.NewDatabaseClient(dbClientAddr, clientCert)
	if err != nil {
		logger.Panic().Msgf("cannot create mediaserverdb grpc client: %v", err)
	}
	defer dbClientCloser.Close()
	if resp, err := dbClient.Ping(context.Background(), &emptypb.Empty{}); err != nil {
		logger.Error().Msgf("cannot ping mediaserverdb: %v", err)
	} else {
		if resp.GetStatus() != genericproto.ResultStatus_OK {
			logger.Error().Msgf("cannot ping mediaserverdb: %v", resp.GetStatus())
		} else {
			logger.Info().Msgf("mediaserverdb ping response: %s", resp.GetMessage())
		}
	}

	deleterClient, deleterClientCloser, err := mediaserverclient.NewDeleterClient(deleterClientAddr, clientCert)
	if err != nil {
		logger.Panic().Msgf("cannot create mediaserverdeleter grpc client: %v", err)
	}
	defer deleterClientCloser.Close()
	if resp, err := deleterClient.Ping(context.Background(), &emptypb.Empty{}); err != nil {
		logger.Error().Msgf("cannot ping mediaserverdeleter: %v", err)
	} else {
		if resp.GetStatus() != genericproto.ResultStatus_OK {
			logger.Error().Msgf("cannot ping mediaserverdeleter: %v", resp.GetStatus())
		} else {
			logger.Info().Msgf("mediaserverdeleter ping response: %s", resp.GetMessage())
		}
	}

	actionControllerClient, actionControllerClientCloser, err := mediaserverclient.NewActionClient(actionControllerAddr, clientCert)
	if err != nil {
		logger.Panic().Msgf("cannot create mediaserveractionController grpc client: %v", err)
	}
	defer actionControllerClientCloser.Close()
	if resp, err := actionControllerClient.Ping(context.Background(), &emptypb.Empty{}); err != nil {
		logger.Error().Msgf("cannot ping mediaserveractionController: %v", err)
	} else {
		if resp.GetStatus() != genericproto.ResultStatus_OK {
			logger.Error().Msgf("cannot ping mediaserveractionController: %v", resp.GetStatus())
		} else {
			logger.Info().Msgf("mediaserveractionController ping response: %s", resp.GetMessage())
		}
	}

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
