// Regression test for https://github.com/google/go-jsonnet/issues/636
// Closing parenthesis of multiline parameter list should align with
// the start of the function definition, not with the parameters.
local asd(
  x=1
) = {
  hello: x,
};
local f(
  a,
  b,
  c=3
) = a + b + c;
asd(x=4) + f(1, 2)
