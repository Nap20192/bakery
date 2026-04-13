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

type IikoClient interface {
    Auth() error
    Logout() error
    ListProducts() ([]Product, error)
    AssemblyChartsGetAll(dateFrom, dateTo string, includeDeleted, includePrepared bool) (*ChartResultDto, error)
    AssemblyChartsGetAllUpdate(knownRevision int, dateFrom, dateTo string, includeDeleted, includePrepared bool) (*ChartResultDto, error)
    AssemblyChartByID(id string) (*ChartResultDto, error)
    AssemblyChartsGetTree(date, productID, departmentID string) (*ChartResultDto, error)
    AssemblyChartsGetAssembled(date, productID, departmentID string) (*ChartResultDto, error)
    AssemblyChartsGetPrepared(date, productID, departmentID string) (*ChartResultDto, error)
    AssemblyChartsGetHistory(productID, departmentID string) (*ChartResultDto, error)
}

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

// get выполняет авторизованный GET и возвращает тело ответа.
// Требует, чтобы Auth() был вызван ранее (c.key != "").
func (c *Client) get(url string) ([]byte, error) {
	if c.key == "" {
		return nil, fmt.Errorf("iiko: not authenticated")
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("iiko: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("iiko: request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("iiko: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("iiko: unexpected status %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

// getJSON — дженерик-хелпер: GET + JSON-декод.
func getJSON[T any](c *Client, url string) (T, error) {
	var out T
	body, err := c.get(url)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return out, fmt.Errorf("iiko: unmarshal: %w", err)
	}
	return out, nil
}

// ListProducts — GET /resto/api/v2/entities/products/list.
func (c *Client) ListProducts() ([]Product, error) {
	return getJSON[[]Product](c, c.api.ProductsListURL())
}

// AssemblyChartsGetAll — GET /resto/api/v2/assemblyCharts/getAll.
// Возвращает все техкарты, действующие в диапазоне дат.
// dateFrom/dateTo в формате yyyy-MM-dd.
func (c *Client) AssemblyChartsGetAll(dateFrom, dateTo string, includeDeleted, includePrepared bool) (*ChartResultDto, error) {
	r, err := getJSON[ChartResultDto](c, c.api.AssemblyChartsGetAllURL(dateFrom, dateTo, includeDeleted, includePrepared))
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// AssemblyChartsGetAllUpdate — GET /resto/api/v2/assemblyCharts/getAllUpdate.
// Инкрементальное обновление по knownRevision.
func (c *Client) AssemblyChartsGetAllUpdate(knownRevision int, dateFrom, dateTo string, includeDeleted, includePrepared bool) (*ChartResultDto, error) {
	r, err := getJSON[ChartResultDto](c, c.api.AssemblyChartsGetAllUpdateURL(knownRevision, dateFrom, dateTo, includeDeleted, includePrepared))
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// AssemblyChartByID — GET /resto/api/v2/assemblyCharts/byId.
func (c *Client) AssemblyChartByID(id string) (*ChartResultDto, error) {
	if id == "" {
		return nil, fmt.Errorf("iiko: assemblyCharts/byId: empty id")
	}
	r, err := getJSON[ChartResultDto](c, c.api.AssemblyChartsByIDURL(id))
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// AssemblyChartsGetTree — GET /resto/api/v2/assemblyCharts/getTree.
func (c *Client) AssemblyChartsGetTree(date, productID, departmentID string) (*ChartResultDto, error) {
	r, err := getJSON[ChartResultDto](c, c.api.AssemblyChartsGetTreeURL(date, productID, departmentID))
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// AssemblyChartsGetAssembled — GET /resto/api/v2/assemblyCharts/getAssembled.
// Возвращает исходную техкарту блюда на дату.
func (c *Client) AssemblyChartsGetAssembled(date, productID, departmentID string) (*ChartResultDto, error) {
	r, err := getJSON[ChartResultDto](c, c.api.AssemblyChartsGetAssembledURL(date, productID, departmentID))
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// AssemblyChartsGetPrepared — GET /resto/api/v2/assemblyCharts/getPrepared.
// Возвращает техкарту, разложенную до конечных ингредиентов — это то, что
// нужно для расчёта теста по заявке.
func (c *Client) AssemblyChartsGetPrepared(date, productID, departmentID string) (*ChartResultDto, error) {
	r, err := getJSON[ChartResultDto](c, c.api.AssemblyChartsGetPreparedURL(date, productID, departmentID))
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// AssemblyChartsGetHistory — GET /resto/api/v2/assemblyCharts/getHistory.
// История версий техкарты для блюда/подразделения.
func (c *Client) AssemblyChartsGetHistory(productID, departmentID string) (*ChartResultDto, error) {
	r, err := getJSON[ChartResultDto](c, c.api.AssemblyChartsGetHistoryURL(productID, departmentID))
	if err != nil {
		return nil, err
	}
	return &r, nil
}
