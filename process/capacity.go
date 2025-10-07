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

func pollBattery(ctx context.Context, capacityCh chan<- int, errCh chan<- error) {
	previousCapacity := 0
	log.Println("[pollBattery] Starting the poll...")

	// instead of a blocking time.Sleep
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[pollBattery] Context canceled. Shutting down.")
			return

		case <-ticker.C:
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

			if capacity != previousCapacity {
				capacityCh <- capacity
			}

			previousCapacity = capacity
		}
	}
}

func (service CapacityService) Start(opts *mqtt.ClientOptions, quit chan os.Signal) {
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal("Could not establish connection with MQTT server: ", token.Error())
	}
	log.Println("[CapacityService] Connected to the MQTT server.")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	capacityCh := make(chan int)
	errCh := make(chan error)

	go pollBattery(ctx, capacityCh, errCh)

	for {
		select {
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

		case err, ok := <-errCh:
			if !ok {
				return
			}

			log.Fatal(err)

		case <-quit:
			log.Println("[CapacityService] Disconnecting mqtt clien. Quitting.")
			client.Disconnect(0)
			return
		}
	}
}
