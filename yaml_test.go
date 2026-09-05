/*
Copyright 2026 Google Inc. All rights reserved.

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
	"reflect"
	"testing"
)

func TestUnmarshalYAMLBooleanKeys(t *testing.T) {
	got, err := unmarshalYAML([]byte("on: help\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]interface{}{"on": "help"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestUnmarshalYAMLBooleanValues(t *testing.T) {
	got, err := unmarshalYAML([]byte("flag: on\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]interface{}{"flag": true}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}
