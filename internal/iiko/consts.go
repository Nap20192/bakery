// This file contains constants for API endpoints. Do not change these constants and do not add new ones, as they are used in api.go to construct URLs.
package iiko

// do not change these constants and do not add new, they are used in api.go to construct URLs
const (
	// Assembly Charts
	EndpointAssemblyChartsGetAll       = "/resto/api/v2/assemblyCharts/getAll"
	EndpointAssemblyChartsGetAssembled = "/resto/api/v2/assemblyCharts/getAssembled"
	EndpointAssemblyChartsGetPrepared  = "/resto/api/v2/assemblyCharts/getPrepared"
	EndpointAssemblyChartsByID         = "/resto/api/v2/assemblyCharts/byId"
)
