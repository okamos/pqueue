package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/okamos/pqueue"
)

type payloadJSON struct {
	Second int `json:"second"`
}

type worker struct{}

func (w worker) Run(payload json.RawMessage) bool {
	var p payloadJSON
	err := json.Unmarshal(payload, &p)
	if err != nil {
		return false
	}
	time.Sleep(time.Duration(p.Second) * time.Second)
	return true
}

type jobJSON struct {
	Name    string          `json:"name"`
	Payload json.RawMessage `json:"payload"`
	Timeout uint            `json:"timeout"`
}

func main() {
	sigCh := make(chan os.Signal, 1)
	defer close(sigCh)

	err := pqueue.NewDB()
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/job", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		b, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var jJSON jobJSON
		err = json.Unmarshal(b, &jJSON)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		job := pqueue.NewJob(jJSON.Name, jJSON.Payload, jJSON.Timeout)
		err = job.Save()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	http.HandleFunc("/jobs", func(w http.ResponseWriter, r *http.Request) {
		for i := 0; i < 50; i++ {
			job := pqueue.NewJob("sleep", []byte(fmt.Sprintf(`{"second":%d}`, rand.Intn(50)+1)), 20)
			job.Save()
		}
		w.WriteHeader(http.StatusOK)
	})

	srv := &http.Server{
		Addr: ":8080",
	}

	err = pqueue.ReleaseJobs()
	if err != nil {
		log.Fatal(err)
	}
	w := worker{}
	d1 := pqueue.NewDispatcher(6, w)
	d1.Start(200)
	d2 := pqueue.NewDispatcher(4, w)
	d2.Start(500)

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("Error!!: %s", err.Error())
		}
	}()

	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGINT)
	<-sigCh
	log.Println("Shutdown Server and Dispatcher...")

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		ctx, c := context.WithTimeout(context.Background(), 5*time.Second)
		defer c()

		if err = d1.Stop(ctx); err != nil {
			log.Printf("Failed dispatcher 1 stop: %s", err)
		}
		wg.Done()
	}()

	go func() {
		ctx, c := context.WithTimeout(context.Background(), 5*time.Second)
		defer c()

		if err = d2.Stop(ctx); err != nil {
			log.Printf("Failed dispatcher 1 stop: %s", err)
		}
		wg.Done()
	}()

	ctx, c := context.WithTimeout(context.Background(), 5*time.Second)
	defer c()

	if err = srv.Shutdown(ctx); err != nil {
		log.Printf("Failed server shut down: %s", err)
	}

	wg.Wait()

	log.Print("Shutdown Done!!")
}
