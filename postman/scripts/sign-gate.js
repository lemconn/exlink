// Gate signature script
// Usage: Include this script in Pre-request Script tab
// Requires: gate_api_key, gate_secret_key environment variables

const apiKey = (pm.environment.get('gate_api_key') || '').trim();
const secretKey = (pm.environment.get('gate_secret_key') || '').trim();

if (!apiKey || !secretKey) {
    console.error('Missing required environment variables: gate_api_key or gate_secret_key');
    throw new Error('Missing required environment variables');
}

const method = pm.request.method.toUpperCase();
let path = pm.request.url.getPath();
const timestamp = Math.floor(Date.now() / 1000);

// Remove /api/v4 prefix from path if exists (for signature)
let signPath = path;
if (path.startsWith('/api/v4')) {
    signPath = path.substring('/api/v4'.length);
}

// Build query string
const queryParams = {};
pm.request.url.query.all().forEach(param => {
    if (!param.disabled) {
        queryParams[param.key] = param.value;
    }
});
let queryString = '';
if (Object.keys(queryParams).length > 0) {
    queryString = Object.keys(queryParams)
        .sort()
        .map(key => `${key}=${encodeURIComponent(queryParams[key])}`)
        .join('&');
}

// Build request body string
let bodyStr = '';
if (pm.request.body && pm.request.body.raw) {
    bodyStr = pm.request.body.raw.trim();
}

// SHA512 hash of request body
const bodyHash = CryptoJS.SHA512(bodyStr).toString();

// Gate signature format: METHOD\n/api/v4/path\nqueryString\nbodyHash\ntimestamp
const signString = `${method}\n/api/v4${signPath}\n${queryString}\n${bodyHash}\n${timestamp}`;

// HMAC-SHA512 signature
const signature = CryptoJS.HmacSHA512(signString, secretKey).toString();

// Set request headers using setOrUpdateHeaderParam
setOrUpdateHeaderParam('KEY', apiKey);
setOrUpdateHeaderParam('Timestamp', timestamp.toString());
setOrUpdateHeaderParam('SIGN', signature);

function setOrUpdateHeaderParam(key, value) {
    let headers = pm.request.headers;
    let header = headers.find((header) => header.key === key);
    if (header) header.value = value;
    else pm.request.headers.add({ key, value });
}