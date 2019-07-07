package rabbitMQ

import (
	"encoding/xml"
)

// Message -
type Message struct {
  Exchange   string
	RoutingKey string
	Data       FolderWatch `xml:"Data"`
}

// FolderWatch -
type FolderWatch struct {
	XMLName xml.Name `xml:"FolderWatch"`
	Action  string   `xml:"action"`
	Path    string   `xml:"path"`
	IsDir   string   `xml:"isDir"`
}

// SendFolderWatchMessage - posts the XML message to RabbitMQ
func (c *Client) SendFolderWatchMessage(msg Message) error {
  docType := "folderWatch.dtd"
	xmlMsg, err := convertToXML(msg, docType)
	if err != nil {
		return err
	}
	if err := c.Send(msg.Exchange, msg.RoutingKey, xmlMsg); err != nil {
		return err
	}
	return nil
}
