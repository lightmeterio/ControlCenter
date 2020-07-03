// +build release

package newsletter

func NewSubscriber(url string) Subscriber {
	return &HTTPSubscriber{URL: url}
}
