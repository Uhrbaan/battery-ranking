package process

import (
	"context"
	"encoding/json"
	"log"
	"strconv"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jochenvg/go-udev"
)

type CapacityService struct {
	Unit   string
	Status string
	Event  string
}

func (service CapacityService) Start(opts *mqtt.ClientOptions) {
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal("Could not establish connection with MQTT server: ", token.Error())
	}

	// Create the `udev` environment to get the notified when the battery percentage changes.
	u := udev.Udev{}
	m := u.NewMonitorFromNetlink("udev")
	m.FilterAddMatchSubsystem("power_supply")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create the channel that notifies us when the power capacity changes
	devices, errors, err := m.DeviceChan(ctx)
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case event, ok := <-devices:
			if !ok {
				return
			}
			if event.Action() != "change" {
				continue
			}

			properties := event.Properties()
			capacity, err := strconv.Atoi(properties["POWER_SUPPLY_CAPACITY"])
			status := properties["POWER_SUPPLY_STATUS"]

			if err != nil {
				log.Println("[CapacityService] Could not convert the power supply capacity to int.")
				continue
			}

			jsonData, _ := json.Marshal(dataStore{
				DisplayName: service.Unit,
				Percentage:  capacity,
			})

			client.Publish(service.Status, 1, false, string(jsonData))
			log.Println("[CapacityService] Published", string(jsonData), "on topic", service.Status)
			client.Publish(service.Event, 1, false, status)
			log.Println("[CapacityService] Published", status, "on topic", service.Event)

		case err, ok := <-errors:
			if !ok {
				return
			}

			log.Fatal(err)
		}
	}
}
