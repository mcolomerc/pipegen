package docker

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// ServiceCheck represents a service health check configuration
type ServiceCheck struct {
	Name string
	URL  string
	Type string // "http", "kafka", "tcp"
}

// ServiceWaiter handles waiting for services to be ready
type ServiceWaiter struct {
	services []ServiceCheck
}

// NewServiceWaiter creates a new service waiter
func NewServiceWaiter(services []ServiceCheck) *ServiceWaiter {
	return &ServiceWaiter{
		services: services,
	}
}

// WaitForAll waits for all services to be ready
func (w *ServiceWaiter) WaitForAll(ctx context.Context) error {
	for _, service := range w.services {
		fmt.Printf("⏳ Waiting for %s to be ready...\n", service.Name)

		if err := w.waitForService(ctx, service); err != nil {
			return fmt.Errorf("service %s failed to start: %w", service.Name, err)
		}

		fmt.Printf("✅ %s is ready\n", service.Name)
	}

	return nil
}

// waitForService waits for a single service to be ready
func (w *ServiceWaiter) waitForService(ctx context.Context, service ServiceCheck) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			ready, err := w.checkService(service)
			if err != nil {
				// Log error but continue retrying
				fmt.Printf("    %s check failed: %v\n", service.Name, err)
				continue
			}
			if ready {
				return nil
			}
		}
	}
}

// checkService performs a health check for a single service
func (w *ServiceWaiter) checkService(service ServiceCheck) (bool, error) {
	switch service.Type {
	case "http":
		return w.checkHTTP(service.URL)
	case "kafka":
		return w.checkKafka(service.URL)
	case "tcp":
		return w.checkTCP(service.URL)
	default:
		return false, fmt.Errorf("unknown service type: %s", service.Type)
	}
}

// checkHTTP checks if an HTTP service is responding
func (w *ServiceWaiter) checkHTTP(url string) (bool, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 400, nil
}

// checkKafka checks if a Kafka broker is responding
func (w *ServiceWaiter) checkKafka(address string) (bool, error) {
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	// For a more thorough check, we could try to create a Kafka admin client
	// and list topics, but a simple TCP connection is sufficient for startup
	return true, nil
}

// checkTCP checks if a TCP service is accepting connections
func (w *ServiceWaiter) checkTCP(address string) (bool, error) {
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return false, err
	}
	defer conn.Close()
	return true, nil
}
