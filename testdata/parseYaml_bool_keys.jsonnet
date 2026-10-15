// YAML 1.1 treated on/yes/y/n/no/off as booleans. Jsonnet object keys are
// strings, and the other implementations keep the scalar as written.
[
  std.parseYaml(text)
  for text in [
    "on: help",
    "y: 2",
    "yes: 1",
    "n: false",
    "off: 0",
    "true: 1",
    "1: a",
    "flag: yes",
    |||
      base: &b {x: 1}
      child:
        <<: *b
        y: 2
    |||,
    |||
      a: &a {x: 1}
      b: &b {y: 2}
      c:
        <<: [*a, *b]
    |||,
  ]
]
