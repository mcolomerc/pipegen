package cmd

import (
	"context"
	"fmt"
	"os"

	"pipegen/internal/docker"
	"pipegen/internal/pipeline"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// cleanCmd represents the clean command
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean up Docker containers and resources",
	Long: `Clean up Docker containers, networks, and volumes created by PipeGen.

This command will:
- Stop and remove Docker containers
- Remove Docker networks
- Remove Docker volumes (with --volumes flag)
- Clean up any orphaned containers

Examples:
  pipegen clean                    # Stop and remove containers
  pipegen clean --volumes          # Also remove volumes
  pipegen clean --all              # Remove everything including images`,
	RunE: runClean,
}

var (
	cleanVolumes bool
	cleanAll     bool
	cleanForce   bool
	cleanFlink   bool
)

func init() {
	rootCmd.AddCommand(cleanCmd)

	cleanCmd.Flags().BoolVar(&cleanVolumes, "volumes", false, "Remove volumes as well")
	cleanCmd.Flags().BoolVar(&cleanAll, "all", false, "Remove everything including Docker images")
	cleanCmd.Flags().BoolVar(&cleanForce, "force", false, "Force removal without confirmation")
	cleanCmd.Flags().BoolVar(&cleanFlink, "flink", false, "Cancel all running Flink jobs")
}

func runClean(cmd *cobra.Command, args []string) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check if docker-compose.yml exists
	if _, err := os.Stat("docker-compose.yml"); os.IsNotExist(err) {
		return fmt.Errorf("docker-compose.yml not found in current directory. Please run from a PipeGen project directory")
	}

	deployer := docker.NewDeployer(projectDir, "pipegen-project")

	// Check Docker availability
	if err := deployer.CheckDockerAvailability(); err != nil {
		return fmt.Errorf("docker is not available: %w", err)
	}

	// Confirm action unless forced
	if !cleanForce {
		var action string
		if cleanAll {
			action = "stop containers, remove networks, volumes, and images"
		} else if cleanVolumes {
			action = "stop containers, remove networks and volumes"
		} else {
			action = "stop containers and remove networks"
		}

		fmt.Printf("⚠️  This will %s for the current project.\n", action)
		fmt.Print("Are you sure you want to continue? (y/N): ")

		var response string
		_, err := fmt.Scanln(&response)
		if err != nil {
			fmt.Printf("failed to read input: %v\n", err)
		}

		if response != "y" && response != "Y" && response != "yes" && response != "YES" {
			fmt.Println("❌ Operation cancelled")
			return nil
		}
	}

	fmt.Println("🧹 Cleaning up Docker resources...")

	// Clean up Flink jobs if requested or if cleaning all
	if cleanFlink || cleanAll {
		fmt.Println("🔥 Cancelling running Flink jobs...")
		ctx := context.Background()
		flinkURL := viper.GetString("flink_url")
		if err := pipeline.CancelAllRunningJobs(ctx, flinkURL); err != nil {
			fmt.Printf("⚠️  Warning: Failed to cancel Flink jobs: %v\n", err)
		}
	}

	// Stop and remove containers, networks
	if err := deployer.StopStack(); err != nil {
		if err := deployer.CheckDockerAvailability(); err != nil {
			return fmt.Errorf("docker is not available: %w", err)
		}
	}

	// Remove volumes if requested
	if cleanVolumes || cleanAll {
		fmt.Println("🗑️  Removing Docker volumes...")
		if err := deployer.RemoveVolumes(); err != nil {
			fmt.Printf("⚠️  Failed to remove some volumes: %v\n", err)
		}
	}

	// Remove images if requested
	if cleanAll {
		fmt.Println("🗑️  Removing Docker images...")
		if err := deployer.RemoveImages(); err != nil {
			fmt.Printf("⚠️  Failed to remove some images: %v\n", err)
		}
	}

	// Clean up orphaned containers
	fmt.Println("🧹 Cleaning up orphaned containers...")
	if err := deployer.CleanupOrphans(); err != nil {
		fmt.Printf("⚠️  Failed to clean up orphans: %v\n", err)
	}

	fmt.Println("✅ Cleanup completed successfully!")

	return nil
}
