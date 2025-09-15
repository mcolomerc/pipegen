#!/bin/bash
# Copy connectors from subdirectory to main lib for proper classpath loading
if [ -d "/opt/flink/lib/connectors" ]; then
    echo "Copying connector JARs to main lib directory..."
    cp /opt/flink/lib/connectors/*.jar /opt/flink/lib/
    echo "Copied $(ls /opt/flink/lib/connectors/*.jar | wc -l) connector JARs"
fi

# Execute the Flink command
if [ "$1" = "jobmanager" ]; then
    exec /docker-entrypoint.sh jobmanager
elif [ "$1" = "taskmanager" ]; then
    exec /docker-entrypoint.sh taskmanager
else
    # For other commands, just pass through
    exec "$@"
fi
