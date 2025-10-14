package process

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const batDir = "/sys/class/power_supply/"

var batPath = "/sys/class/power_supply/BAT0/capacity"

type CapacityService struct {
	Unit            string
	Status          string
	BatteryProvider func(ctx context.Context, capacityCh chan<- int, errCh chan<- error)
}

// This function runs before the rest. It will make sure that the program picks up on a battery if there is any.
func init() {
	entries, err := os.ReadDir(batDir)
	if err != nil {
		log.Fatal("Device is not compatible:", err)
	}

	for _, entry := range entries {
		// The /sys/class/power_supply/ path is a virtual file system that contains information about kernel objects, specifically devices linked to power.
		// Usually, battery information is stored in a folder named BAT with the battery number.
		// On most laptops, this is either 0 or 1.
		if strings.Contains(entry.Name(), "BAT") {
			batPath = batDir + entry.Name() + "/capacity"
			log.Println("[CapacityService] Initializing the battery path to", batPath)
			return
		}
	}

	log.Fatal("[CapacityService] The program could not find a battery in", batDir)
}

// Function polling the battery at batPath
func PollBattery(ctx context.Context, capacityCh chan<- int, errCh chan<- error) {
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

func SimulateBattery(ctx context.Context, capacityCh chan<- int, errCh chan<- error) {
	capacity := rand.IntN(100 + 1)
	var capacityChange int
	capacityChange = rand.IntN(3+1)*2 - 3
	if capacityChange == 0 {
		capacityChange = 1
	}

	log.Println("[batterySimulation] Starting the simulation...")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Printf("[SimulateBattery] Simulating a computer starting at %d%% and gaining %d%% every 30 seconds.", capacity, capacityChange)

	for {
		select {
		case <-ctx.Done():
			log.Println("[batterySimulation]")
			return

		case <-ticker.C:
			capacity += capacityChange
			capacityCh <- capacity

			if capacity == 0 {
				errCh <- errors.New("simulating process death")
				return
			}

			// Stay at 100% if it reached it.
			if capacity >= 100 {
				capacityChange = 100
			}
		}
	}
}

func connectMqttClient(opts *mqtt.ClientOptions) (client mqtt.Client, err error) {
	client = mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		err = token.Error()
	}
	return
}

func (service CapacityService) Start(opts *mqtt.ClientOptions, quit chan os.Signal) {
	// Connecting to the Mosquitto client
	client, err := connectMqttClient(opts)
	if err != nil {
		log.Fatal("Could not establish connection with MQTT server: ", err)
	}
	log.Println("[CapacityService] Connected to the MQTT server.")

	// Starting the polling function in the background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	capacityCh := make(chan int)
	errCh := make(chan error)

	go service.BatteryProvider(ctx, capacityCh, errCh)

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
