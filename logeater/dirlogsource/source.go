package dirlogsource

import (
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/logeater/dirwatcher"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

type Source struct {
	initialTime time.Time
	dir         dirwatcher.DirectoryContent
}

func New(dirname string, initialTime time.Time) (*Source, error) {
	dir, err := dirwatcher.NewDirectoryContent(dirname)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	func() {
		if initialTime.IsZero() {
			log.Info().Msg("Start importing Postfix logs directory into a new workspace")
			return
		}

		log.Info().Msgf("Importing Postfix logs directory from time %v", initialTime)
	}()

	return &Source{
		initialTime: initialTime,
		dir:         dir,
	}, nil
}

func (s *Source) PublishLogs(p data.Publisher) error {
	watcher := dirwatcher.NewDirectoryImporter(s.dir, p, s.initialTime)

	if err := watcher.Run(); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
