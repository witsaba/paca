//! Categories Page

use leptos::prelude::*;
use crate::components::categories::CategoriesManager;

#[component]
pub fn CategoriesPage() -> impl IntoView {
    view! {
        <main>
            <a href="/" class="back-link">"← Back to Home"</a>
            <CategoriesManager />
        </main>
    }
}
