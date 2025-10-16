#import "@preview/fletcher:0.5.8" as fletcher: diagram, node, edge
#import "@preview/typslides:1.2.6": *
#show: typslides.with(
  ratio: "4-3",
  theme: "bluey",
  font: "Noto Sans",
  link-style: "color",
)

#show raw.where(block: true): set text(size: 16pt)

#front-slide(
  title: "Battery ranking",
  subtitle: [Ranking battery levels of linux machines in an event-driven manner.],
  authors: "Léonard Clément",
  info: [#link("https://github.com/Uhrbaan/battery-ranking")],
)

#table-of-contents()

#slide[
  A system for monitoring the battery status of Linux machines implemented in Go and using an event-driven architecture.
]

#title-slide[Demonstration]

#slide()[
  + Install the go project:
    #grayed[```sh 
    sudo apt install golang # install go
    git clone https://github.com/Uhrbaan/battery-ranking.git # clone the repo
    cd battery-ranking
    go mod tidy # install go packages
    ```]
  
  + #block(sticky:true)[Run the project on real computers]
    #grayed[```sh
    # start the aggregation and visualization services.
    go run . -aggregate -show 

    # start the service reading battery percentage.
    go run . -capacity -display=computer -v
    ```]
  
  + #block(sticky: true)[Run the project as a simulation]
    #grayed[```sh 
    # run at least on aggregate and show service to see the result
    go run . -aggregate -show

    # run as many simulations as you want
    go run . -capacity -display=simulation1 -simulate -v
    ```]
]

#slide(title: [Video])[
  #link("https://kdrive.infomaniak.com/app/share/1618622/4811f6b8-1228-4b97-8b5c-7e84efd27b2c")#footnote[
    The version of the program shown in the program is slightly older. Replace `-store` with `-aggregate`.
  ]
]

#title-slide[Design]

#slide()[
  #figure(image("schema.svg"))

  / #stress[CapacityService]: Sensor which polls local battery capacity via `sysfs`.

  / #stress[AggregateService]: Mediator which collects, and processes data coming from _CapacityService_.

  / #stress[ShowService]: Actuator which displays a ranked list of computers by lowest battery level.

  / #stress[Mosquitto broker]: Central message hub through which all the services communicate.
]

#title-slide[
  Implementation 
]

#show enum: it => {block(sticky: true, it)}

#slide(title: [CapacityService])[
  + The #stress[CapacityService] works by starting the ```go PollBattery()``` process and sending data it receives from it to `status/battery`.
  
  #grayed[```go  
  // getting the data from PollBattery 
  case capacity, ok := <-capacityCh:
      jsonData, _ := json.Marshal(dataAggregate{
          DisplayName: service.Unit,
          Percentage:  capacity,
      })
      client.Publish(service.Status, 1, false, string(jsonData))
  ```]

  #pagebreak()
  #enum(start: 2, spacing: 1em)[The ```go PollBattery()``` uses #stress[`sysfs`], a virtual file system that represent Linux devices as files.
  ][Every thirty seconds, it reads the file and sends the capacity through the `capacityCh` channel]
  
  #grayed[```go  
    ticker := time.NewTicker(30 * time.Second)
    // ...
    case <-ticker.C:
        content, _ := os.ReadFile("/sys/class/power_supply/BAT0/capacity")
        capacity, _ := strconv.Atoi(strings.TrimSpace(string(content)))
        if capacity != previousCapacity {
            capacityCh <- capacity
        }
    ```]  
]

#slide(title: [AggregateService])[
  + The AggregateService starts by #stress[receiving data] from different CapacityServices on the `status/battery/` topic.
  + It then #stress[stores] the data into a ```go map``` along with the data from the other computers.
  + Finally, it #stress[publishes] the ```go map``` as JSON.
  
  #grayed[```go 
  var data dataAggregate
  json.Unmarshal(msg, &data)
  service.readings[data.DisplayName] = data.Percentage
  jsonData, _ := json.Marshal(service.readings)
  client.Publish(service.Status, 1, false, string(jsonData))
  ```]
]

#slide(title: [ShowService])[
  + The ShowService starts by #stress[receiving data] from AggregateService on the `status/aggregate` topic.
  + It then #stress[sorts] the `DeviceNames` into a list according to their battery level in ascending order.
  + Finally, it loops through that list and #stress[prints the data] to the console.
    
  #grayed[```go  
  var data map[string]int
  json.Unmarshal(msg, &data)
  for k := range data { keys = append(keys, k) }
  sort.Slice(keys, func(i, j int) bool {
      return data[keys[i]] < data[keys[j]]
  })
  for i, key := range keys {
			fmt.Printf("%2d %s @ %d%%\n", i+1, key, data[key])
  }
  ```]
]

#title-slide[Recovery]

#slide[
  - CapacityService registers a #stress[Last Will] message on the `event/death` topic before connecting to the broker. 

  - If the client #stress[disconnects unexpectedly], the Mosquitto broker #stress[automatically publishes] the message (the service's `DisplayName`)

  - AggregateService catches on the message on `event/death` and #stress[removes] the `DisplayName` from its list of devices. 

  #grayed[```go 
  delete(service.readings, string(displayName))
  ```]
]

#title-slide[Limitations & Improvements]

#slide(title: [Single Point of Failure: AggregateService])[
  - If #stress[AggregateService crashes], the whole system breaks since CapacityService won't send data to anything, and ShowService won't receive any data to show.

  - To prevent the breakdown, AggregateService could also register a #stress[Last Will] that gets caught by other services. 

    - ShowService could then show an #stress[error message]
    - CapacityService could place itself in low-power mode
]

#slide(title: [ShowService Failure])[
  - If #stress[ShowService fails], then the system can't tell which computer has the lowest battery life.
    We thus wouldn't be able, for example, to plug in the computers with the lowest battery life.

  - ShowService could register a #stress[Last Will] that would tell the other processes to get into a low-power mode, or indicate to another process to send a warning to the staff.
]

#slide(title: [Too many AggregateServices])[
  - If an isolated systems starts #stress[multiple AggregateServices], they will all get data from the same CapacityServices and send the data to the same ShowServices.

  - Each time a CapacityService sends its battery status to all the AggregateServices, ShowServices will receive identical data from every AggregateService, leading to #stress[redundant information] being shown.

  - This could be solved by applying a filter on the ShowService, to only accept one incoming message at the same time.
]

#slide(title: [Hard Linux dependency])[
  - Currently, the CapacityService relies on #stress[`sysfs`] to get the current battery level of the device. 
  
  - Since this virtual file system is specific to the #stress[Linux kernel], the code cannot work on other operating systems.

  - The solution would be to write #stress[platform-specific implementations] for each major operating system, but this would be very time consuming.
]

#focus-slide[
  Questions ?
]

#focus-slide[
  Thank you for your attention !
]