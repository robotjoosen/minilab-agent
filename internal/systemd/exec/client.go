package exec

type Client struct{}

func New() *Client {
	return &Client{}
}

func (c *Client) Run(name string, args ...string) (string, error) {
	return execCommand(name, args...)
}
