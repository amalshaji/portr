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
	case "stub":
		*c = Stub
	case "static":
		*c = Static
	default:
		*c = Http
	}

	return nil
}

// WireType returns the connection type sent to the server. Stub and static
// tunnels are client-side concepts served over the HTTP tunnel protocol; the
// server only ever accepts http and tcp.
func (c ConnectionType) WireType() ConnectionType {
	if c == Stub || c == Static {
		return Http
	}
	return c
}

// IsHTTPLike reports whether the tunnel is addressed by subdomain over HTTP.
func (c ConnectionType) IsHTTPLike() bool {
	return c == Http || c == Stub || c == Static
}

const (
	Http   ConnectionType = "http"
	Tcp    ConnectionType = "tcp"
	Stub   ConnectionType = "stub"
	Static ConnectionType = "static"
)

const ClientUiViteDistDir = "./internal/client/dashboard/ui-v2/dist/static/.vite/manifest.json"
