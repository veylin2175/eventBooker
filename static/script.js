document.addEventListener('DOMContentLoaded', function() {
    loadEvents();
    setupEventListeners();
});

function setupEventListeners() {
    document.getElementById('booking-form').addEventListener('submit', function(e) {
        e.preventDefault();
        bookEvent();
    });

    document.getElementById('confirmation-form').addEventListener('submit', function(e) {
        e.preventDefault();
        confirmBooking();
    });
}

function loadEvents() {
    fetch('/events')
        .then(response => {
            if (!response.ok) {
                throw new Error('Network response was not ok');
            }
            return response.json();
        })
        .then(data => {
            console.log('Received data:', data);
            displayEvents(data.events || []);
        })
        .catch(error => {
            console.error('Error loading events:', error);
            document.getElementById('events-container').innerHTML =
                '<p class="error">Ошибка загрузки мероприятий</p>';
        });
}

function displayEvents(events) {
    const container = document.getElementById('events-container');

    // Отладочный вывод
    console.log('displayEvents called with:', events);

    // Обрабатываем разные форматы ответа
    let eventsArray = [];

    if (Array.isArray(events)) {
        eventsArray = events;
    } else if (events && typeof events === 'object') {
        // Проверяем различные возможные структуры ответа
        if (events.events && Array.isArray(events.events)) {
            eventsArray = events.events;
        } else if (events.data && Array.isArray(events.data)) {
            eventsArray = events.data;
        } else if (events.items && Array.isArray(events.items)) {
            eventsArray = events.items;
        } else {
            // Если это объект, но не массив, пробуем извлечь значения
            eventsArray = Object.values(events);
        }
    }

    console.log('Final events array:', eventsArray);

    if (!Array.isArray(eventsArray) || eventsArray.length === 0) {
        container.innerHTML = '<p>Нет доступных мероприятий</p>';
        return;
    }

    let html = '';
    eventsArray.forEach(event => {
        // Добавляем проверки на существование полей
        const freeSeats = event.total_seats ? event.total_seats - (event.booked_seats || 0) : 0;
        const eventDate = event.date ? new Date(event.date).toLocaleString('ru-RU') : 'Дата не указана';
        const deadline = event.deadline_minutes ? `${event.deadline_minutes} минут` : 'Не указан';

        html += `
            <div class="event-card">
                <div class="event-title">${event.title || 'Без названия'}</div>
                <div class="event-info">
                    <div class="event-date">📅 ${eventDate}</div>
                    <div class="event-seats">🪑 Свободных мест: ${freeSeats} из ${event.total_seats || 0}</div>
                    <div class="event-deadline">⏰ Дедлайн: ${deadline}</div>
                </div>
                <div class="event-actions">
                    <button onclick="showBookingForm(${event.id || 0})">Забронировать</button>
                </div>
            </div>
        `;
    });

    container.innerHTML = html;
}

function showBookingForm(eventId) {
    document.getElementById('event-id').value = eventId;
    document.getElementById('booking-section').style.display = 'block';
    document.getElementById('confirmation-section').style.display = 'none';
    document.getElementById('user-id').focus();
}

function bookEvent() {
    const eventId = document.getElementById('event-id').value;
    const userId = document.getElementById('user-id').value;

    if (!userId.trim()) {
        alert('Пожалуйста, введите ваш ID пользователя');
        return;
    }

    const data = {
        user_id: userId
    };

    fetch(`/events/${eventId}/book`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(data)
    })
        .then(response => response.json())
        .then(result => {
            if (result.status === 'OK') {
                showSuccess('Место успешно забронировано! Не забудьте подтвердить бронь.');
                showConfirmationForm(eventId);
                loadEvents();
            } else {
                showError('Ошибка бронирования: ' + result.error);
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showError('Ошибка сети при бронировании');
        });
}

function showConfirmationForm(eventId) {
    document.getElementById('confirm-event-id').value = eventId;
    document.getElementById('confirmation-section').style.display = 'block';
    document.getElementById('booking-section').style.display = 'none';
    document.getElementById('confirm-user-id').value = document.getElementById('user-id').value;
    document.getElementById('confirm-user-id').focus();
}

function confirmBooking() {
    const eventId = document.getElementById('confirm-event-id').value;
    const userId = document.getElementById('confirm-user-id').value;

    if (!userId.trim()) {
        alert('Пожалуйста, введите ваш ID пользователя');
        return;
    }

    const data = {
        user_id: userId
    };

    fetch(`/events/${eventId}/confirm`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(data)
    })
        .then(response => response.json())
        .then(result => {
            if (result.status === 'OK') {
                showSuccess('Бронь успешно подтверждена!');
                document.getElementById('confirmation-section').style.display = 'none';
                loadEvents();
            } else {
                showError('Ошибка подтверждения: ' + result.error);
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showError('Ошибка сети при подтверждении');
        });
}

function showSuccess(message) {
    const successDiv = document.createElement('div');
    successDiv.className = 'success';
    successDiv.textContent = message;
    document.querySelector('main').insertBefore(successDiv, document.querySelector('main').firstChild);

    setTimeout(() => {
        successDiv.remove();
    }, 5000);
}

function showError(message) {
    const errorDiv = document.createElement('div');
    errorDiv.className = 'error';
    errorDiv.textContent = message;
    document.querySelector('main').insertBefore(errorDiv, document.querySelector('main').firstChild);

    setTimeout(() => {
        errorDiv.remove();
    }, 5000);
}