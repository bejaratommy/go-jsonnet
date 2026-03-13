std.map(std.escapeStringXml, [
    "",
    "<tag>",
    "</tag>",
    "a & b",
    "say \"hello\"",
    "it's fine",
    "<a href=\"url\">link & text</a>",
    "no escaping needed",
    null,
    42,
])
