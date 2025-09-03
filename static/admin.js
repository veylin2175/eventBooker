document.addEventListener('DOMContentLoaded', function() {
    loadAdminEvents();
    setupAdminEventListeners();
});

function setupAdminEventListeners() {
    document.getElementById('create-event-form').addEventListener('submit', function(e) {
        e.preventDefault();
        createEvent();
    });
}

function loadAdminEvents() {
    fetch('/events')
        .then(response => {
            if (!response.ok) {
                throw new Error('Network response was not ok');
            }
            return response.json();
        })
        .then(data => {
            console.log('Received admin data:', data);
            displayAdminEvents(data); // передаем весь объект
        })
        .catch(error => {
            console.error('Error loading events:', error);
            document.getElementById('admin-events-container').innerHTML =
                '<p class="error">Ошибка загрузки мероприятий</p>';
        });
}

function displayAdminEvents(responseData) {
    const container = document.getElementById('admin-events-container');

    console.log('displayAdminEvents called with:', responseData);

    // Получаем массив мероприятий из responseData.events
    let eventsArray = [];

    if (responseData && responseData.events && Array.isArray(responseData.events)) {
        eventsArray = responseData.events;
    } else if (Array.isArray(responseData)) {
        eventsArray = responseData; // на случай, если придет массив напрямую
    } else {
        console.error('Unexpected response format:', responseData);
        container.innerHTML = '<p class="error">Неверный формат данных</p>';
        return;
    }

    console.log('Final admin events array:', eventsArray);

    if (eventsArray.length === 0) {
        container.innerHTML = '<p>Нет созданных мероприятий</p>';
        return;
    }

    let html = '';
    eventsArray.forEach(event => {
        const freeSeats = event.total_seats ? event.total_seats - (event.booked_seats || 0) : 0;
        const date = event.date ? new Date(event.date) : new Date();
        const deadline = event.deadline_minutes ? `${event.deadline_minutes} минут` : 'Не указан';

        html += `
            <div class="event-card">
                <div class="event-title">${event.title || 'Без названия'}</div>
                <div class="event-info">
                    <div class="event-date">📅 ${date.toLocaleString('ru-RU')}</div>
                    <div class="event-seats">🪑 Всего мест: ${event.total_seats || 0}, свободно: ${freeSeats}</div>
                    <div class="event-deadline">⏰ Дедлайн: ${deadline}</div>
                    <div class="event-id">🆔 ID: ${event.id || 'N/A'}</div>
                </div>
            </div>
        `;
    });

    container.innerHTML = html;
}

function createEvent() {
    const title = document.getElementById('title').value;
    const date = document.getElementById('date').value;
    const totalSeats = document.getElementById('total-seats').value;
    const deadline = document.getElementById('deadline').value;

    if (!title || !date || !totalSeats || !deadline) {
        showError('Пожалуйста, заполните все поля');
        return;
    }

    const data = {
        title: title,
        date: new Date(date).toISOString(),
        total_seats: parseInt(totalSeats),
        deadline: parseInt(deadline)
    };

    fetch('/events', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(data)
    })
        .then(response => response.json())
        .then(result => {
            if (result.status === 'OK') {
                showSuccess('Мероприятие успешно создано!');
                document.getElementById('create-event-form').reset();
                loadAdminEvents();
            } else {
                showError('Ошибка создания: ' + result.error);
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showError('Ошибка сети при создании мероприятия');
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