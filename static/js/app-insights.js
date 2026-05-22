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
