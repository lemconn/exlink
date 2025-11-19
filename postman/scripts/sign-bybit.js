// Bybit signature script (v5 API)
// Usage: Include this script in Pre-request Script tab
// Requires: bybit_api_key, bybit_secret_key environment variables

const timestamp = Date.now().toString();
const recvWindow = '5000';

// Build query string
const queryParams = pm.request.url.query.toObject();
let queryString = '';
if (Object.keys(queryParams).length > 0) {
    queryString = Object.keys(queryParams)
        .sort()
        .map(key => `${key}=${queryParams[key]}`)
        .join('&');
}

// For POST requests, use request body
let bodyStr = '';
if (pm.request.method === 'POST' && pm.request.body && pm.request.body.raw) {
    bodyStr = pm.request.body.raw;
}

// Build signature base string: timestamp + apiKey + recvWindow + (queryString or bodyStr)
const authBase = timestamp + pm.environment.get('bybit_api_key') + recvWindow;
const authFull = authBase + (pm.request.method === 'POST' ? bodyStr : queryString);

// HMAC-SHA256 signature
const signature = CryptoJS.HmacSHA256(authFull, pm.environment.get('bybit_secret_key')).toString();

// Set request headers
pm.request.headers.add({
    key: 'X-BAPI-API-KEY',
    value: pm.environment.get('bybit_api_key')
});
pm.request.headers.add({
    key: 'X-BAPI-TIMESTAMP',
    value: timestamp
});
pm.request.headers.add({
    key: 'X-BAPI-RECV-WINDOW',
    value: recvWindow
});
pm.request.headers.add({
    key: 'X-BAPI-SIGN',
    value: signature
});

