package main

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"regexp"
)

const (
	// NOTE: adapted from https://github.com/youyo/postfix-log-parser.git
	timeSmtpSentStatusRegexpFormat = `(?P<Time>[A-Za-z]{3}\s\s?[0-9]{1,2} [0-9]{2}:[0-9]{2}:[0-9]{2})`
	hostSmtpSentStatusRegexpFormat = `(?P<Host>[0-9A-Za-z\.]+)`
	// TODO: the process name can have more slash separated components, such as: postfix/submission/smtpd
	processSmtpSentStatusRegexpFormat = `(postfix(-[^/]+)?/(?P<Process>[a-z]+)\[[0-9]{1,5}\])`
	queueIdSmtpSentStatusRegexpFormat = `(?P<Queue>[0-9A-F]+)`

	procRegexpFormat = `^` + timeSmtpSentStatusRegexpFormat + ` ` + hostSmtpSentStatusRegexpFormat + ` ` + processSmtpSentStatusRegexpFormat + `: `

	anythingExceptCommaRegexpFormat = `[^,]+`

	relayComponentsRegexpFormat = `(?P<RelayName>[^\,[]+)` + `\[(?P<RelayIp>[^\],]+)\]` + `:` + `(?P<RelayPort>[\d]+)`

	messageSentWithStatusSmtpSentStatusRegexpFormat = `(?P<MessageSentWithStatus>` +
		`to=<(?P<To>.+@.+)>` + `, ` +
		`relay=` + relayComponentsRegexpFormat + `, ` +
		`delay=(?P<Delay>` + anythingExceptCommaRegexpFormat + `)` + `, ` +
		`delays=(?P<Delays>` + anythingExceptCommaRegexpFormat + `)` + `, ` +
		`dsn=(?P<Dsn>` + anythingExceptCommaRegexpFormat + `)` + `, ` +
		`status=(?P<Status>[a-z]+)` + ` ` +
		`(?P<ExtraMessage>.*)` +
		`)`

	possiblePayloads = messageSentWithStatusSmtpSentStatusRegexpFormat

	smtpSentStatusRegexpFormat = `^` + queueIdSmtpSentStatusRegexpFormat + `: ` +
		`(` + possiblePayloads + `)`
)

type SmtpSentStatusPublisher interface {
	Publish(SmtpSentStatus)
	Close()
}

type ChannelBasedSmtpSentStatusPublisher struct {
	channel chan []byte
}

func (pub *ChannelBasedSmtpSentStatusPublisher) Publish(status SmtpSentStatus) {
	// FIXME: obviously allocating new buffers and encoders on every message is bad for performance
	// but this is only testing code

	var encoderBuffer bytes.Buffer
	encoder := gob.NewEncoder(&encoderBuffer)

	if err := encoder.Encode(&status); err != nil {
		log.Fatal(err)
	}

	pub.channel <- encoderBuffer.Bytes()
}

func (pub *ChannelBasedSmtpSentStatusPublisher) Close() {
	close(pub.channel)
}

func main() {
	c := make(chan []byte, 10)
	pub := ChannelBasedSmtpSentStatusPublisher{c}
	go parseLogsFromStdin(&pub)

	for m := range c {
		// FIXME: obviously allocating new buffers and encoders on every message is bad for performance
		// but this is only testing code

		buffer := bytes.NewBuffer(m)
		decoder := gob.NewDecoder(buffer)
		var status SmtpSentStatus
		if err := decoder.Decode(&status); err != nil {
			log.Fatal(err)
		}

		fmt.Println("time:", string(status.Header.Time), ", queue:", string(status.Queue), ", to:", string(status.To), ", status:", string(status.Status), "relay name:", string(status.RelayName), ", ip:", string(status.RelayIp), ", port:", string(status.RelayPort))
	}
}

type LogHeader struct {
	Time    []byte
	Host    []byte
	Process []byte
}

type SmtpSentStatus struct {
	Header       LogHeader
	Queue        []byte
	To           []byte
	RelayName    []byte
	RelayIp      []byte
	RelayPort    []byte
	Delay        []byte
	Delays       []byte
	Dsn          []byte
	Status       []byte
	ExtraMessage []byte
}

func indexForGroup(smtpSentStatusRegexp *regexp.Regexp, name string) int {
	e := smtpSentStatusRegexp.SubexpNames()
	for i, v := range e {
		if v == name {
			return i
		}
	}

	panic("Wrong Group Name!")
}

func parseLogsFromStdin(publisher SmtpSentStatusPublisher) {
	scanner := bufio.NewScanner(os.Stdin)

	smtpSentStatusRegexp := regexp.MustCompile(smtpSentStatusRegexpFormat)
	procRegexp := regexp.MustCompile(procRegexpFormat)

	timeIndex := indexForGroup(procRegexp, "Time")
	hostIndex := indexForGroup(procRegexp, "Host")
	processIndex := indexForGroup(procRegexp, "Process")

	smtpQueueIndex := indexForGroup(smtpSentStatusRegexp, "Queue")
	smtpToIndex := indexForGroup(smtpSentStatusRegexp, "To")
	smtpRelayNameIndex := indexForGroup(smtpSentStatusRegexp, "RelayName")
	smtpRelayIpIndex := indexForGroup(smtpSentStatusRegexp, "RelayIp")
	smtpRelayPortIndex := indexForGroup(smtpSentStatusRegexp, "RelayPort")
	smtpDelayIndex := indexForGroup(smtpSentStatusRegexp, "Delay")
	smtpDelaysIndex := indexForGroup(smtpSentStatusRegexp, "Delays")
	smtpDsnIndex := indexForGroup(smtpSentStatusRegexp, "Dsn")
	smtpStatusIndex := indexForGroup(smtpSentStatusRegexp, "Status")
	smtpExtraMessageIndex := indexForGroup(smtpSentStatusRegexp, "ExtraMessage")

	for {
		if !scanner.Scan() {
			break
		}

		logLine := scanner.Bytes()

		headerMatches := procRegexp.FindSubmatch(logLine)

		if len(headerMatches) == 0 {
			continue
		}

		if bytes.Compare(headerMatches[processIndex], []byte("smtp")) != 0 {
			// TODO: implement support for other processes
			continue
		}

		linePayload := logLine[len(headerMatches[0]):]

		payloadMatches := smtpSentStatusRegexp.FindSubmatch(linePayload)

		if len(payloadMatches) == 0 {
			//fmt.Println("New smtp payload: ", string(linePayload))
			// TODO: implement other stuff done by the "smtp" process
			continue
		}

		s := SmtpSentStatus{
			Header: LogHeader{
				Time:    headerMatches[timeIndex],
				Host:    headerMatches[hostIndex],
				Process: headerMatches[processIndex],
			},

			Queue:        payloadMatches[smtpQueueIndex],
			To:           payloadMatches[smtpToIndex],
			RelayName:    payloadMatches[smtpRelayNameIndex],
			RelayIp:      payloadMatches[smtpRelayIpIndex],
			RelayPort:    payloadMatches[smtpRelayPortIndex],
			Delay:        payloadMatches[smtpDelayIndex],
			Delays:       payloadMatches[smtpDelaysIndex],
			Dsn:          payloadMatches[smtpDsnIndex],
			Status:       payloadMatches[smtpStatusIndex],
			ExtraMessage: payloadMatches[smtpExtraMessageIndex],
		}

		publisher.Publish(s)
	}

	publisher.Close()
}
