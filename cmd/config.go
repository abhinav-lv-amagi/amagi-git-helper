package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
)

// Config represents the configuration structure.
type Config struct {
	Abbreviation string `json:"abbreviation"`
}

// configFilePath returns the path to the config file in the user's home directory.
func configFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configDir := filepath.Join(homeDir, ".git-helper-cli")
	// Ensure the config directory exists.
	if err := os.MkdirAll(configDir, os.ModePerm); err != nil {
		return "", err
	}
	return filepath.Join(configDir, "config.json"), nil
}

// saveConfig writes the config struct to a JSON file.
func saveConfig(cfg Config) error {
	configPath, err := configFilePath()
	if err != nil {
		return err
	}
	file, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(cfg)
}

// loadConfig reads the configuration from file.
func loadConfig() (Config, error) {
	var cfg Config
	configPath, err := configFilePath()
	if err != nil {
		return cfg, err
	}

	file, err := os.Open(configPath)
	if err != nil {
		// If the file doesn't exist, return an empty config.
		return cfg, nil
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&cfg)
	return cfg, err
}

// configCmd represents the command to set/update configuration.
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure your git-helper-cli settings",
	Long:  "Set or update your two-letter abbreviation used in branch naming.",
	RunE: func(cmd *cobra.Command, args []string) error {
		var abbrev string

		// Prompt the user for a two-letter abbreviation.
		prompt := &survey.Input{
			Message: "Enter your two-letter abbreviation:",
		}
		// Validate that the input is exactly two letters.
		validator := func(val interface{}) error {
			str, ok := val.(string)
			if !ok {
				return fmt.Errorf("invalid input")
			}
			matched, err := regexp.MatchString("^[A-Za-z]{2}$", str)
			if err != nil {
				return err
			}
			if !matched {
				return fmt.Errorf("abbreviation must be exactly two letters")
			}
			return nil
		}

		if err := survey.AskOne(prompt, &abbrev, survey.WithValidator(validator)); err != nil {
			return err
		}

		cfg := Config{Abbreviation: abbrev}
		if err := saveConfig(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Println("Configuration saved successfully!")
		return nil
	},
}

// showConfigCmd represents the command to display the current configuration.
var showConfigCmd = &cobra.Command{
	Use:   "show-config",
	Short: "Display the current configuration",
	Long:  "Display the currently stored configuration for git-helper-cli.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Check if the configuration is empty.
		if cfg.Abbreviation == "" {
			fmt.Println("No configuration found. Please run 'git-helper-cli config' to set up your configuration.")
			return nil
		}

		fmt.Println("Current Configuration:")
		fmt.Printf("  Two-letter Abbreviation: %s\n", cfg.Abbreviation)
		return nil
	},
}

func init() {
	// Add both commands as subcommands of the root command.
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(showConfigCmd)
}
