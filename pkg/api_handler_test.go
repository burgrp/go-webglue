package webglue

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

//go:embed client
var testClientResources embed.FS

type TestApi struct {
	counter int
}

func (api *TestApi) Add(a, b int) int {
	return a + b
}

func (api *TestApi) Subtract(a, b int) int {
	return a - b
}

func (api *TestApi) Divide(a, b int) (int, error) {
	if b == 0 {
		return 0, errors.New("division by zero")
	}
	return a / b, nil
}

func (api *TestApi) MultipleReturns(a, b int) (int, int, error) {
	if b == 0 {
		return 0, 0, errors.New("division by zero")
	}
	return a / b, a % b, nil
}

func (api *TestApi) GetCounter() int {
	return api.counter
}

func (api *TestApi) IncrementCounter(inc int) int {
	api.counter += inc
	return api.counter
}

func (api *TestApi) WithContext(ctx context.Context, value string) string {
	return "context:" + value
}

type ComplexInput struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func (api *TestApi) ComplexParam(input ComplexInput) string {
	return input.Name + "=" + string(rune(input.Value+'0'))
}

type AuthedTestApi struct {
	TestApi
}

type CustomUser struct {
	ID   int
	Name string
}

func (api *AuthedTestApi) CheckCall(req *http.Request, functionName string) ([]any, error) {
	token := req.Header.Get("Authorization")
	if token == "" {
		return nil, errors.New("unauthorized")
	}
	if token != "Bearer valid-token" {
		return nil, errors.New("invalid token")
	}

	user := &CustomUser{ID: 42, Name: "TestUser"}
	return []any{user}, nil
}

func (api *AuthedTestApi) GetUser(user *CustomUser) *CustomUser {
	return user
}

func (api *AuthedTestApi) PublicMethod() string {
	return "public"
}

func TestApiHandlerBasicCall(t *testing.T) {
	module := &Module{
		Name: "test",
		Api:  &TestApi{},
	}

	handler, err := newApiHandler([]*Module{module})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	body := bytes.NewBufferString("[5, 3]")
	req := httptest.NewRequest("POST", "/api/test/add", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response ResultReply
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if result, ok := response.Result.(float64); !ok || result != 8 {
		t.Errorf("Expected result 8, got %v", response.Result)
	}
}

func TestApiHandlerErrorReturn(t *testing.T) {
	module := &Module{
		Name: "test",
		Api:  &TestApi{},
	}

	handler, err := newApiHandler([]*Module{module})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	body := bytes.NewBufferString("[10, 0]")
	req := httptest.NewRequest("POST", "/api/test/divide", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var response ErrorReply
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Error != "division by zero" {
		t.Errorf("Expected error 'division by zero', got '%s'", response.Error)
	}
}

func TestApiHandlerMultipleReturns(t *testing.T) {
	module := &Module{
		Name: "test",
		Api:  &TestApi{},
	}

	handler, err := newApiHandler([]*Module{module})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	body := bytes.NewBufferString("[10, 3]")
	req := httptest.NewRequest("POST", "/api/test/multipleReturns", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var response ResultReply
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	results, ok := response.Result.([]any)
	if !ok {
		t.Fatalf("Expected array result, got %T", response.Result)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	if quotient, ok := results[0].(float64); !ok || quotient != 3 {
		t.Errorf("Expected quotient 3, got %v", results[0])
	}

	if remainder, ok := results[1].(float64); !ok || remainder != 1 {
		t.Errorf("Expected remainder 1, got %v", results[1])
	}
}

func TestApiHandlerStatefulness(t *testing.T) {
	api := &TestApi{counter: 10}
	module := &Module{
		Name: "test",
		Api:  api,
	}

	handler, err := newApiHandler([]*Module{module})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// First call
	body := bytes.NewBufferString("[5]")
	req := httptest.NewRequest("POST", "/api/test/incrementCounter", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var response ResultReply
	json.Unmarshal(w.Body.Bytes(), &response)

	if result, ok := response.Result.(float64); !ok || result != 15 {
		t.Errorf("Expected result 15, got %v", response.Result)
	}

	// Second call
	body = bytes.NewBufferString("[3]")
	req = httptest.NewRequest("POST", "/api/test/incrementCounter", body)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	json.Unmarshal(w.Body.Bytes(), &response)

	if result, ok := response.Result.(float64); !ok || result != 18 {
		t.Errorf("Expected result 18, got %v", response.Result)
	}
}

func TestApiHandlerComplexParameter(t *testing.T) {
	module := &Module{
		Name: "test",
		Api:  &TestApi{},
	}

	handler, err := newApiHandler([]*Module{module})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	body := bytes.NewBufferString(`[{"name":"foo","value":5}]`)
	req := httptest.NewRequest("POST", "/api/test/complexParam", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var response ResultReply
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if result, ok := response.Result.(string); !ok || result != "foo=5" {
		t.Errorf("Expected result 'foo=5', got %v", response.Result)
	}
}

func TestApiHandlerModuleNotFound(t *testing.T) {
	module := &Module{
		Name: "test",
		Api:  &TestApi{},
	}

	handler, err := newApiHandler([]*Module{module})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	body := bytes.NewBufferString("[]")
	req := httptest.NewRequest("POST", "/api/nonexistent/method", body)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var response ErrorReply
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.Error != "module not found" {
		t.Errorf("Expected 'module not found', got '%s'", response.Error)
	}
}

func TestApiHandlerMethodNotFound(t *testing.T) {
	module := &Module{
		Name: "test",
		Api:  &TestApi{},
	}

	handler, err := newApiHandler([]*Module{module})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	body := bytes.NewBufferString("[]")
	req := httptest.NewRequest("POST", "/api/test/nonexistent", body)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var response ErrorReply
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.Error != "function not found" {
		t.Errorf("Expected 'function not found', got '%s'", response.Error)
	}
}

func TestApiHandlerHeadRequest(t *testing.T) {
	module := &Module{
		Name: "test",
		Api:  &TestApi{},
	}

	handler, err := newApiHandler([]*Module{module})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	req := httptest.NewRequest("HEAD", "/api/test/add", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.Len() > 0 {
		t.Errorf("Expected empty body for HEAD request, got %d bytes", w.Body.Len())
	}

	if ct := w.Header().Get("Content-Type"); ct != ContentTypeJson {
		t.Errorf("Expected Content-Type %s, got %s", ContentTypeJson, ct)
	}
}

func TestApiHandlerCallChecker(t *testing.T) {
	module := &Module{
		Name: "test",
		Api:  &AuthedTestApi{},
	}

	handler, err := newApiHandler([]*Module{module})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Test without auth
	body := bytes.NewBufferString("[]")
	req := httptest.NewRequest("POST", "/api/test/getUser", body)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var errorResp ErrorReply
	json.Unmarshal(w.Body.Bytes(), &errorResp)

	if errorResp.Error != "unauthorized" {
		t.Errorf("Expected 'unauthorized', got '%s'", errorResp.Error)
	}

	// Test with valid auth
	body = bytes.NewBufferString("[]")
	req = httptest.NewRequest("POST", "/api/test/getUser", body)
	req.Header.Set("Authorization", "Bearer valid-token")
	w = httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var response ResultReply
	json.Unmarshal(w.Body.Bytes(), &response)

	result, ok := response.Result.(map[string]any)
	if !ok {
		t.Fatalf("Expected map result, got %T", response.Result)
	}

	if result["ID"].(float64) != 42 || result["Name"].(string) != "TestUser" {
		t.Errorf("Unexpected user data: %v", result)
	}
}

func TestApiHandlerCallCheckerCannotBeCalled(t *testing.T) {
	module := &Module{
		Name: "test",
		Api:  &AuthedTestApi{},
	}

	handler, err := newApiHandler([]*Module{module})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	body := bytes.NewBufferString("[]")
	req := httptest.NewRequest("POST", "/api/test/checkCall", body)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var response ErrorReply
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.Error != "CheckCall function is not allowed" {
		t.Errorf("Expected 'CheckCall function is not allowed', got '%s'", response.Error)
	}
}

func TestApiHandlerBadPath(t *testing.T) {
	module := &Module{
		Name: "test",
		Api:  &TestApi{},
	}

	handler, err := newApiHandler([]*Module{module})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Path too short (less than 3 parts)
	req := httptest.NewRequest("POST", "/api", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestApiHandlerModuleWithoutApi(t *testing.T) {
	module := &Module{
		Name: "test",
		Api:  nil,
	}

	handler, err := newApiHandler([]*Module{module})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	body := bytes.NewBufferString("[]")
	req := httptest.NewRequest("POST", "/api/test/method", body)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var response ErrorReply
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.Error != "module API not found" {
		t.Errorf("Expected 'module API not found', got '%s'", response.Error)
	}
}
