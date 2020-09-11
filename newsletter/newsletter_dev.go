// +build dev !release

package newsletter

import (
	"context"
	"log"
)

type dummySubscriber struct{}

func (*dummySubscriber) Subscribe(context context.Context, email string) error {
	log.Println("A dummy call that would otherwise subscribe email", email, "to Lightmeter newsletter :-)")
	return nil
}

func NewSubscriber(string) Subscriber {
	return &dummySubscriber{}
}
