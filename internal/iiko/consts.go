package iiko

const (
	DefaultScheme = "https"
	DefaultHost   = "pekarnya-gagarina.iiko.it"
	DefaultPort   = "443"
	DefaultPath   = "/resto"

	Login    = "Nikolay"
	Password = "3333"

	DefaultTimeout = 30 // seconds
)

const (
	// Assembly Charts
	EndpointAssemblyChartsGetAll       = "/resto/api/v2/assemblyCharts/getAll"
	EndpointAssemblyChartsGetAllUpdate = "/resto/api/v2/assemblyCharts/getAllUpdate"
	EndpointAssemblyChartsGetTree      = "/resto/api/v2/assemblyCharts/getTree"
	EndpointAssemblyChartsGetAssembled = "/resto/api/v2/assemblyCharts/getAssembled"
	EndpointAssemblyChartsGetPrepared  = "/resto/api/v2/assemblyCharts/getPrepared"
	EndpointAssemblyChartsByID         = "/resto/api/v2/assemblyCharts/byId"
	EndpointAssemblyChartsGetHistory   = "/resto/api/v2/assemblyCharts/getHistory"
)
