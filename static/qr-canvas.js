/**
 * Modern Canvas-based QR Code Generator
 * Lightweight replacement for old qrcode.js
 * Uses qrcode-generator library (Kazuhiko Arase, MIT License)
 * https://github.com/kazuhikoarase/qrcode-generator
 */
(function(global) {
    'use strict';

    /**
     * QRCode constructor - Compatible with old qrcode.js API
     * @param {HTMLElement|string} element - Target element or element ID
     * @param {string|Object} options - QR code text or options object
     */
    function QRCode(element, options) {
        this.element = typeof element === 'string' ?
            document.getElementById(element) : element;

        if (!this.element) {
            throw new Error('QRCode: Invalid element');
        }

        // Parse options - handle undefined/null
        if (!options) {
            options = {};
        } else if (typeof options === 'string') {
            options = { text: options };
        }

        this.options = {
            text: options.text || '',
            width: options.width || 256,
            height: options.height || 256,
            colorDark: options.colorDark || '#000000',
            colorLight: options.colorLight || '#ffffff',
            // Default to 0 (error correction level M)
            correctLevel: (options.correctLevel !== undefined && options.correctLevel !== null)
                ? options.correctLevel
                : 0
        };

        // Generate and render if text provided
        if (this.options.text) {
            this.makeCode(this.options.text);
        }
    }

    /**
     * Error correction levels (matches qrcode-generator library values)
     */
    QRCode.CorrectLevel = {
        L: 1,  // ~7% correction
        M: 0,  // ~15% correction (default)
        Q: 3,  // ~25% correction
        H: 2   // ~30% correction
    };

    /**
     * Generate QR code from text
     * @param {string} text - Text to encode
     */
    QRCode.prototype.makeCode = function(text) {
        // Check if qrcode-generator library is loaded
        if (typeof qrcode === 'undefined') {
            throw new Error('qrcode-generator library not loaded. Include qrcode-generator.min.js before qr-canvas.js');
        }

        this.options.text = text;

        // Determine optimal type number (QR code version) based on text length
        var typeNumber = this._getTypeNumber(text);

        // Get error correction level (should always be set in constructor)
        var errorCorrectionLevel = this.options.correctLevel;

        // Safety check - ensure it's never undefined
        if (errorCorrectionLevel === undefined || errorCorrectionLevel === null) {
            console.error('ERROR: errorCorrectionLevel is undefined, using 0');
            errorCorrectionLevel = 0;
        }

        console.log('QRCode makeCode - typeNumber:', typeNumber, 'errorCorrectionLevel:', errorCorrectionLevel);

        // Create QR code using qrcode-generator library
        // The library expects numeric values: L=1, M=0, Q=3, H=2
        var qr = qrcode(typeNumber, errorCorrectionLevel);
        qr.addData(text);
        qr.make();

        // Render to canvas
        this._renderCanvas(qr);
    };

    /**
     * Determine QR code type number (version) based on text length
     * @private
     */
    QRCode.prototype._getTypeNumber = function(text) {
        // Calculate appropriate type number based on data length
        // Auto-detect (0) has issues with errorCorrectionLevel, so we calculate it
        var length = text.length;

        // These are approximate - for error correction level M
        if (length <= 14) return 1;
        if (length <= 26) return 2;
        if (length <= 42) return 3;
        if (length <= 62) return 4;
        if (length <= 84) return 5;
        if (length <= 106) return 6;
        if (length <= 122) return 7;
        if (length <= 152) return 8;
        if (length <= 180) return 9;
        if (length <= 213) return 10;
        if (length <= 251) return 11;
        if (length <= 287) return 12;
        if (length <= 331) return 13;
        if (length <= 362) return 14;
        if (length <= 412) return 15;
        if (length <= 450) return 16;
        if (length <= 504) return 17;
        if (length <= 560) return 18;
        if (length <= 624) return 19;
        if (length <= 666) return 20;

        // For very long data, use higher versions
        return Math.min(40, Math.ceil(length / 40));
    };

    /**
     * Render QR code to Canvas
     * @private
     */
    QRCode.prototype._renderCanvas = function(qr) {
        var moduleCount = qr.getModuleCount();
        var cellSize = Math.floor(this.options.width / moduleCount);
        var canvasSize = cellSize * moduleCount;

        // Create canvas
        var canvas = document.createElement('canvas');
        canvas.width = canvasSize;
        canvas.height = canvasSize;

        var ctx = canvas.getContext('2d');

        // Draw background
        ctx.fillStyle = this.options.colorLight;
        ctx.fillRect(0, 0, canvasSize, canvasSize);

        // Draw QR code modules
        ctx.fillStyle = this.options.colorDark;

        for (var row = 0; row < moduleCount; row++) {
            for (var col = 0; col < moduleCount; col++) {
                if (qr.isDark(row, col)) {
                    ctx.fillRect(
                        col * cellSize,
                        row * cellSize,
                        cellSize,
                        cellSize
                    );
                }
            }
        }

        // Clear element and append canvas
        this.element.innerHTML = '';
        this.element.appendChild(canvas);
    };

    /**
     * Clear the QR code
     */
    QRCode.prototype.clear = function() {
        this.element.innerHTML = '';
    };

    // Export to global scope
    global.QRCode = QRCode;

})(window);
