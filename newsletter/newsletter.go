package newsletter

import (
	"bytes"
	"errors"
	"gitlab.com/lightmeter/controlcenter/util"
	"log"
	"mime/multipart"
	"net/http"
)

var ErrSubscribingToNewsletter = errors.New(`Failed to Subscribe to newsletter`)

type Subscriber interface {
	Subscribe(email string) error
}

type HTTPSubscriber struct {
	URL string
}

func (s *HTTPSubscriber) Subscribe(email string) error {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	fw, err := w.CreateFormField("item_meta[14]")

	if err != nil {
		return util.WrapError(err)
	}

	if _, err := fw.Write([]byte(email)); err != nil {
		return util.WrapError(err)
	}

	if err := w.Close(); err != nil {
		return util.WrapError(err)
	}

	req, err := http.NewRequest("POST", s.URL, &b)

	if err != nil {
		return util.WrapError(err)
	}

	req.Header.Set("Content-Type", w.FormDataContentType())

	client := &http.Client{}

	res, err := client.Do(req)

	if err != nil {
		return util.WrapError(err)
	}

	if res.StatusCode != http.StatusOK {
		log.Println("Error subscribing email to newsletter!")
		return util.WrapError(ErrSubscribingToNewsletter)
	}

	log.Println("Successfully subscribed email to newsletter!")

	return nil
}
