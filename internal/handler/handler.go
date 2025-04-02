package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"entitlements/internal/model"

	"entitlements/utility"

	"net/http"

	"strconv"

	"github.com/gorilla/mux"

	"github.com/oliveagle/jsonpath"
)

type TypeMap map[string]map[string]string

type JSONMap map[string]any

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
	// Checking if the request has the Policies header
	authHeader := r.Header.Get("Policies")
	if authHeader == "" {
		response := map[string]any{
			"status":  "error",
			"message": "Missing Policies header",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Reading the request body
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		response := map[string]any{
			"status":  "error",
			"message": "Error reading request body",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}
	apiRequestBody := string(body)
	if apiRequestBody == "" {
		response := map[string]any{
			"status":  "error",
			"message": "Request body cannot be empty",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Parsing the schema
	_, allFieldMap, err := utility.ParseSchema()
	if err != nil {
		response := map[string]any{
			"status":  "error",
			"message": "Error parsing schema: " + err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Splitting the policies and removing spaces
	policies := splitPoliciesAndRemoveSpace(authHeader, ",")
	if len(policies) == 0 {
		response := map[string]any{
			"status":  "error",
			"message": "No valid policies provided",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	var data JSONMap
	err = json.Unmarshal([]byte(apiRequestBody), &data)
	if err != nil {
		response := map[string]any{
			"status":  "error",
			"message": "Error parsing JSON body: " + err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Processing the JSON data
	for _, value := range policies {
		data, err = processJsonData(data, value, allFieldMap)
		if err != nil {
			response := map[string]any{
				"status":  "error",
				"message": "Error processing policy: " + err.Error(),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	responseSuccess := map[string]any{
		"status":   "success",
		"data":     data["data"],
		"allfield": allFieldMap,
		"message":  "Successfully parsed JSON",
	}
	if err := json.NewEncoder(w).Encode(responseSuccess); err != nil {
		response := map[string]any{
			"status":  "error",
			"message": "Error encoding response: " + err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
	}
}

func splitPoliciesAndRemoveSpace(policies string, delimeter string) []string {
	parts := strings.Split(policies, delimeter)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func processJsonData(jsonStr JSONMap, Policy string, FieldsMap map[string]string) (JSONMap, error) {
	if jsonStr == nil {
		return nil, fmt.Errorf("input JSON data is nil")
	}
	if Policy == "" {
		return nil, fmt.Errorf("policy cannot be empty")
	}

	policies := splitPoliciesAndRemoveSpace(Policy, ".")

	if len(policies) > 2 {
		return nil, fmt.Errorf("invalid policy format")
	}

	if policies[0] == "Query" {
		if dataField, ok := jsonStr["data"].(map[string]any); ok {
			if _, exists := dataField[policies[1]]; exists {
				delete(dataField, policies[1])
			} else {
				return nil, fmt.Errorf("key '%s' not found in data", policies[0])
			}
		} else {
			return nil, fmt.Errorf("data field is not a valid map")
		}
		return jsonStr, nil
	} else {
		var customerKeys []string
		for key, value := range FieldsMap {
			if value == policies[0] {
				customerKeys = append(customerKeys, key)
			}
		}
		fmt.Println("Customer Keys:", customerKeys)
		fmt.Println("Policies:", policies)
		allKeys := collectKeys(jsonStr, "")

		if len(customerKeys) == 0 {
			return nil, fmt.Errorf("no matching keys found for policy: %s", policies[0])
		}

		for _, key := range allKeys {
			keys := splitPoliciesAndRemoveSpace(key, ".")
			if len(keys) > 1 {
				lastkey := keys[len(keys)-1]
				secondLastKey := keys[len(keys)-2]
				if removeArrayIndices(lastkey) == policies[1] && removeArrayIndices(secondLastKey) == customerKeys[0] {
					fmt.Println("Deleting key:", key)
					err := deleteJSONPath(jsonStr, key)
					if err != nil {
						fmt.Println(err.Error())
					}
					fmt.Println(jsonStr)
				}
			}
		}
		return jsonStr, nil
	}
}

func collectKeys(m map[string]interface{}, prefix string) []string {
	var keys []string
	for key, value := range m {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}
		keys = append(keys, fullKey)
		switch v := value.(type) {
		case map[string]interface{}:
			keys = append(keys, collectKeys(v, fullKey)...)
		case []interface{}:
			for i, item := range v {
				if obj, ok := item.(map[string]interface{}); ok {
					keys = append(keys, collectKeys(obj, fmt.Sprintf("%s[%d]", fullKey, i))...)
				}
			}
		}
	}
	return keys
}

func removeArrayIndices(input string) string {
	re := regexp.MustCompile(`\[\d+\]`)
	return re.ReplaceAllString(input, "")
}

func deleteJSONPath(jsonMap map[string]any, path string) error {

	if !strings.HasPrefix(path, "$") {
		path = "$" + path
	}

	// Find parent structure
	parentPath := path[:len(path)-len(path[strings.LastIndex(path, ".")+1:])-1]
	jp, err := jsonpath.Compile(parentPath)
	if err != nil {
		return err
	}
	parent, err := jp.Lookup(jsonMap)
	if err != nil {
		return err
	}

	// Remove key if parent is a map
	if parentMap, ok := parent.(map[string]any); ok {
		key := path[strings.LastIndex(path, ".")+1:]
		delete(parentMap, key)
	}

	return nil
}
