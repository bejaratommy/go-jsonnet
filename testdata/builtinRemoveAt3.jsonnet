local a = [1, 2, 3];
{
  past_end: std.removeAt(a, 5),
  at_length: std.removeAt(a, 3),
  last: std.removeAt(a, 2),
  negative: std.removeAt(a, -1),
  empty: std.removeAt([], 0),
}
