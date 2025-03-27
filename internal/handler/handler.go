package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"entitlements/internal/model"

	"entitlements/utility"

	"net/http"

	"strconv"

	"github.com/gorilla/mux"
)

type TypeMap map[string]map[string]string

type JSONMap map[string]interface{}

var users = []model.User{
	{ID: 1, Name: "John Doe"},
	{ID: 2, Name: "John Smith"},
	{ID: 3, Name: "Jane Doe"},
	{ID: 4, Name: "Jane Smith"},
}

func GetUsers(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(users)
}

func GetUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	for _, user := range users {
		if user.ID == id {
			json.NewEncoder(w).Encode(user)
			return
		}
	}
	http.NotFound(w, r)
}

func CreateUser(w http.ResponseWriter, r *http.Request) {
	var user model.User
	_ = json.NewDecoder(r.Body).Decode(&user)
	users = append(users, user)
	json.NewEncoder(w).Encode(user)
}

func UpdateUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	for i, user := range users {
		if user.ID == id {
			_ = json.NewDecoder(r.Body).Decode(&user)
			users[i] = user
			json.NewEncoder(w).Encode(user)
			return
		}
	}
	http.NotFound(w, r)
}

func DeleteUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	for i, user := range users {
		if user.ID == id {
			users = append(users[:i], users[i+1:]...)
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
	http.NotFound(w, r)
}

func ParseGraphQLQuery(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Policies")
	if authHeader == "" {
		response := map[string]interface{}{
			"status":  "error",
			"message": "Missing Policies header",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}
	policies := splitPoliciesAndRemoveSpace(authHeader, ",")
	fmt.Println(policies)
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		response := map[string]interface{}{
			"status":  "error",
			"message": "Error reading request body",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}
	apiRequestBody := string(body)
	fmt.Println(apiRequestBody)
	typeMap, allFieldMap, err := utility.ParseSchema()
	fmt.Println(typeMap)
	if err != nil {
		response := map[string]interface{}{
			"status":  "error",
			"message": "Error parsing schema",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}
	data, err := processJsonData(apiRequestBody, authHeader, allFieldMap)
	if err != nil {
		response := map[string]interface{}{
			"status":  "error",
			"message": err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	responseSuccess := map[string]interface{}{
		"status":  "success",
		"data":    data["data"],
		"message": "Successfully parsed json",
	}
	if err := json.NewEncoder(w).Encode(responseSuccess); err != nil {
		response := map[string]interface{}{
			"status":  err.Error(),
			"message": "Error parsing json",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}
}

func splitPoliciesAndRemoveSpace(policies string, delimeter string) []string {
	parts := strings.Split(policies, delimeter)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func processJsonData(jsonStr string, Policy string, FieldsMap map[string]string) (JSONMap, error) {
	var data JSONMap
	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		return nil, err
	}
	policies := splitPoliciesAndRemoveSpace(Policy, ".")
	if len(policies) == 2 {
		fmt.Println(policies)
		var customerKeys []string
		for key, value := range FieldsMap {
			if value == policies[0] {
				customerKeys = append(customerKeys, key)
			}
		}
		fmt.Println("Keys with value 'Customer':", customerKeys)
		if dataField, ok := data["data"].(map[string]interface{}); ok {
			if keys, exists := dataField[customerKeys[0]].(map[string]interface{}); exists {
				delete(keys, policies[1])
			}
		}
		return data, nil
	}
	if len(policies) == 1 {
		if dataField, ok := data["data"].(map[string]interface{}); ok {
			delete(dataField, policies[0])
		}
		return data, nil
	}
	return nil, fmt.Errorf("invalid policy")
}
