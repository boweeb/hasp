package domain

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestProfilePath_StringRoundTrip(t *testing.T) {
	tests := []struct {
		dotted string
		path   ProfilePath
	}{
		{"work.foobarco", ProfilePath{"work", "foobarco"}},
		{"personal", ProfilePath{"personal"}},
		{"", nil},
	}
	for _, tt := range tests {
		if got := tt.path.String(); got != tt.dotted {
			t.Errorf("ProfilePath(%v).String() = %q, want %q", tt.path, got, tt.dotted)
		}
		if got := ParseProfilePath(tt.dotted); !reflect.DeepEqual(got, tt.path) {
			t.Errorf("ParseProfilePath(%q) = %v, want %v", tt.dotted, got, tt.path)
		}
	}
}

func TestProfilePath_MarshalJSON(t *testing.T) {
	got, err := json.Marshal(ProfilePath{"work", "foobarco"})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if string(got) != `"work.foobarco"` {
		t.Errorf("json.Marshal = %s, want %q", got, `"work.foobarco"`)
	}
}

func TestProfilePath_Equal(t *testing.T) {
	a := ProfilePath{"work", "foobarco"}
	b := ProfilePath{"work", "foobarco"}
	c := ProfilePath{"work", "acme"}
	if !a.Equal(b) {
		t.Error("expected a.Equal(b)")
	}
	if a.Equal(c) {
		t.Error("expected !a.Equal(c)")
	}
}

func TestProfilePath_IsDescendantOf(t *testing.T) {
	work := ProfilePath{"work"}
	workFoobarCo := ProfilePath{"work", "foobarco"}
	personal := ProfilePath{"personal"}

	if !workFoobarCo.IsDescendantOf(work) {
		t.Error("work.foobarco should be a descendant of work")
	}
	if !work.IsDescendantOf(work) {
		t.Error("a profile should be a descendant of itself (T21's --no-recurse boundary)")
	}
	if personal.IsDescendantOf(work) {
		t.Error("personal should not be a descendant of work")
	}
	if work.IsDescendantOf(workFoobarCo) {
		t.Error("a parent should not be a descendant of its child")
	}
}
