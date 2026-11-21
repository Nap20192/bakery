package iiko

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

type IikoClient interface {
	Auth(ctx context.Context) error
	Logout(ctx context.Context) error
	ListProducts(ctx context.Context) ([]Product, error)
	ListProductsWithCategories(ctx context.Context) (*NomenclatureResponse, error)
	AssemblyChartsGetAll(ctx context.Context, dateFrom, dateTo string, includeDeleted, includePrepared bool) (*ChartResultDto, error)
	AssemblyChartByID(ctx context.Context, id string) (*ChartResultDto, error)
	AssemblyChartsGetPrepared(ctx context.Context, date, productID, departmentID string) (*ChartResultDto, error)
}

type Client struct {
	login    string
	password string
	key      string
	api      *Api
	client   *http.Client
}

func NewClient(login, password string, api *Api) (*Client, error) {
	jar, err := cookiejar.New(nil)

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

func (c *Client) Auth(ctx context.Context) error {
	url := c.api.AuthURL(c.login, c.password)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("iiko: build auth request: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("iiko: auth request: %w", err)
	}
	defer closeBody(resp.Body)

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

func (c *Client) Logout(ctx context.Context) error {
	if c.key == "" {
		return fmt.Errorf("iiko: logout: not authenticated")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.api.LogoutByKeyURL(c.key), nil)
	if err != nil {
		return fmt.Errorf("iiko: build logout request: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("iiko: logout request: %w", err)
	}
	defer closeBody(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("iiko: logout: unexpected status %d", resp.StatusCode)
	}
	c.key = ""

	return nil
}

// get выполняет авторизованный GET и возвращает тело ответа.
// Требует, чтобы Auth() был вызван ранее (c.key != "").
func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	if c.key == "" {
		return nil, fmt.Errorf("iiko: not authenticated")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("iiko: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("iiko: request: %w", err)
	}
	defer closeBody(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("iiko: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("iiko: unexpected status %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func closeBody(body io.Closer) {
	if err := body.Close(); err != nil {
		return
	}
}

func (c *Client) getJSON(ctx context.Context, url string, out any) error {
	body, err := c.get(ctx, url)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("iiko: unmarshal: %w", err)
	}
	return nil
}

// ListProducts — GET /resto/api/v2/entities/products/list.
func (c *Client) ListProducts(ctx context.Context) ([]Product, error) {
	var products []Product
	if err := c.getJSON(ctx, c.api.ProductsListURL(), &products); err != nil {
		return nil, err
	}
	return products, nil
}

// ListProductsWithCategories — GET /resto/api/v2/entities/products/list.
// Возвращает продукты и категории.
// Поддерживает оба формата ответа iiko:
// 1) NomenclatureResponse (products + productCategories)
// 2) []Product (без productCategories)
func (c *Client) ListProductsWithCategories(ctx context.Context) (*NomenclatureResponse, error) {
	body, err := c.get(ctx, c.api.ProductsListURL())
	if err != nil {
		return nil, err
	}

	var full NomenclatureResponse
	if err := json.Unmarshal(body, &full); err == nil && (full.Products != nil || full.ProductCategories != nil) {
		return &full, nil
	}

	var products []Product
	if err := json.Unmarshal(body, &products); err == nil {
		return &NomenclatureResponse{
			Products: products,
		}, nil
	}

	var wrappedFull NomenclatureAPIResponse
	if err := json.Unmarshal(body, &wrappedFull); err == nil && (wrappedFull.Response.Products != nil || wrappedFull.Response.ProductCategories != nil) {
		return &wrappedFull.Response, nil
	}

	var wrappedProducts ProductsAPIResponse
	if err := json.Unmarshal(body, &wrappedProducts); err == nil && wrappedProducts.Response != nil {
		return &NomenclatureResponse{
			Products: wrappedProducts.Response,
		}, nil
	}

	return nil, fmt.Errorf("iiko: unmarshal products with categories: unsupported payload")
}

// AssemblyChartsGetAll — GET /resto/api/v2/assemblyCharts/getAll.
// Возвращает все техкарты, действующие в диапазоне дат.
// dateFrom/dateTo в формате yyyy-MM-dd.
func (c *Client) AssemblyChartsGetAll(ctx context.Context, dateFrom, dateTo string, includeDeleted, includePrepared bool) (*ChartResultDto, error) {
	var result ChartResultDto
	if err := c.getJSON(ctx, c.api.AssemblyChartsGetAllURL(dateFrom, dateTo, includeDeleted, includePrepared), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// AssemblyChartByID — GET /resto/api/v2/assemblyCharts/byId.
func (c *Client) AssemblyChartByID(ctx context.Context, id string) (*ChartResultDto, error) {
	if id == "" {
		return nil, fmt.Errorf("iiko: assemblyCharts/byId: empty id")
	}
	var result ChartResultDto
	if err := c.getJSON(ctx, c.api.AssemblyChartsByIDURL(id), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// AssemblyChartsGetAssembled — GET /resto/api/v2/assemblyCharts/getAssembled.
// Возвращает исходную техкарту блюда на дату.
func (c *Client) AssemblyChartsGetAssembled(ctx context.Context, date, productID, departmentID string) (*ChartResultDto, error) {
	var result ChartResultDto
	if err := c.getJSON(ctx, c.api.AssemblyChartsGetAssembledURL(date, productID, departmentID), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// AssemblyChartsGetPrepared — GET /resto/api/v2/assemblyCharts/getPrepared.
// Возвращает техкарту, разложенную до конечных ингредиентов — это то, что
// нужно для расчёта теста по заявке.
func (c *Client) AssemblyChartsGetPrepared(ctx context.Context, date, productID, departmentID string) (*ChartResultDto, error) {
	var result ChartResultDto
	if err := c.getJSON(ctx, c.api.AssemblyChartsGetPreparedURL(date, productID, departmentID), &result); err != nil {
		return nil, err
	}
	return &result, nil
}
