// ═══════════════════════════════════════════════════════════════════════════════════════
// ── EVENT TIMELINE ──
// ═══════════════════════════════════════════════════════════════════════════════════════

function changeEventDate(delta) {
    const input = document.getElementById('eventDateFilter');
    const d = new Date(input.value);
    d.setDate(d.getDate() + delta);
    input.value = toLocalDate(d);
    loadEvents();
}

async function loadEvents() {
    const date = document.getElementById('eventDateFilter').value;
    if (!date) return;
    const tzParam = `tz=${encodeURIComponent(TZ)}`;

    try {
        const [feedRes, sleepRes] = await Promise.all([
            fetch(`${API}?date=${date}&${tzParam}`),
            fetch(`${SLEEP_API}?date=${date}&${tzParam}`),
        ]);
        const feedings = await feedRes.json();
        const sleeps = await sleepRes.json();
        renderEvents(feedings, sleeps, date);
    } catch (err) {
        log.error('Failed to load events:', err);
    }
}

function renderEvents(feedings, sleeps, date) {
    const container = document.getElementById('eventsList');

    // Build unified event list
    const events = [];
    if (feedings) {
        feedings.forEach(f => {
            events.push({
                type: 'feeding',
                startTime: f.end_time,
                endTime: f.end_time,
                detail: `${f.amount_ml} ml`,
                badgeClass: 'bg-primary',
            });
        });
    }
    if (sleeps) {
        sleeps.forEach(s => {
            const isActive = s.status === 'active' || !s.end_time;
            const typeLabel = s.sleep_type === 'night' ? 'Night Sleep' : 'Nap';
            events.push({
                type: 'sleep',
                startTime: s.start_time,
                endTime: s.end_time || s.start_time,
                detail: isActive ? 'Active' : formatDuration(s.start_time, s.end_time),
                badgeClass: isActive ? 'bg-warning text-dark' : 'bg-sleep',
                label: typeLabel,
            });
        });
    }
    if (events.length === 0) {
        container.innerHTML = '<p class="text-center text-muted py-3">No events recorded for this date</p>';
        return;
    }

    // Sort chronologically (ascending)
    events.sort((a, b) => new Date(a.startTime) - new Date(b.startTime));

    const dayLabel = new Date(date + 'T00:00:00').toLocaleDateString(undefined, { weekday: 'short', year: 'numeric', month: 'short', day: 'numeric' });
    const feedingCount = events.filter(e => e.type === 'feeding').length;
    const sleepCount = events.filter(e => e.type === 'sleep').length;

    const rows = events.map((ev, idx) => {
        const icons = { feeding: '🍼', sleep: '😴' };
        const defaultLabels = { feeding: 'Feeding', sleep: 'Sleep' };
        const icon = icons[ev.type] || '📌';
        const label = ev.label || defaultLabels[ev.type] || ev.type;
        const timeStr = ev.startTime === ev.endTime
            ? formatTime(ev.startTime)
            : `${formatTime(ev.startTime)} – ${formatTime(ev.endTime)}`;
        let gapHtml = '';
        if (idx > 0) {
            const prevEnd = new Date(events[idx - 1].endTime);
            const currStart = new Date(ev.startTime);
            gapHtml = buildTimeGapRow(prevEnd, currStart, 3);
        }
        return `${gapHtml}
        <tr class="feeding-row">
            <td class="text-nowrap">${timeStr}</td>
            <td>${icon} ${label}</td>
            <td class="text-end"><span class="badge ${ev.badgeClass} rounded-pill">${ev.detail}</span></td>
        </tr>`;
    }).join('');

    container.innerHTML = `
        <div class="card">
            <div class="card-body">
                <div class="d-flex justify-content-between align-items-center bg-light px-3 py-2 rounded-top border mb-0">
                    <strong>${dayLabel}</strong>
                    <span>
                        <span class="badge bg-primary rounded-pill">${feedingCount} feeding${feedingCount !== 1 ? 's' : ''}</span>
                        <span class="badge bg-sleep rounded-pill">${sleepCount} sleep${sleepCount !== 1 ? 's' : ''}</span>
                    </span>
                </div>
                <div class="table-responsive border border-top-0 rounded-bottom">
                    <table class="table table-sm table-striped mb-0">
                        <thead class="table-light">
                            <tr><th>Time</th><th>Type</th><th class="text-end">Detail</th></tr>
                        </thead>
                        <tbody>${rows}</tbody>
                    </table>
                </div>
            </div>
        </div>`;
}
