package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/soiya/chissoku2/gen/sqlc"
	"io"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"time"

	"github.com/alecthomas/kong"
	"go.bug.st/serial"
)

// CO2Data - the data
type CO2Data struct {
	CO2         int64            `json:"co2"`
	Humidity    float64          `json:"humidity"`
	Temperature float64          `json:"temperature"`
	Timestamp   pgtype.Timestamp `json:"timestamp"`
}

// initialize and prepare the device
func prepareDevice(p serial.Port, s *bufio.Scanner) error {
	logInfo("Prepare device...:")
	defer logPrintln("")
	for _, c := range []string{"STP", "ID?", "STA"} {
		logPrintf(" %v", c)
		if _, err := p.Write([]byte(c + "\r\n")); err != nil {
			return err
		}
		time.Sleep(time.Millisecond * 100) // wait
		for s.Scan() {
			t := s.Text()
			if t[:2] == `OK` {
				break
			} else if t[:2] == `NG` {
				return fmt.Errorf(" command `%v` failed", c)
			}
		}
	}
	logPrint(" OK.")
	return nil
}

func main() {
	var opts Options

	kong.Parse(&opts,
		kong.Name(ProgramName),
		kong.Vars{"version": "v" + Version},
		kong.Description(`A CO2 sensor reader`))

	if opts.Quiet {
		logWriter = io.Discard
	}

	pool, err := pgxpool.New(context.Background(), os.Getenv("POSTGRESQL_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create connection pool: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	queries := sqlc.New(pool)

	port, err := serial.Open(opts.Device, &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		StopBits: serial.OneStopBit,
		Parity:   serial.NoParity,
	})
	if err != nil {
		logErrorf("Opening serial: %+v: %v\n", err, opts.Device)
		os.Exit(1)
	}
	defer func() { port.Write([]byte("STP\r\n")); time.Sleep(time.Millisecond * 100); port.Close() }()

	// serial reader
	port.SetReadTimeout(time.Second * 10)
	s := bufio.NewScanner(port)
	s.Split(bufio.ScanLines)

	if err := prepareDevice(port, s); err != nil {
		logErrorln(err.Error())
		port.Close()
		os.Exit(1)
	}

	// trap SIGINT
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)
	// signal handler
	go func() {
		<-sigch
		port.Write([]byte("STP\r\n"))
	}()

	// serial reader channel
	r := make(chan *CO2Data)

	// periodical dispatcher
	go func() {
		var cur *CO2Data // current data
		tick := time.Tick(time.Second * time.Duration(opts.Interval))
		for {
			select {
			case <-tick:
				if cur == nil {
					continue
				}

				err := queries.InsertData(context.Background(), sqlc.InsertDataParams{
					Co2:         cur.CO2,
					Humidity:    cur.Humidity,
					Temperature: cur.Temperature,
					Timestamp:   cur.Timestamp,
				})
				if err != nil {
					logErrorln(err.Error())
					os.Exit(1)
				}

				if err != nil {
					logError(err.Error())
					continue
				}
				fmt.Printf("co2:%v,humidity:%v,temperature:%v,timestamp:%v\n", cur.CO2, cur.Humidity, cur.Temperature, cur.Timestamp.Time.Format(time.RFC3339))

				cur = nil // dismiss
			case cur = <-r:
			}
		}
	}()

	// reader (main)
	re := regexp.MustCompile(`CO2=(\d+),HUM=([0-9\.]+),TMP=([0-9\.-]+)`)
	for s.Scan() {
		d := &CO2Data{}
		text := s.Text()
		m := re.FindAllStringSubmatch(text, -1)
		if len(m) > 0 {
			d.CO2, _ = strconv.ParseInt(m[0][1], 10, 64)
			d.Humidity, _ = strconv.ParseFloat(m[0][2], 64)
			d.Temperature, _ = strconv.ParseFloat(m[0][3], 64)
			d.Timestamp = pgtype.Timestamp{Time: time.Now(), Valid: true}
			r <- d
		} else if text[:6] == `OK STP` {
			return // exit 0
		} else {
			logWarningf("Read unmatched string: %v", text)
		}
	}
	if s.Err() != nil {
		logError(s.Err().Error())
	}
}
