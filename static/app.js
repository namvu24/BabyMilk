// ── Global state ──
const API = '/api/feedings';
const SLEEP_API = '/api/sleeps';
const TZ = Intl.DateTimeFormat().resolvedOptions().timeZone; // e.g. "Asia/Ho_Chi_Minh"

// ── Logger ──
// Logs are shown on localhost / dev hostnames and suppressed in production
const isDev = ['localhost', '127.0.0.1', ''].includes(location.hostname)
    || location.hostname.includes('dev') || location.port !== '';
const log = {
    debug: (...args) => isDev && console.log('[DEBUG]', ...args),
    info:  (...args) => isDev && console.info('[INFO]', ...args),
    warn:  (...args) => console.warn('[WARN]', ...args),   // always show warnings
    error: (...args) => console.error('[ERROR]', ...args),  // always show errors
};

let dailyChart = null;        // Chart.js instance (destroyed & recreated on each render)
let feedingStartTime = null;  // Date object when Start Feeding was pressed (null = idle)
let timerInterval = null;     // setInterval ID for the live stopwatch
let toastTimer = null;        // setTimeout ID so we can cancel overlapping toasts

// Sleep-specific state
let sleepChart = null;        // Chart.js instance for sleep chart
let sleepStartTime = null;    // Date object when Start Sleep was pressed (null = idle)
let sleepTimerInterval = null;// setInterval ID for the sleep stopwatch

// ── Toast notification ──
// Shows a brief message at the top of the screen (green for success, red for error)
function showToast(message, type = 'success') {
    const toast = document.getElementById('toast');
    toast.textContent = message;
    toast.style.backgroundColor = type === 'success' ? '#198754' : '#dc3545';
    toast.style.display = 'block';
    toast.style.opacity = '1';
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => {
        toast.style.opacity = '0';
        setTimeout(() => { toast.style.display = 'none'; }, 300);
    }, 2000);
}

// ── Dark mode toggle ──
function toggleDarkMode() {
    const html = document.documentElement;
    const isDark = html.getAttribute('data-bs-theme') === 'dark';
    const next = isDark ? 'light' : 'dark';
    html.setAttribute('data-bs-theme', next);
    document.getElementById('darkModeBtn').textContent = next === 'dark' ? '☀️' : '🌙';
    localStorage.setItem('theme', next);
}

// Apply saved theme on load
(function () {
    const saved = localStorage.getItem('theme') ||
        (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
    document.documentElement.setAttribute('data-bs-theme', saved);
    document.addEventListener('DOMContentLoaded', () => {
        document.getElementById('darkModeBtn').textContent = saved === 'dark' ? '☀️' : '🌙';
    });
})();

// ── Start / Stop feeding timer ──
// Toggles between recording a feeding (live stopwatch) and stopping it
function toggleFeeding() {
    const btn = document.getElementById('startStopBtn');
    const timerEl = document.getElementById('feedingTimer');

    if (!feedingStartTime) {
        // START
        feedingStartTime = new Date();
        document.getElementById('feedingDate').value = toLocalDate(feedingStartTime);
        document.getElementById('startTime').value = toLocalTime(feedingStartTime);
        document.getElementById('endTime').value = toLocalTime(feedingStartTime);

        btn.textContent = '⏹ Stop Feeding';
        btn.classList.replace('btn-success', 'btn-danger');
        timerEl.classList.remove('d-none');

        timerInterval = setInterval(() => {
            const elapsed = Math.floor((Date.now() - feedingStartTime) / 1000);
            const m = String(Math.floor(elapsed / 60)).padStart(2, '0');
            const s = String(elapsed % 60).padStart(2, '0');
            timerEl.textContent = `⏱ ${m}:${s}`;
        }, 1000);
    } else {
        // STOP
        const now = new Date();
        document.getElementById('endTime').value = toLocalTime(now);

        clearInterval(timerInterval);
        timerInterval = null;
        feedingStartTime = null;

        btn.textContent = '▶ Start Feeding';
        btn.classList.replace('btn-danger', 'btn-success');
        timerEl.classList.add('d-none');
        timerEl.textContent = '';

        // Scroll to form and focus amount
        document.getElementById('amountMl').scrollIntoView({ behavior: 'smooth', block: 'center' });
        document.getElementById('amountMl').focus();
    }
}

// ── Form helpers ──
// Set date and time inputs to "now" (used on page load and after form submit)
function setDefaultTimes() {
    const now = new Date();
    document.getElementById('feedingDate').value = toLocalDate(now);
    document.getElementById('startTime').value = toLocalTime(now);
    document.getElementById('endTime').value = toLocalTime(now);
}

// Set end time = start time ("Set" button)
function setEndTimeFromStart() {
    document.getElementById('endTime').value = document.getElementById('startTime').value;
}

document.addEventListener('DOMContentLoaded', () => {
    setDefaultTimes();
    // Default month picker to current month
    const now = new Date();
    const ym = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
    document.getElementById('monthFilter').value = ym;
    loadFeedings();
    loadDailyTotals();
    loadQuickAmounts();
    document.getElementById('feedingForm').addEventListener('submit', handleSubmit);

    // Sync slider when amount input changes
    document.getElementById('amountMl').addEventListener('input', (e) => {
        document.getElementById('amountSlider').value = e.target.value;
    });

    // Snap time inputs to nearest 5 minutes on change
    ['startTime', 'endTime'].forEach(id => {
        document.getElementById(id).addEventListener('change', (e) => {
            const val = e.target.value; // "HH:MM"
            if (val) {
                const [h, m] = val.split(':').map(Number);
                const rounded = Math.round(m / 5) * 5;
                const hrs = rounded === 60 ? (h + 1) % 24 : h;
                const mins = rounded === 60 ? 0 : rounded;
                e.target.value = `${String(hrs).padStart(2, '0')}:${String(mins).padStart(2, '0')}`;
            }
        });
    });

    // Event delegation for edit/delete buttons
    document.getElementById('feedingsByDay').addEventListener('click', (e) => {
        const editBtn = e.target.closest('[data-edit-id]');
        if (editBtn) {
            editFeeding(
                parseInt(editBtn.dataset.editId),
                parseInt(editBtn.dataset.amount),
                decodeURIComponent(editBtn.dataset.start),
                decodeURIComponent(editBtn.dataset.end)
            );
            return;
        }
        const deleteBtn = e.target.closest('[data-delete-id]');
        if (deleteBtn) {
            deleteFeeding(parseInt(deleteBtn.dataset.deleteId));
        }
    });

    // ── Sleep tab initialisation ──
    setSleepDefaultTimes();
    document.getElementById('sleepMonthFilter').value = ym;
    loadSleeps();
    loadSleepDailyTotals();
    document.getElementById('sleepForm').addEventListener('submit', handleSleepSubmit);

    // Snap sleep time inputs to nearest 5 minutes
    ['sleepStartTime', 'sleepEndTime'].forEach(id => {
        document.getElementById(id).addEventListener('change', (e) => {
            const val = e.target.value;
            if (val) {
                const [h, m] = val.split(':').map(Number);
                const rounded = Math.round(m / 5) * 5;
                const hrs = rounded === 60 ? (h + 1) % 24 : h;
                const mins = rounded === 60 ? 0 : rounded;
                e.target.value = `${String(hrs).padStart(2, '0')}:${String(mins).padStart(2, '0')}`;
            }
        });
    });

    // Event delegation for sleep edit/delete buttons
    document.getElementById('sleepsByDay').addEventListener('click', (e) => {
        const editBtn = e.target.closest('[data-sleep-edit-id]');
        if (editBtn) {
            editSleep(
                parseInt(editBtn.dataset.sleepEditId),
                decodeURIComponent(editBtn.dataset.start),
                decodeURIComponent(editBtn.dataset.end)
            );
            return;
        }
        const deleteBtn = e.target.closest('[data-sleep-delete-id]');
        if (deleteBtn) {
            deleteSleep(parseInt(deleteBtn.dataset.sleepDeleteId));
        }
    });

    // Reload sleep data when the Sleep tab is shown
    document.getElementById('sleep-tab').addEventListener('shown.bs.tab', () => {
        loadSleeps();
        loadSleepDailyTotals();
    });

    // ── Event tab initialisation ──
    document.getElementById('eventDateFilter').value = toLocalDate(now);
    document.getElementById('event-tab').addEventListener('shown.bs.tab', () => {
        loadEvents();
    });

    // ── Development tab initialisation ──
    document.getElementById('dev-tab').addEventListener('shown.bs.tab', () => {
        initDevelopmentTab();
    });
});

// ── Date/time utilities ──
// Convert a Date to "YYYY-MM-DDTHH:MM" for datetime-local inputs
function toLocalISO(date) {
    const off = date.getTimezoneOffset();
    const local = new Date(date.getTime() - off * 60000);
    return local.toISOString().slice(0, 16);
}

// Extract local date string "YYYY-MM-DD" from a Date
function toLocalDate(date) {
    const off = date.getTimezoneOffset();
    const local = new Date(date.getTime() - off * 60000);
    return local.toISOString().slice(0, 10);
}

// Extract local time string "HH:MM" from a Date
function toLocalTime(date) {
    const off = date.getTimezoneOffset();
    const local = new Date(date.getTime() - off * 60000);
    return local.toISOString().slice(11, 16);
}

// Convert datetime-local value to RFC 3339 (UTC) for the API
function toRFC3339(datetimeLocal) {
    return new Date(datetimeLocal).toISOString();
}

// Format ISO timestamp to short 24-hour time string (e.g. "08:30")
function formatTime(iso) {
    return new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', hour12: false });
}

function formatDate(iso) {
    return new Date(iso).toLocaleDateString();
}

// ── Month navigation ──
// Shift the month picker by +1 or -1 month and reload data
function changeMonth(delta) {
    const input = document.getElementById('monthFilter');
    const [y, m] = input.value.split('-').map(Number);
    const d = new Date(y, m - 1 + delta, 1);
    input.value = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
    loadFeedings();
    loadDailyTotals();
}

// ── Feedings list ──
// Fetch feedings from API (filtered by selected month) and render them
async function loadFeedings() {
    const monthFilter = document.getElementById('monthFilter').value;
    let url = API + `?tz=${encodeURIComponent(TZ)}`;
    if (monthFilter) url += `&month=${monthFilter}`;

    try {
        const res = await fetch(url);
        const feedings = await res.json();
        renderFeedings(feedings);
    } catch (err) {
        log.error('Failed to load feedings:', err);
    }
}

// Group feedings by day and render as collapsible day cards with a table per day
function renderFeedings(feedings) {
    const container = document.getElementById('feedingsByDay');

    if (!feedings || feedings.length === 0) {
        container.innerHTML = '<p class="text-center text-muted py-3">No feedings recorded</p>';
        return;
    }

    // Group by local date string
    const groups = {};
    feedings.forEach(f => {
        const day = new Date(f.start_time).toLocaleDateString(undefined, { weekday: 'short', year: 'numeric', month: 'short', day: 'numeric' });
        if (!groups[day]) groups[day] = [];
        groups[day].push(f);
    });

    // Days are in DESC order already; keep that ordering
    const days = Object.keys(groups);

    container.innerHTML = days.map(day => {
        const items = groups[day];
        const dayTotal = items.reduce((s, f) => s + f.amount_ml, 0);
        const rows = items.map(f => {
            const startISO = encodeURIComponent(f.start_time);
            const endISO = encodeURIComponent(f.end_time);
            return `
            <tr class="feeding-row">
                <td class="text-nowrap">${formatTime(f.start_time)} – ${formatTime(f.end_time)}</td>
                <td><span class="badge bg-primary rounded-pill">${f.amount_ml} ml</span></td>
                <td class="text-end text-nowrap">
                    <button class="btn btn-sm btn-outline-primary py-0 px-2" data-edit-id="${f.id}" data-amount="${f.amount_ml}" data-start="${startISO}" data-end="${endISO}">✏️</button>
                    <button class="btn btn-sm btn-outline-danger py-0 px-2" data-delete-id="${f.id}">🗑</button>
                </td>
            </tr>
        `}).join('');
        return `
            <div class="mb-3">
                <div class="d-flex justify-content-between align-items-center bg-light px-3 py-2 rounded-top border">
                    <strong>${day}</strong>
                    <span class="badge bg-primary rounded-pill">${dayTotal} ml</span>
                </div>
                <div class="table-responsive border border-top-0 rounded-bottom">
                    <table class="table table-sm table-striped mb-0">
                        <thead class="table-light">
                            <tr><th>Time</th><th>Amount</th><th class="text-end">Actions</th></tr>
                        </thead>
                        <tbody>${rows}</tbody>
                    </table>
                </div>
            </div>`;
    }).join('');
}

// ── Form submit (add or update) ──
async function handleSubmit(e) {
    e.preventDefault();
    const id = document.getElementById('editId').value;
    const date = document.getElementById('feedingDate').value;
    const data = {
        amount_ml: parseInt(document.getElementById('amountMl').value),
        start_time: toRFC3339(date + 'T' + document.getElementById('startTime').value),
        end_time: toRFC3339(date + 'T' + document.getElementById('endTime').value),
    };

    try {
        const url = id ? `${API}/${id}` : API;
        const method = id ? 'PUT' : 'POST';
        const res = await fetch(url, {
            method,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data),
        });
        if (!res.ok) {
            const err = await res.json();
            alert(err.error || 'Failed to save');
            return;
        }
        document.getElementById('feedingForm').reset();
        setDefaultTimes();
        cancelEdit();
        showToast(id ? 'Feeding updated' : 'Feeding added');
        loadFeedings();
        loadDailyTotals();
        loadQuickAmounts();
    } catch (err) {
        log.error('Failed to save feeding:', err);
    }
}

// ── Edit mode ──
// Populate form with existing feeding data for editing
function editFeeding(id, amount, startTime, endTime) {
    document.getElementById('editId').value = id;
    document.getElementById('amountMl').value = amount;
    document.getElementById('amountSlider').value = amount;
    document.getElementById('feedingDate').value = toLocalDate(new Date(startTime));
    document.getElementById('startTime').value = toLocalTime(new Date(startTime));
    document.getElementById('endTime').value = toLocalTime(new Date(endTime));
    document.getElementById('formTitle').textContent = 'Edit Feeding';
    document.getElementById('submitBtn').textContent = 'Update';
    document.getElementById('cancelBtn').classList.remove('d-none');

    // Scroll to form and focus amount field
    document.getElementById('feedingForm').scrollIntoView({ behavior: 'smooth', block: 'center' });
    document.getElementById('amountMl').focus();
}

// Reset form back to "Add" mode
function cancelEdit() {
    document.getElementById('editId').value = '';
    document.getElementById('formTitle').textContent = 'Add Feeding';
    document.getElementById('submitBtn').textContent = 'Add';
    document.getElementById('cancelBtn').classList.add('d-none');
}

// ── Delete ──
async function deleteFeeding(id) {
    if (!confirm('Delete this feeding?')) return;
    try {
        await fetch(`${API}/${id}`, { method: 'DELETE' });
        showToast('Feeding deleted');
        loadFeedings();
        loadDailyTotals();
        loadQuickAmounts();
    } catch (err) {
        log.error('Failed to delete feeding:', err);
    }
}

// ── Quick amount buttons ──
// Fetch only the last feeding to derive quick-pick amounts (±10, ±20 ml)
async function loadQuickAmounts() {
    try {
        const res = await fetch(API + '/last');
        const feeding = await res.json();
        updateQuickAmounts(feeding);
    } catch (err) {
        log.error('Failed to load quick amounts:', err);
    }
}

// Render 5 quick-pick buttons based on the last feeding's amount
function updateQuickAmounts(feeding) {
    const container = document.getElementById('quickAmounts');
    if (!feeding) {
        container.innerHTML = '';
        return;
    }
    const lastAmount = feeding.amount_ml;
    const steps = [-20, -10, 0, 10, 20];
    container.innerHTML = steps.map(s => {
        const val = Math.max(1, lastAmount + s);
        const active = 'btn-outline-secondary';
        return `<button type="button" class="btn ${active} py-1 px-0" style="flex:1;min-width:0;font-size:0.85rem" onclick="document.getElementById('amountMl').value=${val};document.getElementById('amountSlider').value=${val}">${val}</button>`;
    }).join('');
}

// ── Daily totals chart ──
// Fetch aggregated daily totals for the selected month and render bar chart
async function loadDailyTotals() {
    try {
        log.debug('Loading daily totals...');
        const month = document.getElementById('monthFilter').value;
        const tzParam = `tz=${encodeURIComponent(TZ)}`;
        const url = month ? `${API}/daily?${tzParam}&month=${month}` : `${API}/daily?${tzParam}&days=31`;
        const res = await fetch(url);
        const totals = await res.json();
        renderChart(totals);
    } catch (err) {
        log.error('Failed to load daily totals:', err);
    }
}

// Render bar chart with day-of-month on x-axis and ml totals on y-axis
function renderChart(totals) {
    const ctx = document.getElementById('dailyChart').getContext('2d');
    // Extract day-of-month from "YYYY-MM-DD" date string for x-axis labels
    const labels = totals.map(t => {
        const parts = t.date.split('-');   // ["2026", "03", "07"]
        return parseInt(parts[2], 10);     // 7
    });
    const data = totals.map(t => t.total_ml);

    if (dailyChart) dailyChart.destroy();

    const dataLabelPlugin = {
        id: 'barDataLabels',
        afterDatasetsDraw(chart) {
            const { ctx, data } = chart;
            chart.getDatasetMeta(0).data.forEach((bar, i) => {
                const value = data.datasets[0].data[i];
                if (!value) return;
                ctx.save();
                ctx.fillStyle = '#495057';
                ctx.font = 'bold 12px sans-serif';
                ctx.textAlign = 'center';
                ctx.textBaseline = 'bottom';
                ctx.fillText(value + ' ml', bar.x, bar.y - 4);
                ctx.restore();
            });
        }
    };

    dailyChart = new Chart(ctx, {
        type: 'bar',
        data: {
            labels,
            datasets: [{
                label: 'Total ml',
                data,
                backgroundColor: 'rgba(54, 162, 235, 0.6)',
                borderColor: 'rgba(54, 162, 235, 1)',
                borderWidth: 1,
            }],
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            layout: { padding: { top: 24 } },
            scales: {
                y: {
                    beginAtZero: true,
                    title: { display: true, text: 'Milliliters (ml)' },
                },
                x: {
                    title: { display: true, text: 'Day' },
                },
            },
        },
        plugins: [dataLabelPlugin],
    });
}

// ═══════════════════════════════════════════════════════════════════════════
// ── SLEEP TRACKING ──
// ═══════════════════════════════════════════════════════════════════════════

// ── Sleep timer (Start / Stop) ──
function toggleSleepTimer() {
    const btn = document.getElementById('sleepStartStopBtn');
    const timerEl = document.getElementById('sleepTimerDisplay');

    if (!sleepStartTime) {
        // START
        sleepStartTime = new Date();
        document.getElementById('sleepDate').value = toLocalDate(sleepStartTime);
        document.getElementById('sleepStartTime').value = toLocalTime(sleepStartTime);
        document.getElementById('sleepEndTime').value = toLocalTime(sleepStartTime);

        btn.textContent = '⏹ Stop Sleep';
        btn.classList.replace('btn-sleep', 'btn-danger');
        timerEl.classList.remove('d-none');

        const startLabel = toLocalTime(sleepStartTime);
        sleepTimerInterval = setInterval(() => {
            const elapsed = Math.floor((Date.now() - sleepStartTime) / 1000);
            const h = String(Math.floor(elapsed / 3600)).padStart(2, '0');
            const m = String(Math.floor((elapsed % 3600) / 60)).padStart(2, '0');
            const s = String(elapsed % 60).padStart(2, '0');
            timerEl.textContent = `😴 ${startLabel} — ⏱ ${h}:${m}:${s}`;
        }, 1000);
    } else {
        // STOP
        const now = new Date();
        document.getElementById('sleepEndTime').value = toLocalTime(now);

        clearInterval(sleepTimerInterval);
        sleepTimerInterval = null;
        sleepStartTime = null;

        btn.textContent = '😴 Start Sleep';
        btn.classList.replace('btn-danger', 'btn-sleep');
        timerEl.classList.add('d-none');
        timerEl.textContent = '';

        // Auto-submit the sleep entry
        document.getElementById('sleepForm').requestSubmit();
    }
}

// ── Sleep form helpers ──
function setSleepDefaultTimes() {
    const now = new Date();
    document.getElementById('sleepDate').value = toLocalDate(now);
    document.getElementById('sleepStartTime').value = toLocalTime(now);
    document.getElementById('sleepEndTime').value = toLocalTime(now);
}

function setSleepEndTimeFromStart() {
    document.getElementById('sleepEndTime').value = document.getElementById('sleepStartTime').value;
}

// ── Format duration between two ISO timestamps as "Xh Ym" ──
function formatDuration(startISO, endISO) {
    const diffMs = new Date(endISO) - new Date(startISO);
    const totalMin = Math.round(diffMs / 60000);
    const h = Math.floor(totalMin / 60);
    const m = totalMin % 60;
    if (h > 0) return `${h}h ${m}m`;
    return `${m}m`;
}

// ── Sleep month navigation ──
function changeSleepMonth(delta) {
    const input = document.getElementById('sleepMonthFilter');
    const [y, m] = input.value.split('-').map(Number);
    const d = new Date(y, m - 1 + delta, 1);
    input.value = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
    loadSleeps();
    loadSleepDailyTotals();
}

// ── Load sleeps from API ──
async function loadSleeps() {
    const monthFilter = document.getElementById('sleepMonthFilter').value;
    let url = SLEEP_API + `?tz=${encodeURIComponent(TZ)}`;
    if (monthFilter) url += `&month=${monthFilter}`;

    try {
        const res = await fetch(url);
        const sleeps = await res.json();
        renderSleeps(sleeps);
    } catch (err) {
        log.error('Failed to load sleeps:', err);
    }
}

// ── Render sleeps grouped by day ──
function renderSleeps(sleeps) {
    const container = document.getElementById('sleepsByDay');

    if (!sleeps || sleeps.length === 0) {
        container.innerHTML = '<p class="text-center text-muted py-3">No sleep sessions recorded</p>';
        return;
    }

    const groups = {};
    sleeps.forEach(s => {
        const day = new Date(s.start_time).toLocaleDateString(undefined, { weekday: 'short', year: 'numeric', month: 'short', day: 'numeric' });
        if (!groups[day]) groups[day] = [];
        groups[day].push(s);
    });

    const days = Object.keys(groups);

    container.innerHTML = days.map(day => {
        const items = groups[day];
        // Sum total minutes for the day
        const dayTotalMin = items.reduce((sum, s) => {
            return sum + Math.round((new Date(s.end_time) - new Date(s.start_time)) / 60000);
        }, 0);
        const dayTotalStr = dayTotalMin >= 60
            ? `${Math.floor(dayTotalMin / 60)}h ${dayTotalMin % 60}m`
            : `${dayTotalMin}m`;

        const rows = items.map(s => {
            const startISO = encodeURIComponent(s.start_time);
            const endISO = encodeURIComponent(s.end_time);
            return `
            <tr class="feeding-row">
                <td class="text-nowrap">${formatTime(s.start_time)} – ${formatTime(s.end_time)}</td>
                <td><span class="badge bg-sleep rounded-pill">${formatDuration(s.start_time, s.end_time)}</span></td>
                <td class="text-end text-nowrap">
                    <button class="btn btn-sm btn-outline-primary py-0 px-2" data-sleep-edit-id="${s.id}" data-start="${startISO}" data-end="${endISO}">✏️</button>
                    <button class="btn btn-sm btn-outline-danger py-0 px-2" data-sleep-delete-id="${s.id}">🗑</button>
                </td>
            </tr>`;
        }).join('');

        return `
            <div class="mb-3">
                <div class="d-flex justify-content-between align-items-center bg-light px-3 py-2 rounded-top border">
                    <strong>${day}</strong>
                    <span class="badge bg-sleep rounded-pill">${dayTotalStr}</span>
                </div>
                <div class="table-responsive border border-top-0 rounded-bottom">
                    <table class="table table-sm table-striped mb-0">
                        <thead class="table-light">
                            <tr><th>Time</th><th>Duration</th><th class="text-end">Actions</th></tr>
                        </thead>
                        <tbody>${rows}</tbody>
                    </table>
                </div>
            </div>`;
    }).join('');
}

// ── Sleep form submit (add or update) ──
async function handleSleepSubmit(e) {
    e.preventDefault();
    const id = document.getElementById('sleepEditId').value;
    const date = document.getElementById('sleepDate').value;
    const data = {
        start_time: toRFC3339(date + 'T' + document.getElementById('sleepStartTime').value),
        end_time: toRFC3339(date + 'T' + document.getElementById('sleepEndTime').value),
    };

    try {
        const url = id ? `${SLEEP_API}/${id}` : SLEEP_API;
        const method = id ? 'PUT' : 'POST';
        const res = await fetch(url, {
            method,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data),
        });
        if (!res.ok) {
            const err = await res.json();
            alert(err.error || 'Failed to save');
            return;
        }
        document.getElementById('sleepForm').reset();
        setSleepDefaultTimes();
        cancelSleepEdit();
        showToast(id ? 'Sleep updated' : 'Sleep added');
        loadSleeps();
        loadSleepDailyTotals();
    } catch (err) {
        log.error('Failed to save sleep:', err);
    }
}

// ── Sleep edit mode ──
function editSleep(id, startTime, endTime) {
    document.getElementById('sleepEditId').value = id;
    document.getElementById('sleepDate').value = toLocalDate(new Date(startTime));
    document.getElementById('sleepStartTime').value = toLocalTime(new Date(startTime));
    document.getElementById('sleepEndTime').value = toLocalTime(new Date(endTime));
    document.getElementById('sleepFormTitle').textContent = 'Edit Sleep';
    document.getElementById('sleepSubmitBtn').textContent = 'Update';
    document.getElementById('sleepCancelBtn').classList.remove('d-none');

    document.getElementById('sleepForm').scrollIntoView({ behavior: 'smooth', block: 'center' });
}

function cancelSleepEdit() {
    document.getElementById('sleepEditId').value = '';
    document.getElementById('sleepFormTitle').textContent = 'Add Sleep';
    document.getElementById('sleepSubmitBtn').textContent = 'Add';
    document.getElementById('sleepCancelBtn').classList.add('d-none');
}

// ── Delete sleep ──
async function deleteSleep(id) {
    if (!confirm('Delete this sleep session?')) return;
    try {
        await fetch(`${SLEEP_API}/${id}`, { method: 'DELETE' });
        showToast('Sleep deleted');
        loadSleeps();
        loadSleepDailyTotals();
    } catch (err) {
        log.error('Failed to delete sleep:', err);
    }
}

// ── Sleep daily totals chart ──
async function loadSleepDailyTotals() {
    try {
        log.debug('Loading sleep daily totals...');
        const month = document.getElementById('sleepMonthFilter').value;
        const tzParam = `tz=${encodeURIComponent(TZ)}`;
        const url = month ? `${SLEEP_API}/daily?${tzParam}&month=${month}` : `${SLEEP_API}/daily?${tzParam}&days=31`;
        const res = await fetch(url);
        const totals = await res.json();
        renderSleepChart(totals);
    } catch (err) {
        log.error('Failed to load sleep daily totals:', err);
    }
}

function renderSleepChart(totals) {
    const ctx = document.getElementById('sleepChart').getContext('2d');
    const labels = totals.map(t => {
        const parts = t.date.split('-');
        return parseInt(parts[2], 10);
    });
    const data = totals.map(t => t.total_minutes);

    if (sleepChart) sleepChart.destroy();

    const dataLabelPlugin = {
        id: 'sleepBarDataLabels',
        afterDatasetsDraw(chart) {
            const { ctx, data } = chart;
            chart.getDatasetMeta(0).data.forEach((bar, i) => {
                const value = data.datasets[0].data[i];
                if (!value) return;
                ctx.save();
                ctx.fillStyle = '#495057';
                ctx.font = 'bold 12px sans-serif';
                ctx.textAlign = 'center';
                ctx.textBaseline = 'bottom';
                ctx.fillText(value + 'm', bar.x, bar.y - 4);
                ctx.restore();
            });
        }
    };

    sleepChart = new Chart(ctx, {
        type: 'bar',
        data: {
            labels,
            datasets: [{
                label: 'Total minutes',
                data,
                backgroundColor: 'rgba(199, 85, 10, 0.6)',
                borderColor: 'rgba(199, 85, 10, 1)',
                borderWidth: 1,
            }],
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            layout: { padding: { top: 24 } },
            scales: {
                y: {
                    beginAtZero: true,
                    title: { display: true, text: 'Minutes' },
                },
                x: {
                    title: { display: true, text: 'Day' },
                },
            },
        },
        plugins: [dataLabelPlugin],
    });
}

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
                startTime: f.start_time,
                endTime: f.end_time,
                detail: `${f.amount_ml} ml`,
                badgeClass: 'bg-primary',
            });
        });
    }
    if (sleeps) {
        sleeps.forEach(s => {
            events.push({
                type: 'sleep',
                startTime: s.start_time,
                endTime: s.end_time,
                detail: formatDuration(s.start_time, s.end_time),
                badgeClass: 'bg-sleep',
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

    const rows = events.map(ev => {
        const icon = ev.type === 'feeding' ? '🍼' : '😴';
        const label = ev.type === 'feeding' ? 'Feeding' : 'Sleep';
        return `
        <tr class="feeding-row">
            <td class="text-nowrap">${formatTime(ev.startTime)} – ${formatTime(ev.endTime)}</td>
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

// ═══════════════════════════════════════════════
// ██  DEVELOPMENT TAB                         ██
// ═══════════════════════════════════════════════

const BABY_API = '/api/baby';
const DEV_API = '/api/development';

let babyProfile = null;       // { id, date_of_birth, ... }
let devWeeksData = [];        // array of parsed week content objects
let devCurrentWeek = 0;       // current baby week number
let devViewingIndex = 0;      // index into devWeeksData currently displayed
let devTabInitialized = false; // only load once per session

// ── Tab initialization ──
async function initDevelopmentTab() {
    if (devTabInitialized && babyProfile) {
        return; // already loaded
    }
    await loadBabyProfile();
}

// ── Baby profile ──
async function loadBabyProfile() {
    try {
        const res = await fetch(BABY_API);
        const data = await res.json();
        if (data && data.id) {
            babyProfile = data;
            showBabyInfo();
            await loadDevelopmentContent();
        } else {
            showDobSetup();
        }
    } catch (err) {
        log.error('Failed to load baby profile:', err);
        showDobSetup();
    }
}

async function saveBabyProfile() {
    const dobInput = document.getElementById('babyDob');
    const dob = dobInput.value;
    if (!dob) {
        showToast('Please enter a date of birth', 'error');
        return;
    }
    try {
        const res = await fetch(BABY_API, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ date_of_birth: dob }),
        });
        if (!res.ok) {
            const err = await res.json();
            showToast(err.error || 'Failed to save', 'error');
            return;
        }
        babyProfile = await res.json();
        showToast('Baby profile saved!');
        showBabyInfo();
        await loadDevelopmentContent();
    } catch (err) {
        log.error('Failed to save baby profile:', err);
        showToast('Failed to save profile', 'error');
    }
}

function editBabyDob() {
    document.getElementById('babyDob').value = babyProfile.date_of_birth.slice(0, 10);
    showDobSetup();
}

function showDobSetup() {
    document.getElementById('dobSetup').classList.remove('d-none');
    document.getElementById('babyInfoBar').classList.add('d-none');
    document.getElementById('devContent').classList.add('d-none');
    document.getElementById('devLoading').classList.add('d-none');
    document.getElementById('devError').classList.add('d-none');
}

function showBabyInfo() {
    document.getElementById('dobSetup').classList.add('d-none');
    document.getElementById('babyInfoBar').classList.remove('d-none');

    const dob = new Date(babyProfile.date_of_birth);
    const age = calculateAge(dob);
    document.getElementById('babyAgeDisplay').textContent = age.description;
    document.getElementById('babyDobDisplay').textContent = `Born: ${formatDate(babyProfile.date_of_birth)}`;
}

function calculateAge(dob) {
    const now = new Date();
    const diffMs = now - dob;
    const totalDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
    const weeks = Math.floor(totalDays / 7);
    const months = Math.floor(totalDays / 30.44);
    const remainingWeeks = Math.floor((totalDays - Math.floor(months * 30.44)) / 7);

    let description;
    if (months < 1) {
        description = `${weeks} week${weeks !== 1 ? 's' : ''} old`;
    } else {
        description = `${months} month${months !== 1 ? 's' : ''} and ${remainingWeeks} week${remainingWeeks !== 1 ? 's' : ''} old`;
    }
    return { weeks, months, totalDays, description };
}

// ── Development content loading ──
async function loadDevelopmentContent() {
    const loading = document.getElementById('devLoading');
    const content = document.getElementById('devContent');
    const error = document.getElementById('devError');

    loading.classList.remove('d-none');
    content.classList.add('d-none');
    error.classList.add('d-none');

    try {
        const res = await fetch(DEV_API);
        if (!res.ok) {
            const err = await res.json();
            throw new Error(err.error || `HTTP ${res.status}`);
        }
        const data = await res.json();
        devCurrentWeek = data.current_week;
        devWeeksData = data.weeks || [];
        devViewingIndex = 0;
        devTabInitialized = true;

        loading.classList.add('d-none');

        if (devWeeksData.length > 0) {
            renderDevelopmentWeek(devWeeksData[0]);
            updateWeekBadge();
            content.classList.remove('d-none');
        } else {
            showDevError('No development content available.');
        }
    } catch (err) {
        log.error('Failed to load development content:', err);
        loading.classList.add('d-none');
        showDevError(err.message);
    }
}

function showDevError(message) {
    const error = document.getElementById('devError');
    document.getElementById('devErrorMsg').textContent = message;
    error.classList.remove('d-none');
    document.getElementById('devContent').classList.add('d-none');
}

// ── Week navigation ──
function switchWeek(delta) {
    const newIndex = devViewingIndex + delta;
    if (newIndex < 0 || newIndex >= devWeeksData.length) {
        // Could load more, but for now just clamp
        if (newIndex < 0) showToast('No earlier weeks loaded', 'error');
        else showToast('No later weeks loaded', 'error');
        return;
    }
    devViewingIndex = newIndex;
    renderDevelopmentWeek(devWeeksData[devViewingIndex]);
    updateWeekBadge();
}

function updateWeekBadge() {
    const weekNum = devCurrentWeek + devViewingIndex;
    document.getElementById('weekBadge').textContent = `Week ${weekNum}`;
}

// ── Render development content ──
function renderDevelopmentWeek(weekData) {
    if (!weekData) return;

    // Handle error weeks
    if (weekData.error) {
        document.getElementById('behaviorsContent').innerHTML = `<p class="text-muted">${weekData.error}</p>`;
        document.getElementById('milestonesContent').innerHTML = '';
        document.getElementById('wonderWeekContent').innerHTML = '';
        document.getElementById('exercisesContent').innerHTML = '';
        return;
    }

    renderBehaviors(weekData.behaviors || []);
    renderMilestones(weekData.milestones || []);
    renderWonderWeek(weekData.wonder_week, weekData.upcoming_wonder_weeks || []);
    renderExercises(weekData.exercises || []);
}

function renderBehaviors(behaviors) {
    const el = document.getElementById('behaviorsContent');
    if (behaviors.length === 0) {
        el.innerHTML = '<p class="text-muted">No behavior data available.</p>';
        return;
    }
    el.innerHTML = behaviors.map(b => `
        <div class="d-flex align-items-start mb-3">
            <span class="fs-4 me-3">👁️</span>
            <div>
                <strong>${escapeHtml(b.title)}</strong>
                <p class="mb-0 text-muted">${escapeHtml(b.description)}</p>
            </div>
        </div>
    `).join('');
}

function renderMilestones(milestones) {
    const el = document.getElementById('milestonesContent');
    if (milestones.length === 0) {
        el.innerHTML = '<p class="text-muted">No milestone data available.</p>';
        return;
    }
    el.innerHTML = milestones.map(m => `
        <div class="d-flex align-items-start mb-3">
            <span class="fs-4 me-3">⭐</span>
            <div class="flex-grow-1">
                <div class="d-flex justify-content-between align-items-start">
                    <strong>${escapeHtml(m.title)}</strong>
                    ${m.typical_range ? `<span class="badge bg-secondary ms-2">${escapeHtml(m.typical_range)}</span>` : ''}
                </div>
                <p class="mb-0 text-muted">${escapeHtml(m.description)}</p>
            </div>
        </div>
    `).join('');
}

function renderWonderWeek(wonderWeek, upcoming) {
    const el = document.getElementById('wonderWeekContent');
    let html = '';

    if (wonderWeek) {
        if (wonderWeek.is_active) {
            html += `
                <div class="alert wonder-week-active mb-3">
                    <h6 class="alert-heading">🌟 Active Wonder Week — Leap ${wonderWeek.leap_number || '?'}</h6>
                    <strong>${escapeHtml(wonderWeek.name || '')}</strong>
                    <p class="mt-2 mb-2">${escapeHtml(wonderWeek.description || '')}</p>
                    ${wonderWeek.signs && wonderWeek.signs.length > 0 ? `
                        <h6 class="mt-3">Signs to watch for:</h6>
                        <ul class="mb-2">${wonderWeek.signs.map(s => `<li>${escapeHtml(s)}</li>`).join('')}</ul>
                    ` : ''}
                    ${wonderWeek.handling_tips && wonderWeek.handling_tips.length > 0 ? `
                        <h6>How to help your baby:</h6>
                        <ul class="mb-0">${wonderWeek.handling_tips.map(t => `<li>${escapeHtml(t)}</li>`).join('')}</ul>
                    ` : ''}
                </div>
            `;
        } else {
            html += `
                <div class="alert alert-light mb-3">
                    <span class="text-muted">No active wonder week right now. Your baby is in a calm phase! 😊</span>
                </div>
            `;
        }
    }

    if (upcoming && upcoming.length > 0) {
        html += `<h6 class="mt-3 mb-2">Upcoming Wonder Weeks</h6>`;
        html += `<div class="list-group">`;
        upcoming.forEach(u => {
            html += `
                <div class="list-group-item d-flex justify-content-between align-items-center">
                    <div>
                        <strong>Leap ${u.leap_number || '?'}: ${escapeHtml(u.name || '')}</strong>
                        <br><small class="text-muted">Around week ${u.week || '?'}</small>
                    </div>
                    <span class="badge bg-dev rounded-pill">${u.weeks_away || '?'} weeks away</span>
                </div>
            `;
        });
        html += `</div>`;
    }

    el.innerHTML = html || '<p class="text-muted">No wonder week data available.</p>';
}

function renderExercises(exercises) {
    const el = document.getElementById('exercisesContent');
    if (exercises.length === 0) {
        el.innerHTML = '<p class="text-muted">No exercise data available.</p>';
        return;
    }
    el.innerHTML = `<div class="row g-3">${exercises.map(ex => `
        <div class="col-12 col-md-6">
            <div class="card exercise-card h-100">
                <div class="card-body">
                    <div class="d-flex align-items-center mb-2">
                        <span class="fs-2 me-2">${ex.icon || '🤸'}</span>
                        <h6 class="mb-0">${escapeHtml(ex.name)}</h6>
                    </div>
                    <p class="mb-2">${escapeHtml(ex.instructions || '')}</p>
                    <div class="d-flex gap-2 flex-wrap">
                        ${ex.benefits ? `<span class="badge bg-success-subtle text-success-emphasis">💪 ${escapeHtml(ex.benefits)}</span>` : ''}
                        ${ex.duration ? `<span class="badge bg-info-subtle text-info-emphasis">⏱️ ${escapeHtml(ex.duration)}</span>` : ''}
                    </div>
                </div>
            </div>
        </div>
    `).join('')}</div>`;
}

// ── HTML escaping utility ──
function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}
