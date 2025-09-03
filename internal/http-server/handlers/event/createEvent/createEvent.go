package createEvent

import (
	"errors"
	"eventBooker/internal/lib/api/response"
	"eventBooker/internal/lib/logger/sl"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"log/slog"
	"net/http"
	"time"
)

type EventRequest struct {
	Title      string    `json:"title" validate:"required"`
	Date       time.Time `json:"date" validate:"required"`
	TotalSeats int       `json:"total_seats" validate:"required"`
	Deadline   int       `json:"deadline" validate:"required"`
}

type EventResponse struct {
	response.Response
	EventId int `json:"event_id"`
}

//go:generate go run github.com/vektra/mockery/v2@v2.51.1 --name=EventCreator
type EventCreator interface {
	CreateEvent(title string, date time.Time, totalSeats, deadline int) (int, error)
}

func New(log *slog.Logger, event EventCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.event.createEvent.New"

		log = log.With(
			slog.String("op", op),
		)

		var req EventRequest

		err := render.DecodeJSON(r.Body, &req)
		if err != nil {
			log.Error("failed to decode request body", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("failed to decode request"))

			return
		}

		log.Info("request body decoded", slog.Any("request", req))

		if err = validator.New().Struct(req); err != nil {
			var validateErr validator.ValidationErrors
			errors.As(err, &validateErr)

			log.Error("invalid request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.ValidationError(validateErr))

			return
		}

		eventId, err := event.CreateEvent(req.Title, req.Date, req.TotalSeats, req.Deadline)
		if err != nil {
			log.Error("failed to add event", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("failed to add event"))

			return
		}

		log.Info("event added", slog.Int("id", eventId))

		responseOK(w, r, eventId)
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, eventId int) {
	render.JSON(w, r, EventResponse{
		Response: response.OK(),
		EventId:  eventId,
	})
}
