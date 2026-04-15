package domain

type Ingredient interface {
	Name() string
	Quantity() float64
	Unit() string
	Gross() float64
	Net() float64
	Ingredients() []Ingredient
}

type RawIngredient struct {
	IName     string  `json:"name"`
	IQuantity float64 `json:"quantity"`
	IUnit     string  `json:"unit"`
	IGross    float64 `json:"gross"`
	INet      float64 `json:"net"`
}

func (r RawIngredient) Name() string              { return r.IName }
func (r RawIngredient) Quantity() float64         { return r.IQuantity }
func (r RawIngredient) Unit() string              { return r.IUnit }
func (r RawIngredient) Gross() float64            { return r.IGross }
func (r RawIngredient) Net() float64              { return r.INet }
func (r RawIngredient) Ingredients() []Ingredient { return nil }

type SubProduct struct {
	IName        string       `json:"name"`
	IQuantity    float64      `json:"quantity"`
	IUnit        string       `json:"unit"`
	IGross       float64      `json:"gross"`
	INet         float64      `json:"net"`
	IIngredients []Ingredient `json:"ingredients,omitempty"`
}

func (s SubProduct) Name() string              { return s.IName }
func (s SubProduct) Quantity() float64         { return s.IQuantity }
func (s SubProduct) Unit() string              { return s.IUnit }
func (s SubProduct) Gross() float64            { return s.IGross }
func (s SubProduct) Net() float64              { return s.INet }
func (s SubProduct) Ingredients() []Ingredient { return s.IIngredients }

type Product struct {
	Name        string       `json:"name"`
	Quantity    float64      `json:"quantity"`
	Unit        string       `json:"unit"`
	Ingredients []Ingredient `json:"ingredients,omitempty"`
}
type ProductRepository interface {
	Save(p Product) error
	// Get возвращает продукт, масштабированный на quantity кг.
	Get(name string, quantity float64) (Product, error)
	// GetBase возвращает базовый рецепт продукта (на 1 штуку) без масштабирования.
	GetBase(name string) (Product, error)
	List() ([]string, error)
}
