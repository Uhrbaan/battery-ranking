package capacity

import (
	"context"
	"encoding/json"
	"log"
	"strconv"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jochenvg/go-udev"
)

type data struct {
	DisplayName string `json:"display"`
	Percentage  int    `json:"percentage"`
}

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
			capacity, err := strconv.Atoi(event.SysattrValue("capacity"))
			status := event.SysattrValue("status")

			if err != nil {
				jsonData, _ := json.Marshal(data{
					DisplayName: service.Unit,
					Percentage:  capacity,
				})
				client.Publish(service.Status, 1, false, jsonData)
				client.Publish(service.Event, 1, false, status)
			}
			// fmt.Println("Event:", event.Syspath(), event.Action())

		case err, ok := <-errors:
			if !ok {
				return
			}

			log.Fatal(err)
		}
	}
}
