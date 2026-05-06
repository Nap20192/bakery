package domain

type Group struct {
	ID   string
	Code string
	Name string
}

type GroupInput struct {
	Code string
	Name string
}
