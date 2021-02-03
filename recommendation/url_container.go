// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package recommendation

type Link struct {
	Link string `json:"link"`
	ID   string `json:"id"`
}

type URLContainer interface {
	Get(k string) string
	Set(k string, v string)
	SetForEach(links []Link)
}

func NewURLContainer() URLContainer {
	return &uRLContainer{
		links: map[string]string{},
	}
}

type uRLContainer struct {
	links map[string]string
}

// Get an url from the URL container. Returns the url or empty string
func (c *uRLContainer) Get(k string) string {
	link, found := c.links[k]
	if !found {
		return ""
	}

	return link
}

// Add an url to the url container, replacing any existing url
func (c *uRLContainer) Set(k string, url string) {
	c.links[k] = url
}

func (c *uRLContainer) SetForEach(links []Link) {
	for _, link := range links {
		c.Set(link.ID, link.Link)
	}
}
