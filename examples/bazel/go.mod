module example/go-jsonnet-using-bazel

go 1.24.5

require github.com/google/go-jsonnet v0.22.0

require (
	golang.org/x/crypto v0.45.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/google/go-jsonnet => ../../
