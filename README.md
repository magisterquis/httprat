httprat
=======
Simple but production-grade backdoor.

Not for illegal use.

Well, production-grade after a bit of thorough testing, perhaps.

Each compromised host makes an HTTP(s) beacon back to its own unique path (MAC
address or hostname work well).  If there's tasking, the C2 server sends it to
the compromised host.  Output from tasking (or anything, really) is sent back
to the C2 server from the compromised hosts via further asynchronous HTTP
requests.

```go
/* TODO: Write real documentation */

Please run it with `-h` for more information.
```
The (as-yet rough) [HOWTO](./HOWTO.md) provides a quick guide to getting
up and running with a minimum of fuss.

Installation
------------
C2 server only 
```sh
go get -u github.com/magisterquis/httprat/c2
```

C2 server and sample backdoors
```sh
cd $GOPATH/src/ #(or wherever Go code lives)
mkdir -p github.com/magisterquis
cd github.com/magisterquis
git clone https://github.com/magisterquis/httprat
go install -u github.com/magisterquis/httprat
```

Compiled binaries are available upon request.

C2
--
The C2 server listens for HTTP(s) GET requests from compromised hosts.  When a
host connects and requests a path (endpoint) for which the C2 server has a
command, it sends the command.

Compromised hosts send back command output with POST requests to the same path.

For humans to control the C2 server, a unix domain socket (or, optionally a
TCP socket on the off chance someone decides to run a C2 server on a Windows
host) listens for connections.

```
$ rlwrap nc -vU ./c2.sock  
Welcome.  You are unix-sock-1495170450036702645.
Which endpoint would you like to control?
Enter an endpoint path or ? to list endpoints.
> ?
Endpoint                     Last beacon
--------                     -----------
/client_blue/printserv       2017-05-19 01:07:00.453661914 -0400 EDT
/client_green/ceo_desktop    2017-05-19 01:07:09.811066807 -0400 EDT
/client_green/httpsrv0133    2017-05-19 01:07:23.726051473 -0400 EDT

Which endpoint would you like to control?
Enter an endpoint path or ? to list endpoints.
> /client_blue/printserv
/client_blue/printserv> uname -a
/client_blue/printserv>  -> "uname -a"
OpenBSD omnia.dmz.stuartsapartment.com 6.1 GENERIC.MP#20 amd64
```

At the moment, output isn't very nice, but it gets the job done.  This is
likely to be fixed in the future.

Multiple control connections to the C2 server are possible to enable a team to
control multiple compromised hosts.  It's also possible for multiple control
connections to task an endpoint and view the output of commands.  It works, a
bit like multiplayer, line-oriented bash.  Commands (really just input lines)
are sent one-by-one, and everybody gets notified when a line is sent and sees
all the returned data.

TLS
---
By default, a self-signed TLS certificate is generated when the C2 server
starts.  For serious deployments, a real certificate and key can be specified
with `-cert` and `-key` or a reverse proxy can be used to terminate TLS.

Besides serving straight HTTP(s), the C2 server can also listen on a Unix 
domain socket for FCGI or HTTP requests, to help intergrate with existing
infrastructure.

Backdoors
---------
A selection of simple backdoors are included.  They generally all follow the
pattern of `request | /bin/sh | reply`, though there's no reason in principle
that it has to be limited to shell commands.

Windows
-------
The C2 server should work on Windows, with the caveat that human interactions
have to go over a TCP socket (on localhost, please).

A quick powershell oneliner should do the trick to beacon back.

Gotchas
-------
- Endpoints with `..`'s or multiple slashes will be "fixed" by the HTTP library,
so they should probably be avoided.
- User-visible output is really ugly
- 

Why?
----
Why write something like this when we have Emp[iy]re/Pupy/Meterpreter and so on
for free?

httprat has the following advantages
- Simplicity.  A single binary for the server (plus TLS cert and key), a shell
on-liner on the victim.  Not much to set up, really.
- Flexibility.  It's easy to write backdoors in whatever language bleds in with
the environment with whatever HTTP/TLS client software won't be caught.
- Domain fronting.  The C2 server only really inspects the URL path, making it
easy to put behind larger infrastructure where someone else terminates TLS.
- FCGI.  Easy to integrate with existing http servers or reverse proxies (say
to have your compromised hosts call to your clients' internal webserver or fax
machine).

It's not without its downsides, of course
- Very limited functionality.  It sends and receives (small amounts) of bytes.
'Course, you can probably script up something on top of it.
- The user interface, it's not pretty.
