#import "project-template.typ": template 

// TODO: change date and abstract
#show: template.with(
  title: [Start-Semester project],
  author: [Léonard #smallcaps[Clément]],
  abstract: [
    This project details the design and implementation of a system for monitoring the battery status of Linux machines. Implemented in Go, the system utilizes an event-driven architecture based on the MQTT (Mosquitto) protocol to ensure decoupled communication. The solution is structured around three core processes: the _CapacityService_ (Sensor) which polls local battery capacity via `sysfs`; the _AggregateService_ (Mediator) which collects, and processes data; and the _ShowService_ (Actuator) which displays a ranked list of computers by lowest battery level. The project demonstrated battery ranking across multiple devices, achieving the goal of learning event-driven design principles and Go concurrency.
  ]
)

= Idea
This project allows Linux machines to connect to a Mosquitto network and share their battery capacity. 
The program will then display which machines have the lowest battery level.
Such a system could be expanded to monitor several aspects of the computers, helping for example an IT team to know how and when to service the machines.

= Design
#figure(
  image("schema.svg"),
  caption: [General data flow between the processes.]
) <general-dataflow>

@general-dataflow shows a schema of the overall design of the project.
It is divided into three processes. 
The first one, _CapacityService_ is tasked to continuously check for a change in the laptop's battery capacity.

The second, _AggregateService_ listens for outgoing messages from _CapacityService_ and aggregates the messages coming from different computers and stores their respective battery capacity. 
It will then publish a message containing all the computer identifiers and their respective battery levels. 
The service serves a role of *mediator*.
It processes data from a first service, and sends the result to the next one.

The last service, _ShowService_ receives the full list of computers with their battery percentage and displays it to a console.

All the communications go through the _Mosquitto broker_, which acts like a central message hub to decouple the services.

= Implementation
The project is implemented as a single go project, where each service runs inside its own #link("https://go.dev/tour/concurrency/1")[goroutine].
This allows us to run multiple services at once with the comfort of running a single program. 

The implementation of the processes follows the DUISE approach, where each process presents a
- *Documentation*, which describes the shape the service expects, 
- *Unit*, which is an identifier specific to the process, 
- *Intent*, an ingoing topic which the process listens to, 
- *Status*, which is an outgoing topic that describes the state of the process or some data, 
- *Event*, an outgoing event used to describe physical events. 

The processes are implemented inside the `process` module.
They provide ```go struct``` containing the topics and data needed for the service.
Each process also provides a `Start()` method that connects the process to the Mosquitto broker and starts the service.

A process also may or may not define a ```go type``` with #link("https://go.dev/wiki/Well-known-struct-tags")[struct tags] to define what the shape of the `JSON` data the service expects. 
This corresponds to the Documentation part of the DUISE approach.

@general-dataflow shows how the data flows between the processes. 
Since we are using #link("https://mosquitto.org/")[Mosquitto], processes will publish their messages to a specific _topic_ which other processes can subscribe to get their message. 
@general-dataflow shows only the last part of these topics. 
The topics in use are listed here: 
```go 
const (
	rootTopic = "example.com/sin.04028/battery-ranking"
	batteryTopic     = rootTopic + "/sensor/capacity/status/battery"
	sensorDeathTopic = rootTopic + "/sensor/capacity/event/death"
	aggregateTopic   = rootTopic + "/mediator/aggregate/status/aggregate"
)
```

_CapacityService_ is a sensor#footnote[A sensor gets data from the physical world.] that polls the battery capacity through _syfs_ (see @capacity-service for more details) and sends it to the mosquitto broker under the `status/battery` topic. 

The mediator#footnote[A mediator serves as a midpoint between two processes. It fetches data, processes it and sends the result to the next process.] _AggregateService_ then fetches the data from `status/battery` and aggregates the data to previous data received from other sensors into a single dictionary and publishes it to `status/aggregate`.

Finally, _ShowService_, and actuator#footnote[An actuator is a process that acts upon the physical world, in our case printing text to the screen.] gets the message from `status/aggregate` and simply prints it to the console.

Implementation details of each service follow in their corresponding subsections.

== CapacityService <capacity-service>
When the _CapacityService_ starts, it launches a separate process, ```go PollBattery()``` or ```go SimulateBattery()``` based on what the user sets, as a goroutine that will poll the battery level of the computer every 30 seconds, or simulate the battery changes every 10 seconds respectively.

Checking for the battery capacity works by using the `sysfs` pseudo-filesystem which provides an interface to kernel data structures @sysfs-5 which represent physical devices like the battery.
This enables us to query the battery level of the device simply by reading the virtual file at `/sys/class/power_supply/BAT0/capacity` with a simple ```go os.ReadFile()```.

Once the data is received, it can be sent according to the documentation of the _AggregateService_:
```go  
type dataAggregate struct {
    DisplayName string `json:"display"`
    Percentage  int    `json:"percentage"`
}
```
This means that the service will send its `DisplayName`, which is just its `Unit` field, and the capacity as `Percentage`. 
Once the data is packaged into the data structure, it is converted to JSON and published. 
This is done with: 
```go  
jsonData, _ := json.Marshal(dataAggregate{
    DisplayName: service.Unit,
    Percentage:  capacity,
  })

client.Publish(service.Status, 1, false, string(jsonData))
```

== AggregateService
When the _AggregateService_ receives data from the `status/battery` as a JSON and converts it back to the `dataAggregate` format with ```go json.Unmarshal()```.

The service will then store the data into a dictionary, called a ```go map``` in go, with the `DisplayName` as a key and the `Percentage` as a value.
In code, this looks like the following: 
```go  
service.readings[data.DisplayName] = data.Percentage
```
This ensures that each _CapacityService_ process gets stored exactly once, since a ```go map``` cannot contain duplicate keys.

Once the message has been stored, the ```go map``` is converted to JSON and published, similarly to how it was done in @capacity-service.

The second topic _AggregateService_ listens to is `event/death`.
_CapacityService_ registers a Last Will message on the `event/death` topic before connecting to the broker. 
If the client disconnects unexpectedly, the Mosquitto broker automatically publishes the message (the service's `DisplayName`) to signal its removal from the network.
When the message is received, it means that the computer no longer exists on the network, so its battery level and name should be removed from the ranking.
The removal is done with:
```go 
delete(service.readings, string(displayName))
```

== _ShowService_
The final service gets its data from `status/aggregate``` and converts it back to a ```go map[string]int```.
It then sorts the the map into the `keys` list of `DisplayName`s according to the battery level of the device in an ascending order, and the prints it to the console with:
```go  
for i, key := range keys {
    fmt.Printf("%2d %s @ %d%%\n", i+1, key, data[key])
}
```
Which prints the rank, the `DisplayName` and the battery level of the devices.

= Result
A video showing the program running on two different laptops can be viewed at #link("https://kdrive.infomaniak.com/app/share/1618622/4811f6b8-1228-4b97-8b5c-7e84efd27b2c").
_Please note that the version of the program running is slightly older. Since then, the `-store` option has been renamed to `-aggregate`._

The video shows a terminal with two different tabs open. 
The pink (right) tab is an ssh session to a second laptop, which we named `lenovo` with the `-display` option, 
and the regular (left) tab shows the current computer names `slimbook`. 
The program is correctly reading the battery levels of both computers, and printing them in the correct order to the console.

== Limitations and future work
=== Death of _AggregateService_ and _ShowService_
One of the limitations of the project happens when either the _AggregateService_ or the _ShowService_ exits unexpectedly.

If the first of the mentioned services were to die, then _ShowService_ would not have any data to show and _CapacityService_ would continue running while being unable to share their data, wasting resources. 

To solve this, both services could listen to _AggregateService_'s last will and react accordingly if it were to be published. 
_ShowService_ could simply show an error message informing the IT service what happened, and _CapacityService_ could enter a low-power mode where it stops reading and sending data until it is instructed otherwise. 

If _ShowService_ were to close, then its last will could also instruct the other services to enter a low-power mode. 

=== Running multiple _AggregateServices_
Another important limitation is the unexpected behavior produced when multiple _AggregateServices_ run at the same time. 
Since each service connects to all the _CapacityServices_, all of them will send the same data to the same _ShowServices_, showing redundant information.

This could be fixed by creating a filter in the _ShowServices_, which could, for example, choose to show only the messages coming from the oldest _AggregateService_.

=== Hard dependency on the Linux kernel 
_CapacityService_ is dependent on the Linux operating system and its `sysfs` pseudo-filesystem interface, which provides battery data at `/sys/class/power_supply/BAT0/capacity`. 
This limitation makes it impossible to run the program on non-linux devices.

= Reasoning
The goal of this project is to learn how to design and implement a project in an event-driven architecture, more specifically using the standard #link("https://mosquitto.org/")[Mosquitto] protocol developed by the eclipse foundation. 

Implementing three services, a sensor, a mediator and an actuator allowed to get a greater understanding of what a complex, multiprocess system could look like.

The secondary and personal goal of this project is to learn the #link("https://go.dev/")[Go] programming language and its concurrency model.

Using goroutines and channels to communicate between processes in the _CapacityService_ taught me the basics of concurrency in this programming language.

= Conclusion
The development of this battery-ranking system achieved its objective: the design and implementation of an event-driven architecture using the standard MQTT protocol. 

While the current implementation achieved its goal, limitations such as the Single Point of Failure in the _AggregateService_ leave room for future development, especially to improve robustness and implement strategies in case of failure.

#bibliography("bibliography.bib")
