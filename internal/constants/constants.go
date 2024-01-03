package constants

type ConnectionType string

func (c *ConnectionType) String() string {
	return string(*c)
}

func (c *ConnectionType) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	switch s {
	case "http":
		*c = Http
	case "tcp":
		*c = Tcp
	default:
		*c = Http
	}

	return nil
}

const (
	Http ConnectionType = "http"
	Tcp  ConnectionType = "tcp"
)
