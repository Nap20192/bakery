package iiko

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
)

type Client struct {
	login    string
	password string
	key      string
	api      *Api
	client   *http.Client
}

func NewClient(login, password string, api *Api) (*Client, error) {
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	if err != nil {
		return nil, fmt.Errorf("iiko: cookiejar: %w", err)
	}

	return &Client{
		login:    login,
		password: password,
		api:      api,
		client: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (c *Client) Auth() error {

	url := c.api.AuthURL(c.login, c.password)

	resp, err := c.client.Get(url)

	if err != nil {
		return fmt.Errorf("iiko: auth request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("iiko: auth: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("iiko: auth read body: %w", err)
	}

	c.key = strings.TrimSpace(string(body))
	if c.key == "" {
		return fmt.Errorf("iiko: auth: empty token")
	}

	return nil
}

func (c *Client) Logout() error {
	if c.key == "" {
		return fmt.Errorf("iiko: logout: not authenticated")
	}

	resp, err := c.client.Get(c.api.LogoutByKeyURL(c.key))

	if err != nil {
		return fmt.Errorf("iiko: logout request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("iiko: logout: unexpected status %d", resp.StatusCode)
	}
	c.key = ""

	return nil
}

func (c *Client) ListProducts() ([]Product, error) {
	if c.key == "" {
		return nil, fmt.Errorf("iiko: list products: not authenticated")
	}

	resp, err := c.client.Get(c.api.ProductsListURL())
	if err != nil {
		return nil, fmt.Errorf("iiko: list products request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("iiko: list products: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("iiko: list products read body: %w", err)
	}

	var products []Product
	err = json.Unmarshal(body, &products)
	if err != nil {
		return nil, fmt.Errorf("iiko: list products unmarshal: %w", err)
	}

	return products, nil
}
