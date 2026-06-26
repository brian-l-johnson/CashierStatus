import { encode } from "./encoder.mjs";
import { decode } from "./decoder.mjs";

const simple = {
    "cc": 123,
    "p": "A726",
    "i": [
        {
            "v": 456,
            "q": 2
        },
        {
            "v": 789,
            "q": 3
        }
    ]
}

const complex = {
    "cc": 12345,
    "p": "iOS",
    "txn": 99999,
    "metadata": {
        "timestamp": 1234567890,
        "version": "1.0",
        "strings": [
            "keep it simple"
        ]
    },
    "i": [
        {
            "v": 123,
            "q": 2
        },
        {
            "v": 456,
            "q": 3
        },
        {
            "v": 789,
            "q": 1
        },
        {
            "variant": "I don't exist (but I have interesting characters, like: !@#$%^&*)",
            "q": -1
        },
        {
            "''": "technically, a that's a totally valid field name",
            "q": "get that $$$"
        },
        {
            "urlencoded": "this%20is%20a%20%22safe%22%20string,%20right%3F"
        }
    ]
}

const testObjects = [simple, complex];

testObjects.forEach((obj) => {
    let encoded = encode(obj);
    console.log("Encoded:", encoded);
    try {
        let decoded = decode(encoded);
        console.log("Decoded:", JSON.stringify(decoded));
    } catch (e) {
        console.error("Decoding failed for:", encoded);
        console.error(e);
    }
})