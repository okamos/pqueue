package pqueue

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

type payloadJSON struct {
	Duration int `json:"duration"`
}

type worker struct{}

func (w worker) Run(payload json.RawMessage) bool {
	var p payloadJSON
	err := json.Unmarshal(payload, &p)
	if err != nil {
		return false
	}
	time.Sleep(time.Duration(p.Duration) * time.Millisecond)
	return true
}

func TestStart(t *testing.T) {
	TruncateJob()

	w := worker{}
	d := NewDispatcher(2, w)
	for i := 0; i < 4; i++ {
		j := NewJob("test", []byte(`{"duration": 100}`), 5)
		j.Save()
	}

	d.Start(100)
	time.Sleep(150 * time.Millisecond)
	jobs, _ := ProcessingJobs()
	if len(jobs) != 2 {
		t.Errorf("processing jobs expect 2, actual %d", len(jobs))
	}
	time.Sleep(150 * time.Millisecond)
	jobs, _ = ProcessedJobs(time.Time{}, 0)
	if len(jobs) != 2 {
		t.Errorf("processing jobs expect 2, actual %d", len(jobs))
	}
}

func TestStop(t *testing.T) {
	TruncateJob()

	w := worker{}
	d := NewDispatcher(4, w)
	for i := 0; i < 8; i++ {
		j := NewJob("test", []byte(`{"duration": 100}`), 5)
		j.Save()
	}

	d.Start(100)
	time.Sleep(150 * time.Millisecond)
	jobs, _ := ProcessingJobs()
	if len(jobs) != 4 {
		t.Errorf("processing jobs expect 4, actual %d", len(jobs))
	}
	time.Sleep(150 * time.Millisecond)

	ctx, c := context.WithTimeout(context.Background(), 2*time.Second)
	defer c()
	d.Stop(ctx)

	jobs, _ = ProcessedJobs(time.Time{}, 0)
	if len(jobs) != 8 {
		t.Errorf("processing jobs expect 8, actual %d", len(jobs))
	}
}
