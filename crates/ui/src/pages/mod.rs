//! Pages module

use leptos::prelude::*;

use crate::components::categories::CategoriesManager;
use crate::components::transactions::TransactionsList;
use crate::components::transaction_form::TransactionForm;

/// Home page
#[component]
pub fn HomePage() -> impl IntoView {
    view! {
        <div class="home-page">
            <h1>"Welcome to PACA"</h1>
            <p>"Personal finance management application"</p>
            <div class="quick-links">
                <a href="/categories" class="card">
                    <h3>"Categories"</h3>
                    <p>"Manage your transaction categories"</p>
                </a>
                <a href="/transactions" class="card">
                    <h3>"Transactions"</h3>
                    <p>"View and manage your transactions"</p>
                </a>
            </div>
        </div>
    }
}

/// Categories page
#[component]
pub fn CategoriesPage() -> impl IntoView {
    view! {
        <div class="page">
            <a href="/" class="back-link">"← Back to Home"</a>
            <CategoriesManager />
        </div>
    }
}

/// Transactions page
#[component]
pub fn TransactionsPage() -> impl IntoView {
    view! {
        <div class="page">
            <a href="/" class="back-link">"← Back to Home"</a>
            <TransactionsList />
        </div>
    }
}

/// Transaction form page (for both new and edit)
#[component]
pub fn TransactionFormPage() -> impl IntoView {
    view! {
        <div class="page">
            <a href="/transactions" class="back-link">"← Back to Transactions"</a>
            <TransactionForm />
        </div>
    }
}
