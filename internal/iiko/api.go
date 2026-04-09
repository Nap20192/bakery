package iiko

import (
	"crypto/sha1"
	"fmt"
)

type Api struct {
	Host string
	Port string
}

func NewApi(host, port string) *Api {
	return &Api{
		Host: host,
		Port: port,
	}
}

func (s *Api) base() string {
	return fmt.Sprintf("%s:%s", s.Host, s.Port)
}

// POST /api/0/auth/access_token
func (s *Api) AuthURL(login, password string) string {
	h := sha1.New()
	h.Write([]byte(password))
	hash := fmt.Sprintf("%x", h.Sum(nil))
	return fmt.Sprintf("%s/resto/api/auth?login=%s&pass=%s", s.base(), login, hash)
}

// GET /resto/api/v2/entities/products/list
func (s *Api) ProductsListURL() string {
	return fmt.Sprintf("%s/resto/api/v2/entities/products/list", s.base())
}

// GET /resto/api/logout?key=...  (key via query param)
func (s *Api) LogoutByKeyURL(key string) string {
	return fmt.Sprintf("%s/resto/api/logout?key=%s", s.base(), key)
}

// GET /resto/api/logout  (key via Cookie header — caller must set Cookie: key=...)
func (s *Api) LogoutURL() string {
	return fmt.Sprintf("%s/resto/api/logout", s.base())
}

// GET /assemblyCharts/getAll
func (s *Api) AssemblyChartsGetAllURL(dateFrom, dateTo string, includeDeleted, includePrepared bool) string {
	return fmt.Sprintf("%s%s?dateFrom=%s&dateTo=%s&includeDeletedProducts=%t&includePreparedCharts=%t",
		s.base(), EndpointAssemblyChartsGetAll, dateFrom, dateTo, includeDeleted, includePrepared)
}

// GET /assemblyCharts/getAllUpdate
func (s *Api) AssemblyChartsGetAllUpdateURL(knownRevision int, dateFrom, dateTo string, includeDeleted, includePrepared bool) string {
	return fmt.Sprintf("%s%s?knownRevision=%d&dateFrom=%s&dateTo=%s&includeDeletedProducts=%t&includePreparedCharts=%t",
		s.base(), EndpointAssemblyChartsGetAllUpdate, knownRevision, dateFrom, dateTo, includeDeleted, includePrepared)
}

// GET /assemblyCharts/getTree
func (s *Api) AssemblyChartsGetTreeURL(date, productID, departmentID string) string {
	return fmt.Sprintf("%s%s?date=%s&productId=%s&departmentId=%s",
		s.base(), EndpointAssemblyChartsGetTree, date, productID, departmentID)
}

// GET /assemblyCharts/getAssembled
func (s *Api) AssemblyChartsGetAssembledURL(date, productID, departmentID string) string {
	return fmt.Sprintf("%s%s?date=%s&productId=%s&departmentId=%s",
		s.base(), EndpointAssemblyChartsGetAssembled, date, productID, departmentID)
}

// GET /assemblyCharts/getPrepared
func (s *Api) AssemblyChartsGetPreparedURL(date, productID, departmentID string) string {
	return fmt.Sprintf("%s%s?date=%s&productId=%s&departmentId=%s",
		s.base(), EndpointAssemblyChartsGetPrepared, date, productID, departmentID)
}

// GET /assemblyCharts/byId
func (s *Api) AssemblyChartsByIDURL(id string) string {
	return fmt.Sprintf("%s%s?id=%s", s.base(), EndpointAssemblyChartsByID, id)
}

// GET /assemblyCharts/getHistory
func (s *Api) AssemblyChartsGetHistoryURL(productID, departmentID string) string {
	return fmt.Sprintf("%s%s?productId=%s&departmentId=%s",
		s.base(), EndpointAssemblyChartsGetHistory, productID, departmentID)
}
