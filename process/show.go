package process

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type dataShow map[string]int

type ShowService struct {
	Unit   string
	Intent string
}

func (service ShowService) Start(opts *mqtt.ClientOptions) {
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal("Could not establish connection with MQTT server: ", token.Error())
	}

	if token := client.Subscribe(service.Intent, 1, func(client mqtt.Client, message mqtt.Message) {
		msg := message.Payload()
		log.Println("[ShowService] Got message", string(msg), "from topic", service.Intent)
		var data dataShow
		json.Unmarshal(msg, &data)

		keys := make([]string, 0, len(data))
		for k := range data {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			return data[keys[i]] < data[keys[j]]
		})

		fmt.Println("Who has the lowest battery ?\n===========================")
		for i, key := range keys {
			fmt.Printf("%2d %s @ %d%%\n", i+1, key, data[key])
		}
	}); token.Wait() && token.Error() != nil {
		log.Fatalf("Store service %s failed to subscribe to %s: %v", service.Unit, service.Intent, token.Error())
	}
	log.Println("[ShowService] Subscribed to", service.Intent, "successfully.")
}
