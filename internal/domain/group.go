package domain

// Group — универсальный идентификатор сущности, приходящей с UI/бота.
// В текущем мониторинге используется как ссылка на ингредиент (по ID/Code/Name).
type Group struct {
	Name string `json:"name"`
	ID   string `json:"id"`
	Code string `json:"code"`
}
