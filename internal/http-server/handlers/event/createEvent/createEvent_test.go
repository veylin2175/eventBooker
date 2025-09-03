package createEvent

import (
	"bytes"
	"encoding/json"
	"errors"
	"eventBooker/internal/http-server/handlers/event/createEvent/mocks"
	"eventBooker/internal/lib/logger/handlers/slogdiscard"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateEventHandler(t *testing.T) {
	t.Parallel()

	logger := slogdiscard.NewDiscardLogger()

	testTime := time.Date(2024, 12, 25, 18, 0, 0, 0, time.UTC)

	testCases := []struct {
		name           string
		requestBody    string
		mockSetup      func(mock *mocks.EventCreator)
		expectedStatus int
		expectedBody   string
		checkBody      func(t *testing.T, body string)
	}{
		{
			name: "Success",
			requestBody: `{
				"title": "Test Event",
				"date": "2024-12-25T18:00:00Z",
				"total_seats": 100,
				"deadline": 30
			}`,
			mockSetup: func(mock *mocks.EventCreator) {
				mock.On("CreateEvent", "Test Event", testTime, 100, 30).Return(123, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"OK","event_id":123}`,
		},
		{
			name:           "Invalid JSON",
			requestBody:    `invalid json`,
			mockSetup:      func(mock *mocks.EventCreator) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":"Error","error":"failed to decode request"}`,
		},
		{
			name: "Missing title",
			requestBody: `{
				"date": "2024-12-25T18:00:00Z",
				"total_seats": 100,
				"deadline": 30
			}`,
			mockSetup:      func(mock *mocks.EventCreator) {},
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body string) {
				assert.Contains(t, body, `"status":"Error"`)
				assert.Contains(t, body, `"error":`)
				assert.Contains(t, body, "Title")
			},
		},
		{
			name: "Missing date",
			requestBody: `{
				"title": "Test Event",
				"total_seats": 100,
				"deadline": 30
			}`,
			mockSetup:      func(mock *mocks.EventCreator) {},
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body string) {
				assert.Contains(t, body, `"status":"Error"`)
				assert.Contains(t, body, `"error":`)
				assert.Contains(t, body, "Date")
			},
		},
		{
			name: "Missing total_seats",
			requestBody: `{
				"title": "Test Event",
				"date": "2024-12-25T18:00:00Z",
				"deadline": 30
			}`,
			mockSetup:      func(mock *mocks.EventCreator) {},
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body string) {
				assert.Contains(t, body, `"status":"Error"`)
				assert.Contains(t, body, `"error":`)
				assert.Contains(t, body, "TotalSeats")
			},
		},
		{
			name: "Missing deadline",
			requestBody: `{
				"title": "Test Event",
				"date": "2024-12-25T18:00:00Z",
				"total_seats": 100
			}`,
			mockSetup:      func(mock *mocks.EventCreator) {},
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body string) {
				assert.Contains(t, body, `"status":"Error"`)
				assert.Contains(t, body, `"error":`)
				assert.Contains(t, body, "Deadline")
			},
		},
		{
			name: "Empty title",
			requestBody: `{
				"title": "",
				"date": "2024-12-25T18:00:00Z",
				"total_seats": 100,
				"deadline": 30
			}`,
			mockSetup:      func(mock *mocks.EventCreator) {},
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body string) {
				assert.Contains(t, body, `"status":"Error"`)
				assert.Contains(t, body, `"error":`)
				assert.Contains(t, body, "Title")
			},
		},
		{
			name: "Invalid date format",
			requestBody: `{
				"title": "Test Event",
				"date": "invalid-date",
				"total_seats": 100,
				"deadline": 30
			}`,
			mockSetup:      func(mock *mocks.EventCreator) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":"Error","error":"failed to decode request"}`,
		},
		{
			name: "Internal server error",
			requestBody: `{
				"title": "Test Event",
				"date": "2024-12-25T18:00:00Z",
				"total_seats": 100,
				"deadline": 30
			}`,
			mockSetup: func(mock *mocks.EventCreator) {
				mock.On("CreateEvent", "Test Event", testTime, 100, 30).Return(0, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"status":"Error","error":"failed to add event"}`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockCreator := mocks.NewEventCreator(t)
			tc.mockSetup(mockCreator)

			handler := New(logger, mockCreator)

			req, err := http.NewRequest("POST", "/events", bytes.NewBufferString(tc.requestBody))
			require.NoError(t, err)

			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code, "Status code mismatch")

			if tc.expectedBody != "" {
				assert.JSONEq(t, tc.expectedBody, rr.Body.String(), "Response body mismatch")
			} else if tc.checkBody != nil {
				tc.checkBody(t, rr.Body.String())
			}

			if tc.expectedStatus == http.StatusOK || tc.expectedStatus == http.StatusInternalServerError {
				mockCreator.AssertExpectations(t)
			}
		})
	}
}

func TestResponseOK(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	responseOK(rr, req, 456)

	assert.Equal(t, http.StatusOK, rr.Code)

	var actualResponse EventResponse
	err := json.Unmarshal(rr.Body.Bytes(), &actualResponse)
	require.NoError(t, err)

	assert.Equal(t, "OK", actualResponse.Status)
	assert.Equal(t, "", actualResponse.Error)
	assert.Equal(t, 456, actualResponse.EventId)
}

func TestValidationErrors(t *testing.T) {
	t.Parallel()

	logger := slogdiscard.NewDiscardLogger()
	mockCreator := mocks.NewEventCreator(t)
	handler := New(logger, mockCreator)

	testCases := []struct {
		name           string
		requestBody    string
		expectedStatus int
		expectedFields []string
	}{
		{
			name:           "Missing all required fields",
			requestBody:    `{}`,
			expectedStatus: http.StatusBadRequest,
			expectedFields: []string{"Title", "Date", "TotalSeats", "Deadline"},
		},
		{
			name: "Empty title",
			requestBody: `{
				"title": "",
				"date": "2024-12-25T18:00:00Z",
				"total_seats": 100,
				"deadline": 30
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedFields: []string{"Title"},
		},
		{
			name: "Zero total_seats",
			requestBody: `{
				"title": "Test Event",
				"date": "2024-12-25T18:00:00Z",
				"total_seats": 0,
				"deadline": 30
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedFields: []string{"TotalSeats"},
		},
		{
			name: "Zero deadline",
			requestBody: `{
				"title": "Test Event",
				"date": "2024-12-25T18:00:00Z",
				"total_seats": 100,
				"deadline": 0
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedFields: []string{"Deadline"},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/events", bytes.NewBufferString(tc.requestBody))
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)

			body := rr.Body.String()
			assert.Contains(t, body, `"status":"Error"`)
			assert.Contains(t, body, `"error":`)

			for _, field := range tc.expectedFields {
				assert.Contains(t, body, field)
			}
		})
	}
}

// Тест для проверки точного формата успешного ответа
func TestSuccessResponseFormat(t *testing.T) {
	t.Parallel()

	logger := slogdiscard.NewDiscardLogger()
	mockCreator := mocks.NewEventCreator(t)
	handler := New(logger, mockCreator)

	// Mock setup
	testTime := time.Date(2024, 12, 25, 18, 0, 0, 0, time.UTC)
	mockCreator.On("CreateEvent", "Test Event", testTime, 100, 30).Return(789, nil)

	// Create request
	requestBody := `{
		"title": "Test Event",
		"date": "2024-12-25T18:00:00Z",
		"total_seats": 100,
		"deadline": 30
	}`
	req, err := http.NewRequest("POST", "/events", bytes.NewBufferString(requestBody))
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response EventResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "OK", response.Status)
	assert.Equal(t, "", response.Error)
	assert.Equal(t, 789, response.EventId)

	mockCreator.AssertExpectations(t)
}

// Тест для проверки обработки ошибок от EventCreator
func TestEventCreatorErrorHandling(t *testing.T) {
	t.Parallel()

	logger := slogdiscard.NewDiscardLogger()
	mockCreator := mocks.NewEventCreator(t)
	handler := New(logger, mockCreator)

	// Mock setup - возвращаем ошибку
	testTime := time.Date(2024, 12, 25, 18, 0, 0, 0, time.UTC)
	mockCreator.On("CreateEvent", "Test Event", testTime, 100, 30).Return(0, errors.New("some database error"))

	// Create request
	requestBody := `{
		"title": "Test Event",
		"date": "2024-12-25T18:00:00Z",
		"total_seats": 100,
		"deadline": 30
	}`
	req, err := http.NewRequest("POST", "/events", bytes.NewBufferString(requestBody))
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.JSONEq(t, `{"status":"Error","error":"failed to add event"}`, rr.Body.String())

	mockCreator.AssertExpectations(t)
}
