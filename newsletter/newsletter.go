package newsletter

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

var ErrSubscribingToNewsletter = errors.New(`Failed to Subscribe to newsletter`)

type Subscriber interface {
	Subscribe(context context.Context, email string) error
}

type HTTPSubscriber struct {
	URL string
}

func encodeBody(reader io.Reader) (string, error) {
	body, err := ioutil.ReadAll(reader)

	if err != nil {
		return "", errorutil.WrapError(err)
	}

	var base64Writer bytes.Buffer

	encoder := base64.NewEncoder(base64.StdEncoding, &base64Writer)

	if _, err := encoder.Write(body); err != nil {
		return "", errorutil.WrapError(err)
	}

	if err := encoder.Close(); err != nil {
		return "", errorutil.WrapError(err)
	}

	return base64Writer.String(), nil
}

func (s *HTTPSubscriber) Subscribe(context context.Context, email string) error {
	data := url.Values{}

	data.Set("email", email)
	data.Set("htmlemail", "1")
	data.Set("list[11]", "signup")
	data.Set("subscribe", "subscribe")

	content := strings.NewReader(data.Encode())

	req, err := http.NewRequestWithContext(context, "POST", s.URL+"/lists/?p=asubscribe&id=2", content)

	if err != nil {
		return errorutil.WrapError(err)
	}

	req.Header.Set("Content-Type", `application/x-www-form-urlencoded; charset=UTF-8`)

	client := &http.Client{}

	res, err := client.Do(req)

	if err != nil {
		return errorutil.WrapError(err)
	}

	defer func() { errorutil.MustSucceed(res.Body.Close(), "") }()

	body, err := encodeBody(res.Body)

	if err != nil {
		return errorutil.WrapError(err)
	}

	log.Println("Subscribe response:", body)

	if res.StatusCode != http.StatusOK {
		log.Println("Error subscribing email to newsletter!")
		return errorutil.WrapError(ErrSubscribingToNewsletter)
	}

	log.Println("Successfully subscribed email to newsletter!")

	return nil
}
