package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"pipegen/internal/docker"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy local streaming pipeline stack",
	Long: `Deploy starts a local streaming pipeline stack using Docker Compose:
- Kafka broker in KRaft mode (single node, replication factor 1)
- Zookeeper (for Kafka metadata)
- Flink Job Manager and Task Manager
- Schema Registry (optional)
- Creates topics and registers schemas
- Deploys FlinkSQL jobs`,
	RunE: runDeploy,
}

func init() {
	rootCmd.AddCommand(deployCmd)
	deployCmd.Flags().String("project-dir", ".", "Project directory path")
	deployCmd.Flags().Bool("with-schema-registry", true, "Deploy with Schema Registry")
	deployCmd.Flags().Bool("detach", true, "Run containers in detached mode")
	deployCmd.Flags().Duration("startup-timeout", 120*time.Second, "Timeout for stack startup")
	deployCmd.Flags().Bool("clean", false, "Clean existing containers before deploying")
}

func runDeploy(cmd *cobra.Command, args []string) error {
	projectDir, _ := cmd.Flags().GetString("project-dir")
	withSchemaRegistry, _ := cmd.Flags().GetBool("with-schema-registry")
	detach, _ := cmd.Flags().GetBool("detach")
	startupTimeout, _ := cmd.Flags().GetDuration("startup-timeout")
	clean, _ := cmd.Flags().GetBool("clean")

	fmt.Println("🚀 Deploying local streaming pipeline stack...")

	// Check if project directory exists
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return fmt.Errorf("project directory does not exist: %s", projectDir)
	}

	// Check if docker-compose.yml exists, create if not
	composePath := filepath.Join(projectDir, "docker-compose.yml")
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		fmt.Println("📝 Creating docker-compose.yml...")
		if err := createDockerCompose(projectDir, withSchemaRegistry); err != nil {
			return fmt.Errorf("failed to create docker-compose.yml: %w", err)
		}
	}

	// Check if flink-conf.yaml exists, create if not
	flinkConfPath := filepath.Join(projectDir, "flink-conf.yaml")
	if _, err := os.Stat(flinkConfPath); os.IsNotExist(err) {
		fmt.Println("📝 Creating flink-conf.yaml...")
		if err := createFlinkConfig(projectDir); err != nil {
			return fmt.Errorf("failed to create flink-conf.yaml: %w", err)
		}
	}

	// Check if Docker is running
	if err := checkDockerRunning(); err != nil {
		return fmt.Errorf("docker is not running or not accessible: %w", err)
	}

	// Clean existing containers if requested
	if clean {
		fmt.Println("🧹 Cleaning existing containers...")
		if err := dockerComposeDown(projectDir); err != nil {
			fmt.Printf("⚠️  Warning: failed to clean containers: %v\n", err)
		}
	}

	// Start the stack
	fmt.Println("🔄 Starting Docker Compose stack...")
	if err := dockerComposeUp(projectDir, detach); err != nil {
		return fmt.Errorf("failed to start Docker Compose stack: %w", err)
	}

	// Wait for services to be ready
	fmt.Println("⏳ Waiting for services to be ready...")
	ctx, cancel := context.WithTimeout(context.Background(), startupTimeout)
	defer cancel()

	if err := waitForServices(ctx, withSchemaRegistry); err != nil {
		return fmt.Errorf("services failed to start within timeout: %w", err)
	}

	// Create topics and register schemas
	fmt.Println("📝 Setting up topics and schemas...")
	deployer := docker.NewStackDeployer(projectDir)
	if err := deployer.SetupTopicsAndSchemas(ctx, withSchemaRegistry); err != nil {
		return fmt.Errorf("failed to setup topics and schemas: %w", err)
	}

	// Deploy FlinkSQL jobs
	fmt.Println("⚡ Deploying FlinkSQL jobs...")
	if err := deployer.DeployFlinkJobs(ctx); err != nil {
		return fmt.Errorf("failed to deploy FlinkSQL jobs: %w", err)
	}

	fmt.Println("✅ Local streaming pipeline stack deployed successfully!")
	fmt.Println("\n📊 Services:")
	fmt.Println("  Kafka Broker:    localhost:9092")
	fmt.Println("  Flink WebUI:     http://localhost:8081")
	if withSchemaRegistry {
		fmt.Println("  Schema Registry: http://localhost:8082")
	}
	fmt.Println("\n🔧 Management commands:")
	fmt.Printf("  View logs:       docker-compose -f %s logs -f\n", composePath)
	fmt.Printf("  Stop stack:      docker-compose -f %s down\n", composePath)
	fmt.Printf("  Restart stack:   docker-compose -f %s restart\n", composePath)

	return nil
}

func createDockerCompose(projectDir string, withSchemaRegistry bool) error {
	composer := docker.NewDockerComposeGenerator()
	composeContent, err := composer.Generate(withSchemaRegistry)
	if err != nil {
		return err
	}

	composePath := filepath.Join(projectDir, "docker-compose.yml")
	return os.WriteFile(composePath, []byte(composeContent), 0644)
}

func checkDockerRunning() error {
	cmd := exec.Command("docker", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker command failed: %w", err)
	}
	return nil
}

func dockerComposeUp(projectDir string, detach bool) error {
	args := []string{"compose", "up"}
	if detach {
		args = append(args, "-d")
	}

	cmd := exec.Command("docker", args...)
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func dockerComposeDown(projectDir string) error {
	// Stop and remove containers using docker compose
	cmd := exec.Command("docker", "compose", "down", "-v", "--remove-orphans")
	cmd.Dir = projectDir
	if err := cmd.Run(); err != nil {
		// If compose fails, try to remove containers by name
		containerNames := []string{"pipegen-kafka", "pipegen-flink-jobmanager", "pipegen-flink-taskmanager", "pipegen-schema-registry"}
		for _, name := range containerNames {
			stopCmd := exec.Command("docker", "stop", name)
			_ = stopCmd.Run() // Ignore errors
			rmCmd := exec.Command("docker", "rm", name)
			_ = rmCmd.Run() // Ignore errors
		}
	}
	return nil
}

func waitForServices(ctx context.Context, withSchemaRegistry bool) error {
	services := []docker.ServiceCheck{
		{Name: "Kafka", URL: "localhost:9092", Type: "kafka"},
		{Name: "Flink Job Manager", URL: "http://localhost:8081", Type: "http"},
	}

	if withSchemaRegistry {
		services = append(services, docker.ServiceCheck{
			Name: "Schema Registry",
			URL:  "http://localhost:8082",
			Type: "http",
		})
	}

	waiter := docker.NewServiceWaiter(services)
	return waiter.WaitForAll(ctx)
}

func createFlinkConfig(projectDir string) error {
	flinkConfig := `# Flink configuration for local development
jobmanager.rpc.address: flink-jobmanager
jobmanager.rpc.port: 6123
jobmanager.heap.size: 1024m

taskmanager.numberOfTaskSlots: 2
taskmanager.memory.process.size: 1568m

parallelism.default: 1

# High Availability
high-availability: none

# Checkpointing
state.backend: filesystem
state.checkpoints.dir: file:///opt/flink/checkpoints
state.savepoints.dir: file:///opt/flink/savepoints

# Web UI
web.tmpdir: /tmp/flink-web-ui
web.upload.dir: /tmp/flink-web-upload

# Metrics
metrics.reporter.promgateway.class: org.apache.flink.metrics.prometheus.PrometheusReporter
`

	filePath := filepath.Join(projectDir, "flink-conf.yaml")
	return os.WriteFile(filePath, []byte(flinkConfig), 0644)
}
