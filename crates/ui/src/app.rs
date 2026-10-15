//! Main PACA App

use leptos::prelude::*;
use crate::pages::categories::CategoriesPage;

#[component]
pub fn App() -> impl IntoView {
    view! {
        <main>
            <a href="/" class="back-link">"← Back to Home"</a>
            <CategoriesPage />
        </main>
    }
}
