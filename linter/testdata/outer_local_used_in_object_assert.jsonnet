// Regression test for https://github.com/google/go-jsonnet/issues/778
// An outer local variable used only inside an object-level assert should
// NOT be reported as unused.
local myvar = 'xxxxx';

{
  assert std.length(myvar) == 5 : 'yyyyy',
  some_field: 'zzzzz',
}
