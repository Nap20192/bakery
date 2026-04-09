package iiko

import "github.com/google/uuid"

type NomenclatureRequest struct {
	OrganizationID uuid.UUID `json:"organizationId"`
	StartRevision  int64     `json:"startRevision,omitempty"`
}

type TechCardRequest struct {
	OrganizationID  uuid.UUID   `json:"organizationId"`
	NomenclatureIDs []uuid.UUID `json:"nomenclatureItemIds,omitempty"` // фильтр по блюдам
}

type NomenclatureResponse struct {
	CorrelationID     uuid.UUID         `json:"correlationId"`
	Groups            []Group           `json:"groups"`
	ProductCategories []ProductCategory `json:"productCategories"`
	Products          []Product         `json:"products"`
	Sizes             []Size            `json:"sizes"`
	Revision          int64             `json:"revision"`
}

type Group struct {
	ID               uuid.UUID `json:"id"`
	Code             string    `json:"code"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	AdditionalInfo   string    `json:"additionalInfo"`
	ParentGroup      string    `json:"parentGroup"`
	ImageLinks       []string  `json:"imageLinks"`
	Tags             []string  `json:"tags"`
	Order            int       `json:"order"`
	IsIncludedInMenu bool      `json:"isIncludedInMenu"`
	IsGroupModifier  bool      `json:"isGroupModifier"`
	IsDeleted        bool      `json:"isDeleted"`
	SeoDescription   string    `json:"seoDescription"`
	SeoText          string    `json:"seoText"`
	SeoKeywords      string    `json:"seoKeywords"`
	SeoTitle         string    `json:"seoTitle"`
}

type ProductCategory struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	IsDeleted bool      `json:"isDeleted"`
}

type Product struct {
	ID                      uuid.UUID       `json:"id"`
	Code                    string          `json:"code"`
	Name                    string          `json:"name"`
	Description             string          `json:"description"`
	AdditionalInfo          string          `json:"additionalInfo"`
	Tags                    []string        `json:"tags"`
	GroupID                 uuid.UUID       `json:"groupId"`
	ProductCategoryID       uuid.UUID       `json:"productCategoryId"`
	Type                    string          `json:"type"`          // "Dish", "Good", "Modifier"
	OrderItemType           string          `json:"orderItemType"` // "Product", "Modifier"
	MeasureUnit             string          `json:"measureUnit"`   // "кг", "шт", "л"
	Weight                  float64         `json:"weight"`
	FatAmount               float64         `json:"fatAmount"`
	ProteinsAmount          float64         `json:"proteinsAmount"`
	CarbohydratesAmount     float64         `json:"carbohydratesAmount"`
	EnergyAmount            float64         `json:"energyAmount"`
	FatFullAmount           float64         `json:"fatFullAmount"`
	ProteinsFullAmount      float64         `json:"proteinsFullAmount"`
	CarbohydratesFullAmount float64         `json:"carbohydratesFullAmount"`
	EnergyFullAmount        float64         `json:"energyFullAmount"`
	SizePrices              []SizePrices    `json:"sizePrices"`
	Modifiers               []Modifier      `json:"modifiers"`
	GroupModifiers          []GroupModifier `json:"groupModifiers"`
	ImageLinks              []string        `json:"imageLinks"`
	ParentGroup             string          `json:"parentGroup"`
	Order                   int             `json:"order"`
	FullNameEnglish         string          `json:"fullNameEnglish"`
	IsDeleted               bool            `json:"isDeleted"`
	DoNotPrintInCheque      bool            `json:"doNotPrintInCheque"`
	UseBalanceForSell       bool            `json:"useBalanceForSell"`
	CanSetOpenPrice         bool            `json:"canSetOpenPrice"`
	Splittable              bool            `json:"splittable"`
	ModifierSchemaID        *uuid.UUID      `json:"modifierSchemaId,omitempty"`
	ModifierSchemaName      string          `json:"modifierSchemaName"`
	SeoDescription          string          `json:"seoDescription"`
	SeoText                 string          `json:"seoText"`
	SeoKeywords             string          `json:"seoKeywords"`
	SeoTitle                string          `json:"seoTitle"`
}

type Size struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Priority  int       `json:"priority"`
	IsDefault bool      `json:"isDefault"`
}

type Price struct {
	CurrentPrice       float64 `json:"currentPrice"`
	IsIncludedInMenu   bool    `json:"isIncludedInMenu"`
	NextPrice          float64 `json:"nextPrice"`
	NextIncludedInMenu bool    `json:"nextIncludedInMenu"`
	NextDatePrice      string  `json:"nextDatePrice"`
}

type SizePrices struct {
	SizeID uuid.UUID `json:"sizeId"`
	Price  Price     `json:"price"`
}

type Modifier struct {
	ID                  uuid.UUID `json:"id"`
	DefaultAmount       int       `json:"defaultAmount"`
	MinAmount           int       `json:"minAmount"`
	MaxAmount           int       `json:"maxAmount"`
	Required            bool      `json:"required"`
	HideIfDefaultAmount bool      `json:"hideIfDefaultAmount"`
	Splittable          bool      `json:"splittable"`
	FreeOfChargeAmount  int       `json:"freeOfChargeAmount"`
}

type GroupModifier struct {
	ID                                   uuid.UUID  `json:"id"`
	MinAmount                            int        `json:"minAmount"`
	MaxAmount                            int        `json:"maxAmount"`
	Required                             bool       `json:"required"`
	DefaultAmount                        int        `json:"defaultAmount"`
	Splittable                           bool       `json:"splittable"`
	FreeOfChargeAmount                   int        `json:"freeOfChargeAmount"`
	HideIfDefaultAmount                  bool       `json:"hideIfDefaultAmount"`
	ChildModifiersHaveMinMaxRestrictions bool       `json:"childModifiersHaveMinMaxRestrictions"`
	ChildModifiers                       []Modifier `json:"childModifiers"`
}

type TechCardResponse struct {
	CorrelationID uuid.UUID  `json:"correlationId"`
	TechCards     []TechCard `json:"techCards"`
}

type TechCard struct {
	ItemID   uuid.UUID `json:"itemId"`
	ItemName string    `json:"itemName"`

	Versions []TechCardVersion `json:"techCardVersions"`
}

type TechCardVersion struct {
	ID          uuid.UUID      `json:"id"`
	Name        string         `json:"name"`        // название версии
	IsBasic     bool           `json:"isBasic"`     // базовая версия
	OutputValue float64        `json:"outputValue"` // выход (кол-во готового продукта)
	OutputUnit  string         `json:"outputUnit"`  // единица измерения выхода
	Ingredients []TechCardItem `json:"items"`       // состав
	Modifiers   []TechCardItem `json:"modifiers"`   // модификаторы/добавки
}

type TechCardItem struct {
	ProductID   uuid.UUID `json:"productId"`
	ProductName string    `json:"productName"`

	GrossAmount float64 `json:"grossAmount"` // брутто (до обработки)
	NetAmount   float64 `json:"netAmount"`   // нетто (после обработки)
	Unit        string  `json:"unit"`        // единица: "кг", "г", "л", "шт"

	ColdLossPercent float64 `json:"coldLossPercent"` // потери при холодной обработке %
	HotLossPercent  float64 `json:"hotLossPercent"`  // потери при тепловой обработке %

	IsSemifinished bool `json:"isSemifinished"`

	IsRequired bool `json:"isRequired"`
	IsHidden   bool `json:"isHidden"`
	MinAmount  int  `json:"minAmount"`
	MaxAmount  int  `json:"maxAmount"`
}
