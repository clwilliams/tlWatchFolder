package rabbitMQ

import (
	"fmt"
	"time"

	"encoding/xml"

	"github.com/streadway/amqp"
)

// Client wrapper for rabbitMQ
type Client struct {
	Connection *amqp.Connection
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

// Send - posts a message to RabbitMQ
func (client *Client) Send(exchange, routingKey, body string) error {
	ch, err := client.Connection.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := ch.Publish(
		exchange,   // exchange
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
func convertToXML(msg Message, docType string) (string, error) {
	startMsg := `<?xml version="1.0" encoding="UTF-8"?>`
	startMsg += fmt.Sprintf(`<!DOCTYPE %s SYSTEM "%s.dtd"`, docType, docType)
	output, err := xml.MarshalIndent(msg.Data, "  ", "    ")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s\n%s", startMsg, output), nil
}
