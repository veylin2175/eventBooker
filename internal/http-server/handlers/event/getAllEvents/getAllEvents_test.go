package getAllEvents

import (
	"encoding/json"
	"errors"
	"eventBooker/internal/http-server/handlers/event/getAllEvents/mocks"
	"eventBooker/internal/lib/logger/handlers/slogdiscard"
	"eventBooker/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAllEventsHandler(t *testing.T) {
	t.Parallel()

	logger := slogdiscard.NewDiscardLogger()

	testTime := time.Date(2024, 12, 25, 18, 0, 0, 0, time.UTC)
	testEvents := []models.Event{
		{
			ID:          1,
			Title:       "Test Event 1",
			Date:        testTime,
			TotalSeats:  100,
			BookedSeats: 50,
		},
		{
			ID:          2,
			Title:       "Test Event 2",
			Date:        testTime.Add(24 * time.Hour),
			TotalSeats:  200,
			BookedSeats: 75,
		},
	}

	testCases := []struct {
		name           string
		mockSetup      func(mock *mocks.EventsGetter)
		expectedStatus int
		expectedBody   string
		checkBody      func(t *testing.T, body string)
	}{
		{
			name: "Success with events",
			mockSetup: func(mock *mocks.EventsGetter) {
				mock.On("GetAllEvents").Return(testEvents, nil)
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				var response EventsResponse
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)

				assert.Equal(t, "OK", response.Status)
				assert.Equal(t, "", response.Error)
				assert.Len(t, response.Events, 2)
				assert.Equal(t, 1, response.Events[0].ID)
				assert.Equal(t, "Test Event 1", response.Events[0].Title)
				assert.Equal(t, 2, response.Events[1].ID)
				assert.Equal(t, "Test Event 2", response.Events[1].Title)
			},
		},
		{
			name: "Success with empty events",
			mockSetup: func(mock *mocks.EventsGetter) {
				mock.On("GetAllEvents").Return([]models.Event{}, nil)
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				var response EventsResponse
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)

				assert.Equal(t, "OK", response.Status)
				assert.Equal(t, "", response.Error)
				assert.Empty(t, response.Events)
			},
		},
		{
			name: "Internal server error",
			mockSetup: func(mock *mocks.EventsGetter) {
				mock.On("GetAllEvents").Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"status":"Error","error":"failed to get events"}`,
		},
		{
			name: "Nil events with error",
			mockSetup: func(mock *mocks.EventsGetter) {
				mock.On("GetAllEvents").Return(nil, errors.New("connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"status":"Error","error":"failed to get events"}`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockGetter := mocks.NewEventsGetter(t)
			tc.mockSetup(mockGetter)

			handler := New(logger, mockGetter)

			req, err := http.NewRequest("GET", "/events", nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code, "Status code mismatch")

			if tc.expectedBody != "" {
				assert.JSONEq(t, tc.expectedBody, rr.Body.String(), "Response body mismatch")
			} else if tc.checkBody != nil {
				tc.checkBody(t, rr.Body.String())
			}

			mockGetter.AssertExpectations(t)
		})
	}
}

func TestResponseOK(t *testing.T) {
	t.Parallel()

	// Test data
	testTime := time.Date(2024, 12, 25, 18, 0, 0, 0, time.UTC)
	testEvents := []models.Event{
		{
			ID:          1,
			Title:       "Test Event",
			Date:        testTime,
			TotalSeats:  100,
			BookedSeats: 50,
		},
	}

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	responseOK(rr, req, testEvents)

	assert.Equal(t, http.StatusOK, rr.Code)

	var actualResponse EventsResponse
	err := json.Unmarshal(rr.Body.Bytes(), &actualResponse)
	require.NoError(t, err)

	assert.Equal(t, "OK", actualResponse.Status)
	assert.Equal(t, "", actualResponse.Error)
	require.Len(t, actualResponse.Events, 1)
	assert.Equal(t, 1, actualResponse.Events[0].ID)
	assert.Equal(t, "Test Event", actualResponse.Events[0].Title)
	assert.Equal(t, 100, actualResponse.Events[0].TotalSeats)
	assert.Equal(t, 50, actualResponse.Events[0].BookedSeats)
}

func TestEmptyEventsResponse(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	responseOK(rr, req, []models.Event{})

	assert.Equal(t, http.StatusOK, rr.Code)

	var actualResponse EventsResponse
	err := json.Unmarshal(rr.Body.Bytes(), &actualResponse)
	require.NoError(t, err)

	assert.Equal(t, "OK", actualResponse.Status)
	assert.Equal(t, "", actualResponse.Error)
	assert.Empty(t, actualResponse.Events)
}

func TestNilEventsResponse(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	responseOK(rr, req, nil)

	assert.Equal(t, http.StatusOK, rr.Code)

	var actualResponse EventsResponse
	err := json.Unmarshal(rr.Body.Bytes(), &actualResponse)
	require.NoError(t, err)

	assert.Equal(t, "OK", actualResponse.Status)
	assert.Equal(t, "", actualResponse.Error)
	assert.Empty(t, actualResponse.Events)
}

func TestErrorScenarios(t *testing.T) {
	t.Parallel()

	logger := slogdiscard.NewDiscardLogger()

	testCases := []struct {
		name           string
		mockError      error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Database connection error",
			mockError:      errors.New("database connection failed"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"status":"Error","error":"failed to get events"}`,
		},
		{
			name:           "Timeout error",
			mockError:      errors.New("request timeout"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"status":"Error","error":"failed to get events"}`,
		},
		{
			name:           "Unknown error",
			mockError:      errors.New("unknown error occurred"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"status":"Error","error":"failed to get events"}`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mockGetter := mocks.NewEventsGetter(t)
			mockGetter.On("GetAllEvents").Return(nil, tc.mockError)

			handler := New(logger, mockGetter)

			req, err := http.NewRequest("GET", "/events", nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)
			assert.JSONEq(t, tc.expectedBody, rr.Body.String())

			mockGetter.AssertExpectations(t)
		})
	}
}

func TestHandlerWorksWithAnyHTTPMethod(t *testing.T) {
	t.Parallel()

	logger := slogdiscard.NewDiscardLogger()
	mockGetter := mocks.NewEventsGetter(t)

	testEvents := []models.Event{
		{ID: 1, Title: "Test Event"},
	}
	mockGetter.On("GetAllEvents").Return(testEvents, nil)

	handler := New(logger, mockGetter)

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req, err := http.NewRequest(method, "/events", nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)

			var response EventsResponse
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, "OK", response.Status)
			assert.Len(t, response.Events, 1)
			assert.Equal(t, 1, response.Events[0].ID)
		})
	}

	mockGetter.AssertNumberOfCalls(t, "GetAllEvents", len(methods))
}

func TestHandlerWorksWithDifferentURLs(t *testing.T) {
	t.Parallel()

	logger := slogdiscard.NewDiscardLogger()
	mockGetter := mocks.NewEventsGetter(t)

	testEvents := []models.Event{}
	mockGetter.On("GetAllEvents").Return(testEvents, nil)

	handler := New(logger, mockGetter)

	urls := []string{
		"/events",
		"/events/",
		"/api/events",
		"/",
		"/some/path",
	}

	for _, url := range urls {
		t.Run(url, func(t *testing.T) {
			req, err := http.NewRequest("GET", url, nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
		})
	}

	mockGetter.AssertNumberOfCalls(t, "GetAllEvents", len(urls))
}
