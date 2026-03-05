const API = '/api/feedings';
let dailyChart = null;
let feedingStartTime = null;
let timerInterval = null;
let toastTimer = null;

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

function toggleFeeding() {
    const btn = document.getElementById('startStopBtn');
    const timerEl = document.getElementById('feedingTimer');

    if (!feedingStartTime) {
        // START
        feedingStartTime = new Date();
        document.getElementById('startTime').value = toLocalISO(feedingStartTime);
        document.getElementById('endTime').value = toLocalISO(feedingStartTime);

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
        document.getElementById('endTime').value = toLocalISO(now);

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

function setDefaultTimes() {
    const now = toLocalISO(new Date());
    document.getElementById('startTime').value = now;
    document.getElementById('endTime').value = now;
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

    // Snap time inputs to nearest 5 minutes on change
    ['startTime', 'endTime'].forEach(id => {
        document.getElementById(id).addEventListener('change', (e) => {
            const val = e.target.value;
            if (val) {
                e.target.value = toLocalISO(roundTo5Min(new Date(val)));
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
});

function toLocalISO(date) {
    const off = date.getTimezoneOffset();
    const local = new Date(date.getTime() - off * 60000);
    return local.toISOString().slice(0, 16);
}

function toRFC3339(datetimeLocal) {
    return new Date(datetimeLocal).toISOString();
}

function formatTime(iso) {
    return new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

function formatDate(iso) {
    return new Date(iso).toLocaleDateString();
}

async function loadFeedings() {
    const monthFilter = document.getElementById('monthFilter').value;
    let url = API;
    if (monthFilter) url += `?month=${monthFilter}`;

    try {
        const res = await fetch(url);
        const feedings = await res.json();
        renderFeedings(feedings);
    } catch (err) {
        console.error('Failed to load feedings:', err);
    }
}

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
            <tr>
                <td data-label="Start">${formatTime(f.start_time)}</td>
                <td data-label="End">${formatTime(f.end_time)}</td>
                <td data-label="Amount">${f.amount_ml} ml</td>
                <td>
                    <button class="btn btn-sm btn-outline-primary" data-edit-id="${f.id}" data-amount="${f.amount_ml}" data-start="${startISO}" data-end="${endISO}">Edit</button>
                    <button class="btn btn-sm btn-outline-danger" data-delete-id="${f.id}">Delete</button>
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
                            <tr><th>Start</th><th>End</th><th>Amount</th><th>Actions</th></tr>
                        </thead>
                        <tbody>${rows}</tbody>
                    </table>
                </div>
            </div>`;
    }).join('');
}

async function handleSubmit(e) {
    e.preventDefault();
    const id = document.getElementById('editId').value;
    const data = {
        amount_ml: parseInt(document.getElementById('amountMl').value),
        start_time: toRFC3339(document.getElementById('startTime').value),
        end_time: toRFC3339(document.getElementById('endTime').value),
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
        console.error('Failed to save feeding:', err);
    }
}

function editFeeding(id, amount, startTime, endTime) {
    document.getElementById('editId').value = id;
    document.getElementById('amountMl').value = amount;
    document.getElementById('startTime').value = toLocalISO(new Date(startTime));
    document.getElementById('endTime').value = toLocalISO(new Date(endTime));
    document.getElementById('formTitle').textContent = 'Edit Feeding';
    document.getElementById('submitBtn').textContent = 'Update';
    document.getElementById('cancelBtn').classList.remove('d-none');

    // Scroll to form and focus amount field
    document.getElementById('feedingForm').scrollIntoView({ behavior: 'smooth', block: 'center' });
    document.getElementById('amountMl').focus();
}

function cancelEdit() {
    document.getElementById('editId').value = '';
    document.getElementById('formTitle').textContent = 'Add Feeding';
    document.getElementById('submitBtn').textContent = 'Add';
    document.getElementById('cancelBtn').classList.add('d-none');
}

async function deleteFeeding(id) {
    if (!confirm('Delete this feeding?')) return;
    try {
        await fetch(`${API}/${id}`, { method: 'DELETE' });
        showToast('Feeding deleted');
        loadFeedings();
        loadDailyTotals();
        loadQuickAmounts();
    } catch (err) {
        console.error('Failed to delete feeding:', err);
    }
}

async function loadQuickAmounts() {
    try {
        const res = await fetch(API + '?_t=' + Date.now());
        const feedings = await res.json();
        updateQuickAmounts(feedings);
    } catch (err) {
        console.error('Failed to load quick amounts:', err);
    }
}

function updateQuickAmounts(feedings) {
    const container = document.getElementById('quickAmounts');
    if (!feedings || feedings.length === 0) {
        container.innerHTML = '';
        return;
    }
    const lastAmount = feedings[0].amount_ml;
    const steps = [-20, -10, 0, 10, 20];
    container.innerHTML = steps.map(s => {
        const val = Math.max(1, lastAmount + s);
        const active = 'btn-outline-secondary';
        return `<button type="button" class="btn ${active} px-3 py-2" style="min-width:56px;font-size:1.1rem" onclick="document.getElementById('amountMl').value=${val}">${val}</button>`;
    }).join('');
}

async function loadDailyTotals() {
    try {
        console.log('Loading daily totals...');
        const month = document.getElementById('monthFilter').value;
        const url = month ? `${API}/daily?month=${month}` : `${API}/daily?days=31`;
        const res = await fetch(url);
        const totals = await res.json();
        renderChart(totals);
    } catch (err) {
        console.error('Failed to load daily totals:', err);
    }
}

function renderChart(totals) {
    const ctx = document.getElementById('dailyChart').getContext('2d');
    const labels = totals.map(t => t.date);
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
            layout: { padding: { top: 24 } },
            scales: {
                y: {
                    beginAtZero: true,
                    title: { display: true, text: 'Milliliters (ml)' },
                },
                x: {
                    title: { display: true, text: 'Date' },
                },
            },
        },
        plugins: [dataLabelPlugin],
    });
}
