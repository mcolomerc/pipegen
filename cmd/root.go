package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pipegen",
	Short: "A streaming pipeline generator for Kafka and FlinkSQL",
	Long: `PipeGen is a CLI tool for creating and managing streaming pipelines 
using Kafka and FlinkSQL. It provides:

- Project skeleton with SQL statements
- AVRO-based Kafka producer with configurable message rate
- Kafka consumer for output validation  
- FlinkSQL statement deployment via REST API
- Local Docker stack for development (default)
- Confluent Cloud support for production
- Dynamic resource creation to avoid naming conflicts`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.pipegen.yaml)")
	rootCmd.PersistentFlags().String("bootstrap-servers", "localhost:9092", "Kafka bootstrap servers")
	rootCmd.PersistentFlags().String("flink-url", "http://localhost:8081", "Flink Job Manager URL")
	rootCmd.PersistentFlags().String("schema-registry-url", "http://localhost:8082", "Schema Registry URL")
	rootCmd.PersistentFlags().Bool("local-mode", true, "Use local Docker stack (no authentication)")

	viper.BindPFlag("bootstrap_servers", rootCmd.PersistentFlags().Lookup("bootstrap-servers"))
	viper.BindPFlag("flink_url", rootCmd.PersistentFlags().Lookup("flink-url"))
	viper.BindPFlag("schema_registry_url", rootCmd.PersistentFlags().Lookup("schema-registry-url"))
	viper.BindPFlag("local_mode", rootCmd.PersistentFlags().Lookup("local-mode"))

	// Set defaults for local mode
	viper.SetDefault("local_mode", true)
	viper.SetDefault("bootstrap_servers", "localhost:9092")
	viper.SetDefault("schema_registry_url", "http://localhost:8082")
	viper.SetDefault("flink_url", "http://localhost:8081")
}

// initConfig reads in config file and ENV variables.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".pipegen")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
