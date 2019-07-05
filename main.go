package main

import (
  "fmt"
  "log"
  "time"
  "github.com/radovskyb/watcher"
  "github.com/alecthomas/kingpin"
)

const (
	defaultrabbitMqPort = "3000"
  defaultrabbitMqURL = "TODO"
)

var (
  rabbitMqPort = kingpin.Flag("rabbitMqPort", "RabbitMQ port").Default(defaultrabbitMqPort).Envar("DEFAULT_RABBITMQ_PORT").String()
  rabbitMqURL = kingpin.Flag("rabbitMqUrl", "RabbitMQ url").Default(defaultrabbitMqURL).Envar("DEFAULT_RABBITMQ_URL").String()
  watchFolderPath = kingpin.Flag("watchFolderPath", "Path to watch file changes within").String()
)

func main() {
  // watchFolderPath := os.Args[1]
  // watchFolderPath := "/Users/clairew/watch_me/2019/March"

  w := watcher.New()

	// SetMaxEvents to 1 to allow at most 1 event's to be received
	// on the Event channel per watching cycle.
	//
	// If SetMaxEvents is not set, the default is to send all events.
	w.SetMaxEvents(1)

	// the events we want to be notified of
	w.FilterOps(watcher.Rename, watcher.Move, watcher.Create, watcher.Remove)

  go func() {
		for {
			select {
			case event := <-w.Event:
				fmt.Printf("here %v", event) // Print the event's info.
			case err := <-w.Error:
				log.Fatalln(err)
			case <-w.Closed:
				return
			}
		}
	}()

	// Watch the given folder for changes
	if err := w.Add(*watchFolderPath); err != nil {
		log.Fatalln(err)
	}

	// Print a list of all of the files and folders currently
	// being watched and their paths.
	for path, f := range w.WatchedFiles() {
		fmt.Printf("%s: %s\n", path, f.Name())
	}

	fmt.Println()

	// Start the watching process - it'll check for changes every 100ms.
	if err := w.Start(time.Millisecond * 100); err != nil {
		log.Fatalln(err)
	}
}
