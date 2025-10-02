package process

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Service interface {
	Start(opts *mqtt.ClientOptions) // starts the service
	Description() string            // gives back a description (json template) of the data this service expects
	Unit() string                   // gives the identity of the service
	Intent() string                 // gives back the topic of the intent the services listens to
	Status() string                 // gives back the topic where another service can get the status at
	Event() string                  // gives back the topic where another service can get the events at
}
