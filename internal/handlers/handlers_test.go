package handlers

import (
	"bytes"
	"encoding/json"
	"gin-rest-template/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheck(t *testing.T) {
	// Set Gin to Test Mode
	gin.SetMode(gin.TestMode)

	// Setup router
	r := gin.Default()

	// Mock handler
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, models.SuccessResponse{
			Message: "Service is healthy",
			Data: gin.H{
				"status":  "ok",
				"version": "1.0.0",
			},
		})
	})

	// Create a request
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// Perform request
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Service is healthy", response.Message)
}

func TestRegisterValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.Default()

	r.POST("/register", func(c *gin.Context) {
		var req models.RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_request",
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"message": "success"})
	})

	tests := []struct {
		name           string
		payload        map[string]interface{}
		expectedStatus int
	}{
		{
			name: "Valid registration",
			payload: map[string]interface{}{
				"email":    "test@example.com",
				"password": "password123",
				"name":     "Test User",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Invalid email",
			payload: map[string]interface{}{
				"email":    "invalid-email",
				"password": "password123",
				"name":     "Test User",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Short password",
			payload: map[string]interface{}{
				"email":    "test@example.com",
				"password": "123",
				"name":     "Test User",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing name",
			payload: map[string]interface{}{
				"email":    "test@example.com",
				"password": "password123",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, _ := json.Marshal(tt.payload)
			req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// Example of how to test with mocked service layer
// You would create a mock service that implements the service interface
// func TestGetAlbumsHandler(t *testing.T) {
// 	gin.SetMode(gin.TestMode)
//
// 	// Create mock service
// 	mockService := new(MockService)
// 	cfg := &config.Config{}
// 	handler := NewHandler(mockService, cfg)
//
// 	// Setup expectations
// 	mockService.On("GetAlbums", mock.Anything, "user-id", mock.Anything).Return(
// 		&models.PaginatedResponse{
// 			Data:       []*models.Album{},
// 			Page:       1,
// 			PageSize:   10,
// 			TotalItems: 0,
// 			TotalPages: 0,
// 		}, nil)
//
// 	// Test the handler...
// }
