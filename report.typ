#import "project-template.typ": template 

// TODO: change date and abstract
#show: template.with(
  title: [Start-Semester project],
  author: [Léonard #smallcaps[Clément]],
)

_
Notes sur le projet.
Il faut aller juste un peu plus loin que le tutoriel sur moodle.
Il faut donner le sujet du projet jusqu'à dimanche.
La semaine prochaine on pourra travailler dessus.
On doit *simuler* un senseur et un actuateur. 
Il n'y a pas besoin de préparer toute une infrastructure.
On attend la longueur 4-6 pages.
Il est conseillé de faire une vidéo du projet fonctionnant.
\
Ce serait également bien de réfléchir à comment est ce qu'il faut évoluer le système lorsu'un service crashe.
_

= Idea
This project allows linux machines to connect to a mosquitto network and share their battery capacity. 
The program will then display which machines have the lowest battery level.
Such a system could be expanded to monitor several aspects of the computers, helping for example an IT team to know how and when to service the machines.

The goal of this project is to learn how to design and implement a project in an event-driven architecture, more specifically using the standard #link("https://mosquitto.org/")[Mosquitto] protocol developped by the eclipse foundation. 
The secondary and personal goal of this project is to learn the #link("https://go.dev/")[Go] programming language and its concurrency model.

= Design
The Project is divided into three processes, which are all located in the `process` module. 
The first one, `CapacityService` is tasked to continously check for the change in the laptop's batterry capacity.

The second, `StoreService` listens for outgoing messages from `CapacityService` and aggregates the messages comming from different computers as to store their respective battery capacity. 
It will then publish a message conttaining all the computers identifiers and their battery levels. 
The service serves a role of *mediator*.
It processes data from a first service, and sends the result to the next one.

The last service, `ShowService` recieves the full list of computers with their battery percentage and displays it to a console.

== Limitations
While as many `capacity` or `show` services can be used at once, using multiple `store` services might produce some unexpected behavior. 
The `ShowService` will display the received data from _each_ active `StoreService`, leading to redundant information being shown.

// TODO: implement a system when storeservice dies, since it breaks the flow of information.

= Implementation
The project is implemented as a single go project, where each service runs inside its own #link("https://go.dev/tour/concurrency/1")[goroutine].
This allows us to run multiple services at once with the comfort of running a single program. 

In the project, each process is implemented inside the `process` module.
Each process provides essential data like related topics inside a ```go struct``` named after the service. Each processa also provides a `Start()` method that runs the code ralted to that service.

A process also may or may not define a ```go type``` with #link("https://go.dev/wiki/Well-known-struct-tags")[struct tags] to define what the shape of the `JSON` data the service expects. 

These two implementation details help us follow the `DUISE` -- Documentation, Unit, Intent, Status, Event -- approach. 
The data the service `struct` holds corresponds to the different intent, status, or event topics, and the unit field is used to identify the process.
The documentation part, or the description of the data a service expects, is handled by the ```go type``` we just talked about.

== `CapacityService`
Our first service is defined with only the `Unit` and `Status` fields. Since do not need to listen to other topics, we don't define an `Intent`, and since we are only interested in the battery level of the device, we simply define a `Status` topic where the service will publish that information to.

When the `Start()` method is called, the service first establishes a connection with the Mosquitto client, after which we start a function that polls the battery capacity every thirty seconds in the background, and wait for it to notify us through a #link("https://go.dev/tour/concurrency/2")[go channel] when the battery changes.

Checking for the battery capacity works by using the `sysfs` pseudo-filesystem which provides an interface to kernel data structures @sysfs-5.
This enables us to get the battery level of the device simply by reading the virtual file at `/sys/class/power_supply/BAT0/capacity`.
One important limitation of this method is (1) it is restricted to linux machines (2) which have a single battery on slot at `BAT0`. This will work with most linux laptops but cannot be guranteed.
The code reading the data is fairly straight forward, once it is reduced to the bare minimum, as shown in @pollBattery-simplified.
The code waits for a timer to expire (which will send a signal through the `ticker.C` channel, which is caught by a matchig ```go case``` in the ```go select``` block), the reads the virutal file, converts it to an integer, and sends it back through the `capacityCh` channel for the main process which will send it to the other processes through Mosquitto.

#figure(
  ```go 
  func pollBattery(ctx context.Context, capacityCh chan<- int, errCh chan<- error) {
    // ...
    for {
      select {
        // ... 
      case <-ticker.C: // Waiting for the timer to expire
        content, err := os.ReadFile(batPath) // Reading virtual file

        capacity, err := strconv.Atoi(strings.TrimSpace(string(content))) // Converting the capacity to an integer

        if capacity != previousCapacity { // Checking if battery levels changed
          capacityCh <- capacity // Sending current capacity to the main process
        }

        previousCapacity = capacity
      }
    }
  }
  ```,
  caption: [Code reponsible for reading battery levels.]
) <pollBattery-simplified>

Once we are notified of the current capacity, `CapacityService` will package the display name of the device and its battery capacity in a json object, and send it through mosquitto to the topic stored in `Status`.

Note that at first, this service was using `udev`, the Linux Kernel device manager to get its data.
The service was connecting to `udev`'s event system, which would have been more in line with the course's objective. 
Hovever, that proved unreliable since battery capacity changes did not reliably emit events. 
If this implementation is of interest, you may checkout commit `d788604`.

== `StoreService`
This service aggregates messages comming of different `CapacityService`s from different computers.
It combines them into a ```go map``` containing both the computer's identifier and battery level. 
It does this every time it recieves a message ont the `capacity` topic. 
All of this is done within the subscription described in @StoreService-simplified.

First, we subscribe to the intent of the `StoreService` -- which is the status of the previous `CapacityService` -- to execute a function each time the topic recieves a new message.
This function reads the message, adapts it to a usable type, and stores it as a map (`latestReading`) as to differentiate the different computers and their battery levels.
Finally, that map is again converted to JSON and sent to The next service.

#figure(
  ```go 
  // Subscription to the `capacity` topic
  if token := client.Subscribe(service.Intent, 1, func(client mqtt.Client, message mqtt.Message) {
      // Getting the data from Intent and converting it to a usable type
      msg := message.Payload()
      var data dataStore
      err := json.Unmarshal(msg, &data)

      // Updating the stored data with the new data
      if data.DisplayName != "" {
          latestReading[data.DisplayName] = data.Percentage
      }

      // Encoding the merged data and publishing it
      jsonData, _ := json.Marshal(latestReading)
      client.Publish(service.Status, 1, false, string(jsonData))
  }); // ...
  ```,
  caption: [Code responsible for getting all the battery levels from different computers and combining them into a single list.]
) <StoreService-simplified>

While running multiple `StoreService` processes is safe -- it will not break the program -- it is generally advised not to, since the next service will recieve more and redundant data if you do so. 

== `ShowService`
The final service is the `ShowService`. It simply gets data from the previous service and displays it on a console. 
@ShowService-simplified shows the code that does it. 
We subscribe to the service's intent, which is where the `StoreService` posts to, extract the message and convert it back into a ```go map```. 
Then, to prepare the display we sort the different computers into a list according to their corresponding battery levels, and finally, we print that list to the console.

#figure(
  ```go 
  // Subscription to the `store` topic
  if token := client.Subscribe(service.Intent, 1, func(client mqtt.Client, message mqtt.Message) {
      // Getting and processing the message from intent
      msg := message.Payload()
      var data dataShow
      json.Unmarshal(msg, &data)

      // Sorting the different entries according to their capacities
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
	});
  ```,
  caption: [Code printing the different computer's battery levels to the console.]
) <ShowService-simplified>

= Result
The program runs as expected on compatible devices.
It has been tested running on both Ubuntu and Fedora laptops. 

A video showing the program running on two different laptops can be viewed at #link("https://kdrive.infomaniak.com/app/share/1618622/4811f6b8-1228-4b97-8b5c-7e84efd27b2c").

= Reasoning
// Pourquoi on a choisi l'exemple
= Conclusion

#pagebreak()
#bibliography("bibliography.bib"))
