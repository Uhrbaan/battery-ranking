package process

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// The shape of the json data from Intent.
type dataShow map[string]int

type ShowService struct {
	Unit   string
	Intent string
}

func (service ShowService) Start(opts *mqtt.ClientOptions, quit chan os.Signal) {
	// Connection to the Mosquitto client.
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal("Could not establish connection with MQTT server: ", token.Error())
	}
	log.Println("[ShowService] Connected to the MQTT server.")

	// Subscription to the intent
	if token := client.Subscribe(service.Intent, 1, func(client mqtt.Client, message mqtt.Message) {
		// Getting and processing the message from intent
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

		// Printing processed data to the console
		fmt.Println("Who has the lowest battery ?\n===========================")
		for i, key := range keys {
			fmt.Printf("%2d %s @ %d%%\n", i+1, key, data[key])
		}
	}); token.Wait() && token.Error() != nil {
		log.Fatalf("Store service %s failed to subscribe to %s: %v", service.Unit, service.Intent, token.Error())
	}
	log.Println("[ShowService] Subscribed to", service.Intent, "successfully.")

	// Blocking until the quit signal is detected for graceful shutdown.
	<-quit
	log.Println("[ShowService] Disconnecting mqtt clien. Quitting.")
	client.Disconnect(0)
}
