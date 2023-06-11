package main

import "github.com/alecthomas/kong"

// Options command line options
type Options struct {
	// The serial device
	Device string `arg:"" help:"specify the serial device like '/dev/ttyACM0'"`
	// Don't outout to STDOUT
	NoStdout bool `short:"n" long:"no-stdout" help:"don't output to stdout"`
	// Output interval (sec)
	Interval int `short:"i" long:"interval" help:"interval (second) for output. default: '1'" default:"1"`
	// Quiet
	Quiet bool `long:"quiet" help:"don't output any process logs to STDERR"`
	// Version
	Version kong.VersionFlag `short:"v" long:"version" help:"show program version"`
}
