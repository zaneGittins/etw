//go:build windows
// +build windows

package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/Velocidex/etw"
)

func main() {
	var ()
	flag.Parse()

	session, err := etw.NewKernelSession(etw.WithKernelKeyword("FileIOInit"))
	if err != nil {
		log.Fatalf("Failed to create etw session; %s", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	cb := func(e *etw.Event) {
		log.Printf("[DBG] Event %d from %s\n", e.Header.ID, e.Header.TimeStamp)

		event := make(map[string]interface{})

		if data, err := e.EventProperties(); err == nil {
			event["EventProperties"] = data
		} else {
			log.Printf("[ERR] Failed to enumerate event properties: %s", err)
		}
		_ = enc.Encode(event)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		log.Println("[DBG] Starting to listen kernel ETW")

		// Block until .Close().
		if err := session.Process(cb); err != nil {
			log.Printf("[ERR] Got error processing events: %s", err)
		} else {
			log.Printf("[DBG] Successfully shut down")
		}

		wg.Done()
	}()

	// Trap cancellation (the only signal values guaranteed to be present in
	// the os package on all systems are os.Interrupt and os.Kill).
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	// Wait for stop and shutdown gracefully.
	for range sigCh {
		log.Printf("[DBG] Shutting the session down")

		err = session.Close()
		if err != nil {
			log.Printf("[ERR] (!!!) Failed to stop session: %s\n", err)
		} else {
			break
		}
	}

	wg.Wait()
}
