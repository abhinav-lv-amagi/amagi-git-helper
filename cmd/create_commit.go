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

// Maximum allowed length for the commit description.
const maxCommitDescLength = 50

// getCurrentBranch returns the current git branch name.
func getCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// extractTicketFromBranch extracts the JIRA ticket from the current branch name.
// It uses the last part after splitting by '/' as the ticket ID.
func extractTicketFromBranch(branch string) (string, error) {
	parts := strings.Split(branch, "/")
	if len(parts) < 1 {
		return "", fmt.Errorf("branch name does not contain a '/' separator")
	}
	ticket := parts[len(parts)-1]
	// Validate ticket format (e.g., ABC-123 or CLI-34343)
	matched, err := regexp.MatchString(`^[A-Za-z]+-\d+$`, ticket)
	if err != nil {
		return "", err
	}
	if !matched {
		return "", fmt.Errorf("extracted ticket ID '%s' does not match expected pattern", ticket)
	}
	return ticket, nil
}

// createCommitCmd represents the command to interactively create a commit message.
var createCommitCmd = &cobra.Command{
	Use:   "create-commit",
	Short: "Interactively create a commit message following company conventions",
	Long: `This command interactively builds a commit message with the following template:
  
git commit -m "${type: fix | feat}(${product: lego | plec}): ${commit desc (short)}" -m "${type: Fixes | Closes} ${JIRA ticket id}"

It prompts for commit type, product, and a short description, and extracts the JIRA ticket id from the current branch name.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 0. Check if there are staged changes.
		stagedCheck := exec.Command("git", "diff", "--cached", "--quiet")
		if err := stagedCheck.Run(); err == nil {
			// If no error, then nothing is staged.
			return fmt.Errorf("no staged changes found. Please stage your changes before committing")
		}

		// Variables to store commit details.
		var commitType string
		var product string
		var commitDesc string

		// 1. Prompt for commit type.
		commitTypeOptions := []string{"fix", "feat"}
		if err := survey.AskOne(&survey.Select{
			Message: "Select commit type:",
			Options: commitTypeOptions,
		}, &commitType); err != nil {
			return err
		}

		// 2. Prompt for product.
		productOptions := []string{"lego", "plec"}
		if err := survey.AskOne(&survey.Select{
			Message: "Select product:",
			Options: productOptions,
		}, &product); err != nil {
			return err
		}

		// 3. Prompt for commit description.
		if err := survey.AskOne(&survey.Input{
			Message: "Enter a short commit description:",
		}, &commitDesc, survey.WithValidator(func(val interface{}) error {
			str, ok := val.(string)
			if !ok {
				return fmt.Errorf("invalid input")
			}
			if len(str) == 0 {
				return fmt.Errorf("commit description cannot be empty")
			}
			if len(str) > maxCommitDescLength {
				return fmt.Errorf("commit description too long (max %d characters)", maxCommitDescLength)
			}
			return nil
		})); err != nil {
			return err
		}

		// 4. Get current branch and extract ticket ID.
		branch, err := getCurrentBranch()
		if err != nil {
			return err
		}
		ticketID, err := extractTicketFromBranch(branch)
		if err != nil {
			return fmt.Errorf("failed to extract JIRA ticket from branch '%s': %w", branch, err)
		}

		// 5. Assemble the commit messages.
		// First message: "<type>(<product>): <commitDesc>"
		// Second message: "<CapitalizedType> <ticketID>"
		firstMsg := fmt.Sprintf("%s(%s): %s", commitType, product, commitDesc)
		var verb string
		switch commitType {
		case "fix":
			verb = "Fixes"
		case "feat":
			verb = "Closes"
		default:
			verb = strings.Title(commitType)
		}
		secondMsg := fmt.Sprintf("%s %s", verb, ticketID)

		fmt.Println("\nThe following commit messages will be created:")
		fmt.Printf("Message 1: %s\n", firstMsg)
		fmt.Printf("Message 2: %s\n", secondMsg)

		// 6. Ask for confirmation.
		confirm := false
		if err := survey.AskOne(&survey.Confirm{
			Message: "Do you want to proceed with this commit?",
		}, &confirm); err != nil {
			return err
		}
		if !confirm {
			fmt.Println("Commit creation aborted.")
			return nil
		}

		// 7. Execute the git commit command.
		gitCmd := exec.Command("git", "commit", "-m", firstMsg, "-m", secondMsg)
		gitCmd.Stdout = os.Stdout
		gitCmd.Stderr = os.Stderr

		fmt.Println("Executing git commit...")
		if err := gitCmd.Run(); err != nil {
			return fmt.Errorf("failed to create commit: %w", err)
		}

		fmt.Println("Commit created successfully!")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(createCommitCmd)
}

// bleh
