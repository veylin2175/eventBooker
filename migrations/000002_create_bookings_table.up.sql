CREATE TABLE IF NOT EXISTS bookings
(
    id         SERIAL PRIMARY KEY,
    event_id   INTEGER NOT NULL,
    user_id    TEXT    NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT TIMEZONE('utc', NOW()) NOT NULL,
    confirmed  BOOLEAN NOT NULL         DEFAULT FALSE,

    CONSTRAINT fk_event
        FOREIGN KEY (event_id)
            REFERENCES events (id)
            ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_bookings_event_id ON bookings (event_id);
CREATE INDEX IF NOT EXISTS idx_bookings_user_id ON bookings (user_id);
CREATE INDEX IF NOT EXISTS idx_bookings_confirmed ON bookings (confirmed);
CREATE INDEX IF NOT EXISTS idx_bookings_created_at ON bookings (created_at);

CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_pending_booking_per_user
    ON bookings (event_id, user_id)
    WHERE confirmed = FALSE;