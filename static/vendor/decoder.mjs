import { isCompact, isEscaped, unescapeJson } from "./util.mjs";

function decodeCompact(str) {
    const [version, ...parts] = str.split(':');
    let order = {};
    let txn = '';
    switch (version) {
        case '1':
            const [cc, p, ...rest] = parts;
            // Find the last semicolon, which separates items from optional txn
            const lastSemicolon = rest.join(':').lastIndexOf(';');
            let itemsPart = rest.join(':');
            let txnPart = '';
            if (lastSemicolon !== -1 && lastSemicolon < itemsPart.length - 1) {
                // There is something after the last semicolon, treat as txn
                txnPart = itemsPart.slice(lastSemicolon + 1);
                itemsPart = itemsPart.slice(0, lastSemicolon);
            }
            const items = itemsPart.split(';').map(pair => {
                const [v, q] = pair.split(':');
                return { v, q };
            });
            order = { cc, p, i: items };
            if (txnPart) {
                order.txn = txnPart;
            }
            break;
        default:
            throw new Error("Unsupported version");
    }

    return order;
}

// Decoder: auto-detect format
export function decode(str) {
    // Try compact first
    if (isCompact(str)) {
        try { return decodeCompact(str); } catch (_) { }
    }

    // Try unescaping JSON
    if (isEscaped(str)) {
        try { return JSON.parse(unescapeJson(str)); } catch (_) { }
    }

    // Try raw JSON
    try { return JSON.parse(str); } catch (_) { }

    throw new Error("Unable to decode input");
}