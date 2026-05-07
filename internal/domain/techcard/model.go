package techcard

import "bakery/internal/outbound/iiko"

type TechCard struct {
	Code     string
	Name     string
	Type     string
	Unit     string
	Assembly *iiko.AssemblyChartDto
	Prepared *iiko.PreparedChartDto
	Products map[string]TechCardProduct
}

type TechCardProduct struct {
	ID   string
	Code string
	Name string
	Type string
	Unit string
}
