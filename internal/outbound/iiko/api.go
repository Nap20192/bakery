package iiko

import (
	"crypto/sha1" //nolint:gosec // iiko API requires SHA-1 password hash.
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
	return fmt.Sprintf("https://%s:%s", s.Host, s.Port)
}

func (s *Api) AuthURL(login, password string) string {
	// iiko expects the password to be sent as a SHA-1 hash.
	//nolint:gosec // this is required by the external iiko API contract.
	h := sha1.New()
	h.Write([]byte(password))
	hash := fmt.Sprintf("%x", h.Sum(nil))
	return fmt.Sprintf("%s/resto/api/auth?login=%s&pass=%s", s.base(), login, hash)
}

func (s *Api) ProductsListURL() string {
	return fmt.Sprintf("%s/resto/api/v2/entities/products/list", s.base())
}

func (s *Api) LogoutByKeyURL(key string) string {
	return fmt.Sprintf("%s/resto/api/logout?key=%s", s.base(), key)
}

func (s *Api) LogoutURL() string {
	return fmt.Sprintf("%s/resto/api/logout", s.base())
}

func (s *Api) AssemblyChartsGetAllURL(dateFrom, dateTo string, includeDeleted, includePrepared bool) string {
	return fmt.Sprintf("%s%s?dateFrom=%s&dateTo=%s&includeDeletedProducts=%t&includePreparedCharts=%t",
		s.base(), EndpointAssemblyChartsGetAll, dateFrom, dateTo, includeDeleted, includePrepared)
}

func (s *Api) AssemblyChartsGetAssembledURL(date, productID, departmentID string) string {
	return fmt.Sprintf("%s%s?date=%s&productId=%s&departmentId=%s",
		s.base(), EndpointAssemblyChartsGetAssembled, date, productID, departmentID)
}

func (s *Api) AssemblyChartsGetPreparedURL(date, productID, departmentID string) string {
	return fmt.Sprintf("%s%s?date=%s&productId=%s&departmentId=%s",
		s.base(), EndpointAssemblyChartsGetPrepared, date, productID, departmentID)
}

func (s *Api) AssemblyChartsByIDURL(id string) string {
	return fmt.Sprintf("%s%s?id=%s", s.base(), EndpointAssemblyChartsByID, id)
}
