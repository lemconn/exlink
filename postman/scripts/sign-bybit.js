// Bybit signature script (v5 API)
// Usage: Include this script in Pre-request Script tab
// Requires: bybit_api_key, bybit_secret_key environment variables
//
// Signature format: timestamp + apiKey + recvWindow + (body for POST or queryString for GET)
// For GET requests: use queryString (URL encoded)
// For POST requests: use JSON body string

const apiKey = (pm.environment.get('bybit_api_key') || '').trim();
const secretKey = (pm.environment.get('bybit_secret_key') || '').trim();

if (!apiKey || !secretKey) {
    console.error('Missing required environment variables: bybit_api_key or bybit_secret_key');
    throw new Error('Missing required environment variables');
}

const method = pm.request.method.toUpperCase();
const timestamp = Date.now().toString();
const recvWindow = '5000';

// Build query string for GET requests
let queryString = '';
if (method === 'GET' || method === 'DELETE') {
    const queryParams = {};
    pm.request.url.query.all().forEach(param => {
        if (!param.disabled) {
            queryParams[param.key] = param.value;
        }
    });
    
    if (Object.keys(queryParams).length > 0) {
        // Sort keys and build query string with URL encoding (equivalent to url.QueryEscape in Go)
        const sortedKeys = Object.keys(queryParams).sort();
        const queryParts = sortedKeys.map(key => {
            const value = queryParams[key];
            // URL encode the value (equivalent to url.QueryEscape in Go)
            return `${key}=${encodeURIComponent(String(value))}`;
        });
        queryString = queryParts.join('&');
    }
}

// Build request body string for POST/PUT requests
let bodyStr = '';
if ((method === 'POST' || method === 'PUT') && pm.request.body && pm.request.body.raw) {
    const rawBody = pm.request.body.raw;
    // Remove any leading/trailing whitespace
    bodyStr = rawBody.trim();
}

// Build signature base string: timestamp + apiKey + recvWindow
const authBase = timestamp + apiKey + recvWindow;

// Build full signature string
let authFull;
if (method === 'POST' || method === 'PUT') {
    authFull = authBase + bodyStr;
} else {
    authFull = authBase + queryString;
}

// HMAC-SHA256 signature (hex encoded, equivalent to SignHMAC256 in Go)
const signature = CryptoJS.HmacSHA256(authFull, secretKey).toString(CryptoJS.enc.Hex);

// Set request headers using setOrUpdateHeaderParam
setOrUpdateHeaderParam('X-BAPI-API-KEY', apiKey);
setOrUpdateHeaderParam('X-BAPI-TIMESTAMP', timestamp);
setOrUpdateHeaderParam('X-BAPI-RECV-WINDOW', recvWindow);
setOrUpdateHeaderParam('X-BAPI-SIGN', signature);
setOrUpdateHeaderParam('Content-Type', 'application/json');

function setOrUpdateHeaderParam(key, value) {
    let headers = pm.request.headers;
    let header = headers.find((header) => header.key === key);
    if (header) header.value = value;
    else pm.request.headers.add({ key, value });
}