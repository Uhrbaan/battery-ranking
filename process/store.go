package process

import (
	"encoding/json"
	"log"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Structure the json payload from intent topic must follow.
type dataStore struct {
	DisplayName string `json:"display"`
	Percentage  int    `json:"percentage"`
}

// The first intent is for the connection to the capacity services
// The second intent is for the event of a disconnection of a capacity service.
type StoreService struct {
	Unit   string
	Intent [2]string
	Status string
}

func (service StoreService) Start(opts *mqtt.ClientOptions, quit chan os.Signal) {
	// Storage of the different devices.
	latestReading := make(map[string]int)

	// Connection to the Mosquitto client.
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal("Could not establish connection with MQTT server: ", token.Error())
	}
	log.Println("[StoreService] Connected to the MQTT server.")

	// Subscription to the Intent topic.
	if token := client.Subscribe(service.Intent[0], 1, func(client mqtt.Client, message mqtt.Message) {
		// Getting the data from Intent
		msg := message.Payload()
		log.Println("[StoreService] got message", string(msg), "on topic", service.Intent)
		var data dataStore
		err := json.Unmarshal(msg, &data)
		if err != nil {
			log.Println("[StoreService] Could not unmarshal json object", data)
			return
		}

		// Updating the stored data with the new data
		if data.DisplayName != "" {
			latestReading[data.DisplayName] = data.Percentage
		}
		log.Println("[StoreService] currently stored values:", latestReading)

		// Publishing the new data
		jsonData, _ := json.Marshal(latestReading)
		client.Publish(service.Status, 1, false, string(jsonData))
		log.Println("[StoreService] Published message", string(jsonData), "to topic", service.Status)
	}); token.Wait() && token.Error() != nil {
		log.Fatalf("Store service %s failed to subscribe to %s: %v", service.Unit, service.Intent, token.Error())
	}
	log.Println("[StoreService] Subscribed to", service.Intent, "successfully.")

	if token := client.Subscribe(service.Intent[1], 1, func(client mqtt.Client, message mqtt.Message) {
		// Getting the data from Intent
		msg := message.Payload()
		log.Println("[StoreService] got LastWill", string(msg), "on topic", service.Intent)

		// The message is supposed to contain the unit of the service that died.
		delete(latestReading, string(msg))
	}); token.Wait() && token.Error() != nil {
		log.Fatalf("Store service %s failed to subscribe to %s: %v", service.Unit, service.Intent, token.Error())
	}

	// Graceful shutdown when the process is interupted.
	<-quit
	log.Println("[StoreService] Disconnecting from mqtt client. Quitting.")
	client.Disconnect(0)
}
