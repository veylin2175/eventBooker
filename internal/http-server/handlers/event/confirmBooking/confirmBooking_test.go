package confirmBooking

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"eventBooker/internal/http-server/handlers/event/confirmBooking/mocks"
	"eventBooker/internal/lib/api/response"
	"eventBooker/internal/lib/logger/handlers/slogdiscard"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfirmBookingHandler(t *testing.T) {
	t.Parallel()

	logger := slogdiscard.NewDiscardLogger()

	testCases := []struct {
		name           string
		eventID        string
		requestBody    string
		mockSetup      func(mock *mocks.BookingConfirmer)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:        "Success",
			eventID:     "1",
			requestBody: `{"user_id": "user123"}`,
			mockSetup: func(mock *mocks.BookingConfirmer) {
				mock.On("ConfirmBooking", 1, "user123").Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"OK"}`,
		},
		{
			name:           "Missing event ID",
			eventID:        "",
			requestBody:    `{"user_id": "user123"}`,
			mockSetup:      func(mock *mocks.BookingConfirmer) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":"Error","error":"event id is required"}`,
		},
		{
			name:           "Invalid event ID format",
			eventID:        "invalid",
			requestBody:    `{"user_id": "user123"}`,
			mockSetup:      func(mock *mocks.BookingConfirmer) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":"Error","error":"invalid event id format"}`,
		},
		{
			name:           "Invalid JSON",
			eventID:        "1",
			requestBody:    `invalid json`,
			mockSetup:      func(mock *mocks.BookingConfirmer) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":"Error","error":"failed to decode request"}`,
		},
		{
			name:        "No pending booking found",
			eventID:     "1",
			requestBody: `{"user_id": "user123"}`,
			mockSetup: func(mock *mocks.BookingConfirmer) {
				mock.On("ConfirmBooking", 1, "user123").Return(errors.New("no pending booking found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"status":"Error","error":"no pending booking found for this user"}`,
		},
		{
			name:        "No available seats",
			eventID:     "1",
			requestBody: `{"user_id": "user123"}`,
			mockSetup: func(mock *mocks.BookingConfirmer) {
				mock.On("ConfirmBooking", 1, "user123").Return(errors.New("no available seats"))
			},
			expectedStatus: http.StatusConflict,
			expectedBody:   `{"status":"Error","error":"no available seats"}`,
		},
		{
			name:        "Internal server error",
			eventID:     "1",
			requestBody: `{"user_id": "user123"}`,
			mockSetup: func(mock *mocks.BookingConfirmer) {
				mock.On("ConfirmBooking", 1, "user123").Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"status":"Error","error":"failed to confirm booking"}`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockConfirmer := mocks.NewBookingConfirmer(t)
			tc.mockSetup(mockConfirmer)

			handler := New(logger, mockConfirmer)

			url := "/events/confirm"
			if tc.eventID != "" {
				url = "/events/" + tc.eventID + "/confirm"
			}

			req, err := http.NewRequest("POST", url, bytes.NewBufferString(tc.requestBody))
			require.NoError(t, err)

			router := chi.NewRouter()
			router.Route("/events", func(r chi.Router) {
				r.Route("/{id}", func(r chi.Router) {
					r.Post("/confirm", handler)
				})
				r.Post("/confirm", handler)
			})

			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code, "Status code mismatch")

			if tc.expectedBody != "" {
				assert.JSONEq(t, tc.expectedBody, rr.Body.String(), "Response body mismatch")
			}

			if tc.expectedStatus == http.StatusOK ||
				tc.expectedStatus == http.StatusNotFound ||
				tc.expectedStatus == http.StatusConflict ||
				tc.expectedStatus == http.StatusInternalServerError {
				mockConfirmer.AssertExpectations(t)
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
	mockConfirmer := mocks.NewBookingConfirmer(t)
	handler := New(logger, mockConfirmer)

	req, err := http.NewRequest("POST", "/", bytes.NewBufferString(`{"user_id": "test"}`))
	require.NoError(t, err)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "123")

	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	mockConfirmer.On("ConfirmBooking", 123, "test").Return(nil)

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	mockConfirmer.AssertExpectations(t)
}

func TestHandlerWithoutChiContext(t *testing.T) {
	t.Parallel()

	logger := slogdiscard.NewDiscardLogger()
	mockConfirmer := mocks.NewBookingConfirmer(t)
	handler := New(logger, mockConfirmer)

	req, err := http.NewRequest("POST", "/", bytes.NewBufferString(`{"user_id": "test"}`))
	require.NoError(t, err)

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "event id is required")
}
