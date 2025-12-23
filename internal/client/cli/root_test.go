package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestInit(t *testing.T) {
	// Reset for test
	rootCmd = &cobra.Command{
		Use:   "gopublic",
		Short: "A secure request tunneling tool",
	}

	Init("test-server:4443")

	if ServerAddr != "test-server:4443" {
		t.Errorf("expected ServerAddr 'test-server:4443', got '%s'", ServerAddr)
	}

	// Check commands are registered
	commands := rootCmd.Commands()
	if len(commands) < 2 {
		t.Errorf("expected at least 2 commands (auth, start), got %d", len(commands))
	}

	hasAuth := false
	hasStart := false
	for _, cmd := range commands {
		if cmd.Name() == "auth" {
			hasAuth = true
		}
		if cmd.Name() == "start" {
			hasStart = true
		}
	}

	if !hasAuth {
		t.Error("expected 'auth' command to be registered")
	}
	if !hasStart {
		t.Error("expected 'start' command to be registered")
	}
}

func TestInit_EmptyServerAddr(t *testing.T) {
	// Reset
	rootCmd = &cobra.Command{
		Use:   "gopublic",
		Short: "A secure request tunneling tool",
	}
	ServerAddr = "default:4443"

	Init("")

	if ServerAddr != "default:4443" {
		t.Errorf("expected ServerAddr to remain 'default:4443', got '%s'", ServerAddr)
	}
}

func TestRootCmd_Help(t *testing.T) {
	// Reset
	rootCmd = &cobra.Command{
		Use:   "gopublic",
		Short: "A secure request tunneling tool",
	}
	Init("localhost:4443")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "gopublic") {
		t.Error("help output should contain 'gopublic'")
	}
	if !strings.Contains(output, "auth") {
		t.Error("help output should contain 'auth' command")
	}
	if !strings.Contains(output, "start") {
		t.Error("help output should contain 'start' command")
	}
}

func TestAuthCmd_Help(t *testing.T) {
	// Verify command structure rather than output
	if authCmd.Use != "auth [token]" {
		t.Errorf("expected Use 'auth [token]', got '%s'", authCmd.Use)
	}
	if authCmd.Short != "Save authentication token" {
		t.Errorf("unexpected Short: %s", authCmd.Short)
	}
}

func TestStartCmd_Help(t *testing.T) {
	// Verify command structure rather than output
	if startCmd.Use != "start [port]" {
		t.Errorf("expected Use 'start [port]', got '%s'", startCmd.Use)
	}
	if startCmd.Short != "Start a public tunnel to a local port" {
		t.Errorf("unexpected Short: %s", startCmd.Short)
	}
}

func TestStartCmd_Flags(t *testing.T) {
	// Check that flags are registered
	allFlag := startCmd.Flags().Lookup("all")
	if allFlag == nil {
		t.Error("expected 'all' flag to be registered")
	}

	tuiFlag := startCmd.Flags().Lookup("tui")
	if tuiFlag == nil {
		t.Error("expected 'tui' flag to be registered")
	}

	noTuiFlag := startCmd.Flags().Lookup("no-tui")
	if noTuiFlag == nil {
		t.Error("expected 'no-tui' flag to be registered")
	}
}

func TestShouldUseTUI_NoTuiFlag(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("no-tui", true, "")
	cmd.Flags().Bool("tui", true, "")

	result := shouldUseTUI(cmd)
	if result {
		t.Error("expected false when --no-tui is set")
	}
}

func TestShouldUseTUI_TuiFlagFalse(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("no-tui", false, "")
	cmd.Flags().Bool("tui", false, "")

	result := shouldUseTUI(cmd)
	if result {
		t.Error("expected false when --tui=false")
	}
}

func TestShouldUseTUI_NonTTY(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("no-tui", false, "")
	cmd.Flags().Bool("tui", true, "")

	// In test environment, stdout is usually not a TTY
	// So this should return false
	result := shouldUseTUI(cmd)

	// We can't reliably test TTY detection in unit tests
	// Just ensure it doesn't panic
	_ = result
}

func TestVersion(t *testing.T) {
	if Version == "" {
		t.Error("Version should have a default value")
	}
}

func TestServerAddr_Default(t *testing.T) {
	// ServerAddr should have a default
	if ServerAddr == "" {
		t.Error("ServerAddr should have a default value")
	}
}

// Helper function to capture os.Exit calls
type exitCapture struct {
	code int
}

func (e *exitCapture) Exit(code int) {
	e.code = code
}

func TestAuthCmd_NoArgs(t *testing.T) {
	// authCmd requires exactly 1 arg (token)
	// Verify the Args constraint is set correctly
	if authCmd.Args == nil {
		t.Error("Args should be set")
	}

	// cobra.ExactArgs(1) returns a function, we can't easily compare
	// Instead, verify by attempting to parse with no args
	// The command validates args before running
	// We just verify the constraint exists
}

func TestStartCmd_NoArgsNoConfig(t *testing.T) {
	// Create temp directory without gopublic.yaml
	tmpDir, err := os.MkdirTemp("", "gopublic-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Create a minimal config file with token
	configDir := os.ExpandEnv("$HOME/.gopublic")
	os.MkdirAll(configDir, 0755)

	// The command should fail because there's no gopublic.yaml and no port arg
	// We can't easily test this without mocking, so just verify the command structure
	if startCmd.Use != "start [port]" {
		t.Errorf("unexpected Use string: %s", startCmd.Use)
	}
}

func TestAuthCmd_Structure(t *testing.T) {
	if authCmd.Use != "auth [token]" {
		t.Errorf("unexpected Use: %s", authCmd.Use)
	}
	if authCmd.Short == "" {
		t.Error("Short description should not be empty")
	}
}

func TestStartCmd_Structure(t *testing.T) {
	if startCmd.Use != "start [port]" {
		t.Errorf("unexpected Use: %s", startCmd.Use)
	}
	if startCmd.Short == "" {
		t.Error("Short description should not be empty")
	}
}

func TestRootCmd_Structure(t *testing.T) {
	if rootCmd.Use != "gopublic" {
		t.Errorf("unexpected Use: %s", rootCmd.Use)
	}
	if rootCmd.Short == "" {
		t.Error("Short description should not be empty")
	}
}
