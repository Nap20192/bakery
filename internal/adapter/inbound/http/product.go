package http

import (
	"net/http"

	"github.com/google/uuid"

	"bakery/internal/app"
)

func (s *Server) listProducts(w http.ResponseWriter, r *http.Request) {
	ps, err := s.product.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, productDTOs(ps))
}

func (s *Server) createProduct(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name   string `json:"name"`
		Unit   string `json:"unit"`
		IikoID string `json:"iiko_id"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	p, err := s.product.Create(r.Context(), app.CreateProductInput{
		Name:   body.Name,
		Unit:   body.Unit,
		IikoID: body.IikoID,
	})
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, productDTO(p))
}

func (s *Server) getProduct(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	p, err := s.product.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, productDTO(p))
}
