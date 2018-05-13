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

func (w worker) Run(ctx context.Context, job Job) error {
	var p payloadJSON
	err := json.Unmarshal(job.Payload, &p)
	if err != nil {
		return err
	}
	var elapsed time.Duration
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
Loop:
	for {
		select {
		case <-ticker.C:
			elapsed = time.Duration(elapsed) + 5*time.Millisecond
			if elapsed > time.Duration(p.Duration)*time.Millisecond {
				break Loop
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func TestStart(t *testing.T) {
	TruncateJob()

	w := worker{}
	for i := 0; i < 4; i++ {
		j := NewJob("test", []byte(`{"duration": 50}`), 5)
		j.Save()
	}

	d := NewDispatcher(2, w)
	d.Start(100)
	time.Sleep(110 * time.Millisecond)
	jobs, _ := ProcessingJobs()
	if len(jobs) != 2 {
		t.Errorf("processing jobs expect 2, actual %d", len(jobs))
	}
	time.Sleep(110 * time.Millisecond)
	jobs, _ = ProcessedJobs(time.Time{}, 0)
	if len(jobs) != 2 {
		t.Errorf("processing jobs expect 2, actual %d", len(jobs))
	}
	ctx := context.Background()
	d.Stop(ctx)
}

func TestStop(t *testing.T) {
	TruncateJob()

	w := worker{}
	for i := 0; i < 8; i++ {
		j := NewJob("test", []byte(`{"duration": 50}`), 5)
		j.Save()
	}

	d := NewDispatcher(4, w)
	d.Start(100)
	time.Sleep(110 * time.Millisecond)
	jobs, _ := ProcessingJobs()
	if len(jobs) != 4 {
		t.Errorf("processing jobs expect 4, actual %d", len(jobs))
	}
	time.Sleep(160 * time.Millisecond)

	ctx, c := context.WithTimeout(context.Background(), 1*time.Second)
	defer c()
	d.Stop(ctx)

	jobs, _ = ProcessedJobs(time.Time{}, 0)
	if len(jobs) != 8 {
		t.Errorf("processing jobs expect 8, actual %d", len(jobs))
	}
}
