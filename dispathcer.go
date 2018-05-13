package pqueue

import (
	"context"
	"log"
	"sync"
	"time"
)

// Worker is an interface of a worker
type Worker interface {
	Run(ctx context.Context, job Job) bool
}

// NewDispatcher creates and returns dispatcher
func NewDispatcher(max int, worker Worker) Dispatcher {
	d := Dispatcher{
		jobBuffer: make(chan Job, max),
		sem:       make(chan struct{}, max),
		worker:    worker,
		stopTick:  make(chan struct{}),
		stopLoop:  make(chan struct{}),
		stopped:   make(chan struct{}),
	}

	return d
}

// Dispatcher is used for queue
type Dispatcher struct {
	jobBuffer chan Job
	sem       chan struct{}
	worker    Worker
	stopTick  chan struct{}
	stopLoop  chan struct{}
	stopped   chan struct{}
}

// Start starts a dispatcher
func (d *Dispatcher) Start(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval * time.Millisecond)
		max := cap(d.sem)
		for {
			select {
			case <-ticker.C:
				if len(d.sem) < max {
					d.pop(max - len(d.sem))
				}
			case <-d.stopTick:
				ticker.Stop()
				d.stopLoop <- struct{}{}
			}
		}
	}()

	go func() {
		var wg sync.WaitGroup

	Loop:
		for {
			select {
			case job := <-d.jobBuffer:
				wg.Add(1)
				d.sem <- struct{}{}

				go func(job Job) {
					defer func() { <-d.sem }()
					defer wg.Done()

					start := time.Now()
					ctx, cancel := context.WithTimeout(context.Background(), time.Duration(job.Timeout)*time.Second)
					defer cancel()

					isSuccess := d.worker.Run(ctx, job)
					job.Elapsed = time.Now().Sub(start).Seconds()
					if isSuccess {
						job.Complete()
					} else {
						job.Fail()
					}
				}(job)
			case <-d.stopLoop:
				wg.Wait()
				break Loop
			}
		}

		d.stopped <- struct{}{}
	}()
}

// Stop stops a dispatcher.
// The dispatcher waits done every jobs.
func (d *Dispatcher) Stop(ctx context.Context) error {
	d.stopTick <- struct{}{}

	for {
		select {
		case <-ctx.Done():
			return UnlockJobs()
		case <-d.stopped:
			return nil
		}
	}
}

// Stats logs running worker count
func (d *Dispatcher) Stats() {
	log.Printf("run count: %d", len(d.sem))
}

func (d *Dispatcher) pop(length int) {
	jobs, err := LockJobs(length)
	if err != nil {
		log.Print(err)
		return
	}

	for _, job := range jobs {
		d.jobBuffer <- job
	}
}
