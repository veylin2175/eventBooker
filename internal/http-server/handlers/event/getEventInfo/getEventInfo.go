package getEventInfo

import (
	"eventBooker/internal/lib/api/response"
	"eventBooker/internal/lib/logger/sl"
	"eventBooker/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	"strconv"
)

type EventInfoResponse struct {
	response.Response
	Event   *models.Event    `json:"event"`
	Booking []models.Booking `json:"bookings"`
}

//go:generate go run github.com/vektra/mockery/v2@v2.51.1 --name=EventGetter
type EventGetter interface {
	GetEventWithBookings(eventID int) (*models.Event, []models.Booking, error)
	GetAllEvents() ([]models.Event, error)
}

func New(log *slog.Logger, info EventGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.event.getEventInfo.New"

		log = log.With(slog.String("op", op))

		eventIdStr := chi.URLParam(r, "id")
		if eventIdStr == "" {
			log.Error("event id is required")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("event id is required"))
			return
		}

		eventID, err := strconv.Atoi(eventIdStr)
		if err != nil {
			log.Error("invalid event id format", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid event id format"))
			return
		}

		log = log.With(slog.Int("event_id", eventID))

		event, booking, err := info.GetEventWithBookings(eventID)
		if err != nil {
			log.Error("failed to get event information", sl.Err(err))

			// Обработка специфичной ошибки
			if err.Error() == "event not found" {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, response.Error("event not found"))
				return
			}

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("failed to get event information"))
			return
		}

		log.Info("event info successfully received", slog.Int("event_id", eventID))

		responseOK(w, r, event, booking)
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, event *models.Event, booking []models.Booking) {
	render.JSON(w, r, EventInfoResponse{
		Response: response.OK(),
		Event:    event,
		Booking:  booking,
	})
}
