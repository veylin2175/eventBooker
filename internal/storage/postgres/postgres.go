package postgres

import (
	"database/sql"
	"eventBooker/internal/config"
	"eventBooker/internal/models"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type Storage struct {
	DB *sql.DB
}

func InitDB(dbCfg *config.Database) (*Storage, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		dbCfg.Host,
		dbCfg.Port,
		dbCfg.User,
		dbCfg.Password,
		dbCfg.DBName,
		dbCfg.SSLMode,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the database: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to the database: %w", err)
	}

	return &Storage{DB: db}, nil
}

func (s *Storage) Close() error {
	return s.DB.Close()
}

func (s *Storage) CreateEvent(title string, date time.Time, totalSeats, deadline int) (int, error) {
	query := `
		INSERT INTO events (title, date, total_seats, deadline_minutes)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	var id int
	err := s.DB.QueryRow(query, title, date, totalSeats, deadline).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to create event: %w", err)
	}

	return id, nil
}

func (s *Storage) GetEvent(id int) (*models.Event, error) {
	query := `
		SELECT id, title, date, total_seats, deadline_minutes
		FROM events
		WHERE id = $1`

	var event models.Event
	err := s.DB.QueryRow(query, id).Scan(
		&event.ID,
		&event.Title,
		&event.Date,
		&event.TotalSeats,
		&event.Deadline,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("event not found")
		}
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	bookedQuery := `
		SELECT COUNT(*) 
		FROM bookings 
		WHERE event_id = $1 AND confirmed = true`

	err = s.DB.QueryRow(bookedQuery, id).Scan(&event.BookedSeats)
	if err != nil {
		return nil, fmt.Errorf("failed to get booked seats count: %w", err)
	}

	return &event, nil
}

func (s *Storage) BookEvent(eventID int, userID string) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var totalSeats, bookedSeats int
	countQuery := `
		SELECT e.total_seats, COUNT(b.id)
		FROM events e
		LEFT JOIN bookings b ON e.id = b.event_id AND b.confirmed = true
		WHERE e.id = $1
		GROUP BY e.id, e.total_seats`

	err = tx.QueryRow(countQuery, eventID).Scan(&totalSeats, &bookedSeats)
	if err != nil {
		return fmt.Errorf("failed to get event seats info: %w", err)
	}

	if bookedSeats >= totalSeats {
		return fmt.Errorf("no available seats")
	}

	var existingBooking bool
	checkQuery := `
		SELECT EXISTS(
			SELECT 1 FROM bookings 
			WHERE event_id = $1 AND user_id = $2 AND confirmed = false
		)`

	err = tx.QueryRow(checkQuery, eventID, userID).Scan(&existingBooking)
	if err != nil {
		return fmt.Errorf("failed to check existing booking: %w", err)
	}

	if existingBooking {
		return fmt.Errorf("user already has pending booking for this event")
	}

	insertQuery := `
		INSERT INTO bookings (event_id, user_id, created_at, confirmed)
		VALUES ($1, $2, NOW(), false)`

	_, err = tx.Exec(insertQuery, eventID, userID)
	if err != nil {
		return fmt.Errorf("failed to create booking: %w", err)
	}

	return tx.Commit()
}

func (s *Storage) ConfirmBooking(eventID int, userID string) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var bookingID int
	checkQuery := `
		SELECT id FROM bookings 
		WHERE event_id = $1 AND user_id = $2 AND confirmed = false`

	err = tx.QueryRow(checkQuery, eventID, userID).Scan(&bookingID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no pending booking found")
		}
		return fmt.Errorf("failed to check booking: %w", err)
	}

	var totalSeats, bookedSeats int
	countQuery := `
		SELECT e.total_seats, COUNT(b.id)
		FROM events e
		LEFT JOIN bookings b ON e.id = b.event_id AND b.confirmed = true
		WHERE e.id = $1
		GROUP BY e.id, e.total_seats`

	err = tx.QueryRow(countQuery, eventID).Scan(&totalSeats, &bookedSeats)
	if err != nil {
		return fmt.Errorf("failed to get event seats info: %w", err)
	}

	if bookedSeats >= totalSeats {
		return fmt.Errorf("no available seats")
	}

	updateQuery := `
		UPDATE bookings 
		SET confirmed = true 
		WHERE id = $1`

	_, err = tx.Exec(updateQuery, bookingID)
	if err != nil {
		return fmt.Errorf("failed to confirm booking: %w", err)
	}

	return tx.Commit()
}

func (s *Storage) CancelExpiredBookings() error {
	query := `
		DELETE FROM bookings 
		WHERE confirmed = false 
		AND created_at < NOW() - INTERVAL '1 minute' * (
			SELECT deadline_minutes 
			FROM events 
			WHERE id = bookings.event_id
		)`

	result, err := s.DB.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to cancel expired bookings: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		fmt.Printf("Cancelled %d expired bookings\n", rowsAffected)
	}

	return nil
}

func (s *Storage) GetEventWithBookings(eventID int) (*models.Event, []models.Booking, error) {
	event, err := s.GetEvent(eventID)
	if err != nil {
		return nil, nil, err
	}

	query := `
		SELECT id, event_id, user_id, created_at, confirmed
		FROM bookings
		WHERE event_id = $1
		ORDER BY created_at DESC`

	rows, err := s.DB.Query(query, eventID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get bookings: %w", err)
	}
	defer rows.Close()

	var bookings []models.Booking
	for rows.Next() {
		var booking models.Booking
		err = rows.Scan(
			&booking.ID,
			&booking.EventID,
			&booking.UserID,
			&booking.CreatedAt,
			&booking.Confirmed,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to scan booking: %w", err)
		}
		bookings = append(bookings, booking)
	}

	if err = rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating bookings: %w", err)
	}

	return event, bookings, nil
}

func (s *Storage) GetAllEvents() ([]models.Event, error) {
	query := `
        SELECT id, title, date, total_seats, deadline_minutes
        FROM events
        ORDER BY date ASC`

	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var event models.Event
		err := rows.Scan(
			&event.ID,
			&event.Title,
			&event.Date,
			&event.TotalSeats,
			&event.Deadline,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		bookedQuery := `
            SELECT COUNT(*) 
            FROM bookings 
            WHERE event_id = $1 AND confirmed = true`

		err = s.DB.QueryRow(bookedQuery, event.ID).Scan(&event.BookedSeats)
		if err != nil {
			return nil, fmt.Errorf("failed to get booked seats count: %w", err)
		}

		events = append(events, event)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events: %w", err)
	}

	return events, nil
}
