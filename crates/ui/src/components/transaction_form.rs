//! Transaction form UI (create/edit)

use leptos::{prelude::*, task::spawn_local};
use crate::components::api::{self, CreateTransaction};
use shared::{TransactionType, CategoryWithPath};

/// Transaction form component for creating/editing transactions
#[component]
pub fn TransactionForm() -> impl IntoView {
    // Form state
    let transaction_type = RwSignal::new(TransactionType::Outcome);
    let amount = RwSignal::new(String::new());
    let currency = RwSignal::new(String::from("COP"));
    let category_id = RwSignal::new(Option::<i64>::None);
    let description = RwSignal::new(String::new());
    let label = RwSignal::new(String::new());
    let date = RwSignal::new(String::new());
    
    // Categories for dropdown
    let categories = RwSignal::new(Vec::<CategoryWithPath>::new());
    let loading_categories = RwSignal::new(true);
    
    // Form state
    let submitting = RwSignal::new(false);
    let error = RwSignal::new(Option::<String>::None);
    let success = RwSignal::new(false);
    
    // Validation errors
    let amount_error = RwSignal::new(Option::<String>::None);
    let category_error = RwSignal::new(Option::<String>::None);
    let date_error = RwSignal::new(Option::<String>::None);

    // Load categories on mount
    let load_categories = {
        let categories = categories.clone();
        let loading_categories = loading_categories.clone();
        
        move || {
            loading_categories.set(true);
            
            spawn_local(async move {
                match api::list_categories(None).await {
                    Ok(cats) => {
                        categories.set(cats);
                        loading_categories.set(false);
                    }
                    Err(e) => {
                        error.set(Some(format!("Failed to load categories: {}", e)));
                        loading_categories.set(false);
                    }
                }
            });
        }
    };

    // Set today's date as default
    load_categories();

    // Validate form
    let validate = {
        let amount = amount.clone();
        let category_id = category_id.clone();
        let date = date.clone();
        let amount_error = amount_error.clone();
        let category_error = category_error.clone();
        let date_error = date_error.clone();
        
        move || -> bool {
            let mut is_valid = true;
            
            // Validate amount
            let amount_val = amount.get();
            let parsed: Result<f64, _> = amount_val.parse();
            match parsed {
                Ok(v) if v > 0.0 => {
                    amount_error.set(None);
                }
                Ok(_) => {
                    amount_error.set(Some("Amount must be positive".into()));
                    is_valid = false;
                }
                Err(_) => {
                    amount_error.set(Some("Invalid amount".into()));
                    is_valid = false;
                }
            }
            
            // Validate category
            if category_id.get().is_none() {
                category_error.set(Some("Please select a category".into()));
                is_valid = false;
            } else {
                category_error.set(None);
            }
            
            // Validate date
            let date_val = date.get();
            if date_val.is_empty() {
                date_error.set(Some("Date is required".into()));
                is_valid = false;
            } else {
                date_error.set(None);
            }
            
            is_valid
        }
    };

    // Submit form
    let handle_submit = {
        let transaction_type = transaction_type.clone();
        let amount = amount.clone();
        let currency = currency.clone();
        let category_id = category_id.clone();
        let description = description.clone();
        let label = label.clone();
        let date = date.clone();
        let submitting = submitting.clone();
        let error = error.clone();
        let success = success.clone();
        let validate = validate.clone();
        
        move || {
            // Validate first
            if !validate() {
                return;
            }
            
            submitting.set(true);
            error.set(None);
            
            let amount_val: f64 = amount.get().parse().unwrap_or(0.0);
            let tx_type = transaction_type.get();
            let currency_val = currency.get();
            let cat_id = category_id.get().unwrap_or(0);
            let desc = if description.get().is_empty() { None } else { Some(description.get()) };
            let lbl = if label.get().is_empty() { None } else { Some(label.get()) };
            let date_val = date.get();
            
            spawn_local(async move {
                let payload = CreateTransaction {
                    transaction_type: tx_type,
                    amount: amount_val,
                    currency: currency_val,
                    category_id: cat_id,
                    description: desc,
                    label: lbl,
                    date: date_val,
                };
                
                match api::create_transaction(payload).await {
                    Ok(_) => {
                        success.set(true);
                        submitting.set(false);
                        // Reset form
                        transaction_type.set(TransactionType::Outcome);
                        amount.set(String::new());
                        currency.set(String::from("COP"));
                        category_id.set(None);
                        description.set(String::new());
                        label.set(String::new());
                        // Note: keep the date
                    }
                    Err(e) => {
                        error.set(Some(e));
                        submitting.set(false);
                    }
                }
            });
        }
    };

    view! {
        <div class="transaction-form">
            <header>
                <h2>"New Transaction"</h2>
                <a href="/transactions" class="btn-secondary">"Cancel"</a>
            </header>

            // Success message
            <Show when={move || success.get()}>
                <div class="success-message">
                    <span>"Transaction created successfully!"</span>
                    <button on:click={move |_| success.set(false)}>"×"</button>
                </div>
            </Show>

            // Error display
            <Show when={move || error.get().is_some()}>
                <div class="error-message">
                    <span>{move || error.get().unwrap_or_default()}</span>
                    <button class="btn-close" on:click={move |_| error.set(None)}>"×"</button>
                </div>
            </Show>

            // Loading categories
            <Show when={move || loading_categories.get()}>
                <p class="loading">"Loading categories..."</p>
            </Show>

            // Form
            <Show when={move || !loading_categories.get()}>
                <form on:submit={move |e| {
                    e.prevent_default();
                    handle_submit();
                }}>
                    <div class="form-group">
                        <label for="type">"Type"</label>
                        <select
                            id="type"
                            on:change={move |e| {
                                let val = event_target_value(&e);
                                transaction_type.set(
                                    if val == "income" { TransactionType::Income }
                                    else { TransactionType::Outcome }
                                );
                            }}
                        >
                            <option value="outcome">"Outcome (Expense)"</option>
                            <option value="income">"Income"</option>
                        </select>
                    </div>

                    <div class="form-group">
                        <label for="amount">"Amount"</label>
                        <input
                            id="amount"
                            type="number"
                            step="0.01"
                            min="0.01"
                            placeholder="0.00"
                            prop:value={move || amount.get()}
                            on:input={move |e| amount.set(event_target_value(&e))}
                        />
                        <Show when={move || amount_error.get().is_some()}>
                            <span class="field-error">{move || amount_error.get().unwrap_or_default()}</span>
                        </Show>
                    </div>

                    <div class="form-group">
                        <label for="currency">"Currency"</label>
                        <select
                            id="currency"
                            on:change={move |e| currency.set(event_target_value(&e))}
                        >
                            <option value="COP">"COP - Colombian Peso"</option>
                            <option value="USD">"USD - US Dollar"</option>
                            <option value="EUR">"EUR - Euro"</option>
                            <option value="GBP">"GBP - British Pound"</option>
                        </select>
                    </div>

                    <div class="form-group">
                        <label for="category">"Category"</label>
                        <select
                            id="category"
                            on:change={move |e| {
                                let val = event_target_value(&e);
                                category_id.set(if val.is_empty() { None } else { val.parse().ok() });
                            }}
                        >
                            <option value="">"Select a category"</option>
                            {move || categories.get().into_iter().map(|cat| {
                                view! { <option value={cat.category.id.to_string()}>{cat.path.clone()}</option> }
                            }).collect_view()}
                        </select>
                        <Show when={move || category_error.get().is_some()}>
                            <span class="field-error">{move || category_error.get().unwrap_or_default()}</span>
                        </Show>
                    </div>

                    <div class="form-group">
                        <label for="description">"Description"</label>
                        <input
                            id="description"
                            type="text"
                            placeholder="Optional description"
                            prop:value={move || description.get()}
                            on:input={move |e| description.set(event_target_value(&e))}
                        />
                    </div>

                    <div class="form-group">
                        <label for="label">"Label / Counterparty"</label>
                        <input
                            id="label"
                            type="text"
                            placeholder="e.g., Grocery store, Salary"
                            prop:value={move || label.get()}
                            on:input={move |e| label.set(event_target_value(&e))}
                        />
                    </div>

                    <div class="form-group">
                        <label for="date">"Date"</label>
                        <input
                            id="date"
                            type="date"
                            prop:value={move || date.get()}
                            on:input={move |e| date.set(event_target_value(&e))}
                        />
                        <Show when={move || date_error.get().is_some()}>
                            <span class="field-error">{move || date_error.get().unwrap_or_default()}</span>
                        </Show>
                    </div>

                    <div class="form-actions">
                        <a href="/transactions" class="btn-secondary">"Cancel"</a>
                        <button
                            type="submit"
                            class="btn-primary"
                            disabled={move || submitting.get()}
                        >
                            {move || if submitting.get() { "Saving..." } else { "Save Transaction" }}
                        </button>
                    </div>
                </form>
            </Show>
        </div>
    }
}
