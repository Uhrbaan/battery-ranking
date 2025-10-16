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
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"example.com/sin.04028/project1/process"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
)

const (
	brokerURL = "tcp://test.mosquitto.org:1883"
	rootTopic = "example.com/sin.04028/battery-ranking"

	batteryTopic     = rootTopic + "/sensor/capacity/status/battery"
	sensorDeathTopic = rootTopic + "/sensor/capacity/event/death"
	aggregateTopic   = rootTopic + "/mediator/aggregate/status/aggregate"
)

var (
	id = uuid.NewString()

	// setting up variables to parse command-line arguments.
	broker           = flag.String("broker", brokerURL, "Custom broker url. Should be shaped like \"tcp://broker.emqx.io:1883\"")
	displayName      = flag.String("display", "capacity-"+id, "The displayed name of your computer.")
	capacityService  = flag.Bool("capacity", false, "Use this flag to start the capacity service.")
	simulateBattery  = flag.Bool("simulate", false, "Use this flag to simulate the battery capacity instead of reading it.")
	aggregateService = flag.Bool("aggregate", false, "Use this flag to start the storage/aggregation service.")
	showService      = flag.Bool("show", false, "Use this flag to start the show service, which will display the ranking in your terminal.")
	allServices      = flag.Bool("all", false, "Use this flag to start all the services.")
	verbose          = flag.Bool("v", false, "Enable logging")
)

func main() {
	flag.Parse()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	if !*verbose {
		log.SetOutput(io.Discard)
	}

	if *capacityService || *allServices {
		log.Println("Starting CapacityService")
		cfg := mqtt.NewClientOptions()
		cfg.AddBroker(*broker)
		cfg.SetClientID("capacity-" + id)
		cfg.SetWill(sensorDeathTopic, *displayName, 1, false)

		service := process.CapacityService{
			Unit:   *displayName,
			Status: batteryTopic,
		}

		if *simulateBattery {
			service.BatteryProvider = process.SimulateBattery
		} else {
			service.BatteryProvider = process.PollBattery
		}

		go service.Start(cfg, quit)
	}

	if *aggregateService || *allServices {
		log.Println("Starting StoreService")
		storeCfg := mqtt.NewClientOptions()
		storeCfg.AddBroker(*broker)
		storeCfg.SetClientID("store-" + id)
		go process.AggregateService{
			Unit:   storeCfg.ClientID,
			Intent: [2]string{batteryTopic, sensorDeathTopic},
			Status: aggregateTopic,
		}.Start(storeCfg, quit)
	}

	if *showService || *allServices {
		log.Println("Starting ShowService")
		showCfg := mqtt.NewClientOptions()
		showCfg.AddBroker(*broker)
		showCfg.SetClientID("show-" + id)
		go process.ShowService{
			Unit:   showCfg.ClientID,
			Intent: aggregateTopic,
		}.Start(showCfg, quit)
	}

	log.Println("Services are runnning...")
	<-quit // block until quit signal
	log.Println("Recieved quit signal. Exiting.")
}
