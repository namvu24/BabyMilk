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

