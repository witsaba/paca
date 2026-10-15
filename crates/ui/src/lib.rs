//! PACA UI (Leptos Frontend)

pub mod pages;
pub mod components;

use leptos::prelude::*;
use pages::HomePage;

/// Root app component
#[component]
pub fn App() -> impl IntoView {
    view! {
        <div class="app">
            <nav class="app-nav">
                <a href="/">"PACA"</a>
                <div class="nav-links">
                    <a href="/categories">"Categories"</a>
                    <a href="/transactions">"Transactions"</a>
                </div>
            </nav>
            <main>
                <HomePage />
            </main>
        </div>
    }
}
