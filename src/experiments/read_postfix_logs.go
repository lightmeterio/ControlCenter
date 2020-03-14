package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/gob"
	"encoding/json"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
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
		`to=<(?P<RecipientLocalPart>[^@]+)@(?P<RecipientDomainPart>[^>]+)>` + `, ` +
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

func fillDatabase(db *sql.DB, c chan []byte) {
	stmt, err := db.Prepare("insert into smtp(recipient_local_part, recipient_domain_part, relay_name, status) values(?, ?, ?, ?)")

	if err != nil {
		log.Fatal("error preparing insert statement")
	}

	for m := range c {
		// FIXME: obviously allocating new buffers and encoders on every message is bad for performance
		// but this is only testing code

		buffer := bytes.NewBuffer(m)
		decoder := gob.NewDecoder(buffer)
		var status SmtpSentStatus

		if err := decoder.Decode(&status); err != nil {
			log.Fatal(err)
		}

		_, err := stmt.Exec(status.RecipientLocalPart, status.RecipientDomainPart, status.RelayName, status.Status)

		if err != nil {
			log.Fatal("error inserting value")
		}
	}
}

func main() {
	c := make(chan []byte, 10)
	pub := ChannelBasedSmtpSentStatusPublisher{c}
	go parseLogsFromStdin(&pub)

	os.Remove("lm.db")

	db, err := sql.Open("sqlite3", "lm.db")

	if err != nil {
		log.Fatal("error opening database")
	}

	defer db.Close()

	if _, err := db.Exec("create table smtp(recipient_local_part text, recipient_domain_part text, relay_name text, status text)"); err != nil {
		log.Fatal("error creating database: ", err)
	}

	go fillDatabase(db, c)

	countByStatus := func(status string) int {
		stmt, err := db.Prepare(`select count(status) from smtp where cast(status as text) = ?`)

		if err != nil {
			log.Fatal("error preparing query")
		}

		sentResult, err := stmt.Query(status)

		if err != nil {
			log.Fatal("error querying")
		}

		var countValue int

		sentResult.Next()

		sentResult.Scan(&countValue)

		return countValue
	}

	type domainAndCount struct {
		Domain string
		Count  int
	}

	listDomainAndCount := func(queryStr string) []domainAndCount {
		var r []domainAndCount

		query, err := db.Query(queryStr)

		if err != nil {
			log.Fatal("Error query")
		}

		for query.Next() {
			var domain string
			var countValue int

			query.Scan(&domain, &countValue)

			r = append(r, domainAndCount{domain, countValue})
		}

		return r
	}

	type deliveryValue struct {
		Status string
		Value  float64
	}

	deliveryStatus := func() []deliveryValue {
		var r []deliveryValue

		query, err := db.Query(`select status, cast(count(status) as float) / cast((select count(status) from smtp) as float) from smtp group by status`)

		if err != nil {
			log.Fatal("Error query")
		}

		for query.Next() {
			var status string
			var value float64

			query.Scan(&status, &value)

			r = append(r, deliveryValue{status, value * 100})
		}

		return r
	}

	serveJson := func(w http.ResponseWriter, r *http.Request, v interface{}) {
		w.Header().Set("Content-Type", "application/json")
		encoded, _ := json.Marshal(v)
		w.Write(encoded)
	}

	http.HandleFunc("/countByStatus", func(w http.ResponseWriter, r *http.Request) {
		serveJson(w, r, map[string]int{"sent": countByStatus("sent"), "deferred": countByStatus("deferred"), "bounced": countByStatus("bounced")})
	})

	http.HandleFunc("/topBusiestDomains", func(w http.ResponseWriter, r *http.Request) {
		serveJson(w, r, listDomainAndCount(`select recipient_domain_part, count(recipient_domain_part) as c from smtp group by recipient_domain_part order by c desc limit 20`))
	})

	http.HandleFunc("/topBouncedDomains", func(w http.ResponseWriter, r *http.Request) {
		serveJson(w, r, listDomainAndCount(`select recipient_domain_part, count(recipient_domain_part) as c from smtp where cast(status as text) = "bounced" group by recipient_domain_part order by c desc limit 20`))
	})

	http.HandleFunc("/topDeferredDomains", func(w http.ResponseWriter, r *http.Request) {
		serveJson(w, r, listDomainAndCount(`select relay_name, count(relay_name) as c from smtp where cast(status as text) = "deferred" group by relay_name order by c desc limit 20`))
	})

	http.HandleFunc("/deliveryStatus", func(w http.ResponseWriter, r *http.Request) {
		serveJson(w, r, deliveryStatus())
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}

type LogHeader struct {
	Time    []byte
	Host    []byte
	Process []byte
}

type SmtpSentStatus struct {
	Header              LogHeader
	Queue               []byte
	RecipientLocalPart  []byte
	RecipientDomainPart []byte
	RelayName           []byte
	RelayIp             []byte
	RelayPort           []byte
	Delay               []byte
	Delays              []byte
	Dsn                 []byte
	Status              []byte
	ExtraMessage        []byte
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
	smtpRecipientLocalPartIndex := indexForGroup(smtpSentStatusRegexp, "RecipientLocalPart")
	smtpRecipientDomainPartIndex := indexForGroup(smtpSentStatusRegexp, "RecipientDomainPart")
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

			Queue:               payloadMatches[smtpQueueIndex],
			RecipientLocalPart:  payloadMatches[smtpRecipientLocalPartIndex],
			RecipientDomainPart: payloadMatches[smtpRecipientDomainPartIndex],
			RelayName:           payloadMatches[smtpRelayNameIndex],
			RelayIp:             payloadMatches[smtpRelayIpIndex],
			RelayPort:           payloadMatches[smtpRelayPortIndex],
			Delay:               payloadMatches[smtpDelayIndex],
			Delays:              payloadMatches[smtpDelaysIndex],
			Dsn:                 payloadMatches[smtpDsnIndex],
			Status:              payloadMatches[smtpStatusIndex],
			ExtraMessage:        payloadMatches[smtpExtraMessageIndex],
		}

		publisher.Publish(s)
	}

	publisher.Close()
}
