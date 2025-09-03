package createBooking

import (
	"errors"
	"eventBooker/internal/lib/api/response"
	"eventBooker/internal/lib/logger/sl"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"log/slog"
	"net/http"
	"strconv"
)

type BookingRequest struct {
	UserId string `json:"user_id" validate:"required"`
}

type BookingResponse struct {
	response.Response
}

//go:generate go run github.com/vektra/mockery/v2@v2.51.1 --name=BookingCreator
type BookingCreator interface {
	BookEvent(eventID int, userID string) error
}

func New(log *slog.Logger, booking BookingCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.event.createBooking.New"

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

		var req BookingRequest

		err = render.DecodeJSON(r.Body, &req)
		if err != nil {
			log.Error("failed to decode request body", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("failed to decode request"))
			return
		}

		log.Info("request body decoded", slog.Any("request", req))

		if err = validator.New().Struct(req); err != nil {
			var validateErr validator.ValidationErrors
			if errors.As(err, &validateErr) {
				log.Error("invalid request", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, response.ValidationError(validateErr))
				return
			}
		}

		err = booking.BookEvent(eventID, req.UserId)
		if err != nil {
			log.Error("failed to book event", sl.Err(err))

			switch err.Error() {
			case "no available seats":
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, response.Error("no available seats"))
				return
			case "user already has pending booking for this event":
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, response.Error("user already has pending booking for this event"))
				return
			default:
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, response.Error("failed to book event"))
				return
			}
		}

		log.Info("event booked successfully", slog.String("user_id", req.UserId))

		responseOK(w, r)
	}
}

func responseOK(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, BookingResponse{
		Response: response.OK(),
	})
}
