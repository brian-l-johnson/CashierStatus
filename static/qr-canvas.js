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
            // Use 0 (M level) as default since QRCode.CorrectLevel isn't defined yet
            correctLevel: (typeof options.correctLevel !== 'undefined') ? options.correctLevel : 0
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
        // Use 0 for auto-detect - qrcode-generator will choose optimal version
        // This is more reliable than trying to estimate
        return 0;
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
