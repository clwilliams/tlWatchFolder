package main

import (
	"fmt"
	"os"
	"time"

	stdlog "log"

	"git.thebookpeople.com/qwerkity/customer-indexer/version"
	"github.com/alecthomas/kingpin"
	"github.com/radovskyb/watcher"
	"github.com/rs/zerolog"
	log "github.com/rs/zerolog/log"
	"github.com/watch/rabbitMQ"
)

const (
	defaultRabbitMqHost     = "localhost"
	defaultRabbitMqPort     = "5672"
	defaultRabbitMqUser     = "rabbitmq"
	defaultRabbitMqPassword = "rabbitmq"
)

var (
	dev              = kingpin.Flag("dev", "Run app in development mode, no-dev for production").Default("true").Envar("DEV").Bool()
	verbose          = kingpin.Flag("verbose", "Enable verbose mode").Envar("VERBOSE").Bool()
	rabbitMqHost     = kingpin.Flag("rabbit-mq-host", "").Default(defaultRabbitMqHost).String()
	rabbitMqPort     = kingpin.Flag("rabbit-mq-port", "").Default(defaultRabbitMqPort).String()
	rabbitMqUser     = kingpin.Flag("rabbit-mq-user", "").Default(defaultRabbitMqUser).String()
	rabbitMqPassword = kingpin.Flag("rabbit-mq-password", "").Default(defaultRabbitMqPassword).String()
	watchFolderPath  = kingpin.Flag("watchFolderPath", "Path to watch file changes within").String()
)

func init() {
	// Only log the warning severity or above.
	log.Level(zerolog.WarnLevel)
}

func main() {
	// parse the command line arguments
	kingpin.Version(version.Get())
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
	rabbitMqClient, err := rabbitMQ.NewClient(rabbitMqHost, rabbitMqPort, rabbitMqUser, rabbitMqPassword)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to RabbitMQ")
	}
	defer rabbitMqClient.Close()

	// Initialise folder watcher that will raise events for us
	folderWatcher := watcher.New()

	// SetMaxEvents to 1 to allow at most 1 event's to be received
	// on the Event channel per watching cycle.
	// If SetMaxEvents is not set, the default is to send all events.
	folderWatcher.SetMaxEvents(1)

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
				rabbitMqMessage := rabbitMQ.Message{
					RoutingKey: "enquiries",
					Data: rabbitMQ.FolderWatch{
						Action: event.Op.String(),
						Path:   event.Path,
						IsDir:  isDir,
					},
				}
				if *verbose {
					log.Info().Msg(fmt.Sprintf("Sending message to RabbitMQ : %v", rabbitMqMessage))
				}
				err := rabbitMqClient.SendFolderWatchMessage(rabbitMqMessage)
				if err != nil {
					log.Fatal().Err(err).Msg(fmt.Sprintf("Error sending message to RabbitMQ : %v", rabbitMqMessage))
				}
			case err := <-folderWatcher.Error:
				log.Fatal().Err(err).Msg(fmt.Sprintf("Error handling event for : %s", watchFolderPath))
			case <-folderWatcher.Closed:
				return
			}
		}
	}()

	// Watch the given folder for changes
	if err := folderWatcher.AddRecursive(*watchFolderPath); err != nil {
		log.Fatal().Err(err).Msg(fmt.Sprintf("Error initialising folder to watch : %s", watchFolderPath))
	}
	/*
		env := &config.Env{
			FolderPath: watchFolderPath,
			RabbitMQ:   rabbitMq,
			Watcher:    folderWatcher,
		}
	*/
	// Print a list of all of the files and folders currently being watched and their paths
	for path, f := range folderWatcher.WatchedFiles() {
		isDir := "false"
		if f.IsDir() {
			isDir = "true"
		}
		rabbitMqMessage := rabbitMQ.Message{
			RoutingKey: "enquiries",
			Data: rabbitMQ.FolderWatch{
				Action: "CREATE",
				Path:   path,
				IsDir:  isDir,
			},
		}
		if *verbose {
			log.Info().Msg(fmt.Sprintf("Sending message to RabbitMQ : %v", rabbitMqMessage))
		}
		err := rabbitMqClient.SendFolderWatchMessage(rabbitMqMessage)
		if err != nil {
			log.Fatal().Err(err).Msg(fmt.Sprintf("Error sending message to RabbitMQ : %v", rabbitMqMessage))
		}
		//		fmt.Printf(" - Name: %s | Path: %v | Size: %v | isDirectory: %v | ModTime: %v | Mode: %v \n",
		//			f.Name(), path, f.Size(), f.IsDir(), f.ModTime(), f.Mode())
	}

	// Start the watching process - it'll check for changes every 100ms.
	if err := folderWatcher.Start(time.Millisecond * 100); err != nil {
		log.Fatal().Err(err).Msg(fmt.Sprintf("Error watching the folder %s", watchFolderPath))
	}
}
