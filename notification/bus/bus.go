package bus

import (
	"errors"
	"fmt"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"reflect"
	"sync"
)

// The type of the function's first and only argument
// declares the msg to listen for.
type HandlerFunc interface{}

type Msg interface{}

// It is a simple but powerful publish-subscribe event system. It requires object to
// register themselves with the event bus to receive events.
type Interface interface {
	AddEventListener(handler HandlerFunc)
	Publish(msg Msg) error
}

type bus struct {
	listeners *sync.Map
	isInit    bool
}

func New() Interface {
	return &bus{
		listeners: new(sync.Map),
	}
}

// Publish sends an msg to all registered listeners that were declared
// to accept values of a msg
func (b *bus) Publish(msg Msg) error {
	if !b.isInit {
		return errorutil.Wrap(errors.New("listeners aren't registered"))
	}

	nameOfMsg := reflect.TypeOf(msg)

	val, ok := b.listeners.Load(nameOfMsg.String())
	if !ok {
		return nil
	}

	listeners := val.([]reflect.Value)

	params := make([]reflect.Value, 0, 1)
	params = append(params, reflect.ValueOf(msg))

	for _, listenerHandler := range listeners {
		ret := listenerHandler.Call(params)
		v := ret[0].Interface()

		if err, ok := v.(error); ok && err != nil {
			return errorutil.Wrap(err)
		}
	}

	return nil
}

// AddListener registers a listener function that will be called when a matching
// msg is dispatched.
func (b *bus) AddEventListener(handler HandlerFunc) {
	b.isInit = true

	handlerType := reflect.TypeOf(handler)
	validateHandlerFunc(handlerType)
	// the first input parameter is the msg
	typOfMsg := handlerType.In(0)

	listeners := make([]reflect.Value, 0)

	val, ok := b.listeners.Load(typOfMsg.String())
	if ok {
		listeners = val.([]reflect.Value)
	}

	listeners = append(listeners, reflect.ValueOf(handler))
	b.listeners.Store(typOfMsg.String(), listeners)
}

// panic if conditions not met (this is a programming error)
func validateHandlerFunc(handlerType reflect.Type) {
	switch {
	case handlerType.Kind() != reflect.Func:
		panic(BadFuncError("handler func must be a function"))
	case handlerType.NumIn() != 1:
		panic(BadFuncError("handler func must take exactly one input argument"))
	case handlerType.NumOut() != 1:
		panic(BadFuncError("handler func must take exactly one output argument"))
	}
}

// BadFuncError is raised via panic() when AddEventListener or AddHandler is called with an
// invalid listener function.
type BadFuncError string

func (bhf BadFuncError) Error() string {
	return fmt.Sprintf("bad handler func: %s", string(bhf))
}
