package domain

import "time"

type IikoSyncRun struct {
	ID            int64
	Source        string
	DateFrom      string
	DateTo        string
	KnownRevision int
	Status        string
	Error         string
	StartedAt     time.Time
	FinishedAt    *time.Time
}

type IikoProduct struct {
	ID                string
	Code              string
	Name              string
	Type              string
	OrderItemType     string
	MeasureUnit       string
	ProductCategoryID string
	GroupID           string
	ParentGroup       string
	IsDeleted         bool
	RawJSON           string
	UpdatedAt         time.Time
}

type StoreSpecificationValue struct {
	Departments []string
	Inverse     bool
}

type ProductSizeSpecificationValue struct {
	SizeID string
}

type IikoAssemblyChart struct {
	ID                                            string
	AssembledProductID                            string
	DateFrom                                      string
	DateTo                                        *string
	AssembledAmount                               float64
	ProductWriteoffStrategy                       string
	ProductSizeAssemblyStrategy                   string
	EffectiveDirectWriteoffStoreSpecificationJSON string
	TechnologyDescription                         string
	Description                                   string
	Appearance                                    string
	Organoleptic                                  string
	OutputComment                                 string
	RawJSON                                       string
	UpdatedAt                                     time.Time
}

type IikoAssemblyChartItem struct {
	ID                           string
	ChartID                      string
	SortWeight                   float64
	ProductID                    string
	ProductSizeSpecificationJSON string
	StoreSpecificationJSON       string
	AmountIn                     float64
	AmountMiddle                 float64
	AmountOut                    float64
	AmountIn1                    float64
	AmountOut1                   float64
	AmountIn2                    float64
	AmountOut2                   float64
	AmountIn3                    float64
	AmountOut3                   float64
	PackageCount                 float64
	PackageTypeID                *string
	RawJSON                      string
}

type IikoPreparedChart struct {
	ID                                            string
	AssembledProductID                            string
	DateFrom                                      string
	DateTo                                        *string
	ProductSizeAssemblyStrategy                   string
	EffectiveDirectWriteoffStoreSpecificationJSON string
	RawJSON                                       string
	UpdatedAt                                     time.Time
}

type IikoPreparedChartItem struct {
	ID                           string
	PreparedChartID              string
	SortWeight                   float64
	ProductID                    string
	ProductSizeSpecificationJSON string
	StoreSpecificationJSON       string
	Amount                       float64
	RawJSON                      string
}
