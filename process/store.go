package process

import (
	"encoding/json"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type dataStore struct {
	DisplayName string `json:"display"`
	Percentage  int    `json:"percentage"`
}

type StoreService struct {
	Unit   string
	Intent string
	Status string
}

func (service StoreService) Start(opts *mqtt.ClientOptions) {
	latestReading := make(map[string]int)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal("Could not establish connection with MQTT server: ", token.Error())
	}

	if token := client.Subscribe(service.Intent, 1, func(client mqtt.Client, message mqtt.Message) {
		msg := message.Payload()
		log.Println("[StoreService] got message", string(msg), "on topic", service.Intent)
		var data dataStore
		err := json.Unmarshal(msg, &data)
		if err != nil {
			log.Println("[StoreService] Could not unmarshal json object", data)
			return
		}

		if data.DisplayName != "" {
			latestReading[data.DisplayName] = data.Percentage
		}
		log.Println("[StoreService] currently stored values:", latestReading)

		jsonData, _ := json.Marshal(latestReading)
		client.Publish(service.Status, 1, false, string(jsonData))
		log.Println("[StoreService] Published message", string(jsonData), "to topic", service.Status)
	}); token.Wait() && token.Error() != nil {
		log.Fatalf("Store service %s failed to subscribe to %s: %v", service.Unit, service.Intent, token.Error())
	}
	log.Println("[StoreService] Subscribed to", service.Intent, "successfully.")
}
