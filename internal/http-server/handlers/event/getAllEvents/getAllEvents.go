package getAllEvents

import (
	"eventBooker/internal/lib/api/response"
	"eventBooker/internal/lib/logger/sl"
	"eventBooker/internal/models"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
)

type EventsResponse struct {
	response.Response
	Events []models.Event `json:"events"`
}

//go:generate go run github.com/vektra/mockery/v2@v2.51.1 --name=EventsGetter
type EventsGetter interface {
	GetAllEvents() ([]models.Event, error)
}

func New(log *slog.Logger, eventsGetter EventsGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.event.getAllEvents.New"

		log = log.With(slog.String("op", op))

		events, err := eventsGetter.GetAllEvents()
		if err != nil {
			log.Error("failed to get events", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("failed to get events"))
			return
		}

		log.Info("events retrieved successfully", slog.Int("count", len(events)))

		responseOK(w, r, events)
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, events []models.Event) {
	render.JSON(w, r, EventsResponse{
		Response: response.OK(),
		Events:   events,
	})
}
