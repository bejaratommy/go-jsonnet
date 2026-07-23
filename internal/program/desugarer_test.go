/*
Copyright 2016 Google Inc. All rights reserved.

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

package program

import (
	"testing"

	"github.com/google/go-jsonnet/ast"
)

// TestObjectFieldIDNameHasLocation checks that the string literal synthesized
// for a shortcut object field (`{ foo: ... }`) carries the location of the
// field identifier after desugaring. See issue #456.
func TestObjectFieldIDNameHasLocation(t *testing.T) {
	node, err := SnippetToAST(ast.DiagnosticFileName("test"), "test", "{ foo: 42 }")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj, ok := node.(*ast.DesugaredObject)
	if !ok {
		t.Fatalf("expected *ast.DesugaredObject, got %T", node)
	}
	if len(obj.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(obj.Fields))
	}

	name, ok := obj.Fields[0].Name.(*ast.LiteralString)
	if !ok {
		t.Fatalf("expected field name to be *ast.LiteralString, got %T", obj.Fields[0].Name)
	}
	if name.Value != "foo" {
		t.Fatalf("expected field name %q, got %q", "foo", name.Value)
	}

	loc := name.Loc()
	if !loc.IsSet() {
		t.Fatalf("expected field name to have a location, got unset LocationRange")
	}
	// `foo` occupies columns 3-6 (exclusive end) on line 1 of "{ foo: 42 }".
	if loc.Begin.Line != 1 || loc.Begin.Column != 3 || loc.End.Line != 1 || loc.End.Column != 6 {
		t.Fatalf("unexpected field name location: got %v (begin %+v end %+v)", loc.String(), loc.Begin, loc.End)
	}
}
