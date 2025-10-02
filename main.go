/*
Note that this projects uses libudev to notify when the battery capacity changes.
Install libudev (udev should already be installed) with the following command:

```sh
sudo apt-get install libudev-dev
```

This will not run if you do not have a linux device.
*/

package main

import (
	"flag"

	"example.com/sin.04028/project1/process"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
)

const (
	// brokerURL = "tcp://broker.emqx.io:1883"
	// brokerURL = "tcp://localhost:1883"
	brokerURL = "tcp://test.mosquitto.org:1883"
	rootTopic = "example.com/sin.04028/project1"
)

var (
	// help = flag.Bool("help", true, "Show this help message")
	// helpShort = flag.Bool("h", true, "Show this help message")
	id           = uuid.NewString()
	whichService = flag.String("service", "all", "Specify the service you would like to launch. Accepted values are capacity|store|show|all.")
	broker       = flag.String("broker", brokerURL, "Custom broker url. Should be shaped like \"tcp://broker.emqx.io:1883\"")
	deviceName   = flag.String("deviceName", "capacity-"+id, "Custom string to represent your computer.")
)

func main() {
	flag.Parse()

	if *whichService == "capacity" || *whichService == "all" {
		capacityCfg := mqtt.NewClientOptions()
		capacityCfg.AddBroker(*broker)
		capacityCfg.SetClientID("capacity-" + id)
		go process.CapacityService{
			Unit:   *deviceName,
			Status: rootTopic + "/sensor/capacity/status",
			Event:  rootTopic + "/sensor/capacity/event",
		}.Start(capacityCfg)
	}

	if *whichService == "store" || *whichService == "all" {
		storeCfg := mqtt.NewClientOptions()
		storeCfg.AddBroker(*broker)
		storeCfg.SetClientID("store-" + id)
		go process.StoreService{
			Unit:   storeCfg.ClientID,
			Intent: rootTopic + "/sensor/capacity/status",
			Status: rootTopic + "/actuators/store/status",
		}.Start(storeCfg)
	}

	if *whichService == "show" || *whichService == "all" {
		showCfg := mqtt.NewClientOptions()
		showCfg.AddBroker(*broker)
		showCfg.SetClientID("show-" + id)
		go process.ShowService{
			Unit:   showCfg.ClientID,
			Intent: rootTopic + "/actuators/store/status",
		}.Start(showCfg)
	}

	select {}
}
