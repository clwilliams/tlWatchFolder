package rabbitMQ

// Message -
type Message struct {
	Exchange   string
	RoutingKey string
	Data       FolderWatch `xml:"Data"`
}

// FolderWatch -
type FolderWatch struct {
	Action string `json:"action"`
	Path   string `json:"path"`
	IsDir  string `json:"isDir"`
}

// SendFolderWatchMessage - posts the XML message to RabbitMQ
func (c *Client) SendFolderWatchMessage(msg Message) error {
	jsonMsg, err := convertToJSON(msg)
	if err != nil {
		return err
	}
	if err := c.Send(msg.Exchange, msg.RoutingKey, jsonMsg); err != nil {
		return err
	}
	return nil
}
