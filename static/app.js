// State
let config = null;
let refreshInterval = null;
let isLoading = false;

// DOM Elements
const stopsGrid = document.getElementById('stopsGrid');
const lastUpdatedEl = document.getElementById('lastUpdated');
const refreshBtn = document.getElementById('refreshBtn');
const errorBanner = document.getElementById('errorBanner');
const errorText = document.getElementById('errorText');

// Initialize
async function init() {
    try {
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

    stopsGrid.innerHTML = config.stops.map(stop => `
        <div class="stop-card">
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
        </div>
    `).join('');
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
        renderArrivals(data);
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

// Render arrivals data
function renderArrivals(data) {
    lastUpdatedEl.textContent = formatLocalTime();

    stopsGrid.innerHTML = data.stops.map(stop => `
        <div class="stop-card">
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
        </div>
    `).join('');
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

    if (!direction.arrivals || direction.arrivals.length === 0) {
        return `<span class="no-arrivals">No upcoming vehicles</span>`;
    }

    return direction.arrivals.map(arrival => {
        const isImminent = arrival.minutes <= 5;
        const trainType = getTrainTypeLabel(arrival.line_type);
        const trainClass = getTrainTypeClass(arrival.line_type);

        return `
            <div class="arrival-pill ${isImminent ? 'imminent' : ''} ${trainClass}">
                ${trainType ? `<span class="train-type">${trainType}</span>` : ''}
                <span class="minutes">${arrival.minutes}</span>
                <span class="minutes-label">min</span>
            </div>
        `;
    }).join('');
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

// Event listeners
refreshBtn.addEventListener('click', () => {
    fetchArrivals();
});

// Start the app
init();
