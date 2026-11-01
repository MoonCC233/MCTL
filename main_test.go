package main

import (
	"reflect"
	"testing"
)

func TestSplitDemoArgs(t *testing.T) {
	args, page := splitDemoArgs([]string{"--demo", "versions", "--demo-width", "120", "--demo-height", "36"})
	if page != "versions" {
		t.Fatalf("page = %q", page)
	}
	want := []string{"--demo", "--demo-width", "120", "--demo-height", "36"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}

	args, page = splitDemoArgs([]string{"--demo-page", "logs", "--demo"})
	if page != "" {
		t.Fatalf("explicit --demo-page should win, got %q", page)
	}
	if !reflect.DeepEqual(args, []string{"--demo-page", "logs", "--demo"}) {
		t.Fatalf("args = %#v", args)
	}

	args, page = splitDemoArgs([]string{"--dir", "launch", "--demo"})
	if page != "" {
		t.Fatalf("flag value should not become a page, got %q", page)
	}
	if !reflect.DeepEqual(args, []string{"--dir", "launch", "--demo"}) {
		t.Fatalf("args = %#v", args)
	}
}
