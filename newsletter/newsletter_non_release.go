// +build !release

package newsletter

import (
	"log"
)

type dummySubscriber struct{}

func (*dummySubscriber) Subscribe(email string) error {
	log.Println("A dummy call that would otherwise subscribe email", email, "to Lightmeter newsletter :-)")
	return nil
}

func NewSubscriber(string) Subscriber {
	return &dummySubscriber{}
}
