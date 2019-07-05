package rabbitMQ

import (
	"fmt"
	"time"

	"encoding/xml"

	"github.com/streadway/amqp"
)

// Client wrapper for rabbitMQ
type Client struct {
	*amqp.Connection
}

// Message -
type Message struct {
	RoutingKey string
	Data       FolderWatch `xml:"FileChange "`
}

// FolderWatch -
type FolderWatch struct {
	XMLName xml.Name `xml:"FolderWatch"`
	Type    string   `xml:"type"`
	Name    string   `xml:"name"`
}

// NewClient -
func NewClient(rabbitMqHost, rabbitMqPort, rabbitMqUser, rabbitMqPassword *string) (*Client, error) {
	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%s/", *rabbitMqUser, *rabbitMqPassword, *rabbitMqHost, *rabbitMqPort))
	if err != nil {
		return nil, err
	}

	return &Client{
		conn,
	}, nil
}

// SendFolderWatchMessage - posts the XML message to RabbitMQ
func (c *Client) SendFolderWatchMessage(msg Message) error {
	xmlMsg, err := convertToXML(msg)
	if err != nil {
		return err
	}
	if err := send(c.Connection, msg.RoutingKey, xmlMsg); err != nil {
		return err
	}
	return nil
}

// posts a message to RabbitMQ
func send(conn *amqp.Connection, routingKey, body string) error {
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := ch.Publish(
		"magento",  // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType: "text/xml",
			Body:        []byte(body),
			MessageId:   time.Now().String(),
		}); err != nil {
		return err
	}
	return nil
}

// converts the struct into the XML body for the RabbitMQ message
func convertToXML(msg Message) (string, error) {
	startMsg := `<?xml version="1.0" encoding="UTF-8"?>
  <!DOCTYPE maginusQMSCreateNewCall SYSTEM "MaginusQMSCreateNewCall.dtd">`
	output, err := xml.MarshalIndent(msg.Data, "  ", "    ")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s\n%s", startMsg, output), nil
}