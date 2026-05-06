package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/joho/godotenv"

	"bakery/internal/config"
	"bakery/internal/domain"
	"bakery/internal/iiko"
)

type productExport struct {
	domain.Product
	Category   string `json:"category"`
	CategoryID string `json:"categoryId"`
}

func main() {
	_ = godotenv.Load()
	cfg := config.New()

	outPath := flag.String("out", "products.json", "path to output JSON file")
	date := flag.String("date", time.Now().Format("2006-01-02"), "tech card date (yyyy-MM-dd)")
	includeDeleted := flag.Bool("include-deleted", false, "include deleted products in iiko response")
	includePrepared := flag.Bool("include-prepared", true, "include prepared charts in iiko response")
	flag.Parse()

	if cfg.Iiko.Host == "" || cfg.Iiko.Login == "" || cfg.Iiko.Password == "" {
		log.Fatal("IIKO_HOST, IIKO_LOGIN and IIKO_PASSWORD must be set")
	}

	api := iiko.NewApi(cfg.Iiko.Host, itoa(cfg.Iiko.Port))
	client, err := iiko.NewClient(cfg.Iiko.Login, cfg.Iiko.Password, api)
	if err != nil {
		log.Fatalf("create iiko client: %v", err)
	}

	if err := client.Auth(); err != nil {
		log.Fatalf("iiko auth failed: %v", err)
	}

	defer func() {
		if err := client.Logout(); err != nil {
			log.Printf("iiko logout failed: %v", err)
		}
	}()

	catalog, err := client.ListProductsWithCategories()
	if err != nil {
		log.Fatalf("list iiko products with categories: %v", err)
	}
	products := catalog.Products

	chartResult, err := client.AssemblyChartsGetAll(*date, *date, *includeDeleted, *includePrepared)
	if err != nil {
		log.Fatalf("get iiko assembly charts: %v", err)
	}

	productsByID := make(map[string]iiko.Product, len(products))
	for _, p := range products {
		productsByID[p.ID.String()] = p
	}
	categoriesByID := make(map[string]string, len(catalog.ProductCategories))
	for _, c := range catalog.ProductCategories {
		categoriesByID[c.ID.String()] = c.Name
	}

	chartsByProduct := pickActiveCharts(chartResult.AssemblyCharts, *date)

	ids := make([]string, 0, len(chartsByProduct))
	for productID := range chartsByProduct {
		ids = append(ids, productID)
	}
	sort.Slice(ids, func(i, j int) bool {
		left := productDisplayName(ids[i], productsByID)
		right := productDisplayName(ids[j], productsByID)
		if left == right {
			return ids[i] < ids[j]
		}
		return left < right
	})

	out := make([]productExport, 0, len(ids))
	for _, productID := range ids {
		product := buildBaseProduct(productID, productsByID, chartsByProduct, map[string]bool{})
		categoryID := productsByID[productID].ProductCategoryID.String()
		out = append(out, productExport{
			Product:    product,
			Category:   categoriesByID[categoryID],
			CategoryID: categoryID,
		})
	}

	if err := writeJSON(*outPath, out); err != nil {
		log.Fatalf("write output json: %v", err)
	}

	log.Printf("saved %d products with tech cards to %s", len(out), *outPath)
}

func pickActiveCharts(charts []iiko.AssemblyChartDto, date string) map[string]iiko.AssemblyChartDto {
	byProduct := make(map[string]iiko.AssemblyChartDto, len(charts))
	for _, chart := range charts {
		if chart.AssembledProductID == "" {
			continue
		}
		if chart.DateFrom != "" && chart.DateFrom > date {
			continue
		}
		if chart.DateTo != nil && *chart.DateTo != "" && *chart.DateTo < date {
			continue
		}

		current, exists := byProduct[chart.AssembledProductID]
		if !exists || chart.DateFrom > current.DateFrom || (chart.DateFrom == current.DateFrom && chart.ID > current.ID) {
			byProduct[chart.AssembledProductID] = chart
		}
	}
	return byProduct
}

func buildBaseProduct(
	productID string,
	productsByID map[string]iiko.Product,
	chartsByProduct map[string]iiko.AssemblyChartDto,
	stack map[string]bool,
) domain.Product {
	product := productsByID[productID]
	chart, hasChart := chartsByProduct[productID]

	quantity := 1.0
	if hasChart && chart.AssembledAmount > 0 {
		quantity = chart.AssembledAmount
	}

	base := domain.Product{
		Name:     productDisplayName(productID, productsByID),
		Type:     product.Type,
		Quantity: quantity,
		Unit:     productUnit(product, "кг"),
	}

	if !hasChart || stack[productID] {
		return base
	}

	stack[productID] = true
	defer delete(stack, productID)

	ingredients := make([]domain.Ingredient, 0, len(chart.Items))
	for _, item := range chart.Items {
		if item.ProductID == "" {
			continue
		}

		quantity := firstPositive(item.AmountOut, item.AmountMiddle, item.AmountIn)
		gross := firstPositive(item.AmountIn, quantity)
		net := firstPositive(item.AmountOut, item.AmountMiddle, quantity)
		name := productDisplayName(item.ProductID, productsByID)
		unit := productUnit(productsByID[item.ProductID], "кг")

		if _, hasNestedChart := chartsByProduct[item.ProductID]; hasNestedChart && !stack[item.ProductID] {
			child := buildBaseProduct(item.ProductID, productsByID, chartsByProduct, stack)
			factor := safeDiv(quantity, child.Quantity)
			ingredients = append(ingredients, domain.SubProduct{
				IName:        name,
				IQuantity:    quantity,
				IUnit:        unit,
				IGross:       gross,
				INet:         net,
				IIngredients: scaleIngredients(child.Ingredients, factor),
			})
			continue
		}

		ingredients = append(ingredients, domain.RawIngredient{
			IName:     name,
			IQuantity: quantity,
			IUnit:     unit,
			IGross:    gross,
			INet:      net,
		})
	}

	base.Ingredients = ingredients
	return base
}

func scaleIngredients(ingredients []domain.Ingredient, factor float64) []domain.Ingredient {
	scaled := make([]domain.Ingredient, 0, len(ingredients))
	for _, ingredient := range ingredients {
		switch value := ingredient.(type) {
		case domain.RawIngredient:
			scaled = append(scaled, domain.RawIngredient{
				IName:     value.IName,
				IQuantity: value.IQuantity * factor,
				IUnit:     value.IUnit,
				IGross:    value.IGross * factor,
				INet:      value.INet * factor,
			})
		case domain.SubProduct:
			scaled = append(scaled, domain.SubProduct{
				IName:        value.IName,
				IQuantity:    value.IQuantity * factor,
				IUnit:        value.IUnit,
				IGross:       value.IGross * factor,
				INet:         value.INet * factor,
				IIngredients: scaleIngredients(value.IIngredients, factor),
			})
		}
	}
	return scaled
}

func writeJSON(path string, payload any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

func productDisplayName(productID string, productsByID map[string]iiko.Product) string {
	if product, ok := productsByID[productID]; ok && product.Name != "" {
		return product.Name
	}
	return productID
}

func productUnit(product iiko.Product, fallback string) string {
	if product.MeasureUnit != "" {
		return product.MeasureUnit
	}
	return fallback
}

func firstPositive(values ...float64) float64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func safeDiv(num, den float64) float64 {
	if den == 0 {
		return 0
	}
	return num / den
}

func itoa(v int) string {
	return strconv.Itoa(v)
}
