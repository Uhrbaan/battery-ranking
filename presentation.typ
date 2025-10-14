#import "@preview/typslides:1.2.6": *
#show: typslides.with(
  ratio: "4-3",
  theme: "bluey",
  font: "Cy",
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

#slide(title: [Attempt at a _live_ demonstration])[

]

#slide(title: "A short video")[
  #link("https://kdrive.infomaniak.com/app/share/1618622/4811f6b8-1228-4b97-8b5c-7e84efd27b2c")
]

#title-slide[Inner workings]