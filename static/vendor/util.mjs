// Alphanumeric char set
// ' '
// '$'
// '%'
// '*'
// '+'
// '-'
// '.'
// '/'
// ':'

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

export const escapeJson = (str) => str.replace(new RegExp(STRUCT_RE, 'g'), ch => escapeMap[ch]);
export const isEscapable = (str) => /^[0-9A-Z $%*+\-./:]*$/.test(escapeJson(str));

export const unescapeJson = (str) => str.replace(new RegExp(ESCAPE_RE, 'g'), seq => unescapeMap[seq]);;
export const isCompact = (str) => COMPACT_RE.test(str);
export const isEscaped = (str) => ESCAPE_RE.test(str);