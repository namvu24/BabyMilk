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
