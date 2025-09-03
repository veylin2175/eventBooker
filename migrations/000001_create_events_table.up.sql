CREATE TABLE IF NOT EXISTS events
(
    id               SERIAL PRIMARY KEY,
    title            TEXT                                                    NOT NULL,
    date             TIMESTAMP WITH TIME ZONE                                NOT NULL,
    total_seats      INTEGER                                                 NOT NULL CHECK (total_seats > 0),
    deadline_minutes INTEGER                                                 NOT NULL CHECK (deadline_minutes > 0),
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT TIMEZONE('utc', NOW()) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_events_date ON events (date);