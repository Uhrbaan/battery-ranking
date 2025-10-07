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
	// brokerURL = "tcp://broker.emqx.io:1883"
	// brokerURL = "tcp://localhost:1883"
	brokerURL = "tcp://test.mosquitto.org:1883"
	rootTopic = "example.com/sin.04028/project1"
)

var (
	id              = uuid.NewString()
	broker          = flag.String("broker", brokerURL, "Custom broker url. Should be shaped like \"tcp://broker.emqx.io:1883\"")
	displayName     = flag.String("display", "capacity-"+id, "The displayed name of your computer.")
	capacityService = flag.Bool("capacity", false, "Use this flag to start the capacity service.")
	storeService    = flag.Bool("store", false, "Use this flag to start the storage/aggregation service.")
	showService     = flag.Bool("show", false, "Use this flag to start the show service, which will display the ranking in your terminal.")
	allServices     = flag.Bool("all", false, "Use this flag to start all the services.")
	verbose         = flag.Bool("v", false, "Enable logging")
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
		capacityCfg := mqtt.NewClientOptions()
		capacityCfg.AddBroker(*broker)
		capacityCfg.SetClientID("capacity-" + id)
		go process.CapacityService{
			Unit:   *displayName,
			Status: rootTopic + "/sensor/capacity/status",
		}.Start(capacityCfg, quit)
	}

	if *storeService || *allServices {
		log.Println("Starting StoreService")
		storeCfg := mqtt.NewClientOptions()
		storeCfg.AddBroker(*broker)
		storeCfg.SetClientID("store-" + id)
		go process.StoreService{
			Unit:   storeCfg.ClientID,
			Intent: rootTopic + "/sensor/capacity/status",
			Status: rootTopic + "/actuators/store/status",
		}.Start(storeCfg, quit)
	}

	if *showService || *allServices {
		log.Println("Starting ShowService")
		showCfg := mqtt.NewClientOptions()
		showCfg.AddBroker(*broker)
		showCfg.SetClientID("show-" + id)
		go process.ShowService{
			Unit:   showCfg.ClientID,
			Intent: rootTopic + "/actuators/store/status",
		}.Start(showCfg, quit)
	}

	log.Println("Services are runnning...")
	<-quit // block until quit signal
	log.Println("Recieved quit signal. Exiting.")
}
