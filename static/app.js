// ── Global state ──
const API = '/api/feedings';
const SLEEP_API = '/api/sleeps';
const DIAPER_API = '/api/diapers';
const BATH_API = '/api/baths';
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
    document.getElementById('endTime').value = toLocalTime(now);
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
    ['endTime'].forEach(id => {
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
    restoreSleepSession();

    // Event delegation for sleep edit/delete buttons
    document.getElementById('sleepsByDay').addEventListener('click', (e) => {
        const editBtn = e.target.closest('[data-sleep-edit-id]');
        if (editBtn) {
            editSleep(
                parseInt(editBtn.dataset.sleepEditId),
                decodeURIComponent(editBtn.dataset.start),
                decodeURIComponent(editBtn.dataset.end),
                editBtn.dataset.sleepType || 'nap'
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

    // ── Diaper tab initialisation ──
    setDiaperDefaultTimes();
    document.getElementById('diaperMonthFilter').value = ym;
    document.getElementById('diaperForm').addEventListener('submit', handleDiaperSubmit);
    document.getElementById('diaper-tab').addEventListener('shown.bs.tab', () => {
        loadDiapers();
    });
    // Snap diaper time input to nearest 5 minutes
    document.getElementById('diaperTime').addEventListener('change', (e) => {
        const val = e.target.value;
        if (val) {
            const [h, m] = val.split(':').map(Number);
            const rounded = Math.round(m / 5) * 5;
            const hrs = rounded === 60 ? (h + 1) % 24 : h;
            const mins = rounded === 60 ? 0 : rounded;
            e.target.value = `${String(hrs).padStart(2, '0')}:${String(mins).padStart(2, '0')}`;
        }
    });
    // Event delegation for diaper delete buttons
    document.getElementById('diapersByDay').addEventListener('click', (e) => {
        const editBtn = e.target.closest('[data-diaper-edit-id]');
        if (editBtn) {
            editDiaper(
                parseInt(editBtn.dataset.diaperEditId),
                editBtn.dataset.type,
                decodeURIComponent(editBtn.dataset.time)
            );
            return;
        }
        const deleteBtn = e.target.closest('[data-diaper-delete-id]');
        if (deleteBtn) {
            deleteDiaper(parseInt(deleteBtn.dataset.diaperDeleteId));
        }
    });

    // ── Bath tab initialisation ──
    setBathDefaultTimes();
    document.getElementById('bathMonthFilter').value = ym;
    document.getElementById('bathForm').addEventListener('submit', handleBathSubmit);
    document.getElementById('bath-tab').addEventListener('shown.bs.tab', () => {
        loadBaths();
    });
    // Snap bath time input to nearest 5 minutes
    document.getElementById('bathTime').addEventListener('change', (e) => {
        const val = e.target.value;
        if (val) {
            const [h, m] = val.split(':').map(Number);
            const rounded = Math.round(m / 5) * 5;
            const hrs = rounded === 60 ? (h + 1) % 24 : h;
            const mins = rounded === 60 ? 0 : rounded;
            e.target.value = `${String(hrs).padStart(2, '0')}:${String(mins).padStart(2, '0')}`;
        }
    });
    // Event delegation for bath delete buttons
    document.getElementById('bathsByDay').addEventListener('click', (e) => {
        const editBtn = e.target.closest('[data-bath-edit-id]');
        if (editBtn) {
            editBath(
                parseInt(editBtn.dataset.bathEditId),
                decodeURIComponent(editBtn.dataset.time)
            );
            return;
        }
        const deleteBtn = e.target.closest('[data-bath-delete-id]');
        if (deleteBtn) {
            deleteBath(parseInt(deleteBtn.dataset.bathDeleteId));
        }
    });

    // ── Development tab initialisation ──
    document.getElementById('dev-tab').addEventListener('shown.bs.tab', () => {
        initDevelopmentTab();
    });

    // ── Insights tab initialisation ──
    document.getElementById('insights-tab').addEventListener('shown.bs.tab', () => {
        initInsightsTab();
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
        const day = new Date(f.end_time).toLocaleDateString(undefined, { weekday: 'short', year: 'numeric', month: 'short', day: 'numeric' });
        if (!groups[day]) groups[day] = [];
        groups[day].push(f);
    });

    // Days are in DESC order already; keep that ordering
    const days = Object.keys(groups);

    container.innerHTML = days.map(day => {
        const items = groups[day];
        const dayTotal = items.reduce((s, f) => s + f.amount_ml, 0);
        const rows = items.map((f, idx) => {
            const endISO = encodeURIComponent(f.end_time);
            let gapHtml = '';
            if (idx > 0) {
                const prevEnd = new Date(items[idx - 1].end_time);
                const currEnd = new Date(f.end_time);
                gapHtml = buildTimeGapRow(prevEnd, currEnd, 3);
            }
            return `${gapHtml}
            <tr class="feeding-row">
                <td class="text-nowrap">${formatTime(f.end_time)}</td>
                <td><span class="badge bg-primary rounded-pill">${f.amount_ml} ml</span></td>
                <td class="text-end text-nowrap">
                    <button class="btn btn-sm btn-outline-primary py-0 px-2" data-edit-id="${Number(f.id)}" data-amount="${Number(f.amount_ml)}" data-end="${endISO}">✏️</button>
                    <button class="btn btn-sm btn-outline-danger py-0 px-2" data-delete-id="${Number(f.id)}">🗑</button>
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
function editFeeding(id, amount, endTime) {
    document.getElementById('editId').value = id;
    document.getElementById('amountMl').value = amount;
    document.getElementById('amountSlider').value = amount;
    document.getElementById('feedingDate').value = toLocalDate(new Date(endTime));
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
async function toggleSleepTimer() {
    // Debounce: prevent rapid double-clicks
    const btn = document.getElementById('sleepStartStopBtn');
    if (btn.disabled) return;
    btn.disabled = true;
    setTimeout(() => { btn.disabled = false; }, 1000);
    if (!sleepStartTime) {
        // START — call POST /api/sleeps/start
        const now = new Date();
        const sleepType = document.querySelector('input[name="sleepType"]:checked')?.value || 'nap';
        const data = { start_time: now.toISOString(), sleep_type: sleepType };
        try {
            const res = await fetch(`${SLEEP_API}/start`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data),
            });
            if (!res.ok) {
                const err = await res.json();
                alert(err.error || 'Failed to start sleep');
                return;
            }
            const sleep = await res.json();
            sleepStartTime = new Date(sleep.start_time);
            saveSleepToLocalStorage({ id: sleep.id, start_time: sleep.start_time, sleep_type: sleep.sleep_type });
            setSleepActiveUI(sleep);
            showToast('Sleep started');
        } catch (err) {
            log.error('Failed to start sleep:', err);
        }
    } else {
        // STOP — call POST /api/sleeps/{id}/stop
        const stored = JSON.parse(localStorage.getItem('activeSleep'));
        if (!stored || !stored.id) {
            log.error('No active sleep ID found');
            clearSleepActiveUI();
            return;
        }
        const now = new Date();
        try {
            const res = await fetch(`${SLEEP_API}/${stored.id}/stop`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ end_time: now.toISOString() }),
            });
            if (!res.ok) {
                const err = await res.json();
                alert(err.error || 'Failed to stop sleep');
                return;
            }
            clearSleepFromLocalStorage();
            clearSleepActiveUI();
            showToast('Sleep stopped');
            loadSleeps();
            loadSleepDailyTotals();
        } catch (err) {
            log.error('Failed to stop sleep:', err);
        }
    }
}

// ── Sleep form helpers ──
function setSleepDefaultTimes() {
    const now = new Date();
    document.getElementById('sleepStart').value = toLocalISO(now);
    document.getElementById('sleepEnd').value = toLocalISO(now);
}

// ── Active sleep localStorage helpers ──
function saveSleepToLocalStorage(sleepData) {
    localStorage.setItem('activeSleep', JSON.stringify(sleepData));
}

function clearSleepFromLocalStorage() {
    localStorage.removeItem('activeSleep');
}

// ── Restore active sleep on page load ──
async function restoreSleepSession() {
    try {
        // Check server first
        const res = await fetch(`${SLEEP_API}/active`);
        const serverSleep = await res.json();
        const stored = JSON.parse(localStorage.getItem('activeSleep'));

        if (serverSleep && serverSleep.id && serverSleep.status === 'active') {
            // Server has active sleep — use it as source of truth
            sleepStartTime = new Date(serverSleep.start_time);
            saveSleepToLocalStorage({ id: serverSleep.id, start_time: serverSleep.start_time, sleep_type: serverSleep.sleep_type });
            setSleepActiveUI(serverSleep);
        } else if (stored && stored.id) {
            // localStorage has data but server doesn't — stale, clear it
            clearSleepFromLocalStorage();
        }
    } catch (err) {
        log.debug('No active sleep session:', err);
    }
}

// ── Set UI to active-sleep state ──
function setSleepActiveUI(sleep) {
    const btn = document.getElementById('sleepStartStopBtn');
    const timerEl = document.getElementById('sleepTimerDisplay');
    const editRow = document.getElementById('sleepEditStartRow');
    const editInput = document.getElementById('sleepActiveStartEdit');

    sleepStartTime = new Date(sleep.start_time);

    btn.textContent = '⏹ Stop Sleep';
    btn.classList.replace('btn-sleep', 'btn-danger');
    timerEl.classList.remove('d-none');
    editRow.classList.remove('d-none');
    editInput.value = toLocalISO(sleepStartTime);

    // Set the sleep type radio to match
    const typeRadio = document.querySelector(`input[name="sleepType"][value="${sleep.sleep_type || 'nap'}"]`);
    if (typeRadio) typeRadio.checked = true;

    const startLabel = toLocalTime(sleepStartTime);
    clearInterval(sleepTimerInterval);
    sleepTimerInterval = setInterval(() => {
        const elapsed = Math.floor((Date.now() - sleepStartTime) / 1000);
        const h = String(Math.floor(elapsed / 3600)).padStart(2, '0');
        const m = String(Math.floor((elapsed % 3600) / 60)).padStart(2, '0');
        const s = String(elapsed % 60).padStart(2, '0');
        timerEl.textContent = `😴 ${startLabel} — ⏱ ${h}:${m}:${s}`;
    }, 1000);
}

// ── Clear active-sleep UI back to idle ──
function clearSleepActiveUI() {
    const btn = document.getElementById('sleepStartStopBtn');
    const timerEl = document.getElementById('sleepTimerDisplay');
    const editRow = document.getElementById('sleepEditStartRow');

    clearInterval(sleepTimerInterval);
    sleepTimerInterval = null;
    sleepStartTime = null;

    btn.textContent = '😴 Start Sleep';
    btn.classList.replace('btn-danger', 'btn-sleep');
    timerEl.classList.add('d-none');
    timerEl.textContent = '';
    editRow.classList.add('d-none');
}

// ── Save edited start time for active sleep ──
async function saveActiveStartTime() {
    const stored = JSON.parse(localStorage.getItem('activeSleep'));
    if (!stored || !stored.id) return;
    const newStart = document.getElementById('sleepActiveStartEdit').value;
    if (!newStart) return;

    try {
        const res = await fetch(`${SLEEP_API}/${stored.id}`, {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ start_time: toRFC3339(newStart) }),
        });
        if (!res.ok) {
            const err = await res.json();
            alert(err.error || 'Failed to update start time');
            return;
        }
        const updated = await res.json();
        sleepStartTime = new Date(updated.start_time);
        saveSleepToLocalStorage({ id: updated.id, start_time: updated.start_time, sleep_type: updated.sleep_type });
        setSleepActiveUI(updated);
        showToast('Start time updated');
    } catch (err) {
        log.error('Failed to edit active sleep start:', err);
    }
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
        // Sum total minutes for the day (only completed sleeps)
        const dayTotalMin = items.reduce((sum, s) => {
            if (!s.end_time) return sum;
            return sum + Math.round((new Date(s.end_time) - new Date(s.start_time)) / 60000);
        }, 0);
        const dayTotalStr = dayTotalMin >= 60
            ? `${Math.floor(dayTotalMin / 60)}h ${dayTotalMin % 60}m`
            : `${dayTotalMin}m`;

        const rows = items.map((s, idx) => {
            const startISO = encodeURIComponent(s.start_time);
            const endISO = s.end_time ? encodeURIComponent(s.end_time) : '';
            const sleepType = s.sleep_type || 'nap';
            const typeBadge = sleepType === 'night'
                ? '<span class="badge bg-sleep-night rounded-pill">🌙 Night</span>'
                : '<span class="badge bg-sleep-nap rounded-pill">💤 Nap</span>';
            const isActive = s.status === 'active' || !s.end_time;
            const timeLabel = isActive
                ? `${formatTime(s.start_time)} – <em>ongoing</em>`
                : `${formatTime(s.start_time)} – ${formatTime(s.end_time)}`;
            const durationLabel = isActive
                ? '<span class="badge bg-warning text-dark rounded-pill">Active</span>'
                : `<span class="badge bg-sleep rounded-pill">${formatDuration(s.start_time, s.end_time)}</span>`;
            let gapHtml = '';
            if (idx > 0) {
                const prevEnd = items[idx - 1].end_time ? new Date(items[idx - 1].end_time) : new Date(items[idx - 1].start_time);
                const currStart = new Date(s.start_time);
                gapHtml = buildTimeGapRow(prevEnd, currStart, 4);
            }
            return `${gapHtml}
            <tr class="feeding-row">
                <td class="text-nowrap">${timeLabel}</td>
                <td>${typeBadge}</td>
                <td>${durationLabel}</td>
                <td class="text-end text-nowrap">
                    ${!isActive ? `<button class="btn btn-sm btn-outline-primary py-0 px-2" data-sleep-edit-id="${Number(s.id)}" data-start="${startISO}" data-end="${endISO}" data-sleep-type="${sleepType}">✏️</button>` : ''}
                    <button class="btn btn-sm btn-outline-danger py-0 px-2" data-sleep-delete-id="${Number(s.id)}">🗑</button>
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
                            <tr><th>Time</th><th>Type</th><th>Duration</th><th class="text-end">Actions</th></tr>
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
    const sleepType = document.querySelector('input[name="sleepType"]:checked')?.value || 'nap';
    const data = {
        start_time: toRFC3339(document.getElementById('sleepStart').value),
        end_time: toRFC3339(document.getElementById('sleepEnd').value),
        sleep_type: sleepType,
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
function editSleep(id, startTime, endTime, sleepType) {
    document.getElementById('sleepEditId').value = id;
    document.getElementById('sleepStart').value = toLocalISO(new Date(startTime));
    document.getElementById('sleepEnd').value = toLocalISO(new Date(endTime));
    // Set sleep type radio
    const typeRadio = document.querySelector(`input[name="sleepType"][value="${sleepType || 'nap'}"]`);
    if (typeRadio) typeRadio.checked = true;
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
    // Reset sleep type to default
    const napRadio = document.getElementById('sleepNap');
    if (napRadio) napRadio.checked = true;
}

// ── Delete sleep ──
async function deleteSleep(id) {
    if (!confirm('Delete this sleep session?')) return;
    // Clear active session if we're deleting the active sleep
    const stored = localStorage.getItem('activeSleep');
    if (stored) {
        try {
            const parsed = JSON.parse(stored);
            if (parsed && parsed.id === id) {
                clearSleepFromLocalStorage();
                clearSleepActiveUI();
            }
        } catch (_) { /* ignore parse errors */ }
    }
    try {
        const res = await fetch(`${SLEEP_API}/${id}`, { method: 'DELETE' });
        if (!res.ok) {
            showToast('Failed to delete sleep', 'error');
            return;
        }
        showToast('Sleep deleted');
        loadSleeps();
        loadSleepDailyTotals();
    } catch (err) {
        log.error('Failed to delete sleep:', err);
        showToast('Failed to delete sleep', 'error');
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
        const [feedRes, sleepRes, diaperRes, bathRes] = await Promise.all([
            fetch(`${API}?date=${date}&${tzParam}`),
            fetch(`${SLEEP_API}?date=${date}&${tzParam}`),
            fetch(`${DIAPER_API}?date=${date}&${tzParam}`).catch(() => ({ json: () => [] })),
            fetch(`${BATH_API}?date=${date}&${tzParam}`).catch(() => ({ json: () => [] })),
        ]);
        const feedings = await feedRes.json();
        const sleeps = await sleepRes.json();
        const diapers = await diaperRes.json();
        const baths = await bathRes.json();
        renderEvents(feedings, sleeps, diapers, baths, date);
    } catch (err) {
        log.error('Failed to load events:', err);
    }
}

function renderEvents(feedings, sleeps, diapers, baths, date) {
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
    if (diapers) {
        diapers.forEach(d => {
            const typeIcon = d.type === 'pee' ? '💧' : d.type === 'poo' ? '💩' : '💧💩';
            const typeLabel = d.type === 'both' ? 'Pee + Poo' : d.type.charAt(0).toUpperCase() + d.type.slice(1);
            events.push({
                type: 'diaper',
                startTime: d.time,
                endTime: d.time,
                detail: `${typeIcon} ${typeLabel}`,
                badgeClass: 'bg-diaper',
            });
        });
    }
    if (baths) {
        baths.forEach(b => {
            events.push({
                type: 'bath',
                startTime: b.time,
                endTime: b.time,
                detail: 'Bath',
                badgeClass: 'bg-bath',
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
    const diaperCount = events.filter(e => e.type === 'diaper').length;
    const bathCount = events.filter(e => e.type === 'bath').length;

    const rows = events.map((ev, idx) => {
        const icons = { feeding: '🍼', sleep: '😴', diaper: '🧷', bath: '🛁' };
        const defaultLabels = { feeding: 'Feeding', sleep: 'Sleep', diaper: 'Diaper', bath: 'Bath' };
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
                        ${diaperCount > 0 ? `<span class="badge bg-diaper rounded-pill">${diaperCount} diaper${diaperCount !== 1 ? 's' : ''}</span>` : ''}
                        ${bathCount > 0 ? `<span class="badge bg-bath rounded-pill">${bathCount} bath${bathCount !== 1 ? 's' : ''}</span>` : ''}
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
    const nameVal = (document.getElementById('babyName')?.value || '').trim();
    const genderEl = document.querySelector('input[name="babyGender"]:checked');
    const genderVal = genderEl ? genderEl.value : '';
    const milkTypeVal = document.getElementById('babyMilkType')?.value || '';

    try {
        const body = { date_of_birth: dob };
        if (nameVal) body.name = nameVal;
        if (genderVal) body.gender = genderVal;
        if (milkTypeVal) body.milk_type = milkTypeVal;

        const res = await fetch(BABY_API, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body),
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
    if (babyProfile.name) document.getElementById('babyName').value = babyProfile.name;
    if (babyProfile.gender) {
        const rb = document.querySelector(`input[name="babyGender"][value="${babyProfile.gender}"]`);
        if (rb) rb.checked = true;
    }
    if (babyProfile.milk_type) document.getElementById('babyMilkType').value = babyProfile.milk_type;
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
    const namePrefix = babyProfile.name ? `${babyProfile.name} — ` : '';
    document.getElementById('babyAgeDisplay').textContent = namePrefix + age.description;
    document.getElementById('babyDobDisplay').textContent = `Born: ${formatDate(babyProfile.date_of_birth)}`;

    // Show wonder week banner placeholder (will be updated after content loads)
    document.getElementById('wonderWeekBanner').classList.remove('d-none');
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
        document.getElementById('behaviorsContent').innerHTML = `<p class="text-muted">${escapeHtml(weekData.error)}</p>`;
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
    const banner = document.getElementById('wonderWeekBanner');
    const bannerContent = document.getElementById('wonderWeekBannerContent');
    let html = '';

    if (wonderWeek) {
        if (wonderWeek.is_active) {
            // Update top banner — angry face
            if (banner && bannerContent) {
                bannerContent.className = 'd-flex align-items-center gap-2 rounded p-3 wonder-banner-angry';
                bannerContent.innerHTML = `<span class="fs-2">😠</span><div><strong>Baby is on wonder week</strong><br><small class="text-muted">Leap ${wonderWeek.leap_number || '?'}: ${escapeHtml(wonderWeek.name || '')}</small></div>`;
                banner.classList.remove('d-none');
            }
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
            // Update top banner — happy face
            if (banner && bannerContent) {
                bannerContent.className = 'd-flex align-items-center gap-2 rounded p-3 wonder-banner-happy';
                bannerContent.innerHTML = `<span class="fs-2">😊</span><div><strong>Enjoy your easy week!</strong><br><small class="text-muted">No active wonder week right now</small></div>`;
                banner.classList.remove('d-none');
            }
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
                        <span class="fs-2 me-2">${escapeHtml(ex.icon || '🤸')}</span>
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

// ═══════════════════════════════════════════════
// ██  INSIGHTS TAB                            ██
// ═══════════════════════════════════════════════

const GROWTH_API = '/api/growth';
const INSIGHTS_API = '/api/insights';
const WHO_API = '/api/who-data';

let whoChart = null;           // Chart.js instance for WHO growth chart
let insightsInitialized = false;
let currentWHOMetric = 'weight';

// ── Tab initialization ──
async function initInsightsTab() {
    // Always check profile first
    if (!babyProfile) {
        try {
            const res = await fetch(BABY_API);
            const data = await res.json();
            if (data && data.id) {
                babyProfile = data;
            }
        } catch (e) { /* ignore */ }
    }

    if (!babyProfile || !babyProfile.date_of_birth) {
        document.getElementById('insightNoProfile').classList.remove('d-none');
        document.getElementById('insightGrowthCard').classList.add('d-none');
        document.getElementById('insightChartCard').classList.add('d-none');
        document.getElementById('insightContent').classList.add('d-none');
        return;
    }

    document.getElementById('insightNoProfile').classList.add('d-none');
    document.getElementById('insightGrowthCard').classList.remove('d-none');
    document.getElementById('insightChartCard').classList.remove('d-none');

    // Set default growth date to today
    document.getElementById('growthDate').value = toLocalDate(new Date());

    await loadGrowthHistory();
    await loadWHOChart(currentWHOMetric);
    if (!insightsInitialized) {
        await loadInsights();
        insightsInitialized = true;
    }
}

// ── Growth form toggle ──
function toggleGrowthForm() {
    const form = document.getElementById('growthForm');
    form.classList.toggle('d-none');
}

// ── Add Growth Measurement ──
async function addGrowthMeasurement(event) {
    event.preventDefault();
    const date = document.getElementById('growthDate').value;
    const weight = parseFloat(document.getElementById('growthWeight').value);
    const length = parseFloat(document.getElementById('growthLength').value);
    const headEl = document.getElementById('growthHead');
    const head = headEl.value ? parseFloat(headEl.value) : null;

    const body = { date, weight_kg: weight, length_cm: length };
    if (head !== null) body.head_circumference_cm = head;

    try {
        const res = await fetch(GROWTH_API, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body),
        });
        if (!res.ok) {
            const err = await res.json();
            showToast(err.error || 'Failed to save measurement', 'error');
            return;
        }
        showToast('Growth measurement saved!');
        toggleGrowthForm();
        // Reset form
        document.getElementById('growthWeight').value = '';
        document.getElementById('growthLength').value = '';
        document.getElementById('growthHead').value = '';
        // Reload data
        await loadGrowthHistory();
        await loadWHOChart(currentWHOMetric);
        // Invalidate insights so they reload
        insightsInitialized = false;
        await loadInsights();
        insightsInitialized = true;
    } catch (err) {
        log.error('Failed to save growth measurement:', err);
        showToast('Failed to save measurement', 'error');
    }
}

// ── Delete growth measurement ──
async function deleteGrowthMeasurement(id) {
    if (!confirm('Delete this measurement?')) return;
    try {
        const res = await fetch(`${GROWTH_API}/${id}`, { method: 'DELETE' });
        if (!res.ok) throw new Error('Delete failed');
        showToast('Measurement deleted');
        await loadGrowthHistory();
        await loadWHOChart(currentWHOMetric);
    } catch (err) {
        log.error('Failed to delete growth measurement:', err);
        showToast('Failed to delete', 'error');
    }
}

// ── Load growth history ──
async function loadGrowthHistory() {
    try {
        const res = await fetch(GROWTH_API);
        if (!res.ok) throw new Error('Failed to load');
        const measurements = await res.json();

        const infoEl = document.getElementById('latestGrowthInfo');
        const historyEl = document.getElementById('growthHistory');

        if (!measurements || measurements.length === 0) {
            infoEl.innerHTML = '<p class="text-muted">No growth measurements recorded yet. Add your first measurement above.</p>';
            historyEl.innerHTML = '';
            return;
        }

        // Latest measurement summary
        const latest = measurements[0]; // sorted desc by date
        const dob = new Date(babyProfile.date_of_birth);
        const measDate = new Date(latest.date);
        const ageMonths = ((measDate - dob) / (1000 * 60 * 60 * 24 * 30.44));
        const gender = babyProfile.gender || '';

        let percentileHtml = '';
        if (gender && latest.weight_kg) {
            percentileHtml += `<span class="badge badge-percentile badge-percentile-normal ms-1">Weight recorded</span>`;
        }
        if (gender && latest.length_cm) {
            percentileHtml += `<span class="badge badge-percentile badge-percentile-normal ms-1">Length recorded</span>`;
        }

        infoEl.innerHTML = `
            <div class="d-flex flex-wrap gap-3 align-items-center">
                <div>
                    <strong>Latest (${formatDate(latest.date)}):</strong>
                    ${latest.weight_kg ? `⚖️ ${latest.weight_kg} kg` : ''}
                    ${latest.length_cm ? `📏 ${latest.length_cm} cm` : ''}
                    ${latest.head_circumference_cm ? `🧠 ${latest.head_circumference_cm} cm` : ''}
                    ${percentileHtml}
                </div>
            </div>`;

        // History table (last 10)
        const rows = measurements.slice(0, 10).map(m => `
            <tr>
                <td>${formatDate(m.date)}</td>
                <td>${m.weight_kg ? m.weight_kg + ' kg' : '—'}</td>
                <td>${m.length_cm ? m.length_cm + ' cm' : '—'}</td>
                <td>${m.head_circumference_cm ? m.head_circumference_cm + ' cm' : '—'}</td>
                <td><button class="btn btn-outline-danger btn-sm py-0 px-1" onclick="deleteGrowthMeasurement(${m.id})">✕</button></td>
            </tr>`).join('');

        historyEl.innerHTML = `
            <h6 class="mb-2">Recent Measurements</h6>
            <div class="table-responsive">
                <table class="table table-sm growth-table">
                    <thead><tr><th>Date</th><th>Weight</th><th>Length</th><th>Head</th><th></th></tr></thead>
                    <tbody>${rows}</tbody>
                </table>
            </div>`;
    } catch (err) {
        log.error('Failed to load growth history:', err);
    }
}

// ── WHO Growth Chart ──
async function loadWHOChart(metric) {
    currentWHOMetric = metric;
    // Update button states
    document.getElementById('whoWeightBtn').classList.toggle('active', metric === 'weight');
    document.getElementById('whoLengthBtn').classList.toggle('active', metric === 'length');

    try {
        const gender = babyProfile?.gender || 'male';
        const res = await fetch(`${WHO_API}?metric=${metric}&gender=${gender}`);
        if (!res.ok) throw new Error('Failed to load WHO data');
        const data = await res.json();

        if (whoChart) { whoChart.destroy(); whoChart = null; }

        const canvas = document.getElementById('whoChart');
        const ctx = canvas.getContext('2d');

        const labels = data.months || [];
        const unit = metric === 'weight' ? 'kg' : 'cm';

        const datasets = [];

        // WHO percentile bands as filled areas
        if (data.p3) {
            datasets.push({
                label: 'P3',
                data: data.p3,
                borderColor: 'rgba(239,68,68,0.5)',
                backgroundColor: 'transparent',
                borderWidth: 1,
                borderDash: [4, 4],
                pointRadius: 0,
                fill: false,
            });
        }
        if (data.p15) {
            datasets.push({
                label: 'P15',
                data: data.p15,
                borderColor: 'rgba(245,158,11,0.5)',
                backgroundColor: 'rgba(245,158,11,0.06)',
                borderWidth: 1,
                borderDash: [4, 4],
                pointRadius: 0,
                fill: '-1', // fill between P3 and P15
            });
        }
        if (data.p50) {
            datasets.push({
                label: 'P50 (median)',
                data: data.p50,
                borderColor: 'rgba(34,197,94,0.8)',
                backgroundColor: 'rgba(34,197,94,0.06)',
                borderWidth: 2,
                pointRadius: 0,
                fill: '-1',
            });
        }
        if (data.p85) {
            datasets.push({
                label: 'P85',
                data: data.p85,
                borderColor: 'rgba(245,158,11,0.5)',
                backgroundColor: 'rgba(245,158,11,0.06)',
                borderWidth: 1,
                borderDash: [4, 4],
                pointRadius: 0,
                fill: '-1',
            });
        }
        if (data.p97) {
            datasets.push({
                label: 'P97',
                data: data.p97,
                borderColor: 'rgba(239,68,68,0.5)',
                backgroundColor: 'rgba(239,68,68,0.06)',
                borderWidth: 1,
                borderDash: [4, 4],
                pointRadius: 0,
                fill: '-1',
            });
        }

        // Baby's actual data points
        if (data.baby_data && data.baby_data.length > 0) {
            const babyPoints = new Array(labels.length).fill(null);
            data.baby_data.forEach(pt => {
                const monthIdx = Math.round(pt.month);
                if (monthIdx >= 0 && monthIdx < babyPoints.length) {
                    babyPoints[monthIdx] = pt.value;
                }
            });
            datasets.push({
                label: `Baby's ${metric}`,
                data: babyPoints,
                borderColor: '#0d9488',
                backgroundColor: '#0d9488',
                borderWidth: 2,
                pointRadius: 5,
                pointHoverRadius: 7,
                fill: false,
                spanGaps: true,
            });
        }

        whoChart = new Chart(ctx, {
            type: 'line',
            data: { labels, datasets },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                interaction: { mode: 'index', intersect: false },
                plugins: {
                    legend: { position: 'bottom', labels: { usePointStyle: true, padding: 12, font: { size: 11 } } },
                    tooltip: { callbacks: { label: ctx => `${ctx.dataset.label}: ${ctx.parsed.y} ${unit}` } },
                },
                scales: {
                    x: { title: { display: true, text: 'Age (months)' } },
                    y: { title: { display: true, text: metric === 'weight' ? 'Weight (kg)' : 'Length (cm)' }, beginAtZero: false },
                },
            },
        });
    } catch (err) {
        log.error('Failed to load WHO chart:', err);
    }
}

// ── Load AI Insights ──
async function loadInsights() {
    const loading = document.getElementById('insightLoading');
    const content = document.getElementById('insightContent');
    const alerts = document.getElementById('insightAlerts');

    loading.classList.remove('d-none');
    content.classList.add('d-none');
    alerts.classList.add('d-none');

    try {
        const res = await fetch(INSIGHTS_API);
        if (!res.ok) {
            if (res.status === 400) {
                // Profile not complete enough
                loading.classList.add('d-none');
                return;
            }
            throw new Error(`HTTP ${res.status}`);
        }
        const insight = await res.json();
        loading.classList.add('d-none');
        renderInsights(insight);
        content.classList.remove('d-none');
    } catch (err) {
        log.error('Failed to load insights:', err);
        loading.classList.add('d-none');
        content.classList.remove('d-none');
        document.getElementById('insightGrowthContent').innerHTML =
            '<p class="text-muted">Unable to generate insights at this time. Please try again later.</p>';
    }
}

// ── Render AI Insights ──
function renderInsights(insight) {
    // Growth Assessment
    const ga = insight.growth_assessment;
    if (ga) {
        const statusIcon = ga.status === 'on_track' ? '✅' : ga.status === 'concern' ? '⚠️' : '📊';
        document.getElementById('growthStatusIcon').textContent = statusIcon;
        let gaHtml = '';
        if (ga.weight_percentile) gaHtml += `<p><strong>Weight Percentile:</strong> ${escapeHtml(ga.weight_percentile)}</p>`;
        if (ga.length_percentile) gaHtml += `<p><strong>Length Percentile:</strong> ${escapeHtml(ga.length_percentile)}</p>`;
        if (ga.assessment) gaHtml += `<p>${escapeHtml(ga.assessment)}</p>`;
        if (ga.recommendation) gaHtml += `<p class="text-muted"><em>${escapeHtml(ga.recommendation)}</em></p>`;
        document.getElementById('insightGrowthContent').innerHTML = gaHtml || '<p class="text-muted">No growth assessment available.</p>';
    }

    // Feeding Analysis
    const fa = insight.feeding_analysis;
    if (fa) {
        const statusIcon = fa.status === 'on_track' ? '✅' : fa.status === 'concern' ? '⚠️' : '🍼';
        document.getElementById('feedingStatusIcon').textContent = statusIcon;
        let faHtml = '';
        if (fa.daily_average) faHtml += `<p><strong>Daily Average:</strong> ${escapeHtml(fa.daily_average)}</p>`;
        if (fa.assessment) faHtml += `<p>${escapeHtml(fa.assessment)}</p>`;
        if (fa.recommendation) faHtml += `<p class="text-muted"><em>${escapeHtml(fa.recommendation)}</em></p>`;
        document.getElementById('insightFeedingContent').innerHTML = faHtml || '<p class="text-muted">No feeding analysis available.</p>';
    }

    // Sleep Analysis
    const sa = insight.sleep_analysis;
    if (sa) {
        const statusIcon = sa.status === 'on_track' ? '✅' : sa.status === 'concern' ? '⚠️' : '😴';
        document.getElementById('sleepStatusIcon').textContent = statusIcon;
        let saHtml = '';
        if (sa.daily_average) saHtml += `<p><strong>Daily Average:</strong> ${escapeHtml(sa.daily_average)}</p>`;
        if (sa.assessment) saHtml += `<p>${escapeHtml(sa.assessment)}</p>`;
        if (sa.recommendation) saHtml += `<p class="text-muted"><em>${escapeHtml(sa.recommendation)}</em></p>`;
        document.getElementById('insightSleepContent').innerHTML = saHtml || '<p class="text-muted">No sleep analysis available.</p>';
    }

    // Alerts
    const alertsEl = document.getElementById('insightAlerts');
    if (insight.alerts && insight.alerts.length > 0) {
        alertsEl.innerHTML = insight.alerts.map(a => {
            const cls = a.severity === 'warning' ? 'insight-alert-warning' : a.severity === 'danger' ? 'insight-alert-danger' : 'insight-alert-info';
            const icon = a.severity === 'warning' ? '⚠️' : a.severity === 'danger' ? '🚨' : 'ℹ️';
            return `<div class="insight-alert ${cls}">${icon} ${escapeHtml(a.message)}</div>`;
        }).join('');
        alertsEl.classList.remove('d-none');
    }

    // Summary
    if (insight.summary) {
        document.getElementById('insightSummary').textContent = insight.summary;
    }
}

// ═══════════════════════════════════════════════
// ██  TIME GAP UTILITY                        ██
// ═══════════════════════════════════════════════

// Build a table row showing elapsed time between two timestamps
// colSpan = number of table columns to span
function buildTimeGapRow(prevEnd, currStart, colSpan) {
    const diffMs = currStart - prevEnd;
    if (diffMs <= 0) return '';
    const totalMin = Math.round(diffMs / 60000);
    const h = Math.floor(totalMin / 60);
    const m = totalMin % 60;
    let label;
    if (h > 0) label = `${h}h ${m}m`;
    else label = `${m}m`;
    return `<tr class="time-gap-row"><td colspan="${colSpan}" class="text-center"><span class="time-gap-badge">↕ ${label}</span></td></tr>`;
}

// ═══════════════════════════════════════════════
// ██  DIAPER TRACKING                         ██
// ═══════════════════════════════════════════════

// ── Diaper form helpers ──
function setDiaperDefaultTimes() {
    const now = new Date();
    document.getElementById('diaperDate').value = toLocalDate(now);
    document.getElementById('diaperTime').value = toLocalTime(now);
}

function changeDiaperMonth(delta) {
    const input = document.getElementById('diaperMonthFilter');
    const [y, m] = input.value.split('-').map(Number);
    const d = new Date(y, m - 1 + delta, 1);
    input.value = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
    loadDiapers();
}

// ── Load diapers from API ──
async function loadDiapers() {
    const monthFilter = document.getElementById('diaperMonthFilter').value;
    let url = DIAPER_API + `?tz=${encodeURIComponent(TZ)}`;
    if (monthFilter) url += `&month=${monthFilter}`;

    try {
        const res = await fetch(url);
        const diapers = await res.json();
        renderDiapers(diapers);
    } catch (err) {
        log.error('Failed to load diapers:', err);
    }
}

// ── Render diapers grouped by day ──
function renderDiapers(diapers) {
    const container = document.getElementById('diapersByDay');

    if (!diapers || diapers.length === 0) {
        container.innerHTML = '<p class="text-center text-muted py-3">No diaper changes recorded</p>';
        return;
    }

    const groups = {};
    diapers.forEach(d => {
        const day = new Date(d.time).toLocaleDateString(undefined, { weekday: 'short', year: 'numeric', month: 'short', day: 'numeric' });
        if (!groups[day]) groups[day] = [];
        groups[day].push(d);
    });

    const days = Object.keys(groups);

    container.innerHTML = days.map(day => {
        const items = groups[day];
        const dayCount = items.length;

        const rows = items.map((d, idx) => {
            const timeISO = encodeURIComponent(d.time);
            const typeIcon = d.type === 'pee' ? '💧' : d.type === 'poo' ? '💩' : '💧💩';
            const safeType = escapeHtml(d.type);
            const typeLabel = d.type === 'both' ? 'Pee + Poo' : d.type.charAt(0).toUpperCase() + d.type.slice(1);
            let gapHtml = '';
            if (idx > 0) {
                const prevTime = new Date(items[idx - 1].time);
                const currTime = new Date(d.time);
                gapHtml = buildTimeGapRow(prevTime, currTime, 3);
            }
            return `${gapHtml}
            <tr class="feeding-row">
                <td class="text-nowrap">${formatTime(d.time)}</td>
                <td>${typeIcon} <span class="badge bg-diaper rounded-pill">${escapeHtml(typeLabel)}</span></td>
                <td class="text-end text-nowrap">
                    <button class="btn btn-sm btn-outline-primary py-0 px-2" data-diaper-edit-id="${Number(d.id)}" data-type="${safeType}" data-time="${timeISO}">✏️</button>
                    <button class="btn btn-sm btn-outline-danger py-0 px-2" data-diaper-delete-id="${Number(d.id)}">🗑</button>
                </td>
            </tr>`;
        }).join('');

        return `
            <div class="mb-3">
                <div class="d-flex justify-content-between align-items-center bg-light px-3 py-2 rounded-top border">
                    <strong>${day}</strong>
                    <span class="badge bg-diaper rounded-pill">${dayCount} change${dayCount !== 1 ? 's' : ''}</span>
                </div>
                <div class="table-responsive border border-top-0 rounded-bottom">
                    <table class="table table-sm table-striped mb-0">
                        <thead class="table-light">
                            <tr><th>Time</th><th>Type</th><th class="text-end">Actions</th></tr>
                        </thead>
                        <tbody>${rows}</tbody>
                    </table>
                </div>
            </div>`;
    }).join('');
}

// ── Diaper form submit ──
async function handleDiaperSubmit(e) {
    e.preventDefault();
    const id = document.getElementById('diaperEditId').value;
    const date = document.getElementById('diaperDate').value;
    const typeEl = document.querySelector('input[name="diaperType"]:checked');
    const data = {
        time: toRFC3339(date + 'T' + document.getElementById('diaperTime').value),
        type: typeEl ? typeEl.value : 'pee',
    };

    try {
        const url = id ? `${DIAPER_API}/${id}` : DIAPER_API;
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
        document.getElementById('diaperForm').reset();
        setDiaperDefaultTimes();
        document.getElementById('diaperPee').checked = true;
        cancelDiaperEdit();
        showToast(id ? 'Diaper updated' : 'Diaper change added');
        loadDiapers();
    } catch (err) {
        log.error('Failed to save diaper:', err);
    }
}

// ── Diaper edit mode ──
function editDiaper(id, type, timeStr) {
    document.getElementById('diaperEditId').value = id;
    const dt = new Date(timeStr);
    document.getElementById('diaperDate').value = toLocalDate(dt);
    document.getElementById('diaperTime').value = toLocalTime(dt);
    const rb = document.querySelector(`input[name="diaperType"][value="${type}"]`);
    if (rb) rb.checked = true;
    document.getElementById('diaperFormTitle').textContent = 'Edit Diaper Change';
    document.getElementById('diaperSubmitBtn').textContent = 'Update';
    document.getElementById('diaperCancelBtn').classList.remove('d-none');
    document.getElementById('diaperForm').scrollIntoView({ behavior: 'smooth', block: 'center' });
}

function cancelDiaperEdit() {
    document.getElementById('diaperEditId').value = '';
    document.getElementById('diaperFormTitle').textContent = 'Add Diaper Change';
    document.getElementById('diaperSubmitBtn').textContent = 'Add';
    document.getElementById('diaperCancelBtn').classList.add('d-none');
}

// ── Delete diaper ──
async function deleteDiaper(id) {
    if (!confirm('Delete this diaper change?')) return;
    try {
        await fetch(`${DIAPER_API}/${id}`, { method: 'DELETE' });
        showToast('Diaper change deleted');
        loadDiapers();
    } catch (err) {
        log.error('Failed to delete diaper:', err);
    }
}


// ═══════════════════════════════════════════════
// ██  BATH TRACKING                           ██
// ═══════════════════════════════════════════════

// ── Bath form helpers ──
function setBathDefaultTimes() {
    const now = new Date();
    document.getElementById('bathDate').value = toLocalDate(now);
    document.getElementById('bathTime').value = toLocalTime(now);
}

function changeBathMonth(delta) {
    const input = document.getElementById('bathMonthFilter');
    const [y, m] = input.value.split('-').map(Number);
    const d = new Date(y, m - 1 + delta, 1);
    input.value = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
    loadBaths();
}

// ── Load baths from API ──
async function loadBaths() {
    const monthFilter = document.getElementById('bathMonthFilter').value;
    let url = BATH_API + `?tz=${encodeURIComponent(TZ)}`;
    if (monthFilter) url += `&month=${monthFilter}`;

    try {
        const res = await fetch(url);
        const baths = await res.json();
        renderBaths(baths);
    } catch (err) {
        log.error('Failed to load baths:', err);
    }
}

// ── Render baths grouped by day ──
function renderBaths(baths) {
    const container = document.getElementById('bathsByDay');

    if (!baths || baths.length === 0) {
        container.innerHTML = '<p class="text-center text-muted py-3">No baths recorded</p>';
        return;
    }

    const groups = {};
    baths.forEach(b => {
        const day = new Date(b.time).toLocaleDateString(undefined, { weekday: 'short', year: 'numeric', month: 'short', day: 'numeric' });
        if (!groups[day]) groups[day] = [];
        groups[day].push(b);
    });

    const days = Object.keys(groups);

    container.innerHTML = days.map(day => {
        const items = groups[day];
        const dayCount = items.length;

        const rows = items.map((b, idx) => {
            const timeISO = encodeURIComponent(b.time);
            let gapHtml = '';
            if (idx > 0) {
                const prevTime = new Date(items[idx - 1].time);
                const currTime = new Date(b.time);
                gapHtml = buildTimeGapRow(prevTime, currTime, 2);
            }
            return `${gapHtml}
            <tr class="feeding-row">
                <td class="text-nowrap">🛁 ${formatTime(b.time)}</td>
                <td class="text-end text-nowrap">
                    <button class="btn btn-sm btn-outline-primary py-0 px-2" data-bath-edit-id="${Number(b.id)}" data-time="${timeISO}">✏️</button>
                    <button class="btn btn-sm btn-outline-danger py-0 px-2" data-bath-delete-id="${Number(b.id)}">🗑</button>
                </td>
            </tr>`;
        }).join('');

        return `
            <div class="mb-3">
                <div class="d-flex justify-content-between align-items-center bg-light px-3 py-2 rounded-top border">
                    <strong>${day}</strong>
                    <span class="badge bg-bath rounded-pill">${dayCount} bath${dayCount !== 1 ? 's' : ''}</span>
                </div>
                <div class="table-responsive border border-top-0 rounded-bottom">
                    <table class="table table-sm table-striped mb-0">
                        <thead class="table-light">
                            <tr><th>Time</th><th class="text-end">Actions</th></tr>
                        </thead>
                        <tbody>${rows}</tbody>
                    </table>
                </div>
            </div>`;
    }).join('');
}

// ── Bath form submit ──
async function handleBathSubmit(e) {
    e.preventDefault();
    const id = document.getElementById('bathEditId').value;
    const date = document.getElementById('bathDate').value;
    const data = {
        time: toRFC3339(date + 'T' + document.getElementById('bathTime').value),
    };

    try {
        const url = id ? `${BATH_API}/${id}` : BATH_API;
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
        document.getElementById('bathForm').reset();
        setBathDefaultTimes();
        cancelBathEdit();
        showToast(id ? 'Bath updated' : 'Bath added');
        loadBaths();
    } catch (err) {
        log.error('Failed to save bath:', err);
    }
}

// ── Bath edit mode ──
function editBath(id, timeStr) {
    document.getElementById('bathEditId').value = id;
    const dt = new Date(timeStr);
    document.getElementById('bathDate').value = toLocalDate(dt);
    document.getElementById('bathTime').value = toLocalTime(dt);
    document.getElementById('bathFormTitle').textContent = 'Edit Bath';
    document.getElementById('bathSubmitBtn').textContent = 'Update';
    document.getElementById('bathCancelBtn').classList.remove('d-none');
    document.getElementById('bathForm').scrollIntoView({ behavior: 'smooth', block: 'center' });
}

function cancelBathEdit() {
    document.getElementById('bathEditId').value = '';
    document.getElementById('bathFormTitle').textContent = 'Add Bath';
    document.getElementById('bathSubmitBtn').textContent = 'Add';
    document.getElementById('bathCancelBtn').classList.add('d-none');
}

// ── Delete bath ──
async function deleteBath(id) {
    if (!confirm('Delete this bath?')) return;
    try {
        await fetch(`${BATH_API}/${id}`, { method: 'DELETE' });
        showToast('Bath deleted');
        loadBaths();
    } catch (err) {
        log.error('Failed to delete bath:', err);
    }
}
