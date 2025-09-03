package getEventInfo

import (
	"context"
	"encoding/json"
	"errors"
	"eventBooker/internal/http-server/handlers/event/getEventInfo/mocks"
	"eventBooker/internal/lib/logger/handlers/slogdiscard"
	"eventBooker/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEventInfoHandler(t *testing.T) {
	t.Parallel()

	logger := slogdiscard.NewDiscardLogger()

	testTime := time.Date(2024, 12, 25, 18, 0, 0, 0, time.UTC)
	testEvent := &models.Event{
		ID:          1,
		Title:       "Test Event",
		Date:        testTime,
		TotalSeats:  100,
		BookedSeats: 50,
	}
	testBookings := []models.Booking{
		{
			ID:        1,
			EventID:   1,
			UserID:    "user1",
			CreatedAt: testTime,
		},
		{
			ID:        2,
			EventID:   1,
			UserID:    "user2",
			CreatedAt: testTime.Add(1 * time.Hour),
		},
	}

	testCases := []struct {
		name           string
		eventID        string
		mockSetup      func(mock *mocks.EventGetter)
		expectedStatus int
		expectedBody   string
		checkBody      func(t *testing.T, body string)
	}{
		{
			name:    "Success with event and bookings",
			eventID: "1",
			mockSetup: func(mock *mocks.EventGetter) {
				mock.On("GetEventWithBookings", 1).Return(testEvent, testBookings, nil)
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				var response EventInfoResponse
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)

				assert.Equal(t, "OK", response.Status)
				assert.Equal(t, "", response.Error)
				require.NotNil(t, response.Event)
				assert.Equal(t, 1, response.Event.ID)
				assert.Equal(t, "Test Event", response.Event.Title)
				assert.Len(t, response.Booking, 2)
				assert.Equal(t, "user1", response.Booking[0].UserID)
				assert.Equal(t, "user2", response.Booking[1].UserID)
			},
		},
		{
			name:    "Success with event but no bookings",
			eventID: "1",
			mockSetup: func(mock *mocks.EventGetter) {
				mock.On("GetEventWithBookings", 1).Return(testEvent, []models.Booking{}, nil)
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				var response EventInfoResponse
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)

				assert.Equal(t, "OK", response.Status)
				assert.Equal(t, "", response.Error)
				require.NotNil(t, response.Event)
				assert.Equal(t, 1, response.Event.ID)
				assert.Empty(t, response.Booking)
			},
		},
		{
			name:           "Missing event ID",
			eventID:        "",
			mockSetup:      func(mock *mocks.EventGetter) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":"Error","error":"event id is required"}`,
		},
		{
			name:           "Invalid event ID format",
			eventID:        "invalid",
			mockSetup:      func(mock *mocks.EventGetter) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":"Error","error":"invalid event id format"}`,
		},
		{
			name:    "Event not found",
			eventID: "999",
			mockSetup: func(mock *mocks.EventGetter) {
				mock.On("GetEventWithBookings", 999).Return(nil, nil, errors.New("event not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"status":"Error","error":"event not found"}`,
		},
		{
			name:    "Internal server error",
			eventID: "1",
			mockSetup: func(mock *mocks.EventGetter) {
				mock.On("GetEventWithBookings", 1).Return(nil, nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"status":"Error","error":"failed to get event information"}`,
		},
		{
			name:    "Other specific error",
			eventID: "1",
			mockSetup: func(mock *mocks.EventGetter) {
				mock.On("GetEventWithBookings", 1).Return(nil, nil, errors.New("connection timeout"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"status":"Error","error":"failed to get event information"}`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockGetter := mocks.NewEventGetter(t)
			tc.mockSetup(mockGetter)

			handler := New(logger, mockGetter)

			url := "/events/info"
			if tc.eventID != "" {
				url = "/events/" + tc.eventID + "/info"
			}

			req, err := http.NewRequest("GET", url, nil)
			require.NoError(t, err)

			router := chi.NewRouter()
			router.Route("/events", func(r chi.Router) {
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/info", handler)
				})
				r.Get("/info", handler)
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
				tc.expectedStatus == http.StatusNotFound ||
				tc.expectedStatus == http.StatusInternalServerError {
				mockGetter.AssertExpectations(t)
			}
		})
	}
}

func TestResponseOK(t *testing.T) {
	t.Parallel()

	testTime := time.Date(2024, 12, 25, 18, 0, 0, 0, time.UTC)
	testEvent := &models.Event{
		ID:          1,
		Title:       "Test Event",
		Date:        testTime,
		TotalSeats:  100,
		BookedSeats: 50,
	}
	testBookings := []models.Booking{
		{
			ID:        1,
			EventID:   1,
			UserID:    "user1",
			CreatedAt: testTime,
		},
	}

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	responseOK(rr, req, testEvent, testBookings)

	assert.Equal(t, http.StatusOK, rr.Code)

	var actualResponse EventInfoResponse
	err := json.Unmarshal(rr.Body.Bytes(), &actualResponse)
	require.NoError(t, err)

	assert.Equal(t, "OK", actualResponse.Status)
	assert.Equal(t, "", actualResponse.Error)
	require.NotNil(t, actualResponse.Event)
	assert.Equal(t, 1, actualResponse.Event.ID)
	assert.Equal(t, "Test Event", actualResponse.Event.Title)
	require.Len(t, actualResponse.Booking, 1)
	assert.Equal(t, "user1", actualResponse.Booking[0].UserID)
}

func TestResponseWithNilEvent(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	responseOK(rr, req, nil, []models.Booking{})

	assert.Equal(t, http.StatusOK, rr.Code)

	var actualResponse EventInfoResponse
	err := json.Unmarshal(rr.Body.Bytes(), &actualResponse)
	require.NoError(t, err)

	assert.Equal(t, "OK", actualResponse.Status)
	assert.Equal(t, "", actualResponse.Error)
	assert.Nil(t, actualResponse.Event)
	assert.Empty(t, actualResponse.Booking)
}

func TestResponseWithNilBookings(t *testing.T) {
	t.Parallel()

	// Test data
	testTime := time.Date(2024, 12, 25, 18, 0, 0, 0, time.UTC)
	testEvent := &models.Event{
		ID:          1,
		Title:       "Test Event",
		Date:        testTime,
		TotalSeats:  100,
		BookedSeats: 50,
	}

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	responseOK(rr, req, testEvent, nil)

	assert.Equal(t, http.StatusOK, rr.Code)

	var actualResponse EventInfoResponse
	err := json.Unmarshal(rr.Body.Bytes(), &actualResponse)
	require.NoError(t, err)

	assert.Equal(t, "OK", actualResponse.Status)
	assert.Equal(t, "", actualResponse.Error)
	require.NotNil(t, actualResponse.Event)
	assert.Equal(t, 1, actualResponse.Event.ID)
	assert.Empty(t, actualResponse.Booking)
}

func TestEventGetterErrorScenarios(t *testing.T) {
	t.Parallel()

	logger := slogdiscard.NewDiscardLogger()

	testCases := []struct {
		name           string
		eventID        string
		mockError      error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Event not found error",
			eventID:        "1",
			mockError:      errors.New("event not found"),
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"status":"Error","error":"event not found"}`,
		},
		{
			name:           "Database error",
			eventID:        "1",
			mockError:      errors.New("database connection failed"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"status":"Error","error":"failed to get event information"}`,
		},
		{
			name:           "Timeout error",
			eventID:        "1",
			mockError:      errors.New("query timeout"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"status":"Error","error":"failed to get event information"}`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mockGetter := mocks.NewEventGetter(t)
			mockGetter.On("GetEventWithBookings", 1).Return(nil, nil, tc.mockError)

			handler := New(logger, mockGetter)

			router := chi.NewRouter()
			router.Route("/events", func(r chi.Router) {
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/info", handler)
				})
			})

			req, err := http.NewRequest("GET", "/events/1/info", nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)
			assert.JSONEq(t, tc.expectedBody, rr.Body.String())

			mockGetter.AssertExpectations(t)
		})
	}
}

func TestHandlerWithChiContext(t *testing.T) {
	t.Parallel()

	logger := slogdiscard.NewDiscardLogger()
	mockGetter := mocks.NewEventGetter(t)
	handler := New(logger, mockGetter)

	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "123")

	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	testEvent := &models.Event{ID: 123, Title: "Test Event"}
	testBookings := []models.Booking{}
	mockGetter.On("GetEventWithBookings", 123).Return(testEvent, testBookings, nil)

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response EventInfoResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "OK", response.Status)
	require.NotNil(t, response.Event)
	assert.Equal(t, 123, response.Event.ID)

	mockGetter.AssertExpectations(t)
}

func TestHandlerWithoutChiContext(t *testing.T) {
	t.Parallel()

	logger := slogdiscard.NewDiscardLogger()
	mockGetter := mocks.NewEventGetter(t)
	handler := New(logger, mockGetter)

	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "event id is required")
}
