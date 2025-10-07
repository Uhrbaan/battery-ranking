This project implements a ranking of the battery level of different computers running the same services.
This project was implemented as part of the course SIN.04028 'Process Control' at the University of Fribourg.
For more information, refer to the report written in typst.

```sh 
typst compile report.typ && xdg-open report.pdf
```

= Battery Ranking 
To build the project, you need to have go installed.

== Usage 
```
  -all
    	Use this flag to start all the services.
  -broker string
    	Custom broker url. Should be shaped like "tcp://broker.emqx.io:1883" (default "tcp://test.mosquitto.org:1883")
  -capacity
    	Use this flag to start the capacity service.
  -display string
    	The displayed name of your computer. (default "capacity-e092626b-8e02-4b37-ac50-f9d9b51380cf")
  -show
    	Use this flag to start the show service, which will display the ranking in your terminal.
  -store
    	Use this flag to start the storage/aggregation service.
  -v	Enable logging
```

To use the project, simply run `go run . --all`.
If you want to add more devices, consider only running `go run . -capacity #-show` services. The results should be clearer.

