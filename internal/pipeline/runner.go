package pipeline

import (
	"context"
	"fmt"
	"time"
)

// Config holds the configuration for pipeline execution
type Config struct {
	ProjectDir        string
	MessageRate       int
	Duration         time.Duration
	Cleanup          bool
	DryRun           bool
	BootstrapServers string
	FlinkURL         string
	SchemaRegistryURL string
	LocalMode        bool
}

// Runner orchestrates the complete pipeline execution
type Runner struct {
	config        *Config
	resourceMgr   *ResourceManager
	producer      *Producer
	consumer      *Consumer
	flinkDeployer *FlinkDeployer
	sqlLoader     *SQLLoader
	schemaLoader  *SchemaLoader
}

// NewRunner creates a new pipeline runner
func NewRunner(config *Config) (*Runner, error) {
	resourceMgr := NewResourceManager(config)
	
	producer, err := NewProducer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	consumer, err := NewConsumer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	flinkDeployer := NewFlinkDeployer(config)
	sqlLoader := NewSQLLoader(config.ProjectDir)
	schemaLoader := NewSchemaLoader(config.ProjectDir)

	return &Runner{
		config:        config,
		resourceMgr:   resourceMgr,
		producer:      producer,
		consumer:      consumer,
		flinkDeployer: flinkDeployer,
		sqlLoader:     sqlLoader,
		schemaLoader:  schemaLoader,
	}, nil
}

// Run executes the complete pipeline
func (r *Runner) Run(ctx context.Context) error {
	fmt.Println("🔄 Starting pipeline execution...")

	// Step 1: Load SQL statements
	fmt.Println("📖 Loading SQL statements...")
	sqlStatements, err := r.sqlLoader.LoadStatements()
	if err != nil {
		return fmt.Errorf("failed to load SQL statements: %w", err)
	}
	fmt.Printf("✅ Loaded %d SQL statements\n", len(sqlStatements))

	// Step 2: Load AVRO schemas
	fmt.Println("📋 Loading AVRO schemas...")
	schemas, err := r.schemaLoader.LoadSchemas()
	if err != nil {
		return fmt.Errorf("failed to load schemas: %w", err)
	}
	fmt.Printf("✅ Loaded %d AVRO schemas\n", len(schemas))

	// Step 3: Generate dynamic resource names
	fmt.Println("🏷️  Generating dynamic resource names...")
	resources, err := r.resourceMgr.GenerateResources()
	if err != nil {
		return fmt.Errorf("failed to generate resources: %w", err)
	}
	fmt.Printf("✅ Generated resources with prefix: %s\n", resources.Prefix)

	// Step 4: Create Kafka topics
	fmt.Println("📝 Creating Kafka topics...")
	if err := r.resourceMgr.CreateTopics(ctx, resources); err != nil {
		return fmt.Errorf("failed to create topics: %w", err)
	}
	fmt.Printf("✅ Created %d Kafka topics\n", len(resources.Topics))

	// Step 5: Register AVRO schemas
	fmt.Println("📋 Registering AVRO schemas...")
	if err := r.resourceMgr.RegisterSchemas(ctx, resources, schemas); err != nil {
		return fmt.Errorf("failed to register schemas: %w", err)
	}
	fmt.Printf("✅ Registered %d schemas\n", len(schemas))

	// Step 6: Deploy FlinkSQL statements
	fmt.Println("⚡ Deploying FlinkSQL statements...")
	deploymentIDs, err := r.flinkDeployer.Deploy(ctx, sqlStatements, resources)
	if err != nil {
		return fmt.Errorf("failed to deploy FlinkSQL: %w", err)
	}
	fmt.Printf("✅ Deployed %d FlinkSQL statements\n", len(deploymentIDs))

	// Step 7: Start consumer (before producer to avoid missing messages)
	fmt.Println("👂 Starting Kafka consumer...")
	consumerCtx, cancelConsumer := context.WithCancel(ctx)
	defer cancelConsumer()
	
	consumerDone := make(chan error, 1)
	go func() {
		consumerDone <- r.consumer.Start(consumerCtx, resources.OutputTopic)
	}()

	// Step 8: Start producer
	fmt.Println("📤 Starting Kafka producer...")
	producerCtx, cancelProducer := context.WithTimeout(ctx, r.config.Duration)
	defer cancelProducer()

	producerDone := make(chan error, 1)
	go func() {
		producerDone <- r.producer.Start(producerCtx, resources.InputTopic, schemas["input"])
	}()

	// Step 9: Monitor execution
	fmt.Printf("⏱️  Running pipeline for %v...\n", r.config.Duration)
	
	select {
	case <-time.After(r.config.Duration):
		fmt.Println("⏰ Pipeline duration reached")
	case err := <-producerDone:
		if err != nil {
			return fmt.Errorf("producer failed: %w", err)
		}
		fmt.Println("✅ Producer completed")
	case err := <-consumerDone:
		if err != nil {
			return fmt.Errorf("consumer failed: %w", err)
		}
		fmt.Println("✅ Consumer completed")
	case <-ctx.Done():
		fmt.Println("🛑 Pipeline cancelled")
	}

	// Step 10: Cleanup if enabled
	if r.config.Cleanup {
		fmt.Println("🧹 Cleaning up resources...")
		if err := r.cleanup(ctx, resources, deploymentIDs); err != nil {
			fmt.Printf("⚠️  Warning: cleanup failed: %v\n", err)
		} else {
			fmt.Println("✅ Cleanup completed")
		}
	}

	return nil
}

// cleanup removes all created resources
func (r *Runner) cleanup(ctx context.Context, resources *Resources, deploymentIDs []string) error {
	// Stop FlinkSQL deployments
	if err := r.flinkDeployer.Cleanup(ctx, deploymentIDs); err != nil {
		return fmt.Errorf("failed to cleanup FlinkSQL deployments: %w", err)
	}

	// Delete Kafka topics
	if err := r.resourceMgr.DeleteTopics(ctx, resources); err != nil {
		return fmt.Errorf("failed to delete topics: %w", err)
	}

	return nil
}
