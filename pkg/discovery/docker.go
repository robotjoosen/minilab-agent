package discovery

type Container struct {
	Name  string
	Image string
	State string // "running", "exited", etc.
}

type DockerClient interface {
	ListContainers() ([]Container, error)
}
