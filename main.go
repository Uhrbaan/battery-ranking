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

	"github.com/google/uuid"
)

const (
	brokerURL = "tcp://broker.emqx.io:1883"
	// brokerURL = "tcp://localhost:1883"
)

var (
	display = flag.String("display", uuid.NewString(), "Will be the identifier of your computer")
)

func main() {
	flag.Parse()
}
