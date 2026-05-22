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
    });
}

