package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
)

// Maximum length allowed for the short description (after replacing spaces with hyphens)
const maxDescLength = 30

// createBranchCmd represents the create-branch command.
var createBranchCmd = &cobra.Command{
	Use:   "create-branch",
	Short: "Create and switch to a new branch following company conventions",
	Long: `Interactively create a new branch that follows the naming convention:
<abbreviation>-<type>-<short_desc>/<JIRA_ticket_id>
For example: lv-fix-user-details-window-width/CPRE-11347`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Load user configuration.
		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		if cfg.Abbreviation == "" {
			return fmt.Errorf("no configuration found. Please run 'git-helper-cli config' to set your two-letter abbreviation")
		}

		// Variables to store the branch details.
		branchType := ""
		description := ""
		ticketID := ""

		// Function to prompt for branch type.
		promptBranchType := func() error {
			options := []string{"fix", "feat"}
			prompt := &survey.Select{
				Message: "Choose branch type:",
				Options: options,
			}
			return survey.AskOne(prompt, &branchType)
		}

		// Function to prompt for branch description.
		promptDescription := func() error {
			prompt := &survey.Input{
				Message: "Enter a short branch description (spaces will be replaced with hyphens):",
			}
			validator := func(val interface{}) error {
				str, ok := val.(string)
				if !ok {
					return fmt.Errorf("invalid input")
				}
				formatted := strings.ReplaceAll(str, " ", "-")
				if len(formatted) > maxDescLength {
					return fmt.Errorf("description too long (max %d characters after formatting)", maxDescLength)
				}
				if len(formatted) == 0 {
					return fmt.Errorf("description cannot be empty")
				}
				return nil
			}
			return survey.AskOne(prompt, &description, survey.WithValidator(validator))
		}

		// Function to prompt for JIRA Ticket ID.
		promptTicketID := func() error {
			prompt := &survey.Input{
				Message: "Enter the JIRA Ticket ID (e.g., CPRE-11347):",
			}
			validator := func(val interface{}) error {
				str, ok := val.(string)
				if !ok {
					return fmt.Errorf("invalid input")
				}
				matched, err := regexp.MatchString(`^[A-Za-z]+-\d+$`, str)
				if err != nil {
					return err
				}
				if !matched {
					return fmt.Errorf("ticket ID must be in format ABC-123")
				}
				return nil
			}
			return survey.AskOne(prompt, &ticketID, survey.WithValidator(validator))
		}

		// Initial prompts
		if err := promptBranchType(); err != nil {
			return err
		}
		if err := promptDescription(); err != nil {
			return err
		}
		// Replace spaces with hyphens for consistency.
		description = strings.ReplaceAll(description, " ", "-")
		if err := promptTicketID(); err != nil {
			return err
		}

		// A helper to assemble the branch name.
		assembleBranchName := func() string {
			return fmt.Sprintf("%s-%s-%s/%s", strings.ToLower(cfg.Abbreviation), branchType, strings.ToLower(description), ticketID)
		}

		// Loop to allow user to review and edit inputs.
		for {
			branchName := assembleBranchName()
			fmt.Printf("\nProposed branch name: %s\n", branchName)

			// Offer options to either confirm or edit details.
			menuOptions := []string{
				"Confirm and create branch",
				"Edit branch type",
				"Edit description",
				"Edit JIRA ticket ID",
				"Cancel",
			}
			var choice string
			menuPrompt := &survey.Select{
				Message: "What would you like to do?",
				Options: menuOptions,
			}
			if err := survey.AskOne(menuPrompt, &choice); err != nil {
				return err
			}

			switch choice {
			case "Confirm and create branch":
				// Confirm and proceed to create the branch.
				confirm := false
				confirmPrompt := &survey.Confirm{
					Message: fmt.Sprintf("Create branch '%s'?", branchName),
				}
				if err := survey.AskOne(confirmPrompt, &confirm); err != nil {
					return err
				}
				if confirm {
					// Execute the Git command: git checkout -b <branchName>
					cmdGit := exec.Command("git", "checkout", "-b", branchName)
					cmdGit.Stdout = os.Stdout
					cmdGit.Stderr = os.Stderr

					fmt.Printf("Executing: git checkout -b %s\n", branchName)
					if err := cmdGit.Run(); err != nil {
						return fmt.Errorf("failed to create branch: %w", err)
					}

					fmt.Println("Branch created and switched successfully!")
					return nil
				}
				// If not confirmed, continue the loop.
			case "Edit branch type":
				if err := promptBranchType(); err != nil {
					return err
				}
			case "Edit description":
				if err := promptDescription(); err != nil {
					return err
				}
				description = strings.ReplaceAll(description, " ", "-")
			case "Edit JIRA ticket ID":
				if err := promptTicketID(); err != nil {
					return err
				}
			case "Cancel":
				fmt.Println("Aborting branch creation.")
				return nil
			}
			// After editing, the loop will reassemble the branch name and present the menu again.
		}
	},
}

func init() {
	rootCmd.AddCommand(createBranchCmd)
}
