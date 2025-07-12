package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestMain_AppInitialization(t *testing.T) {
	app := &cli.App{
		Name:    "portr",
		Usage:   "Expose local ports to the public internet",
		Version: "0.0.0",
		Commands: []*cli.Command{
			startCmd(),
			configCmd(),
			httpCmd(),
			tcpCmd(),
			authCmd(),
		},
	}

	assert.Equal(t, "portr", app.Name)
	assert.Equal(t, "Expose local ports to the public internet", app.Usage)
	assert.Len(t, app.Commands, 5)

	// Check command names
	commandNames := make([]string, len(app.Commands))
	for i, cmd := range app.Commands {
		commandNames[i] = cmd.Name
	}

	expectedCommands := []string{"start", "config", "http", "tcp", "auth"}
	for _, expected := range expectedCommands {
		assert.Contains(t, commandNames, expected)
	}
}

func TestStartCmd(t *testing.T) {
	cmd := startCmd()

	assert.Equal(t, "start", cmd.Name)
	assert.Equal(t, "Start the tunnels from the config file", cmd.Usage)
	assert.NotNil(t, cmd.Action)
}

func TestConfigCmd(t *testing.T) {
	cmd := configCmd()

	assert.Equal(t, "config", cmd.Name)
	assert.Equal(t, "Edit the portr config file", cmd.Usage)
	assert.Len(t, cmd.Subcommands, 1)
	assert.Equal(t, "edit", cmd.Subcommands[0].Name)
}

func TestHttpCmd(t *testing.T) {
	cmd := httpCmd()

	assert.Equal(t, "http", cmd.Name)
	assert.Equal(t, "Expose http/ws port", cmd.Usage)
	assert.Len(t, cmd.Flags, 1)
	assert.NotNil(t, cmd.Action)

	// Check subdomain flag
	subdomainFlag := cmd.Flags[0].(*cli.StringFlag)
	assert.Equal(t, "subdomain", subdomainFlag.Name)
	assert.Contains(t, subdomainFlag.Aliases, "s")
}

func TestTcpCmd(t *testing.T) {
	cmd := tcpCmd()

	assert.Equal(t, "tcp", cmd.Name)
	assert.Equal(t, "Expose tcp port", cmd.Usage)
	assert.NotNil(t, cmd.Action)
}

func TestAuthCmd(t *testing.T) {
	cmd := authCmd()

	assert.Equal(t, "auth", cmd.Name)
	assert.Equal(t, "Setup portr cli auth", cmd.Usage)
	assert.Len(t, cmd.Subcommands, 1)

	setCmd := cmd.Subcommands[0]
	assert.Equal(t, "set", setCmd.Name)
	assert.Len(t, setCmd.Flags, 2)

	// Check flags
	flagNames := make([]string, len(setCmd.Flags))
	for i, flag := range setCmd.Flags {
		flagNames[i] = flag.Names()[0]
	}

	assert.Contains(t, flagNames, "token")
	assert.Contains(t, flagNames, "remote")
}

func TestHttpCmd_Action_InvalidPort(t *testing.T) {
	cmd := httpCmd()
	
	// Create a mock context with invalid port
	app := &cli.App{}
	ctx := cli.NewContext(app, nil, nil)
	ctx.Args().(*cli.Args).Set("invalid-port")

	err := cmd.Action(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "please specify a valid port")
}

func TestTcpCmd_Action_InvalidPort(t *testing.T) {
	cmd := tcpCmd()
	
	// Create a mock context with invalid port
	app := &cli.App{}
	ctx := cli.NewContext(app, nil, nil)
	ctx.Args().(*cli.Args).Set("invalid-port")

	err := cmd.Action(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "please specify a valid port")
}

func TestHttpCmd_Action_ValidPort(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that would try to start tunnels")
	}

	// This test checks that the command creates the right tunnel config
	// We'll mock the startTunnels function by checking the tunnel parameter
	originalStartTunnels := startTunnels

	var capturedTunnel *config.Tunnel
	startTunnels = func(c *cli.Context, tunnelFromCli *config.Tunnel) error {
		capturedTunnel = tunnelFromCli
		return fmt.Errorf("mock error to prevent actual tunnel start")
	}
	defer func() { startTunnels = originalStartTunnels }()

	cmd := httpCmd()
	
	// Create a mock context with valid port
	app := &cli.App{}
	ctx := cli.NewContext(app, nil, nil)
	ctx.Args().(*cli.Args).Set("3000")

	err := cmd.Action(ctx)
	
	// We expect an error because we mocked startTunnels to return an error
	assert.Error(t, err)
	assert.Equal(t, "mock error to prevent actual tunnel start", err.Error())
	
	// Check that the tunnel was configured correctly
	assert.NotNil(t, capturedTunnel)
	assert.Equal(t, 3000, capturedTunnel.Port)
	assert.Equal(t, "http", capturedTunnel.Type)
}

func TestTcpCmd_Action_ValidPort(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that would try to start tunnels")
	}

	// Mock startTunnels function
	originalStartTunnels := startTunnels

	var capturedTunnel *config.Tunnel
	startTunnels = func(c *cli.Context, tunnelFromCli *config.Tunnel) error {
		capturedTunnel = tunnelFromCli
		return fmt.Errorf("mock error to prevent actual tunnel start")
	}
	defer func() { startTunnels = originalStartTunnels }()

	cmd := tcpCmd()
	
	// Create a mock context with valid port
	app := &cli.App{}
	ctx := cli.NewContext(app, nil, nil)
	ctx.Args().(*cli.Args).Set("8080")

	err := cmd.Action(ctx)
	
	// We expect an error because we mocked startTunnels to return an error
	assert.Error(t, err)
	assert.Equal(t, "mock error to prevent actual tunnel start", err.Error())
	
	// Check that the tunnel was configured correctly
	assert.NotNil(t, capturedTunnel)
	assert.Equal(t, 8080, capturedTunnel.Port)
	assert.Equal(t, "tcp", capturedTunnel.Type)
	assert.Equal(t, "", capturedTunnel.Subdomain)
}

func TestStartTunnels_ConfigError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that requires config loading")
	}

	// Create a temporary directory for testing
	tempDir := t.TempDir()
	oldConfigDir := config.DefaultConfigDir
	config.DefaultConfigDir = tempDir
	defer func() { config.DefaultConfigDir = oldConfigDir }()

	// Create app context
	app := &cli.App{}
	ctx := cli.NewContext(app, nil, nil)
	ctx.Context = context.Background()

	// Test with non-existent config file
	err := startTunnels(ctx, nil)
	assert.Error(t, err)
}

func TestGetVersionToUpdate_NoUpdateFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Temporarily change the UpdatesFilePath
	oldUpdatesFilePath := UpdatesFilePath
	UpdatesFilePath = tempDir + "/updates.json"
	defer func() { UpdatesFilePath = oldUpdatesFilePath }()

	// Test with no update file
	versionToUpdate, err := getVersionToUpdate()
	assert.NoError(t, err)
	assert.Equal(t, "", versionToUpdate)
}

func TestCreateUpdatesFileIfNotExists(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Temporarily change the UpdatesFilePath
	oldUpdatesFilePath := UpdatesFilePath
	UpdatesFilePath = tempDir + "/updates.json"
	defer func() { UpdatesFilePath = oldUpdatesFilePath }()

	// Test file creation
	err := createUpdatesFileIfNotExists()
	assert.NoError(t, err)

	// Verify file exists and has correct content
	_, err = os.Stat(UpdatesFilePath)
	assert.NoError(t, err)

	content, err := os.ReadFile(UpdatesFilePath)
	assert.NoError(t, err)
	assert.Equal(t, "{}", string(content))
}

func TestGetUpdateState_ValidFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Temporarily change the UpdatesFilePath
	oldUpdatesFilePath := UpdatesFilePath
	UpdatesFilePath = tempDir + "/updates.json"
	defer func() { UpdatesFilePath = oldUpdatesFilePath }()

	// Create test file with valid JSON
	testContent := `{"checked_at":"2023-01-01T00:00:00Z","version":"v1.0.0"}`
	err := os.WriteFile(UpdatesFilePath, []byte(testContent), 0644)
	require.NoError(t, err)

	// Test reading update state
	updateInfo, err := getUpdateState()
	assert.NoError(t, err)
	assert.Equal(t, "v1.0.0", updateInfo.Version)
	
	expectedTime, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00Z")
	assert.Equal(t, expectedTime, updateInfo.CheckedAt)
}

func TestGetUpdateState_InvalidFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Temporarily change the UpdatesFilePath
	oldUpdatesFilePath := UpdatesFilePath
	UpdatesFilePath = tempDir + "/updates.json"
	defer func() { UpdatesFilePath = oldUpdatesFilePath }()

	// Create test file with invalid JSON
	err := os.WriteFile(UpdatesFilePath, []byte("invalid json"), 0644)
	require.NoError(t, err)

	// Test reading update state should fail
	_, err = getUpdateState()
	assert.Error(t, err)
}

func TestAuthCmd_SetAction_MissingFlags(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that might interact with config")
	}

	cmd := authCmd()
	setCmd := cmd.Subcommands[0]

	// Create context without required flags
	app := &cli.App{}
	ctx := cli.NewContext(app, nil, nil)

	// This should fail because required flags are missing
	// Note: In a real scenario, the CLI framework would catch this before calling Action
	// But we can test the action itself with empty values
	
	// Test with missing token/remote should be handled by the CLI framework
	// The action assumes these are provided since they're marked as required
}

// Integration-style test helpers

func createMockContext(args []string, flags map[string]string) *cli.Context {
	app := &cli.App{}
	set := cli.NewFlagSet("test", 0)
	
	for name, value := range flags {
		set.String(name, value, "test flag")
	}
	
	ctx := cli.NewContext(app, set, nil)
	for _, arg := range args {
		ctx.Args().(*cli.Args).Set(arg)
	}
	
	return ctx
}

func TestConfigCmd_EditAction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that might open editor")
	}

	cmd := configCmd()
	editCmd := cmd.Subcommands[0]

	assert.Equal(t, "edit", editCmd.Name)
	assert.Equal(t, "Edit the default config file", editCmd.Usage)
	assert.NotNil(t, editCmd.Action)
}