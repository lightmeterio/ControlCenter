package filelogsource

import (
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/logeater"
	"os"
	"time"
)

type Source struct {
	file        *os.File
	initialTime time.Time
	year        int
}

func New(file *os.File, initialTime time.Time, year int) (*Source, error) {
	return &Source{
		file:        file,
		initialTime: initialTime,
		year:        year,
	}, nil
}

func (s *Source) PublishLogs(p data.Publisher) error {
	initialLogsTime := logeater.BuildInitialLogsTime(s.initialTime, s.year)
	logeater.ParseLogsFromReader(p, initialLogsTime, s.file)

	return nil
}
