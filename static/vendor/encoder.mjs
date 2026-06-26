import { escapeJson, isEscapable } from "./util.mjs";

function encodeCompact(order) {
    const version = 1;
    const { cc, p, i, txn } = order;
    if (!Array.isArray(i)) throw new Error("Invalid order structure");
    const items = i.map(({ v, q }) => `${v}:${q}`).join(';');
    const compact = `${version}:${cc}:${p}:${items}`;
    if (txn) compact += `:${txn}`;
    return compact;
}

// Encoder: automatic tier selection
export function encode(input) {
    let parsed = input;
    if (typeof input === 'string') {
        try { parsed = JSON.parse(input); } catch (e) { /* malformed input */ }
    }

    try {
        if (typeof parsed === 'object' && parsed !== null &&
            typeof parsed.cc === 'number' &&
            typeof parsed.p === 'string' &&
            Array.isArray(parsed.i) &&
            parsed.i.every(e => typeof e.v === 'number' && typeof e.q === 'number')) {
            return encodeCompact(parsed); // Tier 1
        }

        const minified = JSON.stringify(parsed);
        const escaped = escapeJson(minified);
        if (isEscapable(escaped)) return escaped; // Tier 2

        return minified; // Tier 3
    } catch (e) {
        return typeof input === 'string' ? input : JSON.stringify(input); // Tier 3 fallback
    }
}