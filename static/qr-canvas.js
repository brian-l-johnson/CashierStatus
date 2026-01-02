/**
 * Modern Canvas-based QR Code Generator
 * Lightweight replacement for old qrcode.js
 * Uses qrcode-generator library (Kazuhiko Arase, MIT License)
 * https://github.com/kazuhikoarase/qrcode-generator
 */
(function(global) {
    'use strict';

    // Check if qrcode-generator library is loaded
    if (typeof qrcode === 'undefined') {
        throw new Error('qrcode-generator library not loaded. Include qrcode-generator.min.js before qr-canvas.js');
    }

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

        // Parse options
        if (typeof options === 'string') {
            options = { text: options };
        }

        this.options = {
            text: options.text || '',
            width: options.width || 256,
            height: options.height || 256,
            colorDark: options.colorDark || '#000000',
            colorLight: options.colorLight || '#ffffff',
            correctLevel: options.correctLevel || QRCode.CorrectLevel.M
        };

        // Generate and render if text provided
        if (this.options.text) {
            this.makeCode(this.options.text);
        }
    }

    /**
     * Error correction levels (compatible with qrcode-generator library)
     */
    QRCode.CorrectLevel = {
        L: qrcode.ErrorCorrectLevel.L,  // ~7% correction
        M: qrcode.ErrorCorrectLevel.M,  // ~15% correction
        Q: qrcode.ErrorCorrectLevel.Q,  // ~25% correction
        H: qrcode.ErrorCorrectLevel.H   // ~30% correction
    };

    /**
     * Generate QR code from text
     * @param {string} text - Text to encode
     */
    QRCode.prototype.makeCode = function(text) {
        this.options.text = text;

        // Determine optimal type number (QR code version) based on text length
        var typeNumber = this._getTypeNumber(text, this.options.correctLevel);

        // Create QR code using qrcode-generator library
        var qr = qrcode(typeNumber, this.options.correctLevel);
        qr.addData(text);
        qr.make();

        // Render to canvas
        this._renderCanvas(qr);
    };

    /**
     * Determine QR code type number (version) based on text length and error correction level
     * @private
     */
    QRCode.prototype._getTypeNumber = function(text, errorCorrectLevel) {
        // Simple heuristic - qrcode-generator library will auto-adjust if needed
        var length = text.length;

        if (length <= 20) return 0;   // Auto-detect for short text
        if (length <= 50) return 5;
        if (length <= 100) return 10;
        if (length <= 200) return 15;
        return 20;
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
