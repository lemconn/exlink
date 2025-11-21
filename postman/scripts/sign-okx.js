// OKX signature script
// Usage: Include this script in Pre-request Script tab
// Requires: okx_api_key, okx_secret_key, okx_passphrase environment variables
// 
// Signature format: timestamp + method + requestPath + (queryString for GET or body for POST)
// For GET requests: if query params exist, add "?" + queryString
// For POST/PUT/DELETE requests: if body exists, add body

const apiKey = (pm.environment.get('okx_api_key') || '').trim();
const secretKey = (pm.environment.get('okx_secret_key') || '').trim();
const passphrase = (pm.environment.get('okx_passphrase') || '').trim();

if (!apiKey || !secretKey || !passphrase) {
    console.error('Missing required environment variables: okx_api_key, okx_secret_key, or okx_passphrase');
    throw new Error('Missing required environment variables');
}

const method = pm.request.method.toUpperCase();
const path = pm.request.url.getPath();

// Get ISO8601 format timestamp (UTC)
const now = new Date();
const timestamp = now.toISOString();

// Build signature message: timestamp + method + path
let message = timestamp + method + path;

// For GET requests, add query string if exists
if (method === 'GET') {
    const queryParams = {};
    pm.request.url.query.all().forEach(param => {
        if (!param.disabled) {
            queryParams[param.key] = param.value;
        }
    });
    
    if (Object.keys(queryParams).length > 0) {
        const queryParts = Object.keys(queryParams).map(key => 
            `${key}=${encodeURIComponent(queryParams[key])}`
        );
        const queryString = queryParts.join('&');
        message += '?' + queryString;
    }
} else {
    // For POST/PUT/DELETE requests, add body if exists
    if (pm.request.body && pm.request.body.raw) {
        const bodyStr = pm.request.body.raw;
        // Remove any leading/trailing whitespace
        const trimmedBody = bodyStr.trim();
        if (trimmedBody) {
            message += trimmedBody;
        }
    }
}

// HMAC-SHA256 signature and Base64 encode
const signature = CryptoJS.enc.Base64.stringify(CryptoJS.HmacSHA256(message, secretKey));

// Set request headers using setOrUpdateHeaderParam
setOrUpdateHeaderParam('OK-ACCESS-KEY', apiKey);
setOrUpdateHeaderParam('OK-ACCESS-SIGN', signature);
setOrUpdateHeaderParam('OK-ACCESS-TIMESTAMP', timestamp);
setOrUpdateHeaderParam('OK-ACCESS-PASSPHRASE', passphrase);

// Add simulated trading header if in testnet environment
if (pm.environment.get('okx_testnet') === 'true') {
    setOrUpdateHeaderParam('x-simulated-trading', '1');
} else {
    setOrUpdateHeaderParam('x-simulated-trading', '0');
}

function setOrUpdateHeaderParam(key, value) {
    let headers = pm.request.headers;
    let header = headers.find((header) => header.key === key);
    if (header) header.value = value;
    else pm.request.headers.add({ key, value });
}
