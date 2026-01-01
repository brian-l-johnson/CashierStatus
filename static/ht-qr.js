const ESCAPE_CHAR = `%`;
const STRUCT_CHARS = [
    '{',
    '}',
    '[',
    ']',
    ':', // this is in the alphanumeric char set, but included for completeness
    ',',
    '"'
];

const COMPACT_RE = /^\d+:\d+:[A-Z0-9]+:\d+:\d+(;\d+:\d+)*(?::\d+)?$/;
const ESCAPE_RE = RegExp(`${ESCAPE_CHAR}[0-6]`);
const STRUCT_RE = RegExp(`[\\${STRUCT_CHARS.join("//")}]`);

const entries = Array.from(STRUCT_CHARS, (char, i) => [char, `${ESCAPE_CHAR}${i}`]);
const escapeMap = Object.fromEntries(entries);
const unescapeMap = Object.fromEntries(entries.map(([k, v]) => [v, k]));

const escapeJson = (str) => str.replace(new RegExp(STRUCT_RE, 'g'), ch => escapeMap[ch]);
const isEscapable = (str) => /^[0-9A-Z $%*+\-./:]*$/.test(escapeJson(str));

const unescapeJson = (str) => str.replace(new RegExp(ESCAPE_RE, 'g'), seq => unescapeMap[seq]);;
const isCompact = (str) => COMPACT_RE.test(str);
const isEscaped = (str) => ESCAPE_RE.test(str);

function encodeCompact(order) {
    console.log("in encodeCompact");
    const version = 1;
    const { cc, p, i, txn } = order;
    if (!Array.isArray(i)) throw new Error("Invalid order structure");
    const items = i.map(({ v, q }) => `${v}:${q}`).join(';');
    
    if (txn) {
        compact = `${version}:${cc}:${p}:${items}:${txn}`;    
    }
    else {
        compact = `${version}:${cc}:${p}:${items}`;
    }
        
    return compact;
}

// Encoder: automatic tier selection
function encode(input) {
    console.log(input)
    let parsed = input;
    if (typeof input === 'string') {
        console.log("is a string");
        try { parsed = JSON.parse(input); } catch (e) { /* malformed input */ }
    }

    try {
        console.log("here");
        console.log("typeof data: "+typeof(parsed))
        console.log("data is not null: "+parsed!==null)
        console.log("typeof cc: "+typeof(parsed.cc));
        if (typeof parsed === 'object' && parsed !== null &&
            typeof parsed.cc === 'number' &&
            typeof parsed.p === 'string' &&
            Array.isArray(parsed.i) &&
            parsed.i.every(e => typeof e.v === 'number' && typeof e.q === 'number')) {
            console.log("tier 1");
            return encodeCompact(parsed); // Tier 1
        }

        const minified = JSON.stringify(parsed);
        const escaped = escapeJson(minified);
        if (isEscapable(escaped)) {
            console.log("tier 2")
            return escaped; // Tier 2
        }
        console.log("tier 3");
        return minified; // Tier 3
    } catch (e) {
        return typeof input === 'string' ? input : JSON.stringify(input); // Tier 3 fallback
    }
}



