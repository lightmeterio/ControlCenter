package logsource

import (
	"gitlab.com/lightmeter/controlcenter/data"
)

type Source interface {
	PublishLogs(data.Publisher) error
}
