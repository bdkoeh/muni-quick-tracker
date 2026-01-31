// State
let config = null;
let refreshInterval = null;
let isLoading = false;
let arrivalsData = null;
let displayMode = 'minutes'; // 'minutes' or 'time'

// DOM Elements
const stopsGrid = document.getElementById('stopsGrid');
const lastUpdatedEl = document.getElementById('lastUpdated');
const toggleBtn = document.getElementById('toggleBtn');
const toggleText = document.getElementById('toggleText');
const refreshBtn = document.getElementById('refreshBtn');
const errorBanner = document.getElementById('errorBanner');
const errorText = document.getElementById('errorText');

// Initialize
async function init() {
    try {
        // Load display mode from localStorage
        const savedMode = localStorage.getItem('displayMode');
        if (savedMode === 'time') {
            displayMode = 'time';
            toggleText.textContent = 'time';
            toggleBtn.classList.add('active');
        }

        // Load config first
        const response = await fetch('/api/config');
        config = await response.json();

        // Render initial skeleton
        renderSkeletons();

        // Fetch arrivals
        await fetchArrivals();

        // Set up auto-refresh
        startAutoRefresh();

        // Set up visibility handling
        setupVisibilityHandler();

    } catch (error) {
        console.error('Init error:', error);
        showError('Failed to load configuration');
    }
}

// Render skeleton loaders
function renderSkeletons() {
    if (!config) return;

    stopsGrid.innerHTML = config.stops.map((stop, index) => {
        const isTThird = index === 0 && stop.line.toLowerCase().includes('t third');
        const isSecond = index === 1;
        const isLast = index === config.stops.length - 1;
        let dataAttr = isTThird ? 'data-card="t-third"' : '';
        if (isSecond) dataAttr = 'data-card="second"';
        if (isLast) dataAttr = 'data-card="last"';
        let dripImg = isTThird ? '<img class="card__drip" src="/drip1.png" alt="" aria-hidden="true" /><img class="card__drip-left" src="/drip4.png" alt="" aria-hidden="true" />' : '';
        if (isSecond) dripImg = '<img class="card__drip-center" src="/drip3.png" alt="" aria-hidden="true" />';
        if (isLast) dripImg = '<img class="card__drip-left" src="/drip2.png" alt="" aria-hidden="true" />';
        return `
        <div class="stop-card" ${dataAttr}>
            <div class="stop-header">
                <div class="line-badge ${getLineBadgeClass(stop.line)}">${getLineInitial(stop.line)}</div>
                <div class="stop-info">
                    <h2>${stop.name}</h2>
                    <span class="line-name">${stop.line}</span>
                </div>
            </div>
            ${stop.directions.map(dir => `
                <div class="direction">
                    <div class="direction-label">${dir.label}</div>
                    <div class="arrivals">
                        <div class="skeleton skeleton-pill"></div>
                        <div class="skeleton skeleton-pill"></div>
                        <div class="skeleton skeleton-pill"></div>
                    </div>
                </div>
            `).join('')}
            ${dripImg}
        </div>
    `}).join('');

    // Fix drip positioning after skeleton render
    setTimeout(fixDripPositioning, 50);
}

// Fetch arrivals from API
async function fetchArrivals() {
    if (isLoading) return;

    isLoading = true;
    refreshBtn.classList.add('loading');

    try {
        const response = await fetch('/api/arrivals');

        if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
        }

        const data = await response.json();
        arrivalsData = data;
        renderArrivals();
        hideError();

    } catch (error) {
        console.error('Fetch error:', error);
        showError('Unable to fetch arrivals');
    } finally {
        isLoading = false;
        refreshBtn.classList.remove('loading');
    }
}

// Format current time in user's local timezone
function formatLocalTime() {
    return new Date().toLocaleTimeString('en-US', {
        hour: 'numeric',
        minute: '2-digit',
        second: '2-digit',
        hour12: true
    });
}

// Format arrival time from RFC3339 timestamp
function formatArrivalTime(arrivalTime) {
    const date = new Date(arrivalTime);
    return date.toLocaleTimeString('en-US', {
        hour: 'numeric',
        minute: '2-digit',
        hour12: true
    });
}

// Render quality warning badge
function renderQualityWarning(qualityWarning, qualityLevel) {
    if (!qualityWarning) return '';

    return `
        <div class="quality-warning">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
                <line x1="12" y1="9" x2="12" y2="13"/>
                <line x1="12" y1="17" x2="12.01" y2="17"/>
            </svg>
            <span class="quality-message">${qualityWarning}</span>
        </div>
    `;
}

// Render arrivals data
function renderArrivals() {
    if (!arrivalsData) return;

    lastUpdatedEl.textContent = formatLocalTime();

    stopsGrid.innerHTML = arrivalsData.stops.map((stop, index) => {
        const isTThird = index === 0 && stop.line.toLowerCase().includes('t third');
        const isSecond = index === 1;
        const isLast = index === arrivalsData.stops.length - 1;
        let dataAttr = isTThird ? 'data-card="t-third"' : '';
        if (isSecond) dataAttr = 'data-card="second"';
        if (isLast) dataAttr = 'data-card="last"';
        let dripImg = isTThird ? '<img class="card__drip" src="/drip1.png" alt="" aria-hidden="true" /><img class="card__drip-left" src="/drip4.png" alt="" aria-hidden="true" />' : '';
        if (isSecond) dripImg = '<img class="card__drip-center" src="/drip3.png" alt="" aria-hidden="true" />';
        if (isLast) dripImg = '<img class="card__drip-left" src="/drip2.png" alt="" aria-hidden="true" />';
        return `
        <div class="stop-card" ${dataAttr}>
            <div class="stop-header">
                <div class="line-badge ${getLineBadgeClass(stop.line)}">${getLineInitial(stop.line)}</div>
                <div class="stop-info">
                    <h2>${stop.name}</h2>
                    <span class="line-name">${stop.line}</span>
                </div>
            </div>
            ${stop.directions.map(dir => `
                <div class="direction">
                    <div class="direction-label">${dir.label}</div>
                    <div class="arrivals">
                        ${renderDirectionArrivals(dir)}
                    </div>
                </div>
            `).join('')}
            ${dripImg}
        </div>
    `}).join('');

    // Fix drip positioning after arrivals render
    setTimeout(fixDripPositioning, 50);
}

// Get short train type label
function getTrainTypeLabel(lineType) {
    if (!lineType) return '';
    const l = lineType.toLowerCase();
    if (l.includes('express')) return 'EXP';
    if (l.includes('limited')) return 'LTD';
    if (l.includes('local')) return 'LCL';
    if (l.includes('bullet')) return 'BLT';
    return '';
}

// Get train type CSS class
function getTrainTypeClass(lineType) {
    if (!lineType) return '';
    const l = lineType.toLowerCase();
    if (l.includes('express') || l.includes('bullet')) return 'express';
    if (l.includes('limited')) return 'limited';
    if (l.includes('local')) return 'local';
    return '';
}

// Render arrivals for a single direction
function renderDirectionArrivals(direction) {
    if (direction.error) {
        return `<span class="error-message">${direction.error}</span>`;
    }

    // Render quality warning if present
    const qualityWarning = direction.quality_warning
        ? renderQualityWarning(direction.quality_warning, direction.quality_level)
        : '';

    if (!direction.arrivals || direction.arrivals.length === 0) {
        return qualityWarning || `<span class="no-arrivals">No upcoming vehicles</span>`;
    }

    const arrivalPills = direction.arrivals.map(arrival => {
        const isNow = arrival.minutes <= 0;
        const isImminent = arrival.minutes <= 5 && arrival.minutes > 0;
        const trainType = getTrainTypeLabel(arrival.line_type);
        const trainClass = getTrainTypeClass(arrival.line_type);

        let displayValue, displayLabel;
        if (displayMode === 'time') {
            displayValue = formatArrivalTime(arrival.arrival_time);
            displayLabel = '';
        } else {
            displayValue = isNow ? 'Now' : arrival.minutes;
            displayLabel = isNow ? '' : '<span class="minutes-label">min</span>';
        }

        return `
            <div class="arrival-pill ${isNow ? 'now' : ''} ${isImminent ? 'imminent' : ''} ${trainClass}">
                ${trainType ? `<span class="train-type">${trainType}</span>` : ''}
                <span class="minutes">${displayValue}</span>
                ${displayLabel}
            </div>
        `;
    }).join('');

    return qualityWarning + arrivalPills;
}

// Get line badge CSS class
function getLineBadgeClass(line) {
    const l = line.toLowerCase();
    if (l.includes('caltrain')) return 'caltrain';
    if (l.includes('t ') || l.startsWith('t')) return 't-line';
    if (l.includes('n ') || l.startsWith('n')) return 'n-line';
    return 'default';
}

// Get line initial for badge
function getLineInitial(line) {
    const l = line.toLowerCase();
    if (l.includes('caltrain')) return 'CT';
    // Extract first letter/character that represents the line
    const match = line.match(/^([A-Z])/i);
    return match ? match[1].toUpperCase() : '?';
}

// Auto-refresh handling
function startAutoRefresh() {
    if (refreshInterval) {
        clearInterval(refreshInterval);
    }

    const interval = (config?.refresh_interval || 30) * 1000;
    refreshInterval = setInterval(() => {
        if (!document.hidden) {
            fetchArrivals();
        }
    }, interval);
}

function stopAutoRefresh() {
    if (refreshInterval) {
        clearInterval(refreshInterval);
        refreshInterval = null;
    }
}

// Visibility handling - pause when tab is hidden
function setupVisibilityHandler() {
    document.addEventListener('visibilitychange', () => {
        if (document.hidden) {
            stopAutoRefresh();
        } else {
            // Refresh immediately when becoming visible
            fetchArrivals();
            startAutoRefresh();
        }
    });
}

// Error handling
function showError(message) {
    errorText.textContent = message;
    errorBanner.classList.add('visible');
}

function hideError() {
    errorBanner.classList.remove('visible');
}

// Toggle display mode
function toggleDisplayMode() {
    if (displayMode === 'minutes') {
        displayMode = 'time';
        toggleText.textContent = 'time';
        toggleBtn.classList.add('active');
        localStorage.setItem('displayMode', 'time');
    } else {
        displayMode = 'minutes';
        toggleText.textContent = 'min';
        toggleBtn.classList.remove('active');
        localStorage.setItem('displayMode', 'minutes');
    }
    renderArrivals();
}

// Event listeners
toggleBtn.addEventListener('click', toggleDisplayMode);

refreshBtn.addEventListener('click', () => {
    fetchArrivals();
});

// Force repaint of drip images after they load (fixes mobile positioning bug)
function fixDripPositioning() {
    const drips = document.querySelectorAll('[class*="card__drip"]');
    drips.forEach(img => {
        if (img.complete) {
            img.style.opacity = '0.99';
            requestAnimationFrame(() => {
                img.style.opacity = '1';
            });
        } else {
            img.onload = () => {
                img.style.opacity = '0.99';
                requestAnimationFrame(() => {
                    img.style.opacity = '1';
                });
            };
        }
    });
}

// Start the app
init();

// Fix drip positioning after initial render and on each re-render
document.addEventListener('DOMContentLoaded', fixDripPositioning);
window.addEventListener('load', fixDripPositioning);
