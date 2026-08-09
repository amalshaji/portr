package clientconfig

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/go-resty/resty/v2"
	"gopkg.in/yaml.v3"
)

type downloadResponse struct {
	Message string `json:"message"`
	// HasTemplate reports whether the team configured a client template. When
	// false the server sent its sample tunnel, which must not replace tunnels
	// the member already has.
	HasTemplate bool `json:"has_template"`
}

type apiError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func (e apiError) String() string {
	if e.Error != "" {
		return e.Error
	}
	return e.Message
}

func remoteAddress(remote string) string {
	if strings.HasPrefix(remote, "http://") || strings.HasPrefix(remote, "https://") {
		return remote
	}
	if strings.HasPrefix(remote, "localhost:") {
		return "http://" + remote
	}
	return "https://" + remote
}

func downloadConfig(token string, remote string) (downloadResponse, error) {
	var result downloadResponse
	var failure apiError

	resp, err := resty.New().R().
		SetResult(&result).
		SetError(&failure).
		SetBody(map[string]string{"secret_key": token}).
		Post(remoteAddress(remote) + "/api/v1/config/download")
	if err != nil {
		return downloadResponse{}, err
	}

	if resp.StatusCode() != http.StatusOK {
		message := failure.String()
		if message == "" {
			message = resp.Status()
		}
		return downloadResponse{}, fmt.Errorf("%s", message)
	}

	return result, nil
}

func GetConfig(token string, remote string) error {
	configFileExists := checkDefaultConfigFileExists()

	response, err := downloadConfig(token, remote)
	if err != nil {
		return err
	}

	if configFileExists {
		return updateAuthValues(token, response.Message)
	}

	return SetConfig(response.Message)
}

func updateAuthValues(token string, downloaded string) error {
	var config struct {
		ServerUrl string `yaml:"server_url"`
		SshUrl    string `yaml:"ssh_url"`
	}
	if err := yaml.Unmarshal([]byte(downloaded), &config); err != nil {
		return err
	}

	entries := [][2]string{{"secret_key", token}}
	if config.ServerUrl != "" {
		// the server doesn't send tunnel_url; tunnels are served on the server_url domain
		entries = append(entries,
			[2]string{"server_url", config.ServerUrl},
			[2]string{"tunnel_url", config.ServerUrl},
		)
	}
	if config.SshUrl != "" {
		entries = append(entries, [2]string{"ssh_url", config.SshUrl})
	}

	return updateConfigValues(entries)
}

// TemplateChanges describes what a pull did to the local tunnels and groups.
type TemplateChanges struct {
	Tunnels []string
	Added   []string
	Removed []string
	Groups  []string
}

// PullConfig replaces the local tunnels and groups with the team template.
// Everything else in the config file, including comments, is left alone.
func PullConfig() (TemplateChanges, error) {
	if !checkDefaultConfigFileExists() {
		return TemplateChanges{}, fmt.Errorf("no config file at %s; run `portr auth set` first", DefaultConfigPath)
	}

	current, err := Load(DefaultConfigPath)
	if err != nil {
		return TemplateChanges{}, err
	}
	if current.SecretKey == "" {
		return TemplateChanges{}, fmt.Errorf("no secret_key in %s; run `portr auth set` first", DefaultConfigPath)
	}

	response, err := downloadConfig(current.SecretKey, current.ServerUrl)
	if err != nil {
		return TemplateChanges{}, err
	}
	if !response.HasTemplate {
		return TemplateChanges{}, fmt.Errorf("your team has no client template configured; ask a team admin to add one in the portr dashboard")
	}

	downloaded, err := documentMapping(response.Message)
	if err != nil || downloaded == nil {
		return TemplateChanges{}, fmt.Errorf("invalid config from the server: %w", err)
	}

	var incoming Config
	if err := downloaded.Decode(&incoming); err != nil {
		return TemplateChanges{}, err
	}

	err = editConfigNode(func(mappingNode *yaml.Node) {
		for _, key := range templateKeys {
			if value := configMapNode(downloaded, key); value != nil {
				setConfigMapNode(mappingNode, key, value)
			} else {
				deleteConfigMapKey(mappingNode, key)
			}
		}
	})
	if err != nil {
		return TemplateChanges{}, err
	}

	return templateChanges(current, incoming), nil
}

func templateChanges(current Config, incoming Config) TemplateChanges {
	changes := TemplateChanges{
		Tunnels: tunnelNames(incoming.Tunnels),
		Groups:  make([]string, 0, len(incoming.Groups)),
	}

	currentNames := tunnelNames(current.Tunnels)
	for _, name := range changes.Tunnels {
		if !slices.Contains(currentNames, name) {
			changes.Added = append(changes.Added, name)
		}
	}
	for _, name := range currentNames {
		if !slices.Contains(changes.Tunnels, name) {
			changes.Removed = append(changes.Removed, name)
		}
	}

	for group := range incoming.Groups {
		changes.Groups = append(changes.Groups, group)
	}
	slices.Sort(changes.Groups)

	return changes
}

func tunnelNames(tunnels []Tunnel) []string {
	names := make([]string, 0, len(tunnels))
	for _, tunnel := range tunnels {
		if tunnel.Name != "" {
			names = append(names, tunnel.Name)
		}
	}
	return names
}
