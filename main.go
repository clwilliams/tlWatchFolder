package main

import (
	"fmt"
	"os"
	"time"

	stdlog "log"

	"github.com/alecthomas/kingpin"
	"github.com/radovskyb/watcher"
	"github.com/rs/zerolog"
	log "github.com/rs/zerolog/log"
	"github.com/tlCommonMessaging/rabbitMQ"
)

const (
	defaultRabbitMqHost       = "localhost"
	defaultRabbitMqPort       = "5672"
	defaultRabbitMqUser       = "rabbitmq"
	defaultRabbitMqPassword   = "rabbitmq"
	defaultRabbitMqExchange   = "thirdlight"
	defaultRabbitMqQueue      = "watcher"
	defaultRabbitMqRoutingKey = "crud"
)

var (
	dev                = kingpin.Flag("dev", "Run app in development mode, no-dev for production").Default("true").Envar("DEV").Bool()
	verbose            = kingpin.Flag("verbose", "Enable verbose mode").Envar("VERBOSE").Bool()
	rabbitMqHost       = kingpin.Flag("rabbit-mq-host", "").Default(defaultRabbitMqHost).String()
	rabbitMqPort       = kingpin.Flag("rabbit-mq-port", "").Default(defaultRabbitMqPort).String()
	rabbitMqUser       = kingpin.Flag("rabbit-mq-user", "").Default(defaultRabbitMqUser).String()
	rabbitMqPassword   = kingpin.Flag("rabbit-mq-password", "").Default(defaultRabbitMqPassword).String()
	rabbitMqExchange   = kingpin.Flag("rabbit-mq-exchange", "").Default(defaultRabbitMqExchange).String()
	rabbitMqQueue      = kingpin.Flag("rabbit-mq-queue", "").Default(defaultRabbitMqQueue).String()
	rabbitMqRoutingKey = kingpin.Flag("rabbit-mq-routing-key", "").Default(defaultRabbitMqRoutingKey).String()
	watchFolderPath    = kingpin.Flag("watchFolderPath", "Path to watch file changes within").String()
)

func init() {
	// Only log the warning severity or above.
	log.Level(zerolog.WarnLevel)
}

func main() {
	// parse the command line arguments
	kingpin.Parse()

	// Initialise Logging
	if *dev {
		log.Level(zerolog.InfoLevel)
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
	stdlog.SetFlags(0)
	stdlog.SetOutput(log.Logger)

	if *verbose {
		log.Level(zerolog.DebugLevel)
		log.Debug().Msg("Set logging to verbose")
	}

	// Initialise Rabbit MQ
	rabbitMQClient := rabbitMQ.MessageClient{}
	err := rabbitMQClient.Connect(rabbitMqHost, rabbitMqPort, rabbitMqUser, rabbitMqPassword)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to RabbitMQ")
	}
	defer rabbitMQClient.Connection.Close()

	err = rabbitMQClient.ConfigureChannelAndExchange(rabbitMqExchange)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to configure RabbitMQ Channel / Exchange")
	}
	defer rabbitMQClient.Channel.Close()

	// Initialise folder watcher that will raise events for us
	folderWatcher := watcher.New()

	// ignore hidden files
	folderWatcher.IgnoreHiddenFiles(true)

	// the events we want to be notified of
	folderWatcher.FilterOps(watcher.Rename, watcher.Move, watcher.Create, watcher.Remove)

	go func() {
		for {
			select {
			case event := <-folderWatcher.Event:
				isDir := "false"
				if event.FileInfo.IsDir() {
					isDir = "true"
				}
				rabbitMqMessage := rabbitMQ.FolderWatchMessage{
					WatchFolder: *watchFolderPath,
					Action:      event.Op.String(),
					Path:        event.Path,
					IsDir:       isDir,
				}
				if *verbose {
					log.Info().Msg(fmt.Sprintf("Sending message : %#v", rabbitMqMessage))
				}
				err := rabbitMqMessage.PostToQueue(rabbitMqExchange, rabbitMqRoutingKey, &rabbitMQClient)
				if err != nil {
					log.Fatal().Err(err).Msg(fmt.Sprintf("Error sending message : %#v", rabbitMqMessage))
				}
			case err := <-folderWatcher.Error:
				log.Fatal().Err(err).Msg(fmt.Sprintf("Error handling event for : %s", *watchFolderPath))
			case <-folderWatcher.Closed:
				return
			}
		}
	}()

	// Watch the given folder for changes
	if err := folderWatcher.AddRecursive(*watchFolderPath); err != nil {
		log.Fatal().Err(err).Msg(fmt.Sprintf("Error initialising folder to watch : %s", *watchFolderPath))
	}

	// Send a list of all of the files and folders currently being watched and their paths
	for path, f := range folderWatcher.WatchedFiles() {
		isDir := "false"
		if f.IsDir() {
			isDir = "true"
		}
		rabbitMqMessage := rabbitMQ.FolderWatchMessage{
			WatchFolder: *watchFolderPath,
			Action:      "CREATE",
			Path:        path,
			IsDir:       isDir,
		}
		if *verbose {
			log.Info().Msg(fmt.Sprintf("Sending message to RabbitMQ : %#v", rabbitMqMessage))
		}
		err := rabbitMqMessage.PostToQueue(rabbitMqExchange, rabbitMqRoutingKey, &rabbitMQClient)
		if err != nil {
			log.Fatal().Err(err).Msg(fmt.Sprintf("Error sending message to RabbitMQ : %#v", rabbitMqMessage))
		}
	}

	// Start the watching process - it'll check for changes every 100ms.
	if err := folderWatcher.Start(time.Millisecond * 100); err != nil {
		log.Fatal().Err(err).Msg(fmt.Sprintf("Error watching the folder %s", *watchFolderPath))
	}
}
