//go:build nintendo3ds

package main

import (
	"strconv"

	"github.com/akrck02/go-3ds/network"
	"github.com/akrck02/go-3ds/storage"
)

// The game's log, to see what happened on a console nobody can attach a debugger to: each line
// goes to the debug log (println), to sdmc:/3ds/ships3d/log.txt (read with ftpd), to the options'
// Log tab, and, when built with a log server, over the network as it happens:
//
//	tinygo build ... -ldflags "-X main.logServer=192.168.1.20:7777" ...
//
// and on that computer, for example: nc -lk 7777
var logServer = ""

// The log's sizes: lines kept for the Log tab, and how often the file is written, in seconds.
const (
	logLinesShown   = 40
	logSaveSeconds  = 2
	logFile         = "log.txt"
	logOldFile      = "log-before.txt"
	logRemoteTries  = 1
	figuresLogEvery = 10 // seconds between lines of figures
)

// logbook is the game's log.
type logbook struct {
	lines     []string
	unsaved   []byte
	savedAt   float64
	folder    storage.Folder
	hasFolder bool
	remote    *network.Connection
	tried     int
	now       float64
}

// newLogbook starts this run's log file, keeping the last run's as log-before.txt.
func newLogbook() *logbook {
	book := &logbook{}
	folder, err := storage.Open(profileFolder)
	if err != nil {
		println(err.Error()) // the debug log only: there is no file to write to
		return book
	}
	if earlier, err := folder.Load(logFile); err == nil {
		folder.Save(logOldFile, earlier) //nolint:errcheck // keeping the last run's log is a nicety
	}
	folder.Remove(logFile)
	book.folder, book.hasFolder = folder, true
	return book
}

// add writes a line, stamped with the seconds since the game started.
func (book *logbook) add(text string) {
	line := strconv.FormatFloat(book.now, 'f', 2, 64) + " " + text
	println(line)
	book.lines = append(book.lines, line)
	if len(book.lines) > logLinesShown {
		book.lines = book.lines[len(book.lines)-logLinesShown:]
	}
	book.unsaved = append(book.unsaved, line...)
	book.unsaved = append(book.unsaved, '\n')
	if book.remote != nil {
		if _, err := book.remote.Write([]byte(line + "\n")); err != nil {
			book.remote.Close() //nolint:errcheck // the log server went away; the file keeps logging
			book.remote = nil
		}
	}
}

// keep writes what was logged to the SD card now and then, and keeps the log server's connection
// sending; it runs between frames, at now seconds.
func (book *logbook) keep(now float64) {
	book.now = now
	if book.remote != nil {
		book.remote.Flush() //nolint:errcheck // a failed send shows on the next add
	}
	if book.hasFolder && len(book.unsaved) > 0 && now-book.savedAt >= logSaveSeconds {
		if err := book.folder.Append(logFile, book.unsaved); err != nil {
			println(err.Error()) // the debug log: the file can't take more
		}
		book.unsaved, book.savedAt = book.unsaved[:0], now
	}
}

// connectRemote reaches the log server once, when the game was built with one.
func (book *logbook) connectRemote() {
	if logServer == "" || book.remote != nil || book.tried >= logRemoteTries {
		return
	}
	book.tried++
	host, port, found := cutLast(logServer, ':')
	portNumber, err := strconv.Atoi(port)
	if !found || err != nil {
		book.add("log server " + logServer + ": not address:port")
		return
	}
	connection, err := network.Dial(host, portNumber)
	if err != nil {
		book.add("log server " + logServer + ": " + err.Error())
		return
	}
	book.remote = connection
	for _, line := range book.lines {
		connection.Write([]byte(line + "\n")) //nolint:errcheck // what was logged before the server answered
	}
}

// cutLast splits text at its last separator.
func cutLast(text string, separator byte) (before, after string, found bool) {
	for index := len(text) - 1; index >= 0; index-- {
		if text[index] == separator {
			return text[:index], text[index+1:], true
		}
	}
	return text, "", false
}
