# Netgear Switch Controller API Reference

This document provides a comprehensive API reference for the Netgear Switch Remote Control (`ntgrrc`) application, detailing the HTTP endpoints and request/response formats for each command function.

## Overview

The application communicates with Netgear switches (models GS305EP, GS305EPP, GS308EP, GS308EPP, GS316EP, GS316EPP) via HTTP requests. The API follows these general principles:

- **Protocol**: HTTP (not HTTPS)
- **Authentication**: Session-based using tokens (SID cookie for 30x models, Gambit token for 316 models)
- **Request Format**: Form-encoded POST requests (`application/x-www-form-urlencoded`)
- **Response Format**: HTML pages with embedded data or plain text status messages

## Authentication Flow

### 1. Login Command

The login process involves three steps:

#### Step 1: Fetch Seed Value

**Models GS305EP/GS305EPP/GS308EP/GS308EPP:**
- **Endpoint**: `GET http://{host}/login.cgi`
- **Purpose**: Retrieve random seed value for password encryption
- **Response**: HTML containing element with `id="rand"` and `value` attribute

**Models GS316EP/GS316EPP:**
- **Endpoint**: `GET http://{host}/wmi/login`
- **Purpose**: Retrieve random seed value for password encryption
- **Response**: HTML containing element with `id="rand"` and `value` attribute

#### Step 2: Encrypt Password

The password is encrypted using a special merge algorithm with the seed value:
1. Interleave password and seed characters
2. Apply MD5 hash to the merged string
3. Convert hash to hexadecimal string

#### Step 3: Perform Login

**Models GS305EP/GS305EPP/GS308EP/GS308EPP:**
- **Endpoint**: `POST http://{host}/login.cgi`
- **Headers**: `Content-Type: application/x-www-form-urlencoded`
- **Body**: `password={encrypted_password}`
- **Response**:
  - Success: HTTP 200 with `Set-Cookie: SID={session_token}`
  - Failure: HTTP 200 without session token

**Models GS316EP/GS316EPP:**
- **Endpoint**: `POST http://{host}/redirect.html`
- **Headers**: `Content-Type: application/x-www-form-urlencoded`
- **Body**: `LoginPassword={encrypted_password}`
- **Response**:
  - Success: HTML containing hidden input field with `name="Gambit"` and session token as value
  - Failure: HTTP 200 without Gambit token

## POE Commands

### 2. POE Status Command

Retrieves current Power over Ethernet status for all ports.

**Models GS305EP/GS305EPP/GS308EP/GS308EPP:**
- **Endpoint**: `GET http://{host}/getPoePortStatus.cgi`
- **Headers**: `Cookie: SID={session_token}`
- **Response**: HTML containing `li.poePortStatusListItem` elements with port data

**Models GS316EP/GS316EPP:**
- **Endpoint**: `GET http://{host}/iss/specific/poePortStatus.html?GetData=TRUE`
- **Headers**: `Cookie: Gambit={session_token}`
- **Response**: HTML containing `div.port-wrap` elements with port data

**Response Data Structure:**
- Port Index (1-based)
- Port Name
- POE Status (enabled/disabled)
- Power Class (0-4)
- Voltage (V)
- Current (mA)
- Power (W)
- Temperature (°C)
- Error Status

### 3. POE Settings Command

Retrieves current POE configuration for all ports.

**Models GS305EP/GS305EPP/GS308EP/GS308EPP:**
- **Endpoint**: `GET http://{host}/PoEPortConfig.cgi`
- **Headers**: `Cookie: SID={session_token}`
- **Response**: HTML containing `li.poePortSettingListItem` elements

**Models GS316EP/GS316EPP:**
- **Endpoint**: `GET http://{host}/iss/specific/poePortConf.html`
- **Headers**: `Cookie: Gambit={session_token}`
- **Response**: HTML containing port configuration data

**Response Data Structure:**
- Port Index
- Port Name
- Port Power (enabled/disabled)
- Power Mode (802.3af, legacy, pre-802.3at, 802.3at)
- Priority (low, high, critical)
- Limit Type (none, class, user)
- Power Limit (W)
- Detection Type (IEEE 802, legacy, 4pt 802.3af + Legacy)
- Longer Detection Time (enable/disable)

### 4. POE Set Configuration Command

Updates POE settings for specified ports.

**Models GS305EP/GS305EPP/GS308EP/GS308EPP:**
- **Endpoint**: `POST http://{host}/PoEPortConfig.cgi`
- **Headers**:
  - `Cookie: SID={session_token}`
  - `Content-Type: application/x-www-form-urlencoded`
- **Body Parameters**:
  - `hash`: Security hash from settings page
  - `ACTION`: "Apply"
  - `portID`: Port index (0-based)
  - `ADMIN_MODE`: "1" (enable) or "0" (disable)
  - `PORT_PRIO`: Priority value (0=low, 1=high, 2=critical)
  - `POW_MOD`: Power mode value
  - `POW_LIMT_TYP`: Limit type value
  - `POW_LIMT`: Power limit in watts
  - `DETEC_TYP`: Detection type value
  - `DISCONNECT_TYP`: Longer detection value
- **Response**: "SUCCESS" or error message

**Models GS316EP/GS316EPP:**
- **Endpoint**: `POST http://{host}/iss/specific/poePortConf.html`
- **Headers**:
  - `Cookie: Gambit={session_token}`
  - `Content-Type: application/x-www-form-urlencoded`
- **Body Parameters**:
  - `Gambit`: Session token
  - `TYPE`: "poe"
  - `admin_state_{port}`: "enable" or "disable"
  - `port_priority_{port}`: Priority setting
  - `power_mode_{port}`: Power mode setting
  - `power_limit_type_{port}`: Limit type
  - `power_limit_{port}`: Power limit value
  - `detection_type_{port}`: Detection type
  - `longer_detection_{port}`: "enable" or "disable"
- **Response**: HTML page with updated settings

### 5. POE Cycle Power Command

Power cycles specified POE ports.

**Models GS305EP/GS305EPP/GS308EP/GS308EPP:**
- **Endpoint**: `POST http://{host}/PoEPortConfig.cgi`
- **Headers**:
  - `Cookie: SID={session_token}`
  - `Content-Type: application/x-www-form-urlencoded`
- **Body Parameters**:
  - `hash`: Security hash
  - `ACTION`: "Reset"
  - `port{n}`: "checked" for each port to cycle (n is 0-based)
- **Response**: "SUCCESS" or error message

**Models GS316EP/GS316EPP:**
- **Endpoint**: `POST http://{host}/iss/specific/poePortConf.html`
- **Headers**:
  - `Cookie: Gambit={session_token}`
  - `Content-Type: application/x-www-form-urlencoded`
- **Body Parameters**:
  - `Gambit`: Session token
  - `TYPE`: "resetPoe"
  - `PoePort`: Comma-separated list of port indices to cycle
- **Response**: Confirmation page

## Port Commands

### 6. Port Settings Command

Retrieves current port configuration and status.

**Models GS305EP/GS305EPP/GS308EP/GS308EPP:**
- **Endpoint**: `GET http://{host}/dashboard.cgi`
- **Headers**: `Cookie: SID={session_token}`
- **Response**: HTML containing `li.list_item` elements with port data

**Models GS316EP/GS316EPP:**
- **Endpoint**: `GET http://{host}/iss/specific/dashboard.html`
- **Headers**: `Cookie: Gambit={session_token}`
- **Response**: HTML containing `div.dashboard-port-status` elements

**Response Data Structure:**
- Port Index
- Port Name
- Speed Setting (Auto, 100M full, 100M half, 10M full, 10M half, Disable)
- Ingress Rate Limit
- Egress Rate Limit
- Flow Control (On/Off)
- Port Status (Up/Down)
- Link Speed (actual negotiated speed)

### 7. Port Set Command

Updates port configuration settings.

**Models GS305EP/GS305EPP/GS308EP/GS308EPP:**
- **Endpoint**: `POST http://{host}/port_status.cgi`
- **Headers**:
  - `Cookie: SID={session_token}`
  - `Content-Type: application/x-www-form-urlencoded`
- **Body Parameters**:
  - `hash`: Security hash
  - `port{n}`: "checked" for port to configure (n is port index)
  - `SPEED`: Speed setting value
  - `FLOW_CONTROL`: "0" (off) or "1" (on)
  - `DESCRIPTION`: Port name/description
  - `IngressRate`: Ingress rate limit value
  - `EgressRate`: Egress rate limit value
  - `priority`: "0"
- **Response**: "SUCCESS" or error message

**Models GS316EP/GS316EPP:**
- **Endpoint**: `POST http://{host}/iss/specific/dashboard.html`
- **Headers**:
  - `Cookie: Gambit={session_token}`
  - `Content-Type: application/x-www-form-urlencoded`
- **Body Parameters**:
  - `Gambit`: Session token
  - `TYPE`: "port"
  - `port_name_{port}`: Port name
  - `speed_{port}`: Speed setting
  - `ingress_rate_{port}`: Ingress limit
  - `egress_rate_{port}`: Egress limit
  - `flow_control_{port}`: "enable" or "disable"
- **Response**: Updated dashboard HTML

## Debug Report Command

### 8. Debug Report Command

Collects diagnostic information for troubleshooting.

The debug report command performs multiple GET requests to various endpoints to gather system information:

**Unauthenticated Endpoints (always checked):**
- `GET http://{host}/`
- `GET http://{host}/login.cgi`
- `GET http://{host}/wmi/login`
- `GET http://{host}/redirect.html`

**Authenticated Endpoints for GS305EP/GS305EPP/GS308EP/GS308EPP:**
- `GET http://{host}/getPoePortStatus.cgi`
- `GET http://{host}/PoEPortConfig.cgi`
- `GET http://{host}/port_status.cgi`
- `GET http://{host}/dashboard.cgi`

**Authenticated Endpoints for GS316EP/GS316EPP:**
- `GET http://{host}/iss/specific/poe.html`
- `GET http://{host}/iss/specific/poePortConf.html`
- `GET http://{host}/iss/specific/poePortStatus.html`
- `GET http://{host}/iss/specific/poePortStatus.html?GetData=TRUE`
- `GET http://{host}/iss/specific/getPortRate.html`
- `GET http://{host}/iss/specific/dashboard.html`
- `GET http://{host}/iss/specific/homepage.html`

## Error Handling

### Common Error Responses

1. **Session Expired**: HTML pages may contain login redirect indicators
2. **Invalid Parameters**: Plain text error messages
3. **Network Errors**: Connection timeouts or refused connections
4. **Model Detection Failed**: Error when switch model cannot be determined

### Session Management

- Sessions expire after a period of inactivity
- Expired sessions require re-login
- Token storage is handled locally by the client application

## Value Mappings

### Power Mode Values (GS30x models)
- "0": 802.3af
- "1": Legacy
- "2": Pre-802.3at
- "3": 802.3at

### Priority Values (GS30x models)
- "0": Low
- "1": High
- "2": Critical

### Limit Type Values (GS30x models)
- "0": None
- "1": Class
- "2": User

### Detection Type Values (GS30x models)
- "0": IEEE 802
- "1": Legacy
- "2": 4pt 802.3af + Legacy

### Port Speed Values (GS30x models)
- "0": Auto
- "1": 100M Full
- "2": 100M Half
- "3": 10M Full
- "4": 10M Half
- "5": Disable

## Notes

1. All endpoints require proper session authentication except for login and debug report endpoints
2. The GS316EP model uses different endpoint paths and parameter names compared to GS305EP/GS308EP
3. HTML parsing is required to extract data from responses
4. Rate limits are typically specified in Kbit/s or Mbit/s units
5. Port indices are 1-based in the UI but may be 0-based in API calls depending on the model