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
            scales: {
                x: {
                    title: { display: true, text: 'Day' },
                },
            },
        },
    });
}
