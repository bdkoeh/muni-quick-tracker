# Muni Tram Arrivals App — Product Spec (Behavior Only)

## Overview

A small, locally hosted web app that shows **real-time Muni tram arrivals** for **two preconfigured stops (A and B)**, including **both travel directions**, in a glanceable format.

This spec intentionally excludes implementation details, but it is written to be **unambiguous for an agentic AI builder** (clear inputs, behaviors, and testable outcomes).

---

## Audience and operating context
- **Audience:** an agentic AI that will implement the app end-to-end from this spec.
- **Primary usage moment:** a quick glance while **leaving the house**.
- **Primary devices:** phone first (small screen), desktop/tablet second.
- **Environment:** **self-hosted on a home network**.
- **Packaging preference:** optimized for running as a **single Docker container**.

## Goals

- Provide a fast, low-friction view of **minutes until next departures** for two nearby/favorite tram stops.
- Keep stop selection **static and simple** by reading Stop A/Stop B from a local config file.
- Present information with a **slick, polished UI** using **graphics and subtle animations** that make the page feel instantly legible and premium.
- Avoid unnecessary background activity: the app should **only fetch real-time data when the page is actively loaded and used**.

## Hosting and packaging constraints (Docker-friendly)
- The app should be runnable as a **single container** with a simple start command.
- The app should be **stateless** at runtime (no required database).
- Configuration should come from a **mounted config file** (and optionally environment variables).
- Logs should be written to **stdout/stderr** (so container logs capture them).
- The app should expose a **single HTTP port** for the web UI.

## Non-goals

- Notifications, alerts, or reminders
- Route planning or trip suggestions
- Maps, navigation, or walking-time estimates
- Multi-user accounts
- Historical charts/analytics

---

## Primary user flow

### Normal use

1. User opens the app.
2. App displays arrivals for **Stop A** and **Stop B** immediately.
3. Arrivals refresh automatically on a fixed cadence.
4. User can optionally trigger a manual refresh.

---

## Screens

## 1) Arrivals screen (default)

### UX priorities (leaving-the-house mode)
- **Zero friction:** open page and immediately see the next departures.
- **At-a-glance readability:** large, high-contrast minutes; minimal clutter.
- **Stable layout:** values update without shifting elements around.
- **Polished feel:** tasteful motion and graphics, but never at the cost of speed.

### Layout

- Two “stop cards”&#x20;
- Each stop card contains:
  - Stop name
  - Line name (or identifier) as context
  - Two direction sections (clearly labeled)
  - A list of upcoming departures expressed in **minutes**
  - “Last updated” timestamp

### Direction section behavior

- Each stop shows **both directions**. (i.e. 4th & King street (T line) tram stop goes to Sunnydale direction or China Town direction.
- Each direction shows:
  - Direction label (e.g., “Inbound” / “Outbound” or equivalent)
  - Next departures as a list, e.g., `3, 9, 18` minutes within the same row

### Refresh behavior
- Fetch real-time data **only while the Arrivals page is actively in use**.
- Refresh occurs:
  - On initial page load.
  - On a fixed cadence **only while the page is visible/foreground** (every 30 seconds).
  - On explicit user action (manual refresh).
- If the tab is hidden/backgrounded, the window is minimized, or the device is locked, **pause automatic refresh**.
- If the user navigates away or closes the tab, fetching stops.

---

## 2) Configuration (config file)

### Purpose
Define the two stops (A and B) in a local config file. This app is purpose-built for a single household and does not require end-user configuration.

### Requirements
- A config file defines, for each stop:
  - A human-friendly display name (what shows in the UI)
  - The line/route context (if needed for disambiguation)
  - Two directions (each with a direction label that matches what the user expects, e.g., “Chinatown” / “Sunnydale”)
- Configuration changes take effect on app restart (or next reload), without any in-app UI.

### Config validity behavior
- If the config file is missing or invalid, the app shows a clear error screen explaining what is wrong and that the fix is to update the config.

---

## Data display rules

- Show **minutes until departure** (not absolute timestamps) for quick scanning.
- Display at least the **next 3 departures** per direction when available.
- If fewer than 3 are available, show what exists.
- If no upcoming departures exist, show **“No upcoming vehicles”** for that direction.

---

## Error and outage behavior (graceful degradation)

- The app should always load the UI, even if real-time data is unavailable.
- Failures should be localized:
  - If Stop A data fails, Stop B should still show normally.
  - If one direction fails, the other direction can still show.

### User-visible states

- **Loading:** show lightweight placeholders per direction.
- **No upcoming vehicles:** explicit message.
- **No data / error:** explicit message such as “No data” (optionally with a retry).
- **Stale data:** if refresh fails, keep last known values but indicate staleness via the “Last updated” time.

---

## UI and visual design expectations

### Visual style
- The Arrivals screen should feel **slick and modern**.
- Use **polished graphics** (icons, typography, spacing) to maximize at-a-glance readability.
- Use **subtle, purposeful animations** on page load and during refresh (e.g., smooth value transitions, gentle skeleton loading), avoiding distracting motion.
- Prefer a calm, minimal aesthetic over dense UI.

### Readability requirements
- Minutes should be the dominant visual element.
- Each stop’s two directions must be visually distinct and easy to scan.
- Avoid long text wrapping that forces vertical scrolling on a typical phone screen.

## Performance expectations

- The Arrivals screen should feel **slick and modern**.
- Use **polished graphics** (icons, typography, spacing) to maximize at-a-glance readability.
- Use **subtle, purposeful animations** on page load and during refresh (e.g., smooth value transitions, gentle skeleton loading), avoiding distracting motion.

## Performance expectations

- Arrivals screen should become readable quickly after page load.
- Animations should not block interaction or delay initial data visibility.
- Updates should feel near-instant when refresh completes.

---

## Acceptance criteria (testable)

### Functional
1. **Preconfigured two stops**

   - Stop A and Stop B are defined in a local config file.
   - The app loads successfully with a valid config file and shows both stops.

2. **Both directions per stop**

   - For each configured stop, two direction sections are always visible.

3. **Minute-based arrivals**

   - Each direction displays upcoming departures in minutes.
   - The list updates when data refreshes.

4. **Fetch only while in use**

   - No API calls occur unless the Arrivals page is loaded.
   - Automatic refresh runs only while the page is **visible/foreground**.
   - Hiding the tab/backgrounding the app pauses automatic refresh.

5. **Manual refresh**

   - User can manually refresh from the Arrivals screen.

6. **Resilient UI**

   - Feed failure results in a clear message, not a blank page.
   - Partial failure does not block other stops/directions.

### UX
7. **Fast to first useful paint**

   - The page becomes legible quickly, with placeholders that preserve layout until data arrives.

8. **Stable layout during updates**

   - Refreshes do not cause jarring reflow; values update smoothly in place.

### Deployment
9. **Docker-first operation**

   - The app runs correctly as a single container on a home network.
   - All configuration is provided via a mounted file (and/or environment variables), with no interactive setup.
   - The app starts cleanly and serves the UI on a single HTTP port.

---

## Edge cases (explicit behaviors) **Stable layout during updates**

   - Refreshes do not cause jarring reflow; values update smoothly in place.

---

## Edge cases (explicit behaviors)

1. **Preconfigured two stops**

   - Stop A and Stop B are defined in a local config file.
   - The app loads successfully with a valid config file and shows both stops.

2. **Both directions per stop**

   - For each configured stop, two direction sections are always visible.

3. **Minute-based arrivals**

   - Each direction displays upcoming departures in minutes.
   - The list updates when data refreshes.

4. **Fetch only while in use**

   - No API calls occur unless the Arrivals page is loaded.
   - Closing the tab or navigating away stops automatic refresh.

5. **Manual refresh**

   - User can manually refresh from the Arrivals screen.

6. **Resilient UI**

   - Feed failure results in a clear message, not a blank page.
   - Partial failure does not block other stops/directions.

---

## Edge cases (explicit behaviors)

- Late night service gaps: show “No upcoming vehicles.”
- One direction missing: show “No data” for that direction.
- Temporary network outage: show stale data with last updated time.
- Invalid/missing config: show a clear error state explaining the config is missing/invalid.

