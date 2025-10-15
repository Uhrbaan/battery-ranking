#import "@preview/fletcher:0.5.8" as fletcher: diagram, node, edge
#import "@preview/typslides:1.2.6": *
#show: typslides.with(
  ratio: "4-3",
  theme: "bluey",
  font: "Noto Sans",
  link-style: "color",
)

// Contenu: 
// - présentation du truc qui fonctionne
//   TODO: faire un système de simulation 
// - Montrer le data flow qui explique le système
// - Qu'est ce qui se passe si un système ne réussit pas
// - Limitations, améliorations

#front-slide(
  title: "Battery ranking",
  subtitle: [Ranking battery levels in an event-driven manner.],
  authors: "Léonard Clément",
  info: [#link("https://github.com/Uhrbaan/battery-ranking")],
)

#table-of-contents()

#title-slide[Demonstration]

#slide(title: [_Live_ Demo])[
  + Install the go project:
    #grayed[```sh 
    sudo apt install golang # install go
    git clone https://github.com/Uhrbaan/battery-ranking.git # clone the repo
    cd battery-ranking
    go mod tidy # install go packages
    ```]
  
  + #block(sticky:true)[Run the project]
    #grayed[```sh
    # start the store and show services on one computer.
    # they will respectively aggregate and display the collected data.
    go run . -store -show 

    # start the service reading your battery percentage.
    # -v enable extensive logging
    # -display=computer sets the diplayname of you laptop to "computer"
    # use -simulate to simulate a random charge or discharge amount.
    go run . -capacity -display=computer -v
    ```]
]

#slide(title: "A short video")[
  #link("https://kdrive.infomaniak.com/app/share/1618622/4811f6b8-1228-4b97-8b5c-7e84efd27b2c")
]

#title-slide[Inner workings]

#slide(title: [Data Flow])[
  #figure(image("schema.svg"))
]