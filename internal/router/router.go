package router

import (
	"entitlements/internal/handler"

	"github.com/gorilla/mux"
)

func NewRouter() *mux.Router {

	r := mux.NewRouter()

	r.HandleFunc("/users", handler.GetUsers).Methods("GET")

	r.HandleFunc("/users/{id}", handler.GetUser).Methods("GET")

	r.HandleFunc("/users", handler.CreateUser).Methods("POST")

	r.HandleFunc("/users/{id}", handler.UpdateUser).Methods("PUT")

	r.HandleFunc("/users/{id}", handler.DeleteUser).Methods("DELETE")

	r.HandleFunc("/parse-graphql", handler.ParseGraphQLQuery).Methods("POST")

	return r

}
