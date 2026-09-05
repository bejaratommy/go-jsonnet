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
	"strconv"
	"strings"

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
// the YAML spec) into its own chunk, converting it to JSON via
// yaml.YAMLToJSON, and then passing it to json.Decoder.
func NewYAMLToJSONDecoder(r io.Reader) *YAMLToJSONDecoder {
	reader := bufio.NewReader(r)
	return &YAMLToJSONDecoder{
		reader: NewYAMLReader(reader),
	}
}

// Decode reads a YAML document as JSON from the stream or returns
// an error. The decoding rules match json.Unmarshal, not
// yaml.Unmarshal.
func (d *YAMLToJSONDecoder) Decode(into interface{}) error {
	bytes, err := d.reader.read()
	if err != nil && err != io.EOF {
		return err
	}

	if len(bytes) != 0 {
		converted, convErr := unmarshalYAML(bytes)
		if convErr != nil {
			return convErr
		}
		ptr, ok := into.(*interface{})
		if !ok {
			return fmt.Errorf("YAML decode target must be *interface{}")
		}
		*ptr = converted
	}
	return err
}

// unmarshalYAML converts a YAML document to JSON-compatible Go values.
// Mapping keys are always taken from the original scalar text so YAML 1.1
// booleans such as "on"/"off"/"yes"/"no" stay those strings when used as keys,
// matching cjsonnet and jrsonnet (https://github.com/google/go-jsonnet/issues/766).
func unmarshalYAML(src []byte) (interface{}, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(src, &node); err != nil {
		return nil, err
	}
	return yamlNodeToInterface(&node)
}

func yamlNodeToInterface(n *yaml.Node) (interface{}, error) {
	if n == nil {
		return nil, nil
	}
	switch n.Kind {
	case yaml.DocumentNode:
		if len(n.Content) == 0 {
			return nil, nil
		}
		return yamlNodeToInterface(n.Content[0])
	case yaml.MappingNode:
		m := make(map[string]interface{}, len(n.Content)/2)
		for i := 0; i < len(n.Content); i += 2 {
			key, err := yamlMappingKey(n.Content[i])
			if err != nil {
				return nil, err
			}
			val, err := yamlNodeToInterface(n.Content[i+1])
			if err != nil {
				return nil, err
			}
			m[key] = val
		}
		return m, nil
	case yaml.SequenceNode:
		s := make([]interface{}, 0, len(n.Content))
		for _, c := range n.Content {
			v, err := yamlNodeToInterface(c)
			if err != nil {
				return nil, err
			}
			s = append(s, v)
		}
		return s, nil
	case yaml.ScalarNode:
		return yamlScalarValue(n)
	case yaml.AliasNode:
		return yamlNodeToInterface(n.Alias)
	default:
		return nil, nil
	}
}

func yamlMappingKey(n *yaml.Node) (string, error) {
	if n == nil {
		return "", nil
	}
	if n.Kind == yaml.AliasNode {
		n = n.Alias
	}
	if n.Kind == yaml.ScalarNode {
		return n.Value, nil
	}
	v, err := yamlNodeToInterface(n)
	if err != nil {
		return "", err
	}
	return fmt.Sprint(v), nil
}

func yamlScalarValue(n *yaml.Node) (interface{}, error) {
	quoted := n.Style == yaml.DoubleQuotedStyle || n.Style == yaml.SingleQuotedStyle ||
		n.Style == yaml.LiteralStyle || n.Style == yaml.FoldedStyle
	switch n.ShortTag() {
	case "!!null":
		return nil, nil
	case "!!bool":
		return parseYAMLBool(n.Value)
	case "!!int":
		if i, err := strconv.ParseInt(n.Value, 0, 64); err == nil {
			return i, nil
		}
		if u, err := strconv.ParseUint(n.Value, 0, 64); err == nil {
			return u, nil
		}
		return n.Value, nil
	case "!!float":
		return strconv.ParseFloat(n.Value, 64)
	default:
		// Preserve YAML 1.1 boolean values (on/off/yes/no) for unquoted scalars.
		if !quoted {
			if b, err := parseYAMLBool(n.Value); err == nil {
				return b, nil
			}
		}
		return n.Value, nil
	}
}

func parseYAMLBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "true", "yes", "on", "y":
		return true, nil
	case "false", "no", "off", "n":
		return false, nil
	default:
		return strconv.ParseBool(s)
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
