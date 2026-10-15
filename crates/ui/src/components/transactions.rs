//! Transactions list UI with filters

use leptos::{prelude::*, task::spawn_local};
use crate::components::api::{self, TransactionListParams};
use shared::TransactionType;

/// Transaction display with formatted data
#[derive(Debug, Clone)]
pub struct TransactionDisplay {
    pub id: i64,
    pub transaction_type: TransactionType,
    pub amount: f64,
    pub currency: String,
    pub category_id: i64,
    pub description: Option<String>,
    pub label: Option<String>,
    pub date: String,
    pub created_at: String,
    pub updated_at: String,
}

/// Transactions list component with filters
#[component]
pub fn TransactionsList() -> impl IntoView {
    // State signals
    let transactions = RwSignal::new(Vec::<TransactionDisplay>::new());
    let loading = RwSignal::new(false);
    let error = RwSignal::new(Option::<String>::None);
    
    // Filter signals
    let filter_type = RwSignal::new(Option::<String>::None);
    let filter_category = RwSignal::new(Option::<i64>::None);
    let filter_currency = RwSignal::new(Option::<String>::None);
    let filter_date_from = RwSignal::new(Option::<String>::None);
    let filter_date_to = RwSignal::new(Option::<String>::None);
    let sort_by = RwSignal::new(String::from("date"));
    let sort_order = RwSignal::new(String::from("desc"));

    // Load transactions
    let load_transactions = {
        let transactions = transactions.clone();
        let loading = loading.clone();
        let error = error.clone();
        let filter_type = filter_type.clone();
        let filter_category = filter_category.clone();
        let filter_currency = filter_currency.clone();
        let filter_date_from = filter_date_from.clone();
        let filter_date_to = filter_date_to.clone();
        let sort_by = sort_by.clone();
        let sort_order = sort_order.clone();
        
        move || {
            loading.set(true);
            error.set(None);
            
            let params = TransactionListParams {
                transaction_type: filter_type.get(),
                category_id: filter_category.get(),
                currency: filter_currency.get(),
                date_from: filter_date_from.get(),
                date_to: filter_date_to.get(),
                sort_by: Some(sort_by.get()),
                sort_order: Some(sort_order.get()),
            };
            
            spawn_local(async move {
                match api::list_transactions(Some(params)).await {
                    Ok(txs) => {
                        let display_txs: Vec<TransactionDisplay> = txs.into_iter().map(|t| {
                            TransactionDisplay {
                                id: t.id,
                                transaction_type: t.transaction_type,
                                amount: t.amount,
                                currency: t.currency,
                                category_id: t.category_id,
                                description: t.description,
                                label: t.label,
                                date: t.date,
                                created_at: t.created_at,
                                updated_at: t.updated_at,
                            }
                        }).collect();
                        transactions.set(display_txs);
                        loading.set(false);
                    }
                    Err(e) => {
                        error.set(Some(e));
                        loading.set(false);
                    }
                }
            });
        }
    };

    // Initial load
    load_transactions();

    // Delete transaction handler
    let handle_delete = {
        let error = error.clone();
        let load_transactions = load_transactions.clone();
        
        move |id: i64| {
            spawn_local(async move {
                match api::delete_transaction(id).await {
                    Ok(_) => {
                        load_transactions();
                    }
                    Err(e) => {
                        error.set(Some(e));
                    }
                }
            });
        }
    };

    // Apply filters
    let apply_filters = {
        let load_transactions = load_transactions.clone();
        
        move || {
            load_transactions();
        }
    };

    // Reset filters
    let reset_filters = {
        let filter_type = filter_type.clone();
        let filter_category = filter_category.clone();
        let filter_currency = filter_currency.clone();
        let filter_date_from = filter_date_from.clone();
        let filter_date_to = filter_date_to.clone();
        let sort_by = sort_by.clone();
        let sort_order = sort_order.clone();
        let load_transactions = load_transactions.clone();
        
        move || {
            filter_type.set(None);
            filter_category.set(None);
            filter_currency.set(None);
            filter_date_from.set(None);
            filter_date_to.set(None);
            sort_by.set(String::from("date"));
            sort_order.set(String::from("desc"));
            load_transactions();
        }
    };

    // Format transaction type for display
    let format_type = |t: &TransactionType| -> &'static str {
        match t {
            TransactionType::Income => "Income",
            TransactionType::Outcome => "Outcome",
        }
    };

    // Format amount with currency
    let format_amount = |amount: f64, currency: &str| -> String {
        let symbol = match currency {
            "COP" => "$",
            "USD" => "$",
            "EUR" => "€",
            "GBP" => "£",
            _ => "",
        };
        format!("{} {:.2}", symbol, amount)
    };

    view! {
        <div class="transactions-list">
            <header>
                <h2>"Transactions"</h2>
                <a href="/transactions/new" class="btn-primary">"New Transaction"</a>
            </header>

            // Error display
            <Show when={move || error.get().is_some()}>
                <div class="error-message">
                    <span>{move || error.get().unwrap_or_default()}</span>
                    <button class="btn-close" on:click={move |_| error.set(None)}>"×"</button>
                </div>
            </Show>

            // Filters
            <div class="filters">
                <div class="filter-group">
                    <label>"Type"</label>
                    <select
                        on:change={move |e| {
                            let val = event_target_value(&e);
                            filter_type.set(if val.is_empty() { None } else { Some(val) });
                        }}
                    >
                        <option value="">"All"</option>
                        <option value="income">"Income"</option>
                        <option value="outcome">"Outcome"</option>
                    </select>
                </div>
                <div class="filter-group">
                    <label>"Currency"</label>
                    <select
                        on:change={move |e| {
                            let val = event_target_value(&e);
                            filter_currency.set(if val.is_empty() { None } else { Some(val) });
                        }}
                    >
                        <option value="">"All"</option>
                        <option value="COP">"COP"</option>
                        <option value="USD">"USD"</option>
                        <option value="EUR">"EUR"</option>
                    </select>
                </div>
                <div class="filter-group">
                    <label>"From"</label>
                    <input
                        type="date"
                        on:change={move |e| {
                            let val = event_target_value(&e);
                            filter_date_from.set(if val.is_empty() { None } else { Some(val) });
                        }}
                    />
                </div>
                <div class="filter-group">
                    <label>"To"</label>
                    <input
                        type="date"
                        on:change={move |e| {
                            let val = event_target_value(&e);
                            filter_date_to.set(if val.is_empty() { None } else { Some(val) });
                        }}
                    />
                </div>
                <div class="filter-group">
                    <label>"Sort By"</label>
                    <select
                        on:change={move |e| {
                            sort_by.set(event_target_value(&e));
                        }}
                    >
                        <option value="date">"Date"</option>
                        <option value="amount">"Amount"</option>
                    </select>
                </div>
                <div class="filter-group">
                    <label>"Order"</label>
                    <select
                        on:change={move |e| {
                            sort_order.set(event_target_value(&e));
                        }}
                    >
                        <option value="desc">"Descending"</option>
                        <option value="asc">"Ascending"</option>
                    </select>
                </div>
                <div class="filter-actions">
                    <button class="btn-primary" on:click={move |_| apply_filters()}>"Filter"</button>
                    <button class="btn-secondary" on:click={move |_| reset_filters()}>"Reset"</button>
                </div>
            </div>

            // Transactions table
            <Show when={move || loading.get()}>
                <p class="loading">"Loading transactions..."</p>
            </Show>

            <Show when={move || !loading.get() && transactions.get().is_empty()}>
                <p class="empty">"No transactions found. Create your first one above."</p>
            </Show>

            <Show when={move || !loading.get() && !transactions.get().is_empty()}>
                <table class="transactions-table">
                    <thead>
                        <tr>
                            <th>"Date"</th>
                            <th>"Type"</th>
                            <th>"Amount"</th>
                            <th>"Description"</th>
                            <th>"Label"</th>
                            <th>"Actions"</th>
                        </tr>
                    </thead>
                    <tbody>
                        {move || transactions.get().into_iter().map(|tx| {
                            let tx_id = tx.id;
                            let type_str = format_type(&tx.transaction_type).to_string();
                            let amount_str = format_amount(tx.amount, &tx.currency);
                            let desc = tx.description.clone().unwrap_or_default();
                            let label = tx.label.clone().unwrap_or_default();
                            let date = tx.date.clone();
                            let is_income = matches!(tx.transaction_type, TransactionType::Income);
                            
                            view! {
                                <tr class:income={is_income} class:outcome={!is_income}>
                                    <td>{date}</td>
                                    <td><span class={"type-badge ".to_string() + if is_income { "income" } else { "outcome" }}>{type_str}</span></td>
                                    <td class="amount">{amount_str}</td>
                                    <td>{desc}</td>
                                    <td>{label}</td>
                                    <td class="actions">
                                        <a href={format!("/transactions/{}", tx_id)} class="btn-secondary btn-sm">"Edit"</a>
                                        <button class="btn-danger btn-sm" on:click={move |_| handle_delete(tx_id)}>"Delete"</button>
                                    </td>
                                </tr>
                            }
                        }).collect_view()}
                    </tbody>
                </table>
            </Show>
        </div>
    }
}
