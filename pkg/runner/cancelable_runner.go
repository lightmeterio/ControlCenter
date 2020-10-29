package runner

type CancelChan chan struct{}
type DoneChan chan struct{}

type CancelableRunner interface {
	Run() (done func(), cancel func())
}

func NewCancelableRunner(execute func(done DoneChan, cancel CancelChan)) CancelableRunner {
	return &cancelableRunner{
		execute: execute,
	}
}

type cancelableRunner struct {
	execute func(done DoneChan, cancel CancelChan)
}

func (r *cancelableRunner) Run() (func(), func()) {
	cancel := make(CancelChan)
	done := make(DoneChan)

	r.execute(done, cancel)

	return func() {
			<-done
		}, func() {
			cancel <- struct{}{}
		}
}
