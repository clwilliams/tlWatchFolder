package config

import (
	"git.thebookpeople.com/magento/headless/rabbitMQ"
)

// Env - Environmental data & services
type Env struct {
	folderPath *string,
	RabbitMQ *rabbitMQ.Client
	Watcher  *Watcher
}
