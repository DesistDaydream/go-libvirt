package virt

import (
	"reflect"
	"testing"
)

func TestMatchName(t *testing.T) {
	tests := []struct {
		pattern string
		name    string
		want    bool
		wantErr bool
	}{
		{pattern: "web-01", name: "web-01", want: true},
		{pattern: "web-01", name: "web-02", want: false},
		{pattern: "web-*", name: "web-01", want: true},
		{pattern: "web-*", name: "db-01", want: false},
		{pattern: "*-01", name: "db-01", want: true},
		{pattern: "web-[0-9][0-9]", name: "web-01", want: true},
		{pattern: "web-[", name: "web-01", wantErr: true},
	}

	for _, test := range tests {
		got, err := matchName(test.pattern, test.name)
		if test.wantErr {
			if err == nil {
				t.Errorf("matchName(%q, %q) 期望错误，实际没有", test.pattern, test.name)
			}
			continue
		}
		if err != nil {
			t.Errorf("matchName(%q, %q) 返回错误: %v", test.pattern, test.name, err)
			continue
		}
		if got != test.want {
			t.Errorf("matchName(%q, %q) = %v，期望 %v", test.pattern, test.name, got, test.want)
		}
	}
}

func TestDedupeStrings(t *testing.T) {
	got := dedupeStrings([]string{"b", "a", "b", "", "a"})
	want := []string{"b", "a"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("dedupeStrings = %v，期望 %v", got, want)
	}
}
