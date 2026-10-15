/*
Copyright 2019 Google Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package jsonnet

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// YAMLToJSONDecoder decodes YAML documents from an io.Reader by
// separating individual documents. It first converts the YAML
// body to JSON, then unmarshals the JSON.
type YAMLToJSONDecoder struct {
	reader *YAMLReader
}

// NewYAMLToJSONDecoder decodes YAML documents from the provided
// stream in chunks by converting each document (as defined by
// the YAML spec) into its own chunk.
func NewYAMLToJSONDecoder(r io.Reader) *YAMLToJSONDecoder {
	reader := bufio.NewReader(r)
	return &YAMLToJSONDecoder{
		reader: NewYAMLReader(reader),
	}
}

// Decode reads a YAML document from the stream or returns an error.
//
// Mapping keys are kept as their written scalars. go-yaml 1.1 (and the
// previous sigs.k8s.io/yaml path) resolved unquoted keys such as "on",
// "yes", and "y" to booleans, so std.parseYaml turned `{on: help}` into
// `{"true": "help"}`. Jsonnet objects can only have string fields, and
// the C++ and Rust implementations keep those keys. Decoding with
// YAML 1.2 rules (only true/false are booleans) matches them.
// Non-string keys, such as the boolean true or an integer, are still
// rendered with their scalar text so the result stays a JSON object.
func (d *YAMLToJSONDecoder) Decode(into interface{}) error {
	doc, err := d.reader.read()
	if err != nil && err != io.EOF {
		return err
	}

	if len(doc) != 0 {
		var parsed interface{}
		if uerr := yaml.Unmarshal(doc, &parsed); uerr != nil {
			return uerr
		}
		if err := assignDecoded(into, normalizeYAML(parsed)); err != nil {
			return err
		}
	}
	return err
}

// assignDecoded stores v in *into. Decode's caller passes *interface{}.
func assignDecoded(into interface{}, v interface{}) error {
	target, ok := into.(*interface{})
	if !ok {
		return fmt.Errorf("YAML decode target must be *interface{}")
	}
	*target = v
	return nil
}

// normalizeYAML turns yaml.v3's decoded tree into the types jsonToValue accepts.
// Map keys that yaml.v3 left as bools or numbers become their decimal/literal text.
func normalizeYAML(v interface{}) interface{} {
	switch v := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(v))
		for k, val := range v {
			out[k] = normalizeYAML(val)
		}
		return out
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(v))
		for k, val := range v {
			out[yamlKeyString(k)] = normalizeYAML(val)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(v))
		for i, val := range v {
			out[i] = normalizeYAML(val)
		}
		return out
	case int, int64, float64, bool, string, nil:
		return v
	case uint64:
		if v > math.MaxInt64 {
			return float64(v)
		}
		return int64(v)
	case time.Time:
		return v.UTC().Format(time.RFC3339Nano)
	default:
		return v
	}
}

func yamlKeyString(k interface{}) string {
	switch k := k.(type) {
	case string:
		return k
	case bool:
		if k {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(k)
	case int64:
		return strconv.FormatInt(k, 10)
	case uint64:
		return strconv.FormatUint(k, 10)
	case float64:
		return strconv.FormatFloat(k, 'f', -1, 64)
	default:
		return fmt.Sprint(k)
	}
}

func (d *YAMLToJSONDecoder) IsStream() bool {
	return d.reader.isStream()
}

// Reader reads bytes
type Reader interface {
	Read() ([]byte, error)
}

// YAMLReader reads YAML
type YAMLReader struct {
	reader Reader
	buffer bytes.Buffer
	stream bool
}

// NewYAMLReader creates a new YAMLReader
func NewYAMLReader(r *bufio.Reader) *YAMLReader {
	return &YAMLReader{
		reader: &LineReader{reader: r},
	}
}

var docStartMarker = []byte("---")

// Read returns a full YAML document.
func (r *YAMLReader) read() ([]byte, error) {
	for {
		line, err := r.reader.Read()
		if err != nil && err != io.EOF {
			return nil, err
		}

		// Per https://yaml.org/spec/1.2.2/#912-document-markers
		// Document content lines are forbidden to contain marker sequences.
		// The marker sequences are `---` or `...` at the start of the line,
		// followed by space, tab, CR, LF, or EOF.
		if bytes.HasPrefix(line, docStartMarker) {
			end := len(docStartMarker)
			if end == len(line) || line[end] == '\n' || line[end] == ' ' || line[end] == '\t' {
				r.stream = true
				if r.buffer.Len() != 0 {
					out := append([]byte(nil), r.buffer.Bytes()...)
					r.buffer.Reset()
					// The document start marker should be included in the next document.
					r.buffer.Write(line)
					return out, nil
				}
			}
		}

		if err == io.EOF {
			if r.buffer.Len() != 0 {
				// If we're at EOF, we have a final, non-terminated line. Return it.
				out := append([]byte(nil), r.buffer.Bytes()...)
				r.buffer.Reset()
				return out, nil
			}
			return nil, err
		}
		r.buffer.Write(line)
	}
}

func (r *YAMLReader) isStream() bool {
	return r.stream
}

// LineReader reads single lines.
type LineReader struct {
	reader *bufio.Reader
}

// Read returns a single line (with '\n' ended) from the underlying reader.
// An error is returned iff there is an error with the underlying reader.
func (r *LineReader) Read() ([]byte, error) {
	var (
		isPrefix bool  = true
		err      error = nil
		line     []byte
		buffer   bytes.Buffer
	)

	for isPrefix && err == nil {
		line, isPrefix, err = r.reader.ReadLine()
		buffer.Write(line)
	}
	buffer.WriteByte('\n')
	return buffer.Bytes(), err
}
