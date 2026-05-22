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
