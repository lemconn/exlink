// OKX signature script
// Usage: Include this script in Pre-request Script tab
// Requires: okx_api_key, okx_secret_key, okx_passphrase environment variables

const method = pm.request.method;
const path = pm.request.url.getPath();

// Get ISO8601 format timestamp
const now = new Date();
const timestamp = now.toISOString();

// Build query string
const queryParams = pm.request.url.query.toObject();
let queryString = '';
if (Object.keys(queryParams).length > 0) {
    queryString = '?' + Object.keys(queryParams)
        .sort()
        .map(key => `${key}=${encodeURIComponent(queryParams[key])}`)
        .join('&');
}

// Build request body string
let bodyStr = '';
if (pm.request.body && pm.request.body.raw) {
    bodyStr = pm.request.body.raw;
}

// Build signature message: timestamp + method + path + queryString + body
const message = timestamp + method + path + queryString + bodyStr;

// HMAC-SHA256 signature and Base64 encode
const signature = CryptoJS.enc.Base64.stringify(CryptoJS.HmacSHA256(message, pm.environment.get('okx_secret_key')));

// Set request headers
pm.request.headers.add({
    key: 'OK-ACCESS-KEY',
    value: pm.environment.get('okx_api_key')
});
pm.request.headers.add({
    key: 'OK-ACCESS-SIGN',
    value: signature
});
pm.request.headers.add({
    key: 'OK-ACCESS-TIMESTAMP',
    value: timestamp
});
pm.request.headers.add({
    key: 'OK-ACCESS-PASSPHRASE',
    value: pm.environment.get('okx_passphrase')
});

// Add simulated trading header if in testnet environment
if (pm.environment.get('okx_testnet') === 'true') {
    pm.request.headers.add({
        key: 'x-simulated-trading',
        value: '1'
    });
}

