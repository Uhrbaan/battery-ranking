#import "@preview/latex-lookalike:0.1.4"

#let template(
  title: [Title],
  author: [Léonard #smallcaps[Clément]],
  date: [May 27, 2000],
  abstract: lorem(100),
  body
) = {
  set page(
    paper: "a4",
    columns: 1,
  ) 

  set par(justify: true)
  set text(lang: "en")
  
  set align(center)
  v(4em)
  text(size: 24pt, title)
  v(0pt)
  
  text(size: 16pt, author)
  v(0em)

  text(size: 14pt, date)
  v(2em)

  text(weight: "bold", [Abstract])
  set align(left)
  set par(first-line-indent: 1em)
  abstract
  v(4em)

  set text(hyphenate: auto)
  set figure(placement: auto)
  set heading(numbering: "1.1.")

  show: latex-lookalike.style-outline

  outline(depth: 2)

  set page(
    numbering: "1",
    header: title + h(1fr) + author,
  )
  let style-number(number) = text(gray)[#number]
  show raw.where(block: true): it => grid(
    columns: 2,
    align: (right, left),
    gutter: 0.5em,
    ..it.lines
      .enumerate()
      .map(((i, line)) => (style-number(i + 1), line))
      .flatten()
  )

  // show raw.where(block: false): it => box(fill: luma(240), outset: 2pt, radius: 2pt, it)

  show link: it => underline(it, stroke: blue+1.5pt)
  set par(first-line-indent: 1em)

  body
}

#show: template