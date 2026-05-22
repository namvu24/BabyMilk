document.addEventListener('DOMContentLoaded', async () => {
    // ── Load tab partials ──
    await Promise.all([
        { id: 'milkPane',        url: '/static/tabs/tab-feeding.html' },
        { id: 'sleepPane',       url: '/static/tabs/tab-sleep.html' },
        { id: 'eventPane',       url: '/static/tabs/tab-event.html' },
        { id: 'developmentPane', url: '/static/tabs/tab-development.html' },
        { id: 'insightsPane',    url: '/static/tabs/tab-insights.html' },
    ].map(async ({ id, url }) => {
        const res = await fetch(url);
        document.getElementById(id).innerHTML = await res.text();
    }));

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

    // ── Development tab initialisation ──
    document.getElementById('dev-tab').addEventListener('shown.bs.tab', () => {
        initDevelopmentTab();
    });

    // ── Insights tab initialisation ──
    document.getElementById('insights-tab').addEventListener('shown.bs.tab', () => {
        initInsightsTab();
    });
});
