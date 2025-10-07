package process

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const batPath = "/sys/class/power_supply/BAT0/capacity"

type CapacityService struct {
	Unit   string
	Status string
}

// Function polling the battery at batPath
func pollBattery(ctx context.Context, capacityCh chan<- int, errCh chan<- error) {
	previousCapacity := 0
	log.Println("[pollBattery] Starting the poll...")

	// Using ticker instead of time to make the loop non-blocking (good concurrency practices)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		// Gets notified if the context has been canceled (stopped)
		case <-ctx.Done():
			log.Println("[pollBattery] Context canceled. Shutting down.")
			return

		// Gets notified when the timer reaches 0 (every 30s)
		case <-ticker.C:
			// Checking the file corresponding to the battery capacity
			content, err := os.ReadFile(batPath)
			if err != nil {
				errCh <- err
				continue
			}

			capacity, err := strconv.Atoi(strings.TrimSpace(string(content)))
			if err != nil {
				errCh <- err
				continue
			}
			log.Println("[pollBattery] previous capacity:", previousCapacity, "\tcapacity:", capacity)

			// Sending the capacity back to the Start function through a channel.
			if capacity != previousCapacity {
				capacityCh <- capacity
			}

			previousCapacity = capacity
		}
	}
}

func (service CapacityService) Start(opts *mqtt.ClientOptions, quit chan os.Signal) {
	// Connecting to the Mosquitto client
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal("Could not establish connection with MQTT server: ", token.Error())
	}
	log.Println("[CapacityService] Connected to the MQTT server.")

	// Starting the polling function in the background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	capacityCh := make(chan int)
	errCh := make(chan error)

	go pollBattery(ctx, capacityCh, errCh)

	// Listning for the responses from the polling function
	for {
		select {
		// Listening for notifications from the polling function
		case capacity, ok := <-capacityCh:
			if !ok {
				return
			}

			jsonData, _ := json.Marshal(dataStore{
				DisplayName: service.Unit,
				Percentage:  capacity,
			})

			client.Publish(service.Status, 1, false, string(jsonData))
			log.Println("[CapacityService] Published", string(jsonData), "on topic", service.Status)

		// Listening for potential errors by the polling function
		case err, ok := <-errCh:
			if !ok {
				return
			}

			log.Fatal(err)

		// Listening quit signal produced when interrupting the program for graceful shutdown.
		case <-quit:
			log.Println("[CapacityService] Disconnecting mqtt clien. Quitting.")
			client.Disconnect(0)
			return
		}
	}
}
