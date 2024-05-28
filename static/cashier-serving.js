class CashierServing extends HTMLElement {
    connectedCallback() {
        const station = "Cashier 1"
        //this.textContent = station
    }
}

customElements.define('cashier-serving', CashierServing)