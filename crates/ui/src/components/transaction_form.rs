//! Transaction form UI (create/edit)
//!
//! TODO: Implement form with Leptos reactivity

use leptos::prelude::*;

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum TransactionType {
    Income,
    Outcome,
}

#[component]
pub fn TransactionForm() -> impl IntoView {
    view! {
        <div class="transaction-form">
            <header>
                <h2>"Transaction"</h2>
            </header>

            <form>
                <div class="form-group">
                    <label>"Type"</label>
                    <select>
                        <option value="income">"Income"</option>
                        <option value="outcome">"Outcome"</option>
                    </select>
                </div>

                <div class="form-group">
                    <label>"Amount"</label>
                    <input type="number" step="0.01" placeholder="0.00" />
                </div>

                <div class="form-group">
                    <label>"Currency"</label>
                    <select>
                        <option value="COP">"COP - Colombian Peso"</option>
                        <option value="USD">"USD - US Dollar"</option>
                        <option value="EUR">"EUR - Euro"</option>
                    </select>
                </div>

                <div class="form-group">
                    <label>"Category"</label>
                    <select>
                        <option value="">"Select a category"</option>
                    </select>
                </div>

                <div class="form-group">
                    <label>"Description"</label>
                    <input type="text" placeholder="Optional description" />
                </div>

                <div class="form-group">
                    <label>"Label / Counterparty"</label>
                    <input type="text" placeholder="e.g., Grocery store, Salary" />
                </div>

                <div class="form-group">
                    <label>"Date"</label>
                    <input type="date" />
                </div>

                <div class="form-actions">
                    <a href="/transactions" class="btn-secondary">"Cancel"</a>
                    <button type="submit" class="btn-primary">"Save"</button>
                </div>
            </form>
        </div>
    }
}
