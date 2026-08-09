package clientconfig

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/labstack/gommon/color"
	"gopkg.in/yaml.v3"
)

var UNABLE_TO_OPEN_EDITOR = color.Yellow("Unable to open editor. Please edit the config file manually at " + DefaultConfigPath)

var homedir, _ = os.UserHomeDir()
var DefaultConfigDir = homedir + "/.portr"
var DefaultConfigPath = DefaultConfigDir + "/config.yaml"

func checkDefaultConfigFileExists() bool {
	_, err := os.Stat(DefaultConfigPath)
	return !os.IsNotExist(err)
}

func initConfig() error {
	if checkDefaultConfigFileExists() {
		return nil
	}

	f, err := os.OpenFile(DefaultConfigPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	return os.Chmod(DefaultConfigPath, 0o600)
}

func EditConfig() error {
	if !checkDefaultConfigFileExists() {
		err := initConfig()
		if err != nil {
			return err
		}
	}

	var editorCmd string

	switch os := runtime.GOOS; os {
	case "darwin":
		editorCmd = "open"
	case "linux":
		editorCmd = "xdg-open"
	case "windows":
		editorCmd = "start"
	default:
		fmt.Println(UNABLE_TO_OPEN_EDITOR)
		return nil
	}

	cmd := exec.Command(editorCmd, DefaultConfigPath)
	if err := cmd.Run(); err != nil {
		fmt.Println(UNABLE_TO_OPEN_EDITOR)
		return nil
	}

	return nil
}

func writeDefaultConfigFile(data []byte) error {
	if err := os.WriteFile(DefaultConfigPath, data, 0o600); err != nil {
		return err
	}

	return os.Chmod(DefaultConfigPath, 0o600)
}

func SetConfig(config string) error {
	if !checkDefaultConfigFileExists() {
		err := initConfig()
		if err != nil {
			return err
		}
	}

	return writeDefaultConfigFile([]byte(config))
}

func writeConfigNode(configNode *yaml.Node) error {
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(configNode); err != nil {
		_ = encoder.Close()
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}

	return writeDefaultConfigFile(buf.Bytes())
}

func setConfigMapValue(mappingNode *yaml.Node, key, value string) {
	setConfigMapNode(mappingNode, key, &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: value,
	})
}

func setConfigMapNode(mappingNode *yaml.Node, key string, value *yaml.Node) {
	for i := 0; i < len(mappingNode.Content)-1; i += 2 {
		if mappingNode.Content[i].Value == key {
			mappingNode.Content[i+1] = value
			return
		}
	}

	mappingNode.Content = append(mappingNode.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		value,
	)
}

func configMapNode(mappingNode *yaml.Node, key string) *yaml.Node {
	for i := 0; i < len(mappingNode.Content)-1; i += 2 {
		if mappingNode.Content[i].Value == key {
			return mappingNode.Content[i+1]
		}
	}
	return nil
}

func deleteConfigMapKey(mappingNode *yaml.Node, key string) {
	for i := 0; i < len(mappingNode.Content)-1; i += 2 {
		if mappingNode.Content[i].Value == key {
			mappingNode.Content = append(mappingNode.Content[:i], mappingNode.Content[i+2:]...)
			return
		}
	}
}

// editConfigNode applies edit to the config file's top-level mapping, keeping
// comments, key order and unknown keys intact.
func editConfigNode(edit func(mappingNode *yaml.Node)) error {
	configBytes, err := os.ReadFile(DefaultConfigPath)
	if err != nil {
		return err
	}

	var configNode yaml.Node
	if err := yaml.Unmarshal(configBytes, &configNode); err != nil {
		return err
	}

	if configNode.Kind == 0 || len(configNode.Content) == 0 {
		configNode.Kind = yaml.DocumentNode
		configNode.Content = []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}
	}

	if configNode.Kind != yaml.DocumentNode || len(configNode.Content) == 0 || configNode.Content[0].Kind != yaml.MappingNode {
		return fmt.Errorf("config file must contain a YAML mapping")
	}

	edit(configNode.Content[0])

	return writeConfigNode(&configNode)
}

func updateConfigValues(entries [][2]string) error {
	return editConfigNode(func(mappingNode *yaml.Node) {
		for _, entry := range entries {
			setConfigMapValue(mappingNode, entry[0], entry[1])
		}
	})
}
