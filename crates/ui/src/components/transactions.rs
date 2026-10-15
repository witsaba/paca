//! Transactions list UI with filters
//!
//! TODO: Implement filters with Leptos reactivity

use leptos::prelude::*;

#[component]
pub fn TransactionsList() -> impl IntoView {
    view! {
        <div class="transactions-list">
            <header>
                <h2>"Transactions"</h2>
                <a href="/transactions/new" class="btn-primary">"New Transaction"</a>
            </header>

            <div class="filters">
                <div class="filter-group">
                    <label>"Type"</label>
                    <select>
                        <option value="">"All"</option>
                        <option value="income">"Income"</option>
                        <option value="outcome">"Outcome"</option>
                    </select>
                </div>
                <div class="filter-group">
                    <label>"Currency"</label>
                    <select>
                        <option value="">"All"</option>
                        <option value="COP">"COP"</option>
                        <option value="USD">"USD"</option>
                        <option value="EUR">"EUR"</option>
                    </select>
                </div>
                <div class="filter-group">
                    <label>"From"</label>
                    <input type="date" />
                </div>
                <div class="filter-group">
                    <label>"To"</label>
                    <input type="date" />
                </div>
                <div class="filter-group">
                    <label>&nbsp;</label>
                    <button class="btn-secondary">"Filter"</button>
                </div>
            </div>

            <table class="transactions-table">
                <thead>
                    <tr>
                        <th>"Date"</th>
                        <th>"Type"</th>
                        <th>"Amount"</th>
                        <th>"Description"</th>
                        <th>"Actions"</th>
                    </tr>
                </thead>
                <tbody>
                    <tr>
                        <td colspan="5">"Loading transactions..."</td>
                    </tr>
                </tbody>
            </table>
        </div>
    }
}
