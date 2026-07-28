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
	"io"

	"sigs.k8s.io/yaml"
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
		err := yaml.Unmarshal(bytes, into)
		if err != nil {
			return err
		}
	}
	return err
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
				firstMarker := !r.stream
				r.stream = true
				if r.buffer.Len() != 0 {
					// Content preceding the very first document-start marker is an
					// implicit bare document. When that leading content consists
					// only of comments and/or blank lines it is not a document, so
					// it must be discarded rather than emitted as a spurious null.
					// See https://github.com/google/go-jsonnet/issues/660.
					if firstMarker && isBlankOrComment(r.buffer.Bytes()) {
						r.buffer.Reset()
						r.buffer.Write(line)
						continue
					}
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

// isBlankOrComment reports whether the given YAML chunk contains only blank
// lines and comment lines, i.e. no document content.
func isBlankOrComment(b []byte) bool {
	for _, line := range bytes.Split(b, []byte("\n")) {
		trimmed := bytes.TrimLeft(line, " \t\r")
		if len(trimmed) == 0 || trimmed[0] == '#' {
			continue
		}
		return false
	}
	return true
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
