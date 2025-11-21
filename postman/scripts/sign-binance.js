// Binance signature script
// Usage: Include this script in Pre-request Script tab
// Requires: binance_api_key, binance_secret_key environment variables
const apiKey = (pm.environment.get('binance_api_key') || '').trim();
const apiSecret = (pm.environment.get('binance_secret_key') || '').trim();
if (!apiKey || !apiSecret) {
    console.error('Missing required environment variables: binance_api_key or binance_secret_key');
    throw new Error('Missing required environment variables');
}

const ts = Date.now();
let paramsObject = {};

const parameters = pm.request.url.query;
parameters.each((param) => {
    if (param.key !== 'signature' && !is_empty(param.value) && !is_disabled(param.disabled)) {
        paramsObject[param.key] = param.value;
    }
});

Object.assign(paramsObject, { timestamp: ts });

setOrUpdateHeaderParam('X-MBX-APIKEY', apiKey);

if (parameters.has('signature') && apiSecret) {
    const queryString = buildQueryString(paramsObject);
    const signature = CryptoJS.HmacSHA256(queryString, apiSecret).toString(CryptoJS.enc.Hex);
    pm.request.url.query.clear();
    pm.request.url.query = `${queryString}&signature=${signature}`;
}

function buildQueryString(params) {
    if (!params) return '';
    const pairs = [];
    Object.keys(params).sort().forEach((key) => {
        const value = params[key];
        if (value !== null && value !== undefined) {
            const serializedValue = serializeValue(value);
            pairs.push(`${key}=${encodeURIComponent(serializedValue)}`);
        }
    });
    return pairs.join('&');
}

function serializeValue(value) {
    if (value === null || value === undefined) return '';
    if (Array.isArray(value) || (typeof value === 'object' && value !== null)) return JSON.stringify(value);
    return String(value);
}

function setOrUpdateHeaderParam(key, value) {
    let headers = pm.request.headers;
    let header = headers.find((header) => header.key === key);
    if (header) header.value = value;
    else pm.request.headers.add({ key, value });
}

function is_disabled(str) {
    return str === true;
}

function is_empty(str) {
    return (
        typeof str === 'undefined' ||
        !str ||
        str.length === 0 ||
        str === '' ||
        !/[^\s]/.test(str) ||
        /^\s*$/.test(str) ||
        str.replace(/\s/g, '') === ''
    );
}