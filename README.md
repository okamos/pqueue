pqueue is a lightweight, job queue eco system.

* Using RDBMS (PostgreSQL)
* Multiple queues
* Delayed jobs
* Job retrying

# Setup
* CREATE TABLE [job.sql](https://github.com/okamos/pqueue/blob/master/data/schema/job.sql)

# Configuration
* Set a data source name for the job queue. example below ...
    `postgresql://[user[:password]@][netloc][:port][,...][/dbname][?param1=value1&...]`  
    If you want more information, see [Connection Strings](https://www.postgresql.org/docs/current/static/libpq-connect.html#LIBPQ-CONNSTRING)

# Usage

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/okamos/pqueue"
)

type payloadJSON struct {
	Type    string `json:"type"`
	Ammount int    `json:"ammount"`
}

type worker struct{}

func (w worker) Run(job pqueue.Job) bool {
	var p payloadJSON
	err := json.Unmarshal(job.Payload, &p)
	if err != nil {
		return false
	}
	// do something
	return true
}

func main() {
	pqueue.SetConfig("psql_dsn", "host=localhost user=postgres dbname=development sslmode=disable")
	err := pqueue.NewDB()
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		// name, payload(JSON), timeout
		job := pqueue.NewJob("test job", []byte(fmt.Sprintf(`{"type": "mini", "ammount": %d}`, i)), 5)
		// Set RunAfter if you want schedule the job.
		job.RunAfter = time.Now().Add(5 * time.Hour) // Default time.Now()
		// Set Priority if you want control the job. Large number is low latency.
		job.Priority = 20 // Default 0
		err := job.Save()
		if err != nil {
			log.Print(err)
		}
	}

	w := worker{}
	d := pqueue.NewDispatcher(8, w) // concurrency, worker
	d.Start(200)                    // interval(ms)

	sigCh := make(chan os.Signal, 1)
	defer close(sigCh)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGINT)
	// Ctrl-C or quit signal etc ...
	<-sigCh

	ctx, c := context.WithTimeout(context.Background(), 5*time.Second)
	defer c()

	if err = d.Stop(ctx); err != nil {
		log.Fatalf("Failed dispatcher 1 stop: %s", err)
	}
}
```
