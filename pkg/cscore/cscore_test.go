package cscore

import (
	"sync"
	"testing"
	"testing/fstest"
)

func TestInit(t *testing.T) {
	resetForTesting()
	err := Init(testSheetFS(), testDetailFS())
	if err != nil {
		t.Fatalf("Init() error: %v", err)
	}
}

func TestInit_DoubleCallFails(t *testing.T) {
	resetForTesting()
	if err := Init(testSheetFS(), testDetailFS()); err != nil {
		t.Fatal(err)
	}
	err := Init(testSheetFS(), testDetailFS())
	if err == nil {
		t.Fatal("expected error on second Init()")
	}
}

func TestInit_EmptyFS(t *testing.T) {
	resetForTesting()
	empty := fstest.MapFS{}
	err := Init(empty, empty)
	if err != nil {
		t.Fatalf("Init with empty FS should not error: %v", err)
	}
}

func TestInit_ConcurrentSafety(t *testing.T) {
	resetForTesting()
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- Init(testSheetFS(), testDetailFS())
		}()
	}
	wg.Wait()
	close(errs)

	var successes, failures int
	for err := range errs {
		if err == nil {
			successes++
		} else {
			failures++
		}
	}
	if successes != 1 || failures != 1 {
		t.Errorf("expected 1 success + 1 failure, got %d successes, %d failures", successes, failures)
	}
}

func TestSetDataDir(t *testing.T) {
	resetForTesting()
	SetDataDir("/tmp/test-cs")
	if got := GetDataDir(); got != "/tmp/test-cs" {
		t.Errorf("GetDataDir() = %q, want %q", got, "/tmp/test-cs")
	}
}

func TestMustReg_Panics(t *testing.T) {
	resetForTesting()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic from mustReg() when not initialized")
		}
	}()
	mustReg()
}

func TestJsonMarshal(t *testing.T) {
	got := jsonMarshal(map[string]int{"count": 42})
	if got != `{"count":42}` {
		t.Errorf("jsonMarshal = %q", got)
	}
}

func TestErrorJSON(t *testing.T) {
	err := &validationError{Field: "topic", Message: "empty topic name"}
	got := errorJSON(err)
	if got == "" || got[0] != '{' {
		t.Errorf("errorJSON returned invalid JSON: %q", got)
	}
}
