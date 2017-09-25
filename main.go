package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// filename Path for apcpi light control
var filename = "/proc/acpi/ibm/light"

// pidfile path where pidfile can be found
var pidfile = "/var/run/golight.pid"

// ErrAlreadyLocked error
var ErrAlreadyLocked = errors.New("Locked by other process")

// LockFile object
type LockFile struct {
	name string
	file *os.File
}

// Lock automatically checks if the file already exists, if so, reads the process ID
// from the file and checks if the process is running.
func Lock(name string) (*LockFile, error) {
	var err error

	lock := LockFile{name: name}

	if lock.file, err = os.OpenFile(lock.name, os.O_CREATE|os.O_RDWR, os.ModeTemporary|0640); err == nil {
		var pid int
		if _, err = fmt.Fscanf(lock.file, "%d\n", &pid); err == nil {
			if pid != os.Getpid() {
				if ProcessRunning(pid) {
					return nil, ErrAlreadyLocked
				}
			}
		}
		_, err = lock.file.Seek(0, 0)
		check(err)
		if n, err := fmt.Fprintf(lock.file, "%d\n", os.Getpid()); err == nil {
			err = lock.file.Truncate(int64(n))
			check(err)
			return &lock, nil
		}
		return nil, err
	}
	return nil, err
}

// Unlock closes and deletes the lock file previously created by Lock()
func (l *LockFile) Unlock() {
	err := l.file.Close()
	check(err)
	err = os.Remove(l.name)
	check(err)
}

// ProcessRunning find  and check pid from file
func ProcessRunning(pid int) bool {
	p, e := os.FindProcess(pid) // On unix the FindProcess never returns an error
	check(e)
	err := p.Signal(syscall.Signal(0)) // Returns error if process is not running
	return err == nil
}
func check(e error) {
	if e != nil {
		fmt.Println(e)
		os.Exit(125)
	}
}
func changeState(state string) {
	switch state {
	case "on":
		err := ioutil.WriteFile(filename, []byte("on"), 0644)
		check(err)
	case "off":
		err := ioutil.WriteFile(filename, []byte("off"), 0644)
		check(err)
	}

}

func main() {
	l, err := Lock(pidfile)
	check(err)
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		changeState("off")
		l.Unlock()
		os.Exit(1)
	}()
	timesPtr := flag.Int("times", 1, "How many times blink")
	intervalPtr := flag.Int64("interval", 300, "Milliseconds between blinks")
	flag.Parse()
	if *intervalPtr <= 15 {
		*intervalPtr = int64(15)
	}
	for i := 1; i <= *timesPtr; i++ {
		changeState("on")
		time.Sleep(time.Duration(*intervalPtr) * time.Millisecond)
		changeState("off")
		time.Sleep(time.Duration(*intervalPtr) * time.Millisecond)

	}
	l.Unlock()
}
