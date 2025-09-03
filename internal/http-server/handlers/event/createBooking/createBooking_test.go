package createBooking

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"eventBooker/internal/http-server/handlers/event/createBooking/mocks"
	"eventBooker/internal/lib/api/response"
	"eventBooker/internal/lib/logger/handlers/slogdiscard"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateBookingHandler(t *testing.T) {
	t.Parallel()

	logger := slogdiscard.NewDiscardLogger()

	testCases := []struct {
		name           string
		eventID        string
		requestBody    string
		mockSetup      func(mock *mocks.BookingCreator)
		expectedStatus int
		expectedBody   string
		checkBody      func(t *testing.T, body string)
	}{
		{
			name:        "Success",
			eventID:     "1",
			requestBody: `{"user_id": "user123"}`,
			mockSetup: func(mock *mocks.BookingCreator) {
				mock.On("BookEvent", 1, "user123").Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"OK"}`,
		},
		{
			name:           "Missing event ID",
			eventID:        "",
			requestBody:    `{"user_id": "user123"}`,
			mockSetup:      func(mock *mocks.BookingCreator) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":"Error","error":"event id is required"}`,
		},
		{
			name:           "Invalid event ID format",
			eventID:        "invalid",
			requestBody:    `{"user_id": "user123"}`,
			mockSetup:      func(mock *mocks.BookingCreator) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":"Error","error":"invalid event id format"}`,
		},
		{
			name:           "Invalid JSON",
			eventID:        "1",
			requestBody:    `invalid json`,
			mockSetup:      func(mock *mocks.BookingCreator) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":"Error","error":"failed to decode request"}`,
		},
		{
			name:           "Missing user_id",
			eventID:        "1",
			requestBody:    `{}`,
			mockSetup:      func(mock *mocks.BookingCreator) {},
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body string) {
				assert.Contains(t, body, `"status":"Error"`)
				assert.Contains(t, body, `"error":`)
				assert.Contains(t, body, "UserId")
			},
		},
		{
			name:           "Empty user_id",
			eventID:        "1",
			requestBody:    `{"user_id": ""}`,
			mockSetup:      func(mock *mocks.BookingCreator) {},
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body string) {
				assert.Contains(t, body, `"status":"Error"`)
				assert.Contains(t, body, `"error":`)
				assert.Contains(t, body, "UserId")
			},
		},
		{
			name:        "No available seats",
			eventID:     "1",
			requestBody: `{"user_id": "user123"}`,
			mockSetup: func(mock *mocks.BookingCreator) {
				mock.On("BookEvent", 1, "user123").Return(errors.New("no available seats"))
			},
			expectedStatus: http.StatusConflict,
			expectedBody:   `{"status":"Error","error":"no available seats"}`,
		},
		{
			name:        "User already has pending booking",
			eventID:     "1",
			requestBody: `{"user_id": "user123"}`,
			mockSetup: func(mock *mocks.BookingCreator) {
				mock.On("BookEvent", 1, "user123").Return(errors.New("user already has pending booking for this event"))
			},
			expectedStatus: http.StatusConflict,
			expectedBody:   `{"status":"Error","error":"user already has pending booking for this event"}`,
		},
		{
			name:        "Internal server error",
			eventID:     "1",
			requestBody: `{"user_id": "user123"}`,
			mockSetup: func(mock *mocks.BookingCreator) {
				mock.On("BookEvent", 1, "user123").Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"status":"Error","error":"failed to book event"}`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockCreator := mocks.NewBookingCreator(t)
			tc.mockSetup(mockCreator)

			handler := New(logger, mockCreator)

			url := "/events/book"
			if tc.eventID != "" {
				url = "/events/" + tc.eventID + "/book"
			}

			req, err := http.NewRequest("POST", url, bytes.NewBufferString(tc.requestBody))
			require.NoError(t, err)

			router := chi.NewRouter()
			router.Route("/events", func(r chi.Router) {
				r.Route("/{id}", func(r chi.Router) {
					r.Post("/book", handler)
				})
				r.Post("/book", handler)
			})

			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code, "Status code mismatch")

			if tc.expectedBody != "" {
				assert.JSONEq(t, tc.expectedBody, rr.Body.String(), "Response body mismatch")
			} else if tc.checkBody != nil {
				tc.checkBody(t, rr.Body.String())
			}

			if tc.expectedStatus == http.StatusOK ||
				tc.expectedStatus == http.StatusConflict ||
				tc.expectedStatus == http.StatusInternalServerError {
				mockCreator.AssertExpectations(t)
			}
		})
	}
}

func TestResponseOK(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	responseOK(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	expectedResponse := response.OK()
	var actualResponse BookingResponse
	err := json.Unmarshal(rr.Body.Bytes(), &actualResponse)
	require.NoError(t, err)

	assert.Equal(t, expectedResponse.Status, actualResponse.Status)
	assert.Equal(t, expectedResponse.Error, actualResponse.Error)
}

func TestHandlerWithChiContext(t *testing.T) {
	t.Parallel()

	logger := slogdiscard.NewDiscardLogger()
	mockCreator := mocks.NewBookingCreator(t)
	handler := New(logger, mockCreator)

	req, err := http.NewRequest("POST", "/", bytes.NewBufferString(`{"user_id": "test"}`))
	require.NoError(t, err)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "123")

	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	mockCreator.On("BookEvent", 123, "test").Return(nil)

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	mockCreator.AssertExpectations(t)
}

func TestHandlerWithoutChiContext(t *testing.T) {
	t.Parallel()

	logger := slogdiscard.NewDiscardLogger()
	mockCreator := mocks.NewBookingCreator(t)
	handler := New(logger, mockCreator)

	req, err := http.NewRequest("POST", "/", bytes.NewBufferString(`{"user_id": "test"}`))
	require.NoError(t, err)

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "event id is required")
}

func TestValidationErrors(t *testing.T) {
	t.Parallel()

	logger := slogdiscard.NewDiscardLogger()
	mockCreator := mocks.NewBookingCreator(t)
	handler := New(logger, mockCreator)

	testCases := []struct {
		name           string
		requestBody    string
		expectedStatus int
	}{
		{
			name:           "Empty user_id",
			requestBody:    `{"user_id": ""}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing user_id field",
			requestBody:    `{}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/events/1/book", bytes.NewBufferString(tc.requestBody))
			require.NoError(t, err)

			router := chi.NewRouter()
			router.Route("/events", func(r chi.Router) {
				r.Route("/{id}", func(r chi.Router) {
					r.Post("/book", handler)
				})
			})

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)

			assert.Contains(t, rr.Body.String(), `"status":"Error"`)
			assert.Contains(t, rr.Body.String(), `"error":`)
			assert.Contains(t, rr.Body.String(), "UserId")
		})
	}
}

func TestExactErrorMessage(t *testing.T) {
	t.Parallel()

	logger := slogdiscard.NewDiscardLogger()
	mockCreator := mocks.NewBookingCreator(t)
	handler := New(logger, mockCreator)

	req, err := http.NewRequest("POST", "/events/1/book", bytes.NewBufferString(`{}`))
	require.NoError(t, err)

	router := chi.NewRouter()
	router.Route("/events", func(r chi.Router) {
		r.Route("/{id}", func(r chi.Router) {
			r.Post("/book", handler)
		})
	})

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	t.Logf("Actual error message: %s", rr.Body.String())

	body := rr.Body.String()
	validMessages := []string{
		"field UserId is a required field",
		"Key: 'BookingRequest.UserId' Error:Field validation for 'UserId' failed on the 'required' tag",
		"UserId is a required field",
	}

	hasValidMessage := false
	for _, msg := range validMessages {
		if strings.Contains(body, msg) {
			hasValidMessage = true
			break
		}
	}

	assert.True(t, hasValidMessage, "Expected validation error about UserId, got: %s", body)
}
